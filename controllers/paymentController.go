package controllers

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/khaylila/go-trendDecoration/config"
	"github.com/khaylila/go-trendDecoration/initializers"
	"github.com/khaylila/go-trendDecoration/models"
	"github.com/khaylila/go-trendDecoration/validation"
	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/coreapi"
	"github.com/midtrans/midtrans-go/snap"
)

func Transaction(c *fiber.Ctx) error {
	// get data from request
	var body struct {
		DateStart    string `json:"dateStart" xml:"dateStart" form:"dateStart"  validate:"required,min=10,max=10"`
		DateEnd      string `json:"dateEnd" xml:"dateEnd" form:"dateEnd"  validate:"required,min=10,max=10"`
		Qty          uint   `json:"qty" xml:"qty" form:"qty" validate:"required,numeric,gte=1"`
		Address      string `json:"address" xml:"address" form:"address" validate:"required,min=10,max=128"`
		MerchantSlug string `json:"merchantSlug" xml:"merchantSlug" form:"merchantSlug" validate:"required,max=256"`
		ItemSlug     string `json:"itemSlug" xml:"itemSlug" form:"itemSlug" validate:"required,max=256"`
	}

	if c.BodyParser(&body) != nil {
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":   "400",
				"status": "BAD_REQUEST",
				"error": fiber.Map{
					"err":     err.Error(),
					"message": "failed to read body.",
				},
			}, "application/vnd.api+json")
		}
	}

	errors := validation.ReturnValidation(body)

	if len(errors) != 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":   "400",
			"status": "BAD_REQUEST",
			"errors": errors,
		})
	}
	// prepare
	// insert order to db

	// insert item to db
	// get user
	user := c.Locals("user").(models.User)

	// prepare Data
	invoiceId, err := generateInvNumber()
	fmt.Println(invoiceId)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "Failed to fetch invoice",
			},
		}, "application/vnd.api+json")
	}
	orderData := snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  invoiceId,
			GrossAmt: 0,
		},
		CustomerDetail: &midtrans.CustomerDetails{
			FName: user.FirstName,
			LName: user.LastName,
			Email: user.Email,
		},
	}

	// insert into table project
	tx := initializers.DB.Begin()
	var projectId uint
	if result := tx.Raw("INSERT INTO project(user_id, address, range_date, status, created_at, updated_at) VALUES (?, ?, ?, ?, NOW(), NOW()) RETURNING id;", user.ID, body.Address, fmt.Sprintf("[%s, %s)", body.DateStart, body.DateEnd), "Menunggu Pembayaran").Scan(&projectId); result.Error != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "Failed to save project",
			},
		}, "application/vnd.api+json")
	}

	// query get detail merchant
	var merchant models.Merchant
	if result := initializers.DB.Raw("SELECT * FROM merchants WHERE slug=? AND deleted_at IS null", body.MerchantSlug).Scan(&merchant); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "Unable to fetch merchant.",
			},
		}, "application/vnd.api+json")
	}
	if merchant.ID == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"code":   "404",
			"status": "NOT_FOUND",
			"erorr": fiber.Map{
				"message": "merchant not found.",
			},
		}, "application/vnd.api+json")
	}

	// query get detail item
	var item models.Items
	if result := initializers.DB.Raw("SELECT * FROM items WHERE merchant_id = ? AND slug = ? AND deleted_at IS null", merchant.ID, body.ItemSlug).Scan(&item); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "Unable to fetch items.",
			},
		}, "application/vnd.api+json")
	}
	if item.ID == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"code":   "404",
			"status": "NOT_FOUND",
			"erorr": fiber.Map{
				"message": "items not found.",
			},
		}, "application/vnd.api+json")
	}

	items := make([]midtrans.ItemDetails, 1)

	// Iterate over requestItem and populate orderData.Items

	if result := tx.Exec("INSERT INTO project_item(project_id, item_id, qty, price) VALUES (?, ?, ?, ?);", projectId, item.ID, body.Qty, item.Price); result.Error != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "Failed to save project Item",
			},
		}, "application/vnd.api+json")
	}

	orderData.TransactionDetails.GrossAmt += (int64(body.Qty) * int64(item.Price))
	items[0] = midtrans.ItemDetails{
		ID:           strconv.FormatUint(uint64(item.ID), 10),
		Name:         item.Name,
		Price:        int64(item.Price),
		Qty:          int32(body.Qty),
		MerchantName: merchant.Name,
	}

	orderData.Items = &items

	//
	url, err := config.GenerateSnapURL(orderData)
	if err != nil {
		return c.Status(fiber.StatusGatewayTimeout).JSON(fiber.Map{
			"code":   "504",
			"status": "GATEWAY_TIMEOUT",
			"erorr": fiber.Map{
				"message": "Failed to generate snap_url",
			},
		}, "application/vnd.api+json")
	}

	// save invoice
	if result := tx.Exec("INSERT INTO invoice(amount, status, snap_url, id, expiry_time, project_id, created_at) VALUES (?, ?, ?, ?, ?, ?, NOW());", orderData.TransactionDetails.GrossAmt, "pending", url, invoiceId, time.Now().Add(time.Minute*5), projectId); result.Error != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "Failed to save invoice",
			},
		}, "application/vnd.api+json")
	}

	tx.Commit()

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":   "200",
		"status": "OK",
		"data": fiber.Map{
			"snap_url": url,
		},
	}, "application/vnd.api+json")
}

