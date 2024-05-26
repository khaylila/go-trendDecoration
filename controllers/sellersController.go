package controllers

import (
	"fmt"
	"log"
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
	"github.com/lib/pq"
	"github.com/sethvargo/go-password/password"
	"golang.org/x/crypto/bcrypt"
)

func ListSeller(c *fiber.Ctx) error {
	// get query
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	filter := c.Query("filter", "name")
	search := strings.ToLower(c.Query("search", ""))

	offset := ((page - 1) * limit)

	var merchants []models.Merchant
	result := initializers.DB.Raw("SELECT *, CONCAT(CAST(? AS TEXT), avatar) AS avatar FROM merchants WHERE lower("+filter+") LIKE ? AND deleted_at IS null ORDER BY id ASC LIMIT ? OFFSET ?", fmt.Sprintf("%s/img/", c.BaseURL()), fmt.Sprintf("%%%s%%", search), limit, offset).Scan(&merchants)
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"error": fiber.Map{
				"message": "Unable to fetch merchants.",
			},
		})
	}

	log.Printf("%s", filter)
	log.Printf("%%%s%%", search)

	for i, merchant := range merchants {
		result = initializers.DB.Raw("SELECT users.id, users.first_name, users.last_name, users.updated_at FROM users WHERE id=?", merchant.UserID).Scan(&merchants[i].User)
		if result.Error != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code":   "500",
				"status": "INTERNAL_SERVER_ERROR",
				"error": fiber.Map{
					"message": "Unable to fetch user data.",
				},
			})
		}
	}

	var countMerchants uint
	if result = initializers.DB.Raw("SELECT COUNT(id) as countMerchants FROM merchants WHERE LOWER("+filter+") LIKE ? AND deleted_at IS NULL", fmt.Sprintf("%%%s%%", search)).Scan(&countMerchants); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"error": fiber.Map{
				"message": "Unable to count merchants.",
			},
		})
	}

	lastPage := int(countMerchants) % limit
	if lastPage == 0 {
		lastPage = int(countMerchants) / limit
	} else {
		lastPage = (int(countMerchants) / limit) + 1
	}

	// respond
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":   "200",
		"status": "OK",
		"data":   merchants,
		"page": fiber.Map{
			"limit":     limit,
			"total":     countMerchants,
			"totalPage": lastPage,
			"current":   page,
		},
	}, "application/vnd.api+json")
}

func DetailSeller(c *fiber.Ctx) error {
	merchantID := c.Params("id")

	// query get detail item
	var merchant models.Merchant
	if result := initializers.DB.Raw("SELECT *, CONCAT(CAST(? AS TEXT), avatar) AS avatar FROM merchants WHERE id=? AND deleted_at IS null", fmt.Sprintf("%s/img/", c.BaseURL()), merchantID).Scan(&merchant); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"error": fiber.Map{
				"message": "Unable to fetch merchant.",
			},
		})
	}
	if merchant.ID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":   "400",
			"status": "BAD_REQUEST",
			"error": fiber.Map{
				"message": "Unable to find merchant.",
			},
		})
	}

	if result := initializers.DB.Raw("SELECT users.id, users.first_name, users.last_name, users.updated_at, created_at FROM users WHERE id=?", merchant.UserID).Scan(&merchant.User); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"error": fiber.Map{
				"message": "Unable to fetch user data.",
			},
		})
	}

	// respond
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":   "200",
		"status": "OK",
		"data":   merchant,
	}, "application/vnd.api+json")
}

func RegisterSeller(c *fiber.Ctx) error {
	// get the email/pass off req body
	type SellerRegister struct {
		Email        string `json:"email" form:"email" validate:"required,email,min=6,max=64"`
		FirstName    string `json:"firstName" form:"firstName" validate:"required,min=2,max=32"`
		LastName     string `json:"lastName" form:"lastName" validate:"required,min=2,max=32"`
		MerchantName string `json:"merchantName" form:"merchantName" validate:"required,min=6,max=64"`
		PhoneNumber  string `json:"phoneNumber" form:"phoneNumber" validate:"required,number,min=11,max=20"`
		Address      string `json:"address" form:"address" validate:"required,min=6,max=128"`
	}

	body := new(SellerRegister)
	if c.BodyParser(&body) != nil {
		log.Println(c.BodyParser(&body))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"error": fiber.Map{
				"message": "failed to read body.",
			},
		})
	}

	errors := validation.ReturnValidation(body)

	// check unique email
	var resultEmail uint
	if result := initializers.DB.Raw("SELECT 1 FROM users WHERE email=?;", body.Email).Scan(&resultEmail); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "failed to query user.",
			},
		}, "application/vnd.api+json")
	}

	if resultEmail == 1 {
		errors["Email"] = "Email sudah pernah ditambahkan sebelumnya."
	}

	// check unique phone
	var resultPhone uint
	if result := initializers.DB.Raw("SELECT 1 FROM merchants WHERE phone_number=CAST(? AS TEXT);", body.PhoneNumber).Scan(&resultPhone); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "failed to query merchant.",
			},
		}, "application/vnd.api+json")
	}

	if resultPhone == 1 {
		errors["PhoneNumber"] = "Nomor sudah pernah ditambahkan sebelumnya."
	}

	// check unique phone
	var resultMerchant uint
	if result := initializers.DB.Raw("SELECT 1 FROM merchants WHERE name=?;", body.MerchantName).Scan(&resultMerchant); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "failed to query merchant.",
			},
		}, "application/vnd.api+json")
	}

	if resultMerchant == 1 {
		errors["MerchantName"] = "Nama Merchant sudah pernah ditambahkan sebelumnya."
	}

	if len(errors) != 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":   "400",
			"status": "BAD_REQUEST",
			"errors": errors,
		})
	}

	res, err := password.Generate(16, 5, 0, false, false)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "failed to generate password.",
			},
		}, "application/vnd.api+json")
	}

	// hash the password
	hash, err := bcrypt.GenerateFromPassword([]byte(res), 10)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "failed to hash password.",
			},
		}, "application/vnd.api+json")
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
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "failed to save user.",
			},
		}, "application/vnd.api+json")
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
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "failed to save user role.",
			},
		}, "application/vnd.api+json")
	}

	result = tx.Exec("INSERT INTO merchants(created_at, updated_at, name, address, phone_number, user_id, slug) VALUES (NOW(), NOW(), ?, ?, ?, ?, ?);", body.MerchantName, body.Address, body.PhoneNumber, lastInsertID, slug.Make(body.MerchantName))
	if result.Error != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "failed to save user merchant.",
			},
		}, "application/vnd.api+json")
	}

	statusEmail := config.SendToEmail(body.Email, "Register new reseller", "Akun anda telah berhasil dibuat, berikut informasi akun anda:<br><b>email:<b/> "+body.Email+"<br><b>password:</b> "+res)

	if !statusEmail {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "failed to send email to seller.",
			},
		}, "application/vnd.api+json")
	}

	tx.Commit()

	// respond
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"code":   "201",
		"status": "CREATED",
		"data": fiber.Map{
			"message": "Berhasil menambahkan user merchant.",
		},
	}, "application/vnd.api+json")
}

