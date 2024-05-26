package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/khaylila/go-trendDecoration/initializers"
	"github.com/khaylila/go-trendDecoration/models"
	"github.com/khaylila/go-trendDecoration/validation"
	"github.com/lib/pq"
	"github.com/sethvargo/go-password/password"
)

func ListProject(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)

	// check status
	if result := initializers.DB.Exec("UPDATE project SET status='Pesanan Dibatalkan', updated_at=CURRENT_TIMESTAMP WHERE project.id IN (SELECT project.id FROM project WHERE user_id=? AND status='Menunggu Pembayaran' AND deleted_at IS NULL AND created_at < (CURRENT_TIMESTAMP - INTERVAL '24 hours'));", user.ID); result.Error != nil {
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

	if result := initializers.DB.Raw("SELECT project.id,invoice,lower(range_date) AS start_date,status,project_item.qty,project_item.price,items.name as item_name FROM project JOIN project_item ON project_item.project_id = project.id JOIN items ON items.id = project_item.item_id WHERE user_id=? AND project.deleted_at IS NULL ORDER BY id ASC;", user.ID).Scan(&UserProject); result.Error != nil {
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

func DetailProject(c *fiber.Ctx) error {
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

func CheckProject(c *fiber.Ctx) error {
	// get the req body
	var body struct {
		DateStart    string `json:"dateStart" xml:"dateStart" form:"dateStart"  validate:"required,min=10,max=10"`
		DateEnd      string `json:"dateEnd" xml:"dateEnd" form:"dateEnd"  validate:"required,min=10,max=10"`
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
	if result := initializers.DB.Raw("SELECT date_series::date AS rent_date, (items.qty > COALESCE(SUM (project_item.qty), 0))::boolean AS available  FROM project p JOIN project_item on p.id = project_item.project_id JOIN items ON project_item.item_id = items.id JOIN generate_series(lower(p.range_date), upper(p.range_date), '1 day'::interval) AS date_series  ON date_series::date >= ?::date AND date_series::date <= ?::date  WHERE project_item.item_id = ? GROUP BY date_series::date, items.qty ORDER BY date_series::date;", body.DateStart, body.DateEnd, item.ID).Scan(&projects); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"error": fiber.Map{
				"message": "failed to query chart item.",
			},
		}, "application/vnd.api+json")
	}

	for _, project := range projects {
		if !project.Available {
			return c.Status(fiber.StatusNotAcceptable).JSON(fiber.Map{
				"code":   "406",
				"status": "STATUS_NOT ACCEPTABLE",
				"error": fiber.Map{
					"message": "Items not available.",
				},
			}, "application/vnd.api+json")
		} else {
			if (project.ItemQty-project.ProjectQty)-int(body.Qty) < 0 {
				return c.Status(fiber.StatusNotAcceptable).JSON(fiber.Map{
					"code":   "406",
					"status": "STATUS_NOT ACCEPTABLE",
					"error": fiber.Map{
						"message": "Maksimal item yang tersedia adalah " + strconv.FormatUint(uint64(project.ItemQty)-uint64(project.ProjectQty), 10),
					},
				}, "application/vnd.api+json")
			}
		}
	}

	return c.Next()
}

type StringSlice []string

// Scan implements the sql.Scanner interface.
func (ss *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*ss = nil
		return nil
	}

	// Convert the database representation to a byte array
	bytes, ok := value.([]byte)
	if !ok {
		return nil // or return an error if you prefer
	}

	// Unmarshal the byte array into the StringSlice
	return json.Unmarshal(bytes, ss)
}

func ProjectTimeline(c *fiber.Ctx) error {
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
	if result := initializers.DB.Raw("SELECT 1 FROM project WHERE id=? AND user_id=? AND deleted_at IS NULL;", projectID, c.Locals("user").(models.User).ID).Scan(&resultProject); result.Error != nil {
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

func ProjectTimelineAdd(c *fiber.Ctx) error {
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
	if result := initializers.DB.Raw("SELECT 1 FROM project WHERE id=? AND user_id=? AND deleted_at IS NULL;", projectID, c.Locals("user").(models.User).ID).Scan(&resultProject); result.Error != nil {
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
	if result := tx.Exec("INSERT INTO project_timeline(message, images, created_at, msg_from, project_id) VALUES (?, ?, NOW(), ?, ?);", body.Message, pq.Array(sliceFiles), 1, projectID); result.Error != nil {
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