func VerifyPayment(c *fiber.Ctx) error {
	var body struct {
		OrderID string `json:"order_id" xml:"order_id" form:"order_id"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "failed to read body.",
		})
	}

	var client coreapi.Client
	envi := midtrans.Sandbox
	if os.Getenv("MIDTRANS.ENV") == "production" {
		envi = midtrans.Production
	}
	client.New(os.Getenv("MIDTRANS.SERVERKEY"), envi)
	// 4. Check transaction to Midtrans with param orderId
	fmt.Println(body.OrderID)
	transactionStatusResp, e := client.CheckTransaction(body.OrderID)
	fmt.Println(transactionStatusResp)
	if e != nil {
		fmt.Println("gagal")
		fmt.Println(e)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": e.GetMessage()})
	} else {
		fmt.Println("sukses")
		if transactionStatusResp != nil {
			tx := initializers.DB.Begin()
			// 5. Do set transaction status based on response from check transaction status
			if transactionStatusResp.TransactionStatus == "capture" {
				if transactionStatusResp.FraudStatus == "challenge" {
					// TODO set transaction status on your database to 'challenge'
					// e.g: 'Payment status challenged. Please take action on your Merchant Administration Portal
				} else if transactionStatusResp.FraudStatus == "accept" {
					// TODO set transaction status on your database to 'success'
					var projectId uint
					if result := tx.Raw("UPDATE invoice SET status=? WHERE id=? RETURNING project_id;", transactionStatusResp.TransactionStatus, body.OrderID).Scan(&projectId); result.Error != nil {
						tx.Rollback()
						log.Println("Unable to update invoice.")
					}

					if result := tx.Exec("UPDATE project SET payment_status=?, status=?, updated_at=NOW() WHERE id=?;", 1, "Menunggu konfirmasi vendor", projectId); result.Error != nil {
						tx.Rollback()
						log.Println("Unable to update project.")
					}
				}
			} else if transactionStatusResp.TransactionStatus == "settlement" {
				// TODO set transaction status on your databaase to 'success'
				var projectId uint
				if result := tx.Raw("UPDATE invoice SET status=? WHERE id=? RETURNING project_id;", transactionStatusResp.TransactionStatus, body.OrderID).Scan(&projectId); result.Error != nil {
					tx.Rollback()
					log.Println("Unable to update invoice.")
				}

				if result := tx.Exec("UPDATE project SET payment_status=?, status=?, updated_at=NOW() WHERE id=?;", 1, "Menunggu konfirmasi vendor", projectId); result.Error != nil {
					tx.Rollback()
					log.Println("Unable to update project.")
				}
			} else if transactionStatusResp.TransactionStatus == "deny" {
				// TODO you can ignore 'deny', because most of the time it allows payment retries
				// and later can become success
			} else if transactionStatusResp.TransactionStatus == "cancel" || transactionStatusResp.TransactionStatus == "expire" {
				// TODO set transaction status on your databaase to 'failure'
				var projectId uint
				if result := tx.Raw("UPDATE invoice SET status=? WHERE id=? RETURNING project_id;", transactionStatusResp.TransactionStatus, body.OrderID).Scan(&projectId); result.Error != nil {
					tx.Rollback()
					log.Println("Unable to update invoice.")
				}

				if result := tx.Exec("UPDATE project SET payment_status=?, status=?, updated_at=NOW() WHERE id=?;", 0, "pesanan dibatalkan", projectId); result.Error != nil {
					tx.Rollback()
					log.Println("Unable to update project.")
				}
			} else if transactionStatusResp.TransactionStatus == "pending" {
				// TODO set transaction status on your databaase to 'pending' / waiting payment
				expTime, _ := time.Parse("2006-01-02 15:04:05", transactionStatusResp.ExpiryTime)
				if transactionStatusResp.PaymentType == "bank_transfer" {
					if result := tx.Exec("UPDATE invoice SET status=?, expiry_time=?, payment_type=?, bank=?, va_number=? WHERE id=?;", transactionStatusResp.TransactionStatus, expTime, transactionStatusResp.PaymentType, transactionStatusResp.VaNumbers[0].Bank, transactionStatusResp.VaNumbers[0].VANumber, body.OrderID); result.Error != nil {
						tx.Rollback()
						log.Println("Unable to update invoice.")
					}
				} else {
					if result := tx.Exec("UPDATE invoice SET status=?, expiry_time=?, payment_type=? WHERE id=?;", transactionStatusResp.TransactionStatus, expTime, transactionStatusResp.PaymentType, body.OrderID); result.Error != nil {
						tx.Rollback()
						log.Println("Unable to update invoice.")
					}
				}
			}
			tx.Commit()
		}
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "Ok"}, "application/json")
}

func generateInvNumber() (string, error) {
	var countCreatedAt uint
	if result := initializers.DB.Raw("SELECT COUNT(id) AS count_created_at FROM invoice WHERE DATE(created_at) = CURRENT_DATE;").Scan(&countCreatedAt); result.Error != nil {
		return "", errors.New("unable to fetch invoice")
	}
	return "INV-" + time.Now().Format("20060102") + "-" + fmt.Sprintf("%04d", countCreatedAt+1), nil
}
