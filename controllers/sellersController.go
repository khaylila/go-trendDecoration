package controllers

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gosimple/slug"
	"github.com/khaylila/go-trendDecoration/config"
	"github.com/khaylila/go-trendDecoration/initializers"
	"github.com/khaylila/go-trendDecoration/models"
	"github.com/khaylila/go-trendDecoration/validation"
	"github.com/sethvargo/go-password/password"
	"golang.org/x/crypto/bcrypt"
)

func RegisterSeller(c *fiber.Ctx) error {
	// get the email/pass off req body
	var body struct {
		Email string `json:"email" form:"email"`
		// Password       string `json:"password" form:"password"`
		// RepeatPassword string `json:"repeatPassword" form:"repeatPassword"`
		FirstName    string `json:"firstName" form:"firstName"`
		LastName     string `json:"lastName" form:"lastName"`
		MerchantName string `json:"merchantName" form:"merchantName"`
		PhoneNumber  string `json:"phoneNumber" form:"PhoneNumber"`
		Address      string `json:"address" form:"address"`
	}

	if c.BodyParser(&body) != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "failed to read body.",
		})
	}

	res, err := password.Generate(16, 5, 0, false, false)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "failed to generate password.",
		})
	}

	// hash the password
	hash, err := bcrypt.GenerateFromPassword([]byte(res), 10)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "failed to hash password.",
		})
	}

	// create the user
	// var role models.Role
	// searchRole := 3
	// initializers.DB.Raw("SELECT * FROM roles WHERE id=?", searchRole).Scan(&role)
	// if role.ID == 0 {
	// 	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
	// 		"error": "failed to query roles.",
	// 	})
	// }

	var lastInsertID int
	tx := initializers.DB.Begin()
	result := tx.Raw("INSERT INTO users(created_at, updated_at, deleted_at, first_name, last_name, email, password, is_active, is_banned, message) VALUES (NOW(), NOW(), null, ?, ?, ?, ?, ?, ?, ?) RETURNING id;", body.FirstName, body.LastName, body.Email, string(hash), true, false, "").Scan(&lastInsertID)
	if result.Error != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to save user.",
		})
	}

	// result = tx.Raw("SELECT LAST_INSERT_ID()").Scan(&lastInsertID)
	// if result.Error != nil {
	// 	tx.Rollback()
	// 	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
	// 		"error": "failed to find user.",
	// 	})
	// }

	result = tx.Exec("INSERT INTO user_role (user_id, role_id) VALUES (?, ?);", lastInsertID, 3)
	if result.Error != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to save user role.",
		})
	}

	result = tx.Exec("INSERT INTO merchants(created_at, updated_at, name, address, phone_number, user_id, slug) VALUES (NOW(), NOW(), ?, ?, ?, ?, ?);", body.MerchantName, body.Address, body.PhoneNumber, lastInsertID, slug.Make(body.MerchantName))
	if result.Error != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to save user merchant.",
		})
	}

	statusEmail := config.SendToEmail(body.Email, "Register new reseller", "Akun anda telah berhasil dibuat, berikut informasi akun anda:<br><b>email:<b/> "+body.Email+"<br><b>password:</b> "+res)

	if !statusEmail {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to send email to seller.",
		})
	}

	tx.Commit()

	// respond
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "201", "message": "Success create user with role seller!"}, "application/vnd.api+json")
}

