package controllers

import (
	"log"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
)

func GetImg(c *fiber.Ctx) error {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}

	fileName := c.Params("name")

	imgPath := filepath.Join(dir, "/public/image/", fileName)
	return c.SendFile(imgPath)
}

func GetDir(path string) string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}

	return filepath.Join(dir, path)
}