func UpdateSeller(c *fiber.Ctx) error {
	// get the email/pass off req body
	type sellerRegister struct {
		ID           uint   `json:"id" form:"id" validate:"required"`
		FirstName    string `json:"firstName" form:"firstName" validate:"required,min=2,max=32"`
		LastName     string `json:"lastName" form:"lastName" validate:"required,min=2,max=32"`
		MerchantName string `json:"merchantName" form:"merchantName" validate:"required,min=6,max=64"`
		PhoneNumber  string `json:"phoneNumber" form:"phoneNumber" validate:"required,number,min=11,max=20"`
		Address      string `json:"address" form:"address" validate:"required,min=6,max=128"`
		// Avatar       byte `json:"avatar" form:"avatar"`
	}

	var body sellerRegister
	if err := c.BodyParser(&body); err != nil {
		log.Println(err.Error())
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"error": fiber.Map{
				"message": "failed to read body.",
			},
		})
	}

	log.Println(body)
	errors := validation.ReturnValidation(body)

	// check unique phone
	var resultPhone uint
	if result := initializers.DB.Raw("SELECT 1 FROM merchants WHERE phone_number=? AND id!=?;", body.PhoneNumber, body.ID).Scan(&resultPhone); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "failed to query merchant.",
			},
		}, "application/vnd.api+json")
	}

	if resultPhone == 1 {
		errors["PhoneNumber"] = "Nomor sudah pernah ditambahkan sebelumnya."
	}

	// check unique phone
	var resultMerchant uint
	if result := initializers.DB.Raw("SELECT 1 FROM merchants WHERE name=? AND id!=?;", body.MerchantName, body.ID).Scan(&resultMerchant); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "failed to query merchant.",
			},
		}, "application/vnd.api+json")
	}

	if resultMerchant == 1 {
		errors["MerchantName"] = "Nama Merchant sudah pernah ditambahkan sebelumnya."
	}

	if len(errors) != 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":   "400",
			"status": "BAD_REQUEST",
			"errors": errors,
		})
	}

	var structMerch struct {
		UserID      uint
		MerchAvatar string
		UserAvatar  string
	}

	// getUserLastMerchantPict
	if result := initializers.DB.Raw("SELECT avatar AS merch_avatar, user_id FROM merchants WHERE id=?;", body.ID).Scan(&structMerch); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "failed to query user merchant.",
			},
		}, "application/vnd.api+json")
	}

	// getUserLastPict
	if result := initializers.DB.Raw("SELECT avatar AS user_avatar FROM users WHERE id=?;", structMerch.UserID).Scan(&structMerch); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "failed to query user.",
			},
		}, "application/vnd.api+json")
	}

	// get image from request
	file, err := c.FormFile("avatar")
	if file != nil {
		if err != nil {
			log.Println(err.Error())
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code":   "500",
				"status": "INTERNAL_SERVER_ERROR",
				"error": fiber.Map{
					"message": err.Error(),
				},
			})
		}

		fmt.Println(file.Filename, file.Size, file.Header["Content-Type"][0])
		// => "tutorial.pdf" 360641 "application/pdf"
		log.Println(validation.CheckFileMime(file.Header["Content-Type"][0]))
		log.Println(validation.CheckFileSize(uint64(file.Size), 1))
		if !validation.CheckFileMime(file.Header["Content-Type"][0]) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":   "400",
				"status": "BAD_REQUEST",
				"errors": fiber.Map{
					"avatar": "Format gambar tidak sesuai.",
				},
			})
		}

		if !validation.CheckFileSize(uint64(file.Size), 1) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":   "400",
				"status": "BAD_REQUEST",
				"errors": fiber.Map{
					"avatar": "Ukuran gambar terlalu besar.",
				},
			})
		}

		res, err := password.Generate(16, 16, 0, false, true)
		if err != nil {
			log.Println(err.Error())
			fmt.Println("error password")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code":   "500",
				"status": "INTERNAL_SERVER_ERROR",
				"error":  "unable to create random name.",
			})
		}
		// get ext file
		filenameList := strings.Split(file.Filename, ".")
		ext := filenameList[len(filenameList)-1]

		filename := strconv.FormatInt(time.Now().Unix(), 10) + "_" + res + "." + ext

		// Save the files to disk:
		if err := c.SaveFile(file, GetDir(fmt.Sprintf("/public/image/%s", filename))); err != nil {
			fmt.Println(err.Error())
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code":   "500",
				"status": "INTERNAL_SERVER_ERROR",
				"erorr": fiber.Map{
					"message": "Unable to save image.",
				},
			})
		}

		tx := initializers.DB.Begin()
		if result := tx.Exec("UPDATE merchants SET updated_at=NOW(), name=?, address=?, phone_number=?, slug=?, avatar=? WHERE id=?;", body.MerchantName, body.Address, body.PhoneNumber, slug.Make(body.MerchantName), filename, body.ID); result.Error != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code":   "500",
				"status": "INTERNAL_SERVER_ERROR",
				"erorr": fiber.Map{
					"message": "failed to save user merchant.",
				},
			}, "application/vnd.api+json")
		}

		if result := tx.Exec("UPDATE users SET updated_at=NOW(), first_name=?, last_name=?, avatar=? WHERE id=?;", body.FirstName, body.LastName, filename, structMerch.UserID); result.Error != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code":   "500",
				"status": "INTERNAL_SERVER_ERROR",
				"erorr": fiber.Map{
					"message": "failed to update user.",
				},
			}, "application/vnd.api+json")
		}

		// check userAvatar
		if structMerch.UserAvatar != "user.png" {
			if ok := RemoveFile([]string{structMerch.UserAvatar}); !ok {
				log.Println("gagal menghapus gambar user")
			}
		}

		if structMerch.MerchAvatar != "logo.png" {
			if ok := RemoveFile([]string{structMerch.MerchAvatar}); !ok {
				log.Println("gagal menghapus gambar merchant")
			}
		}
		tx.Commit()
	} else {
		tx := initializers.DB.Begin()
		if result := tx.Exec("UPDATE merchants SET updated_at=NOW(), name=?, address=?, phone_number=?, slug=? WHERE id=?;", body.MerchantName, body.Address, body.PhoneNumber, slug.Make(body.MerchantName), body.ID); result.Error != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code":   "500",
				"status": "INTERNAL_SERVER_ERROR",
				"erorr": fiber.Map{
					"message": "failed to save user merchant.",
				},
			}, "application/vnd.api+json")
		}

		if result := tx.Exec("UPDATE users SET updated_at=NOW(), first_name=?, last_name=? WHERE id=?;", body.FirstName, body.LastName, structMerch.UserID); result.Error != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code":   "500",
				"status": "INTERNAL_SERVER_ERROR",
				"erorr": fiber.Map{
					"message": "failed to update user.",
				},
			}, "application/vnd.api+json")
		}
		tx.Commit()
	}

	// respond
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"code":   "200",
		"status": "OK",
		"data": fiber.Map{
			"message": "Berhasil mengubah user merchant.",
		},
	}, "application/vnd.api+json")
}

