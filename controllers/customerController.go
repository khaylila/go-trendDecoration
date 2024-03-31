package controllers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/khaylila/go-trendDecoration/initializers"
	"github.com/khaylila/go-trendDecoration/models"
)

func ListItemFromMerchant(c *fiber.Ctx) error {
	// // get query
	page := c.QueryInt("page", 1)
	max := c.QueryInt("limit", 5)

	merchantSlug := c.Params("merchant")

	var merchant models.Merchant
	if result := initializers.DB.Raw("SELECT * FROM merchants WHERE slug = ? AND deleted_at IS NULL LIMIT 1;", merchantSlug).Scan(&merchant); result.Error != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Unable to fetch merchant.",
		})
	}

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
		"merchant": merchant,
		"links": fiber.Map{
			"self":  fmt.Sprintf("%s/%s?page=%d&limit=%d", c.BaseURL(), merchantSlug, page, max),
			"first": fmt.Sprintf("%s/%s?page=%d&limit=%d", c.BaseURL(), merchantSlug, 1, max),
			"prev":  fmt.Sprintf("%s/%s?page=%d&limit=%d", c.BaseURL(), merchantSlug, page-1, max),
			"next":  fmt.Sprintf("%s/%s?page=%d&limit=%d", c.BaseURL(), merchantSlug, page+1, max),
			"last":  fmt.Sprintf("%s/%s?page=%d&limit=%d", c.BaseURL(), merchantSlug, lastPage, max),
		},
	}, "application/vnd.api+json")
}

func DetailItemWithSlug(c *fiber.Ctx) error {
	merchantSlug := c.Params("merchant")
	itemSlug := c.Params("itemSlug")

	// query get detail item
	var item models.Items
	if result := initializers.DB.Raw("SELECT items.* FROM items JOIN merchants ON merchants.id = items.merchant_id WHERE items.slug=? AND merchants.slug=? AND items.deleted_at IS null", itemSlug, merchantSlug).Scan(&item); result.Error != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Unable to find items.",
		})
	}
	if item.ID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Unable to find items.",
		})
	}

	if result := initializers.DB.Raw("SELECT * FROM merchants WHERE id = ?", item.MerchantID).Scan(&item.Merchant); result.Error != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Unable to fetch merchants.",
		})
	}

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

func SearchItem(c *fiber.Ctx) error {
	// get query
	page := c.QueryInt("page", 1)
	max := c.QueryInt("limit", 5)
	search := c.Query("search", "")
	fmt.Println(search)
	offset := ((page - 1) * max)

	var items []models.Items
	result := initializers.DB.Raw("SELECT * FROM items WHERE items.name LIKE ? AND deleted_at IS NULL ORDER BY id ASC LIMIT ? OFFSET ?", "%"+search+"%", max, offset).Scan(&items)
	fmt.Println("asdf")
	fmt.Println(items)
	if result.Error != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Unable to fetch items.",
		})
	}

	for i, item := range items {
		if result := initializers.DB.Raw("SELECT * FROM merchants WHERE id = ?;", item.MerchantID).Scan(&items[i].Merchant); result.Error != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Unable to fetch merchant.",
			})
		}
		result = initializers.DB.Raw("SELECT items_id, CONCAT(CAST(? AS TEXT), title) AS title FROM images WHERE items_id=?", fmt.Sprintf("%s/img/", c.BaseURL()), item.ID).Scan(&items[i].Image)
		if result.Error != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Unable to fetch image.",
			})
		}
	}

	var countItems uint
	if result = initializers.DB.Raw("SELECT COUNT(items.id) as countItems FROM items WHERE items.name LIKE ? AND deleted_at IS NULL;", "%"+search+"%").Scan(&countItems); result.Error != nil {
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
		"data": items,
		"links": fiber.Map{
			"self":  fmt.Sprintf("%s/search?page=%d&limit=%d", c.BaseURL(), page, max),
			"first": fmt.Sprintf("%s/search?page=%d&limit=%d", c.BaseURL(), 1, max),
			"prev":  fmt.Sprintf("%s/search?page=%d&limit=%d", c.BaseURL(), page-1, max),
			"next":  fmt.Sprintf("%s/search?page=%d&limit=%d", c.BaseURL(), page+1, max),
			"last":  fmt.Sprintf("%s/search?page=%d&limit=%d", c.BaseURL(), lastPage, max),
		},
	}, "application/vnd.api+json")
}

func CheckChart(c *fiber.Ctx) error {
	// get the req body
	var body struct {
		ItemsID   uint   `json:"itemId" form:"itemId"`
		RentStart string `json:"rentStart" form:"rentStart"`
		RentEnd   string `json:"rentEnd" form:"rentEnd"`
	}

	if c.BodyParser(&body) != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "failed to read body.",
		})
	}

	var carts []struct {
		rentDate  time.Time
		available bool
	}
	if result := initializers.DB.Raw("SELECT date_series::date AS rent_date, (items.qty > COALESCE(SUM(c.qty), 0))::boolean AS available FROM public.carts c JOIN items ON item_id = items.id JOIN generate_series(lower(c.rent_range), upper(c.rent_range), '1 day'::interval) AS date_series ON date_series::date >= ?::date AND date_series::date <= ?::date WHERE item_id = ? GROUP BY date_series::date, items.qty ORDER BY date_series::date;", body.RentStart, body.RentEnd, body.ItemsID).Scan(&carts); result.Error != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "failed to query chart item.",
		})
	}

	for _, cart := range carts {
		if !cart.available {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Items not available.",
			})
		}
	}

	return c.Next()
}

func InsertToChart(c *fiber.Ctx) error {
	// get the req body
	var body struct {
		Qty       uint   `json:"qty" form:"qty"`
		ItemsID   uint   `json:"itemId" form:"itemId"`
		RentStart string `json:"rentStart" form:"rentStart"`
		RentEnd   string `json:"rentEnd" form:"rentEnd"`
	}

	if c.BodyParser(&body) != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "failed to read body.",
		})
	}

	var checkItem uint8
	if result := initializers.DB.Raw("SELECT 1 FROM items WHERE id = ? AND deleted_at IS NULL;", body.ItemsID).Scan(&checkItem); result.Error != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "failed to fetch item.",
		})
	}

	if checkItem != 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Item not found.",
		})
	}

	user := c.Locals("user").(models.User)
	if result := initializers.DB.Exec("INSERT INTO carts(item_id, qty, rent_range, user_id) VALUES (?, ?, ?, ?);", body.ItemsID, body.Qty, fmt.Sprintf("[%s, %s)", body.RentStart, body.RentEnd), user.ID); result.Error != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "failed to save cart item.",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"status": "Berhasil menambahkan data"})
}
