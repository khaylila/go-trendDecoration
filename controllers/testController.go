package controllers

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/khaylila/go-trendDecoration/initializers"
	"github.com/khaylila/go-trendDecoration/models"
)

func CreateEvent(c *fiber.Ctx) error {
	var event []models.Event
	if result := initializers.DB.Exec("INSERT INTO event VALUES(?,?, ?);", 6, "Event 6", "['2024-03-10', '2024-03-15')"); result.Error != nil {
		fmt.Println(result.Error)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error parse event"})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"status": "Success created", "data": event})
}