func ResetPassword(c *fiber.Ctx) error {
	// get the Merchant ID
	merchantID := c.Params("id")

	var user struct {
		UserID uint
		Email  string
	}

	tx := initializers.DB.Begin()

	if result := initializers.DB.Raw("SELECT user_id FROM merchants WHERE id=?;", merchantID).Scan(&user.UserID); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "failed to query merchant.",
			},
		}, "application/vnd.api+json")
	}

	if result := initializers.DB.Raw("SELECT email FROM users WHERE id=?;", user.UserID).Scan(&user.Email); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "failed to query user.",
			},
		}, "application/vnd.api+json")
	}

	// generate password
	res, err := password.Generate(16, 5, 0, false, false)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "failed to generate password.",
			},
		}, "application/vnd.api+json")
	}

	// hash the password
	hash, err := bcrypt.GenerateFromPassword([]byte(res), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "failed to hash password.",
			},
		}, "application/vnd.api+json")
	}

	log.Println(string(hash))

	if result := tx.Exec("UPDATE users SET updated_at=NOW(),password=? WHERE id=?;", string(hash), user.UserID); result.Error != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "failed to save user.",
			},
		}, "application/vnd.api+json")
	}

	log.Println(user.Email)
	log.Println(res)
	statusEmail := config.SendToEmail(user.Email, "Seller Reset Password", "---Ini adalah pesan otomatis---<br><br>Password telah berhasil direset, berikut informasi akun anda:<br><b>email:<b/> "+user.Email+"<br><b>password:</b> "+res)

	if !statusEmail {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "failed to send email to seller.",
			},
		}, "application/vnd.api+json")
	}

	tx.Commit()

	// respond
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"code":   "200",
		"status": "OK",
		"data": fiber.Map{
			"message": "Password berhasil direset, cek email untuk melihat password.",
		},
	}, "application/vnd.api+json")
}