func CreateNewItem(c *fiber.Ctx) error {
	// get data from request
	var body struct {
		Name        string `json:"name" xml:"name" form:"name"`
		Description string `json:"description" xml:"description" form:"description"`
		Qty         uint   `json:"qty" xml:"qty" form:"qty"`
		Images      []byte `json:"images" xml:"images" form:"images"`
	}

	if c.BodyParser(&body) != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "failed to read body.",
		})
	}

	var sliceFiles []string

	// get array image
	if form, err := c.MultipartForm(); err == nil {
		// => *multipart.Form

		// Get all files from "documents" key:
		files := form.File["images"]
		// => []*multipart.FileHeader
		if len(files) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "please upload an image.",
			})
		}

		// Loop through files:
		for _, file := range files {
			fmt.Println(file.Filename, file.Size, file.Header["Content-Type"][0])
			// => "tutorial.pdf" 360641 "application/pdf"
			if !validation.CheckFileMime(file.Header["Content-Type"][0]) || !validation.CheckFileSize(uint64(file.Size), 1) {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "validation error.",
				})
			}

			res, err := password.Generate(16, 16, 0, false, true)
			if err != nil {
				fmt.Println(err.Error())
				fmt.Println("error password")
			}
			// get ext file
			filenameList := strings.Split(file.Filename, ".")
			ext := filenameList[len(filenameList)-1]

			filename := strconv.FormatInt(time.Now().Unix(), 10) + "_" + res + "." + ext
			sliceFiles = append(sliceFiles, filename)

			// Save the files to disk:
			if err := c.SaveFile(file, fmt.Sprintf("./public/image/%s", filename)); err != nil {
				fmt.Println(err.Error())
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "Unable to save image.",
				})
			}
		}
	}

	// save items
	tx := initializers.DB.Begin()
	var merchantId int
	result := tx.Raw("SELECT id FROM merchants WHERE user_id = ?", c.Locals("user").(models.User).ID).Scan(&merchantId)
	if result.Error != nil {
		tx.Rollback()
		if ok := RemoveFile(sliceFiles); !ok {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "unable to find merchants.",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "unable to find merchants.",
		})
	}

	var lastItemsInsertId uint
	result = tx.Raw("INSERT INTO items(created_at, updated_at, deleted_at, name, description, qty, merchant_id, slug) VALUES (NOW(), NOW(), null, ?, ?, ?, ?, ?) RETURNING id;", body.Name, body.Description, body.Qty, merchantId, slug.Make(body.Name)).Scan(&lastItemsInsertId)
	if result.Error != nil {
		tx.Rollback()
		if ok := RemoveFile(sliceFiles); !ok {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "unable to save items.",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "unable to save items.",
		})
	}
	for _, file := range sliceFiles {
		result = tx.Exec("INSERT INTO images VALUES (?, ?);", lastItemsInsertId, file)
		if result.Error != nil {
			tx.Rollback()
			if ok := RemoveFile(sliceFiles); !ok {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "unable to save items.",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "unable to save items.",
			})
		}
	}
	tx.Commit()

	// return success
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "201", "http_code": 201, "message": "Success create new items."}, "application/vnd.api+json")
}

func ListItem(c *fiber.Ctx) error {
	// // get query
	page := c.QueryInt("page", 1)
	max := c.QueryInt("limit", 2)

	merchant := c.Locals("merchant").(models.Merchant)

	offset := ((page - 1) * max)

	var items []models.Items
	result := initializers.DB.Raw("SELECT * FROM items WHERE merchant_id=? AND deleted_at IS null ORDER BY id ASC LIMIT ? OFFSET ?", merchant.ID, max, offset).Scan(&items)
	if result.Error != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Unable to fetch items.",
		})
	}

	for i, item := range items {
		result = initializers.DB.Raw("SELECT items_id, CONCAT(CAST(? AS TEXT), title) AS title FROM images WHERE items_id=?", fmt.Sprintf("%s/img/", c.BaseURL()), item.ID).Scan(&items[i].Image)
		if result.Error != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Unable to fetch image.",
			})
		}
	}

	var countItems uint
	if result = initializers.DB.Raw("SELECT COUNT(id) as countItems FROM items WHERE merchant_id=? AND deleted_at IS NULL", merchant.ID).Scan(&countItems); result.Error != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Unable to count items.",
		})
	}

	lastPage := int(countItems) % max
	if lastPage == 0 {
		lastPage = int(countItems) / max
	} else {
		lastPage = (int(countItems) / max) + 1
	}

	// respond
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"meta": fiber.Map{
			"totalPages": lastPage,
		},
		"data":     items,
		"merchant": c.Locals("merchant"),
		"links": fiber.Map{
			"self":  fmt.Sprintf("%s/seller/items?page=%d&limit=%d", c.BaseURL(), page, max),
			"first": fmt.Sprintf("%s/seller/items?page=%d&limit=%d", c.BaseURL(), 1, max),
			"prev":  fmt.Sprintf("%s/seller/items?page=%d&limit=%d", c.BaseURL(), page-1, max),
			"next":  fmt.Sprintf("%s/seller/items?page=%d&limit=%d", c.BaseURL(), page+1, max),
			"last":  fmt.Sprintf("%s/seller/items?page=%d&limit=%d", c.BaseURL(), lastPage, max),
		},
	}, "application/vnd.api+json")
}

func DetailItem(c *fiber.Ctx) error {
	itemID := c.Params("id")

	// query get detail item
	var item models.Items
	if result := initializers.DB.Raw("SELECT * FROM items WHERE id=? AND deleted_at IS null", itemID).Scan(&item); result.Error != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Unable to find items.",
		})
	}
	if item.ID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Unable to find items.",
		})
	}

	item.Merchant = c.Locals("merchant").(models.Merchant)

	if result := initializers.DB.Raw("SELECT items_id, CONCAT(CAST(? AS TEXT), title) AS title FROM images WHERE items_id=?", fmt.Sprintf("%s/img/", c.BaseURL()), item.ID).Scan(&item.Image); result.Error != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Unable to fetch image.",
		})
	}

	// respond
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": item,
	}, "application/vnd.api+json")
}

