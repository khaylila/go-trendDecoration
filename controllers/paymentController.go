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
		DateStart    string `json:"dateStart" xml:"date_start" form:"dateStart"  validate:"required,min=10"`
		DateEnd      string `json:"dateEnd" xml:"date_end" form:"dateEnd"  validate:"required,min=10"`
		Qty          uint   `json:"qty" xml:"qty" form:"qty" validate:"required,numeric,gte=1"`
		Address      string `json:"address" xml:"address" form:"address" validate:"required,min=10,max=128"`
		MerchantSlug string `json:"merchantSlug" xml:"merchantSlug" form:"merchantSlug" validate:"required,max=256"`
		ItemSlug     string `json:"itemSlug" xml:"itemSlug" form:"itemSlug" validate:"required,max=256"`
	}

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

	errors := validation.ReturnValidation(body)

	if len(errors) != 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":   "400",
			"status": "BAD_REQUEST",
			"errors": errors,
		})
	}

	// validate datestart<=dateend
	dateStart, err := time.Parse("2006-01-02", body.DateStart)
	if err != nil {
		errors["datestart"] = err.Error()
	}
	dateEnd, err := time.Parse("2006-01-02", body.DateStart)
	if err != nil {
		errors["dateend"] = err.Error()
	}

	if dateStart.Unix() > dateEnd.Unix() {
		errors["datestart"] = "Tanggal mulai harus lebih kecil dari tanggal selesai."
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
	var projects []struct {
		RentDate   time.Time
		Available  bool
		ItemQty    int
		ProjectQty int
	}
	if result := initializers.DB.Raw("SELECT date_series::date AS rent_date, (items.qty > COALESCE(SUM (project_item.qty), 0))::boolean AS available, items.qty AS item_qty, SUM (project_item.qty) AS project_qty FROM project p JOIN project_item on p.id = project_item.project_id JOIN items ON project_item.item_id = items.id JOIN generate_series(lower(p.range_date), upper(p.range_date), '1 day'::interval) AS date_series  ON date_series::date >= ?::date AND date_series::date <= ?::date  WHERE project_item.item_id = ? GROUP BY date_series::date, items.qty ORDER BY date_series::date;", body.DateStart, body.DateEnd, item.ID).Scan(&projects); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"error": fiber.Map{
				"message": "failed to query item.",
			},
		}, "application/vnd.api+json")
	}
	itemAvailable := 0
	for _, project := range projects {
		if !project.Available {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"code":   "200",
				"status": "OK",
				"data": fiber.Map{
					"item_qty": 0,
				},
			}, "application/vnd.api+json")
		} else {
			if itemAvailable == 0 || itemAvailable > (project.ItemQty-project.ProjectQty) {
				itemAvailable = project.ItemQty - project.ProjectQty
			}
		}
	}

	if len(projects) == 0 {
		itemAvailable = int(item.Qty)
	}

	if body.Qty > uint(itemAvailable) {
		errors["qty"] = "Maksimal pemesanan item adalah " + strconv.FormatInt(int64(itemAvailable), 10)
	}

	if len(errors) != 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":   "400",
			"status": "BAD_REQUEST",
			"errors": errors,
		})
	}

	// begin payment
	// get user
	user := c.Locals("user").(models.User)

	// prepare Data
	paymentID, err := generateNumber("payment")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": err,
			},
		}, "application/vnd.api+json")
	}
	orderData := snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  paymentID,
			GrossAmt: 0,
		},
		CustomerDetail: &midtrans.CustomerDetails{
			FName: user.FirstName,
			LName: user.LastName,
			Email: user.Email,
		},
	}

	invoiceID, errInv := generateNumber("inv")
	if errInv != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": errInv,
			},
		}, "application/vnd.api+json")
	}

	// insert into table project
	tx := initializers.DB.Begin()

	items := make([]midtrans.ItemDetails, 1)
	orderData.TransactionDetails.GrossAmt += (int64(body.Qty) * int64(item.Price))
	items[0] = midtrans.ItemDetails{
		ID:           strconv.FormatUint(uint64(item.ID), 10),
		Name:         item.Name,
		Price:        int64(item.Price),
		Qty:          int32(body.Qty),
		MerchantName: merchant.Name,
	}
	// insert into midtrans request
	orderData.Items = &items

	// send request to midtrans
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

	// save payment
	if result := tx.Exec("INSERT INTO payment(amount, status, snap_url, id, expiry_time, created_at) VALUES (?, ?, ?, ?, ?, NOW());", orderData.TransactionDetails.GrossAmt, "pending", url, paymentID, time.Now().Add(time.Minute*5)); result.Error != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "Failed to save invoice",
			},
		}, "application/vnd.api+json")
	}

	log.Print(item)

	var projectId uint
	if result := tx.Raw("INSERT INTO project(user_id, address, range_date, status, created_at, updated_at, invoice, payment_id, merchant_id) VALUES (?, ?, ?, ?, NOW(), NOW(), ?, ?, ?) RETURNING id;", user.ID, body.Address, fmt.Sprintf("[%s, %s)", body.DateStart, body.DateEnd), "Menunggu Pembayaran", invoiceID, paymentID, item.MerchantID).Scan(&projectId); result.Error != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "Failed to save project",
			},
		}, "application/vnd.api+json")
	}

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

	if result := tx.Exec("INSERT INTO project_timeline (message,msg_from,project_id,created_at) VALUES (?,?,?,now())", "Pesanan diterima, menunggu pembayaran,", 4, projectId); result.Error != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "Failed to save project Item",
			},
		}, "application/vnd.api+json")
	}

	// commit transaction
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
		// return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		// 	"error": "failed to read body.",
		// })
		return c.SendStatus(fiber.StatusBadRequest)
	}

	var client coreapi.Client
	envi := midtrans.Sandbox
	if os.Getenv("MIDTRANS.ENV") == "production" {
		envi = midtrans.Production
	}
	client.New(os.Getenv("MIDTRANS.SERVERKEY"), envi)
	transactionStatusResp, e := client.CheckTransaction(body.OrderID)
	if e != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": e.GetMessage()})
	} else {
		if transactionStatusResp != nil {
			tx := initializers.DB.Begin()
			// 5. Do set transaction status based on response from check transaction status
			if transactionStatusResp.TransactionStatus == "capture" {
				if transactionStatusResp.FraudStatus == "challenge" {
					// TODO set transaction status on your database to 'challenge'
					// e.g: 'Payment status challenged. Please take action on your Merchant Administration Portal
				} else if transactionStatusResp.FraudStatus == "accept" {
					// TODO set transaction status on your database to 'success'
					if result := tx.Exec("UPDATE payment SET status=? WHERE id=?;", transactionStatusResp.TransactionStatus, body.OrderID); result.Error != nil {
						tx.Rollback()
						log.Println("Unable to update payment.")
					}

					if result := tx.Exec("UPDATE project SET status=?, updated_at=NOW() WHERE payment_id=?;", "Menunggu konfirmasi vendor", body.OrderID); result.Error != nil {
						tx.Rollback()
						log.Println("Unable to update project.")
					}
				}
			} else if transactionStatusResp.TransactionStatus == "settlement" {
				var projectID uint
				// TODO set transaction status on your databaase to 'success'
				settlementTime, _ := time.Parse("2006-01-02 15:04:05", transactionStatusResp.SettlementTime)
				if result := tx.Exec("UPDATE payment SET status=?, settlement_time=? WHERE id=?;", transactionStatusResp.TransactionStatus, settlementTime, body.OrderID); result.Error != nil {
					tx.Rollback()
					log.Println("Unable to update payment.")
				}

				if result := tx.Raw("UPDATE project SET status=?, updated_at=NOW() WHERE payment_id=? RETURNING id;", "Menunggu konfirmasi vendor", body.OrderID).Scan(&projectID); result.Error != nil {
					tx.Rollback()
					log.Println("Unable to update project.")
				}

				if result := tx.Exec("INSERT INTO project_timeline (message,msg_from,project_id,created_at) VALUES (?,?,?,now())", "Pembayaran Diterima,", 4, projectID); result.Error != nil {
					tx.Rollback()
					log.Println("Unable to update project timeline.")
				}
				// 1 pembeli
				// 2 penjual
				// 3 admin
				// 4 sistem

			} else if transactionStatusResp.TransactionStatus == "deny" {
				// TODO you can ignore 'deny', because most of the time it allows payment retries
				// and later can become success
			} else if transactionStatusResp.TransactionStatus == "cancel" || transactionStatusResp.TransactionStatus == "expire" {
				// TODO set transaction status on your databaase to 'failure'
				var projectID uint
				status := "pesanan kadaluarsa"
				if transactionStatusResp.TransactionStatus == "cancel" {
					status = "pesanan dibatalkan"
				}
				if result := tx.Exec("UPDATE payment SET status=? WHERE id=?;", transactionStatusResp.TransactionStatus, body.OrderID); result.Error != nil {
					tx.Rollback()
					log.Println("Unable to update payment.")
				}

				if result := tx.Raw("UPDATE project SET status=?, updated_at=NOW() WHERE payment_id=? RETURNING id;", status, body.OrderID).Scan(&projectID); result.Error != nil {
					tx.Rollback()
					log.Println("Unable to update project.")
				}

				if result := tx.Exec("INSERT INTO project_timeline (message,msg_from,project_id,created_at) VALUES (?,?,?,now())", "Pesanan dibatalkan oleh sistem", 4, projectID); result.Error != nil {
					tx.Rollback()
					log.Println("Unable to update project timeline.")
				}
			} else if transactionStatusResp.TransactionStatus == "pending" {
				// TODO set transaction status on your databaase to 'pending' / waiting payment
				expTime, _ := time.Parse("2006-01-02 15:04:05", transactionStatusResp.ExpiryTime)
				if transactionStatusResp.PaymentType == "bank_transfer" {
					if result := tx.Exec("UPDATE payment SET status=?, expiry_time=?, payment_type=?, bank=?, va_number=? WHERE id=?;", transactionStatusResp.TransactionStatus, expTime, transactionStatusResp.PaymentType, transactionStatusResp.VaNumbers[0].Bank, transactionStatusResp.VaNumbers[0].VANumber, body.OrderID); result.Error != nil {
						tx.Rollback()
						log.Println("Unable to update payment.")
					}
				} else {
					if result := tx.Exec("UPDATE payment SET status=?, expiry_time=?, payment_type=? WHERE id=?;", transactionStatusResp.TransactionStatus, expTime, transactionStatusResp.PaymentType, body.OrderID); result.Error != nil {
						tx.Rollback()
						log.Println("Unable to update payment.")
					}
				}
			}
			tx.Commit()
		}
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "Ok"}, "application/json")
}

func generateNumber(typeNumber string) (string, error) {
	var countCreatedAt uint
	var tag string
	if typeNumber == "payment" {
		tag = "ORDER"
		if result := initializers.DB.Raw("SELECT COUNT(id) AS count_created_at FROM payment WHERE DATE(created_at) = CURRENT_DATE;").Scan(&countCreatedAt); result.Error != nil {
			return "", errors.New("unable to fetch payment")
		}
	} else if typeNumber == "inv" {
		tag = "INV"
		if result := initializers.DB.Raw("SELECT COUNT(id) AS count_created_at FROM payment WHERE DATE(created_at) = CURRENT_DATE;").Scan(&countCreatedAt); result.Error != nil {
			return "", errors.New("unable to fetch invoice")
		}
	} else {
		return "", errors.New("type not found")
	}
	return tag + "-" + time.Now().Format("20060102") + "-" + fmt.Sprintf("%04d", countCreatedAt+1), nil
}