func CreateNewItem(c *fiber.Ctx) error {
	// get data from request
	var body struct {
		Name        string `json:"name" xml:"name" form:"name" validate:"required,min=3,max=128"`
		Description string `json:"description" xml:"description" form:"description" validate:"required,min=3"`
		Qty         uint   `json:"qty" xml:"qty" form:"qty" validate:"required,numeric,gte=0"`
		Price       uint   `json:"price" xml:"price" form:"price" validate:"required,numeric,gte=0"`
		// Images      []byte `json:"images" xml:"images" form:"images" validate:"required"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"error": fiber.Map{
				"message": "failed to read body.",
			},
		}, "application/vnd.api+json")
	}

	errors := validation.ReturnValidation(body)

	var sliceFiles []string

	// merchantId
	merchantID := c.Locals("merchant").(models.Merchant).ID

	// check unique name
	var resultName uint
	if result := initializers.DB.Raw("SELECT 1 FROM items WHERE name=? AND merchant_id=?;", body.Name, merchantID).Scan(&resultName); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "failed to query items.",
			},
		}, "application/vnd.api+json")
	}

	if resultName == 1 {
		errors["Name"] = "Item sudah pernah ditambahkan sebelumnya."
	}

	// get array image
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "failed to get query.",
			},
		}, "application/vnd.api+json")
	}
	// => *multipart.Form
	// Get all files from "documents" key:
	files := form.File["images"]
	// => []*multipart.FileHeader
	if len(files) == 0 {
		errors["images"] = "Upload minimal 1 (satu) gambar terlebih dahulu."
	}

	// Loop through files:
	for _, file := range files {
		fmt.Println(file.Filename, file.Size, file.Header["Content-Type"][0])
		// => "tutorial.pdf" 360641 "application/pdf"
		if !validation.CheckFileMime(file.Header["Content-Type"][0]) || !validation.CheckFileSize(uint64(file.Size), 1) {
			errors["images"] = "Format tidak sesuai/ukuran gambar terlalu besar."
		}
	}

	if len(errors) != 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":   "400",
			"status": "BAD_REQUEST",
			"errors": errors,
		})
	}

	for _, file := range files {
		res, err := password.Generate(16, 16, 0, false, true)
		if err != nil {
			log.Println(err.Error())
			log.Println("error password")
		}
		// get ext file
		filenameList := strings.Split(file.Filename, ".")
		ext := filenameList[len(filenameList)-1]

		filename := strconv.FormatInt(time.Now().Unix(), 10) + "_" + res + "." + ext
		sliceFiles = append(sliceFiles, filename)

		// Save the files to disk:
		if err := c.SaveFile(file, GetDir(fmt.Sprintf("/public/image/%s", filename))); err != nil {
			fmt.Println(err.Error())
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code":   "500",
				"status": "INTERNAL_SERVER_ERROR",
				"erorr": fiber.Map{
					"message": "Gagal menyimpan gambar.",
				},
			}, "application/vnd.api+json")
		}
	}

	// save items
	tx := initializers.DB.Begin()
	var lastItemsInsertId uint
	if result := tx.Raw("INSERT INTO items(created_at, updated_at, deleted_at, name, description, qty, merchant_id, slug, price) VALUES (NOW(), NOW(), null, ?, ?, ?, ?, ?, ?) RETURNING id;", body.Name, body.Description, body.Qty, merchantID, slug.Make(body.Name), body.Price).Scan(&lastItemsInsertId); result.Error != nil {
		tx.Rollback()
		if ok := RemoveFile(sliceFiles); !ok {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code":   "500",
				"status": "INTERNAL_SERVER_ERROR",
				"erorr": fiber.Map{
					"message": "unable to save items.",
				},
			}, "application/vnd.api+json")
		}
	}
	for _, file := range sliceFiles {
		if result := tx.Exec("INSERT INTO images VALUES (?, ?);", lastItemsInsertId, file); result.Error != nil {
			tx.Rollback()
			if ok := RemoveFile(sliceFiles); !ok {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"code":   "500",
					"status": "INTERNAL_SERVER_ERROR",
					"erorr": fiber.Map{
						"message": "unable to save images.",
					},
				}, "application/vnd.api+json")
			}
		}
	}
	tx.Commit()

	// respond
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"code":   "201",
		"status": "CREATED",
		"data": fiber.Map{
			"message": "Berhasil menambahkan item baru.",
		},
	}, "application/vnd.api+json")
}

func ListItem(c *fiber.Ctx) error {
	// get query
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	// filter := c.Query("filter", "name")
	search := strings.ToLower(c.Query("search", ""))

	merchant := c.Locals("merchant").(models.Merchant)

	offset := ((page - 1) * limit)

	var items []models.Items
	if result := initializers.DB.Raw("SELECT * FROM items WHERE merchant_id=? AND name LIKE ? AND deleted_at IS null ORDER BY id ASC LIMIT ? OFFSET ?", merchant.ID, fmt.Sprintf("%%%s%%", search), limit, offset).Scan(&items); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "Unable to fetch items.",
			},
		}, "application/vnd.api+json")
	}

	for i, item := range items {
		if result := initializers.DB.Raw("SELECT items_id, CONCAT(CAST(? AS TEXT), title) AS title FROM images WHERE items_id=? LIMIT 3;", fmt.Sprintf("%s/img/", c.BaseURL()), item.ID).Scan(&items[i].Image); result.Error != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code":   "500",
				"status": "INTERNAL_SERVER_ERROR",
				"erorr": fiber.Map{
					"message": "Unable to fetch image.",
				},
			}, "application/vnd.api+json")
		}

		if result := initializers.DB.Raw("SELECT * FROM merchants where id=?;", item.MerchantID).Scan(&items[i].Merchant); result.Error != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code":   "500",
				"status": "INTERNAL_SERVER_ERROR",
				"erorr": fiber.Map{
					"message": "Unable to fetch merchant.",
				},
			}, "application/vnd.api+json")
		}

		// set on going
		items[i].OnGoing = 0
		// set closed
		items[i].Closed = 0
	}

	var countItems uint
	if result := initializers.DB.Raw("SELECT COUNT(id) as countItems FROM items WHERE merchant_id=? AND name LIKE ? AND deleted_at IS NULL", merchant.ID, fmt.Sprintf("%%%s%%", search)).Scan(&countItems); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "Unable to count items.",
			},
		}, "application/vnd.api+json")
	}

	lastPage := int(countItems) % limit
	if lastPage == 0 {
		lastPage = int(countItems) / limit
	} else {
		lastPage = (int(countItems) / limit) + 1
	}

	// respond
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":   "200",
		"status": "OK",
		"data":   items,
		"page": fiber.Map{
			"limit":     limit,
			"total":     countItems,
			"totalPage": lastPage,
			"current":   page,
		},
	}, "application/vnd.api+json")
}