func UpdateItem(c *fiber.Ctx) error {
	// get data from request
	var body struct {
		Name        string `json:"name" xml:"name" form:"name"`
		Description string `json:"description" xml:"description" form:"description"`
		Qty         uint   `json:"qty" xml:"qty" form:"qty"`
		Images      []byte `json:"images" xml:"images" form:"images"`
	}

	if c.BodyParser(&body) != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "failed to read body.",
		})
	}

	itemID := c.Params("id")

	// query get detail item
	var resultItem bool
	if result := initializers.DB.Raw("SELECT 1 FROM items WHERE id=? LIMIT 1;", itemID).Scan(&resultItem); result.Error != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Unable to find items.",
		})
	}
	if !resultItem {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Unable to find items.",
		})
	}

	var images []string
	if result := initializers.DB.Raw("SELECT title AS images FROM images WHERE items_id=?", itemID).Scan(&images); result.Error != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Unable to fetch image.",
		})
	}

	var sliceFiles []string
	// get array image
	if form, err := c.MultipartForm(); err == nil {
		// => *multipart.Form

		// Get all files from "documents" key:
		files := form.File["images"]
		// => []*multipart.FileHeader
		if len(files) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "please upload an image.",
			})
		}

		// Loop through files:
		for _, file := range files {
			fmt.Println(file.Filename, file.Size, file.Header["Content-Type"][0])
			// => "tutorial.pdf" 360641 "application/pdf"
			if !validation.CheckFileMime(file.Header["Content-Type"][0]) || !validation.CheckFileSize(uint64(file.Size), 1) {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "validation error.",
				})
			}

			res, err := password.Generate(16, 16, 0, false, true)
			if err != nil {
				fmt.Println(err.Error())
				fmt.Println("error password")
			}
			// get ext file
			filenameList := strings.Split(file.Filename, ".")
			ext := filenameList[len(filenameList)-1]

			filename := strconv.FormatInt(time.Now().Unix(), 10) + "_" + res + "." + ext
			sliceFiles = append(sliceFiles, filename)

			// Save the files to disk:
			if err := c.SaveFile(file, fmt.Sprintf("./public/image/%s", filename)); err != nil {
				fmt.Println(err.Error())
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "Unable to save image.",
				})
			}
		}
	}

	// save items
	tx := initializers.DB.Begin()

	if result := tx.Exec("DELETE FROM images WHERE items_id=?;", itemID); result.Error != nil {
		tx.Rollback()
		if ok := RemoveFile(sliceFiles); !ok {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "unable to save items.",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "unable to save items.",
		})
	}

	result := tx.Exec("UPDATE items SET updated_at=NOW(), name=?, description=?, qty=?, slug=? WHERE id=?;", body.Name, body.Description, body.Qty, slug.Make(body.Name), itemID)
	if result.Error != nil {
		tx.Rollback()
		if ok := RemoveFile(sliceFiles); !ok {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "unable to save items.",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "unable to save items.",
		})
	}
	for _, file := range sliceFiles {
		result = tx.Exec("INSERT INTO images VALUES (?, ?);", itemID, file)
		if result.Error != nil {
			tx.Rollback()
			if ok := RemoveFile(sliceFiles); !ok {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "unable to save items.",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "unable to save items.",
			})
		}
	}
	tx.Commit()

	RemoveFile(images)

	// return success
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": fiber.StatusOK, "http_code": fiber.StatusOK, "message": "Success update items."}, "application/vnd.api+json")
}

func RemoveItem(c *fiber.Ctx) error {
	itemID := c.Params("id")

	// get merchantID
	merchant := c.Locals("merchant").(models.Merchant)

	// query get detail item
	var merchantID uint
	if result := initializers.DB.Raw("SELECT merchant_id FROM items WHERE id=?", itemID).Scan(&merchantID); result.Error != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Unable to find merchant.",
		})
	}
	if merchantID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Unable to find merchant.",
		})
	}

	if merchantID != merchant.ID {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unable to remove items.",
		})
	}

	tx := initializers.DB.Begin()
	if result := tx.Exec("UPDATE items SET deleted_at=NOW() WHERE id=?;", itemID); result.Error != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Unable to remove items.",
		})
	}

	tx.Commit()

	// return success
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": fiber.StatusOK, "http_code": fiber.StatusOK, "message": "Success remove items."}, "application/vnd.api+json")
}

func RemoveFile(files []string) bool {
	for _, file := range files {
		if err := os.Remove(fmt.Sprintf("./public/image/%s", file)); err != nil {
			return false
		}
	}
	return true
}
