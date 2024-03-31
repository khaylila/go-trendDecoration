package controllers

import "github.com/gofiber/fiber/v2"

func GetImg(c *fiber.Ctx) error {
	fileName := c.Params("name")
	return c.SendFile("./public/image/" + fileName)
}