func DetailItem(c *fiber.Ctx) error {
	itemID := c.Params("id")

	// query get detail item
	var item models.Items
	if result := initializers.DB.Raw("SELECT * FROM items WHERE id=? AND deleted_at IS null", itemID).Scan(&item); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "Unable to fetch items.",
			},
		}, "application/vnd.api+json")
	}
	if item.ID == 0 {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "Items not found.",
			},
		}, "application/vnd.api+json")
	}

	item.Merchant = c.Locals("merchant").(models.Merchant)

	if result := initializers.DB.Raw("SELECT items_id, CONCAT(CAST(? AS TEXT), title) AS title FROM images WHERE items_id=?", fmt.Sprintf("%s/img/", c.BaseURL()), item.ID).Scan(&item.Image); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "Unable to fetch image.",
			},
		}, "application/vnd.api+json")
	}

	// respond
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":   "200",
		"status": "OK",
		"data":   item,
	}, "application/vnd.api+json")
}

func UpdateItem(c *fiber.Ctx) error {
	// get data from request
	var body struct {
		ID          uint   `json:"id" xml:"id" form:"id"  validate:"required,numeric,gt=0"`
		Name        string `json:"name" xml:"name" form:"name"  validate:"required,min=3,max=128"`
		Description string `json:"description" xml:"description" form:"description" validate:"required,min=3"`
		Qty         uint   `json:"qty" xml:"qty" form:"qty" validate:"required,numeric,gte=0"`
		Price       uint   `json:"price" xml:"price" form:"price" validate:"required,numeric,gte=0"`
		// Images      []byte `json:"images" xml:"images" form:"images"`
	}

	if c.BodyParser(&body) != nil {
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":   "400",
				"status": "BAD_REQUEST",
				"error": fiber.Map{
					"message": "failed to read body.",
				},
			}, "application/vnd.api+json")
		}
	}

	errors := validation.ReturnValidation(body)

	// merchantId
	merchantID := c.Locals("merchant").(models.Merchant).ID

	// check unique name
	var resultName uint
	if result := initializers.DB.Raw("SELECT 1 FROM items WHERE name=? AND id!=? AND merchant_id=?;", body.Name, body.ID, merchantID).Scan(&resultName); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "failed to query items.",
			},
		}, "application/vnd.api+json")
	}

	if resultName == 1 {
		errors["Name"] = "Item sudah pernah ditambahkan sebelumnya."
	}

	var sliceFiles []string
	// get array image
	form, errForm := c.MultipartForm()
	if errForm != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "failed to query param.",
			},
		}, "application/vnd.api+json")
	}
	// => *multipart.Form

	// Get all files from "documents" key:
	files := form.File["images"]
	// => []*multipart.FileHeader

	lenFiles := len(files)

	// Loop through files:
	for _, file := range files {
		fmt.Println(file.Filename, file.Size, file.Header["Content-Type"][0])
		// => "tutorial.pdf" 360641 "application/pdf"
		if !validation.CheckFileMime(file.Header["Content-Type"][0]) || !validation.CheckFileSize(uint64(file.Size), 1) {
			errors["images"] = "Format tidak sesuai/ukuran gambar terlalu besar."
		}
	}

	if len(errors) != 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":   "400",
			"status": "BAD_REQUEST",
			"errors": errors,
		})
	}

	for _, file := range files {
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
		if err := c.SaveFile(file, GetDir(fmt.Sprintf("/public/image/%s", filename))); err != nil {
			fmt.Println(err.Error())
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code":   "500",
				"status": "INTERNAL_SERVER_ERROR",
				"erorr": fiber.Map{
					"message": "Unable to save image.",
				},
			}, "application/vnd.api+json")
		}
	}

	// query get detail item
	var resultItem bool
	if result := initializers.DB.Raw("SELECT 1 FROM items WHERE id=? LIMIT 1;", body.ID).Scan(&resultItem); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "failed to query items.",
			},
		}, "application/vnd.api+json")
	}
	if !resultItem {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "Item not found.",
			},
		}, "application/vnd.api+json")
	}

	var images []string

	// save items
	tx := initializers.DB.Begin()

	if lenFiles != 0 {
		if result := initializers.DB.Raw("SELECT title AS images FROM images WHERE items_id=?", body.ID).Scan(&images); result.Error != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code":   "500",
				"status": "INTERNAL_SERVER_ERROR",
				"erorr": fiber.Map{
					"message": "Unable to fetch image.",
				},
			}, "application/vnd.api+json")
		}

		if result := tx.Exec("DELETE FROM images WHERE items_id=?;", body.ID); result.Error != nil {
			tx.Rollback()
			if ok := RemoveFile(sliceFiles); !ok {
				log.Println("Gagal menghapus gambar.")
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code":   "500",
				"status": "INTERNAL_SERVER_ERROR",
				"erorr": fiber.Map{
					"message": "Gagal menimpa gambar.",
				},
			}, "application/vnd.api+json")
		}
	}

	result := tx.Exec("UPDATE items SET updated_at=NOW(), name=?, description=?, qty=?, slug=?, price=? WHERE id=?;", body.Name, body.Description, body.Qty, slug.Make(body.Name), body.Price, body.ID)
	if result.Error != nil {
		tx.Rollback()
		if ok := RemoveFile(sliceFiles); !ok {
			log.Println("Gagal menghapus gambar.")
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "unable to update item.",
			},
		}, "application/vnd.api+json")
	}

	for _, file := range sliceFiles {
		result = tx.Exec("INSERT INTO images VALUES (?, ?);", body.ID, file)
		if result.Error != nil {
			tx.Rollback()
			if ok := RemoveFile(sliceFiles); !ok {
				log.Println("Gagal menghapus gambar.")
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code":   "500",
				"status": "INTERNAL_SERVER_ERROR",
				"erorr": fiber.Map{
					"message": "unable to save image.",
				},
			}, "application/vnd.api+json")
		}
	}

	if lenFiles != 0 {
		RemoveFile(images)
	}

	tx.Commit()

	// respond
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":   "200",
		"status": "OK",
		"data": fiber.Map{
			"message": "Berhasil mengubah item.",
		},
	}, "application/vnd.api+json")
}

