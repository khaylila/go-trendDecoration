package controllers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/khaylila/go-trendDecoration/initializers"
	"github.com/khaylila/go-trendDecoration/models"
)

func ListItemFromMerchant(c *fiber.Ctx) error {
	// get query
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 99)

	merchantSlug := c.Params("merchant")

	var merchant struct {
		Merchant models.Merchant `json:"merchant"`
		Items    []models.Items  `json:"items"`
	}
	if result := initializers.DB.Raw("SELECT * FROM merchants WHERE slug=? AND deleted_at IS NULL LIMIT 1;", merchantSlug).Scan(&merchant.Merchant); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "Unable to fetch merchant.",
			},
		}, "application/vnd.api+json")
	}

	if merchant.Merchant.ID == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"code":   "404",
			"status": "NOT_FOUND",
			"erorr": fiber.Map{
				"message": "merchant not found.",
			},
		}, "application/vnd.api+json")
	}

	offset := ((page - 1) * limit)

	// var items []models.Items
	if result := initializers.DB.Raw("SELECT * FROM items WHERE merchant_id=? AND deleted_at IS null ORDER BY id ASC LIMIT ? OFFSET ?", merchant.Merchant.ID, limit, offset).Scan(&merchant.Items); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "Unable to fetch items.",
			},
		}, "application/vnd.api+json")
	}

	for i, item := range merchant.Items {
		if result := initializers.DB.Raw("SELECT items_id, CONCAT(CAST(? AS TEXT), title) AS title FROM images WHERE items_id=?", fmt.Sprintf("%s/img/", c.BaseURL()), item.ID).Scan(&merchant.Items[i].Image); result.Error != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code":   "500",
				"status": "INTERNAL_SERVER_ERROR",
				"erorr": fiber.Map{
					"message": "Unable to fetch image.",
				},
			}, "application/vnd.api+json")
		}
		merchant.Items[i].Merchant = merchant.Merchant
	}

	var countItems uint
	if result := initializers.DB.Raw("SELECT COUNT(id) as countItems FROM items WHERE merchant_id=? AND deleted_at IS NULL", merchant.Merchant.ID).Scan(&countItems); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "Unable to count item.",
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
		"data":   merchant,
		"page": fiber.Map{
			"limit":     limit,
			"total":     countItems,
			"totalPage": lastPage,
			"current":   page,
		},
	}, "application/vnd.api+json")
}

// deleted soon
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

func CustomerItemDetail(c *fiber.Ctx) error {
	merchantSlug := c.Params("merchantSlug", "")
	itemSlug := c.Params("itemSlug", "")

	if merchantSlug == "" || itemSlug == "" {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"code":   "404",
			"status": "NOT_FOUND",
			"erorr": fiber.Map{
				"message": "items not found.",
			},
		}, "application/vnd.api+json")
	}
	// query get detail merchant
	var merchant models.Merchant
	if result := initializers.DB.Raw("SELECT *, CONCAT(CAST(? AS TEXT), avatar) AS avatar FROM merchants WHERE slug=? AND deleted_at IS null", fmt.Sprintf("%s/img/", c.BaseURL()), merchantSlug).Scan(&merchant); result.Error != nil {
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
	if result := initializers.DB.Raw("SELECT * FROM items WHERE merchant_id = ? AND slug = ? AND deleted_at IS null", merchant.ID, itemSlug).Scan(&item); result.Error != nil {
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

	if result := initializers.DB.Raw("SELECT items_id, CONCAT(CAST(? AS TEXT), title) AS title FROM images WHERE items_id=?", fmt.Sprintf("%s/img/", c.BaseURL()), item.ID).Scan(&item.Image); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "Unable to fetch image.",
			},
		}, "application/vnd.api+json")
	}

	merchant.Rating = 4.9

	item.Merchant = merchant
	item.Rent = 0
	item.Rating = 5.0
	item.UserRating = 1

	// respond
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":   "200",
		"status": "OK",
		"data":   item,
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
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"error": fiber.Map{
				"message": "failed to query chart item.",
			},
		}, "application/vnd.api+json")
	}

	for _, cart := range carts {
		if !cart.available {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":   "400",
				"status": "BAD_REQUEST",
				"error": fiber.Map{
					"message": "Items not available.",
				},
			}, "application/vnd.api+json")
		}
	}

	return c.Next()
}

func CheckItemByDate(c *fiber.Ctx) error {
	// get the req body
	getParameter := c.Queries()

	// query get detail merchant
	var merchant models.Merchant
	if result := initializers.DB.Raw("SELECT * FROM merchants WHERE slug=? AND deleted_at IS null", getParameter["merchantSlug"]).Scan(&merchant); result.Error != nil {
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
	if result := initializers.DB.Raw("SELECT * FROM items WHERE merchant_id = ? AND slug = ? AND deleted_at IS null", merchant.ID, getParameter["itemSlug"]).Scan(&item); result.Error != nil {
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
	if result := initializers.DB.Raw("SELECT date_series::date AS rent_date, (items.qty > COALESCE(SUM (project_item.qty), 0))::boolean AS available, items.qty AS item_qty, SUM (project_item.qty) AS project_qty FROM project p JOIN project_item on p.id = project_item.project_id JOIN items ON project_item.item_id = items.id JOIN generate_series(lower(p.range_date), upper(p.range_date), '1 day'::interval) AS date_series  ON date_series::date >= ?::date AND date_series::date <= ?::date  WHERE project_item.item_id = ? GROUP BY date_series::date, items.qty ORDER BY date_series::date;", getParameter["dateStart"], getParameter["dateEnd"], item.ID).Scan(&projects); result.Error != nil {
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

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":   "200",
		"status": "OK",
		"data": fiber.Map{
			"item_qty": itemAvailable,
		},
	}, "application/vnd.api+json")
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
