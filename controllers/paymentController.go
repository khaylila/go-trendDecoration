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
	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/coreapi"
	"github.com/midtrans/midtrans-go/snap"
)

func Transaction(c *fiber.Ctx) error {
	// prepare
	// insert order to db

	// insert item to db
	// get user
	user := c.Locals("user").(models.User)
	address := "Rumah Kita Sendiri"
	// prepare Data
	invoiceId, err := generateInvNumber()
	fmt.Println(invoiceId)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"Error": "Failed to fetch invoice",
		})
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
	if result := tx.Raw("INSERT INTO project(user_id, address, range_date, status, created_at, updated_at) VALUES (?, ?, ?, ?, NOW(), NOW()) RETURNING id;", user.ID, address, fmt.Sprintf("[%s, %s)", "2024-04-12", "2024-04-15"), "Menunggu Pembayaran").Scan(&projectId); result.Error != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"Error": "Failed to save project",
		})
	}

	// get items data
	var requestItem []models.Items
	if result := tx.Raw("SELECT * FROM items WHERE id IN (1,2);").Scan(&requestItem); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"Error": "Failed to fetch items",
		})
	}

	items := make([]midtrans.ItemDetails, len(requestItem))

	// Iterate over requestItem and populate orderData.Items
	for i, item := range requestItem {

		if result := tx.Exec("INSERT INTO project_item(project_id, item_id, qty, price) VALUES (?, ?, ?, ?);", projectId, item.ID, 2, 20000); result.Error != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"Error": "Failed to save project Item",
			})
		}

		orderData.TransactionDetails.GrossAmt += (int64(2) * 20000)
		items[i] = midtrans.ItemDetails{
			ID:           strconv.FormatUint(uint64(item.ID), 10),
			Name:         item.Name,
			Price:        20000,
			Qty:          2,
			Category:     "Dekorasi Pernikahan",
			MerchantName: "Trend Decoration",
		}
	}

	orderData.Items = &items

	fmt.Println(orderData.TransactionDetails.GrossAmt)

	//
	url, err := config.GenerateSnapURL(orderData)
	if err != nil {
		return c.Status(fiber.StatusGatewayTimeout).JSON(fiber.Map{
			"Error": "Failed to generate snap_url",
		})
	}

	// save invoice
	if result := tx.Exec("INSERT INTO invoice(amount, status, snap_url, id, expiry_time, project_id, created_at) VALUES (?, ?, ?, ?, ?, ?, NOW());", orderData.TransactionDetails.GrossAmt, "pending", url, invoiceId, time.Now().Add(time.Minute*5), projectId); result.Error != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"Error": "Failed to save invoice",
		})
	}

	tx.Commit()

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"snap_url": url,
	})
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
	return "INV-" + time.Now().Format("20060102") + "-" + fmt.Sprintf("%04d", countCreatedAt+3), nil
}