func RemoveItem(c *fiber.Ctx) error {
	itemID, err := c.ParamsInt("id", 0)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"error": fiber.Map{
				"message": err.Error(),
			},
		}, "application/vnd.api+json")
	}

	if itemID == 0 {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"error": fiber.Map{
				"message": "Params not found.",
			},
		}, "application/vnd.api+json")
	}

	// get merchantID
	merchant := c.Locals("merchant").(models.Merchant)

	// query get detail item
	var merchantID uint
	if result := initializers.DB.Raw("SELECT merchant_id FROM items WHERE id=?", itemID).Scan(&merchantID); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "failed to query merchant.",
			},
		}, "application/vnd.api+json")
	}
	if merchantID == 0 {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "merchant not found.",
			},
		}, "application/vnd.api+json")
	}

	if merchantID != merchant.ID {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "Unable to remove items.",
			},
		}, "application/vnd.api+json")
	}

	tx := initializers.DB.Begin()
	if result := tx.Exec("UPDATE items SET deleted_at=NOW() WHERE id=?;", itemID); result.Error != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "Unable to remove items.",
			},
		}, "application/vnd.api+json")
	}

	tx.Commit()

	// respond
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":   "200",
		"status": "OK",
		"data": fiber.Map{
			"message": "Berhasil menghapus item.",
		},
	}, "application/vnd.api+json")
}

func ListAllItems(c *fiber.Ctx) error {
	// get query
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 12)
	// filter := c.Query("filter", "name")
	search := strings.ToLower(c.Query("search", ""))

	offset := ((page - 1) * limit)

	var items []models.Items
	if result := initializers.DB.Raw("SELECT * FROM items WHERE LOWER(name) LIKE ? AND deleted_at IS null ORDER BY id ASC LIMIT ? OFFSET ?", fmt.Sprintf("%%%s%%", search), limit, offset).Scan(&items); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "Unable to fetch items.",
			},
		}, "application/vnd.api+json")
	}

	for i, item := range items {
		if result := initializers.DB.Raw("SELECT items_id, CONCAT(CAST(? AS TEXT), title) AS title FROM images WHERE items_id=? LIMIT 1;", fmt.Sprintf("%s/img/", c.BaseURL()), item.ID).Scan(&items[i].Image); result.Error != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code":   "500",
				"status": "INTERNAL_SERVER_ERROR",
				"erorr": fiber.Map{
					"message": "Unable to fetch image.",
				},
			}, "application/vnd.api+json")
		}

		if result := initializers.DB.Raw("SELECT * FROM merchants where id=?;", item.MerchantID).Scan(&items[i].Merchant); result.Error != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code":   "500",
				"status": "INTERNAL_SERVER_ERROR",
				"erorr": fiber.Map{
					"message": "Unable to fetch merchant.",
				},
			}, "application/vnd.api+json")
		}

		// set on going
		items[i].OnGoing = 0
		// set closed
		items[i].Closed = 0
	}

	var countItems uint
	if result := initializers.DB.Raw("SELECT COUNT(id) as countItems FROM items WHERE LOWER(name) LIKE ? AND deleted_at IS NULL", fmt.Sprintf("%%%s%%", search)).Scan(&countItems); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "Unable to count items.",
			},
		}, "application/vnd.api+json")
	}

	lastPage := int(countItems) % limit
	if lastPage == 0 {
		lastPage = int(countItems) / limit
	} else {
		lastPage = (int(countItems) / limit) + 1
	}

	// respond
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":   "200",
		"status": "OK",
		"data":   items,
		"page": fiber.Map{
			"limit":     limit,
			"total":     countItems,
			"totalPage": lastPage,
			"current":   page,
		},
	}, "application/vnd.api+json")
}

func RemoveFile(files []string) bool {
	for _, file := range files {
		if err := os.Remove(GetDir(fmt.Sprintf("/public/image/%s", file))); err != nil {
			return false
		}
	}
	return true
}

func MerchantProject(c *fiber.Ctx) error {
	merchant := c.Locals("merchant").(models.Merchant)

	// check status
	if result := initializers.DB.Exec("UPDATE project SET status='Pesanan Dibatalkan', updated_at=CURRENT_TIMESTAMP WHERE project.id IN (SELECT project.id FROM project WHERE merchant_id=? AND status='Menunggu Pembayaran' AND deleted_at IS NULL AND created_at < (CURRENT_TIMESTAMP - INTERVAL '24 hours'));", merchant.ID); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "Failed to update project",
			},
		}, "application/vnd.api+json")
	}

	var UserProject []struct {
		ID        uint      `gorm:"column:id" json:"id"`
		Invoice   string    `gorm:"column:invoice" json:"invoice"`
		StartDate time.Time `gorm:"column:start_date" json:"start_date"`
		Status    string    `gorm:"column:status" json:"status"`
		Qty       uint      `gorm:"column:qty" json:"qty"`
		Price     uint      `gorm:"column:price" json:"price"`
		ItemName  string    `gorm:"column:item_name" json:"item_name"`
	}

	if result := initializers.DB.Raw("SELECT project.id,invoice,lower(range_date) AS start_date,status,project_item.qty,project_item.price,items.name as item_name FROM project JOIN project_item ON project_item.project_id = project.id JOIN items ON items.id = project_item.item_id WHERE project.merchant_id=? AND project.deleted_at IS NULL ORDER BY id ASC;", merchant.ID).Scan(&UserProject); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "Failed to fetch project",
			},
		}, "application/vnd.api+json")
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":   "200",
		"status": "OK",
		"data":   UserProject,
	}, "application/vnd.api+json")
}

func MerchantDetailProject(c *fiber.Ctx) error {
	projectID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":   "400",
			"status": "BAD_REQUEST",
			"error": fiber.Map{
				"message": err.Error(),
			},
		}, "application/vnd.api+json")
	}

	type Merchant struct {
		ID   uint   `gorm:"primaryKey" json:"id"`
		Name string `gorm:"column:name" json:"name"`
	}

	type Project struct {
		ID         uint      `gorm:"column:id;primaryKey" json:"id"`
		Invoice    string    `gorm:"column:invoice" json:"invoice"`
		StartDate  time.Time `gorm:"column:start_date" json:"start_date"`
		EndDate    time.Time `gorm:"column:end_date" json:"end_date"`
		Status     string    `gorm:"column:status" json:"status"`
		Price      uint      `gorm:"column:price" json:"price"`
		ItemName   string    `gorm:"column:item_name" json:"item_name"`
		MerchantID uint      `gorm:"column:merchant_id" json:"merchant_id"`
		Merchant   Merchant  `gorm:"foreignKey:MerchantID" json:"merchant"`
	}

	var project Project

	// project detail
	if result := initializers.DB.Raw(`
		SELECT 
			project.id, 
			invoice, 
			lower(range_date) AS start_date, 
			upper(range_date) AS end_date, 
			status, 
			project_item, 
			project_item.price, 
			items.name as item_name, 
			project.merchant_id
		FROM project 
		JOIN project_item ON project_item.project_id = project.id 
		JOIN items ON items.id = project_item.item_id 
		WHERE project.id = ? AND project.deleted_at IS NULL;
	`, projectID).Scan(&project); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"error": fiber.Map{
				"message": "Failed to fetch project",
			},
		}, "application/vnd.api+json")
	}

	if result := initializers.DB.Raw(`
		SELECT 
			merchants.id, 
			merchants.name 
		FROM merchants 
		JOIN project ON merchants.id = project.merchant_id 
		WHERE project.id = ? AND project.deleted_at IS NULL;
	`, projectID).Scan(&project.Merchant); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"error": fiber.Map{
				"message": "Failed to fetch merchant",
			},
		}, "application/vnd.api+json")
	}

	// return ok
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":   "200",
		"status": "OK",
		"data":   project,
	}, "application/vnd.api+json")
}

func ConfirmMerchantProject(c *fiber.Ctx) error {
	projectID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":   "400",
			"status": "BAD_REQUEST",
			"error": fiber.Map{
				"message": err.Error(),
			},
		}, "application/vnd.api+json")
	}
	// check project merchant
	var checkMerchantProject int8
	if result := initializers.DB.Raw("SELECT 1 FROM project WHERE id=? AND merchant_id=?;", projectID, c.Locals("merchant").(models.Merchant).ID).Scan(&checkMerchantProject); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "Failed to confirm project",
			},
		}, "application/vnd.api+json")
	}
	if checkMerchantProject != 1 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":   "401",
			"status": "UNAUTHORIZED",
			"erorr": fiber.Map{
				"message": "Project Not Found",
			},
		}, "application/vnd.api+json")
	}
	// confirm merchant
	tx := initializers.DB.Begin()
	if result := tx.Exec("UPDATE project SET status='On Going', confirm=1 WHERE id=?;", projectID); result.Error != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "Failed to confirm project",
			},
		}, "application/vnd.api+json")
	}

	if result := tx.Exec("INSERT INTO project_timeline(message, created_at, msg_from, project_id) VALUES ('Mengkonfirmasi Project', NOW(), 2, ?);", projectID); result.Error != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "Failed to create timeline",
			},
		}, "application/vnd.api+json")
	}
	tx.Commit()

	// return ok
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":   "200",
		"status": "OK",
		"data": fiber.Map{
			"message": "Berhasil konfirmasi project",
		},
	}, "application/vnd.api+json")
}

func ConfirmDoneMerchantProject(c *fiber.Ctx) error {
	projectID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":   "400",
			"status": "BAD_REQUEST",
			"error": fiber.Map{
				"message": err.Error(),
			},
		}, "application/vnd.api+json")
	}
	// check project merchant
	var checkMerchantProject int8
	if result := initializers.DB.Raw("SELECT 1 FROM project WHERE id=? AND merchant_id=?;", projectID, c.Locals("merchant").(models.Merchant).ID).Scan(&checkMerchantProject); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "Failed to confirm project",
			},
		}, "application/vnd.api+json")
	}
	if checkMerchantProject != 1 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":   "401",
			"status": "UNAUTHORIZED",
			"erorr": fiber.Map{
				"message": "Project Not Found",
			},
		}, "application/vnd.api+json")
	}
	// confirm merchant
	tx := initializers.DB.Begin()
	if result := tx.Exec("UPDATE project SET status='Confirm Done', confirm=1 WHERE id=?;", projectID); result.Error != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "Failed to confirm project",
			},
		}, "application/vnd.api+json")
	}

	if result := tx.Exec("INSERT INTO project_timeline(message, created_at, msg_from, project_id) VALUES ('Konfirmasi selesai', NOW(), 2, ?);", projectID); result.Error != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "Failed to create timeline",
			},
		}, "application/vnd.api+json")
	}
	tx.Commit()

	// return ok
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":   "200",
		"status": "OK",
		"data": fiber.Map{
			"message": "Berhasil konfirmasi project",
		},
	}, "application/vnd.api+json")
}

func MerchantProjectTimeline(c *fiber.Ctx) error {
	projectID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":   "400",
			"status": "BAD_REQUEST",
			"error": fiber.Map{
				"message": err.Error(),
			},
		}, "application/vnd.api+json")
	}

	var resultProject uint
	if result := initializers.DB.Raw("SELECT 1 FROM project WHERE id=? AND merchant_id=? AND deleted_at IS NULL;", projectID, c.Locals("merchant").(models.Merchant).ID).Scan(&resultProject); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "failed to query project.",
			},
		}, "application/vnd.api+json")
	}

	if resultProject != 1 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":   "401",
			"status": "UNAUTHORIZED",
			"erorr": fiber.Map{
				"message": "failed to find project",
			},
		}, "application/vnd.api+json")
	}

	var Merchant struct {
		Name   string `gorm:"column:name" json:"name"`
		Avatar string `gorm:"column:avatar" json:"avatar"`
	}

	// type System struct {
	// 	Name   string
	// 	Avatar string
	// }

	var ProjectTimeline []struct {
		ID        uint      `gorm:"column:id" json:"id"`
		Message   string    `gorm:"column:message" json:"message"`
		CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
		// Images    []string  `gorm:"column:images" json:"images"`
		// Images      StringSlice `gorm:"column:images" json:"images"`
		Images      []string `gorm:"-" json:"images"` // Exclude from DB operations
		MessageFrom uint     `gorm:"column:msg_from" json:"message_from"`
		Price       uint     `gorm:"column:price" json:"price"`
		ItemName    string   `gorm:"column:item_name" json:"item_name"`
		// Merchant    Merchant  `json:"merchant"`
		// System      System    `json:"system"`
	}

	if result := initializers.DB.Raw("SELECT * FROM project_timeline WHERE project_id=?", projectID).Scan(&ProjectTimeline); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "Failed to fetch project",
			},
		}, "application/vnd.api+json")
	}

	if result := initializers.DB.Raw("SELECT merchants.name AS name,CONCAT(CAST(? AS TEXT), merchants.avatar) AS avatar FROM project JOIN merchants ON merchants.id=project.merchant_id WHERE project.id=?", fmt.Sprintf("%s/img/", c.BaseURL()), projectID).Scan(&Merchant); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "Failed to fetch merchant",
			},
		}, "application/vnd.api+json")
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":   "200",
		"status": "OK",
		"data": fiber.Map{
			"merchant": Merchant,
			"system": fiber.Map{
				"name":   "Sistem",
				"avatar": fmt.Sprintf("%s/img/logo.png", c.BaseURL()),
			},
			"timeline": ProjectTimeline,
		},
	}, "application/vnd.api+json")
}

func MerchantProjectTimelineAdd(c *fiber.Ctx) error {
	projectID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":   "400",
			"status": "BAD_REQUEST",
			"error": fiber.Map{
				"message": err.Error(),
			},
		}, "application/vnd.api+json")
	}

	var resultProject uint
	if result := initializers.DB.Raw("SELECT 1 FROM project WHERE id=? AND merchant_id=? AND deleted_at IS NULL;", projectID, c.Locals("merchant").(models.Merchant).ID).Scan(&resultProject); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "failed to query project.",
			},
		}, "application/vnd.api+json")
	}

	if resultProject != 1 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":   "401",
			"status": "UNAUTHORIZED",
			"erorr": fiber.Map{
				"message": "failed to find project",
			},
		}, "application/vnd.api+json")
	}

	// get data from request
	var body struct {
		Message string `json:"message" xml:"message" form:"message" validate:"required,min=3,max=128"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"error": fiber.Map{
				"message": "failed to read body.",
			},
		}, "application/vnd.api+json")
	}

	errors := validation.ReturnValidation(body)

	var sliceFiles []string

	// get array image
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "failed to get query.",
			},
		}, "application/vnd.api+json")
	}
	// => *multipart.Form
	// Get all files from "documents" key:
	files := form.File["images"]
	// => []*multipart.FileHeader
	// if len(files) == 0 {
	// 	errors["images"] = "Upload minimal 1 (satu) gambar terlebih dahulu."
	// }

	// Loop through files:
	for _, file := range files {
		fmt.Println(file.Filename, file.Size, file.Header["Content-Type"][0])
		// => "tutorial.pdf" 360641 "application/pdf"
		if !validation.CheckFileMime(file.Header["Content-Type"][0]) || !validation.CheckFileSize(uint64(file.Size), 1) {
			errors["images"] = "Format tidak sesuai/ukuran gambar terlalu besar."
		}
	}

	if len(errors) != 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":   "400",
			"status": "BAD_REQUEST",
			"errors": errors,
		})
	}

	for _, file := range files {
		res, err := password.Generate(16, 16, 0, false, true)
		if err != nil {
			log.Println(err.Error())
			log.Println("error password")
		}
		// get ext file
		filenameList := strings.Split(file.Filename, ".")
		ext := filenameList[len(filenameList)-1]

		filename := strconv.FormatInt(time.Now().Unix(), 10) + "_" + res + "." + ext
		sliceFiles = append(sliceFiles, filename)

		// Save the files to disk:
		if err := c.SaveFile(file, GetDir(fmt.Sprintf("/public/image/%s", filename))); err != nil {
			fmt.Println(err.Error())
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code":   "500",
				"status": "INTERNAL_SERVER_ERROR",
				"erorr": fiber.Map{
					"message": "Gagal menyimpan gambar.",
				},
			}, "application/vnd.api+json")
		}
	}

	// 1 customer
	// 2 seller
	// 3 admin
	// 4 system

	// save timeline
	tx := initializers.DB.Begin()
	if result := tx.Exec("INSERT INTO project_timeline(message, images, created_at, msg_from, project_id) VALUES (?, ?, NOW(), ?, ?);", body.Message, pq.Array(sliceFiles), 2, projectID); result.Error != nil {
		tx.Rollback()
		if ok := RemoveFile(sliceFiles); !ok {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code":   "500",
				"status": "INTERNAL_SERVER_ERROR",
				"erorr": fiber.Map{
					"message": "unable to save images.",
				},
			}, "application/vnd.api+json")
		}
	}
	tx.Commit()

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"code":   "201",
		"status": "CREATED",
		"data": fiber.Map{
			"message": "message uploaded",
		},
	}, "application/vnd.api+json")
}
