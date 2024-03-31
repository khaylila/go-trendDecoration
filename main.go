package main

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/khaylila/go-trendDecoration/config"
	"github.com/khaylila/go-trendDecoration/controllers"
	"github.com/khaylila/go-trendDecoration/initializers"
	"github.com/khaylila/go-trendDecoration/middleware"
)

func init() {
	initializers.LoadEnvVariable()
	initializers.ConnectToDb()
	// initializers.SyncDatabase()
}

func main() {
	app := fiber.New()

	// testing
	app.Get("/validate", middleware.RequireAuth, controllers.Validate)

	app.Get("/event", controllers.CreateEvent)

	// get image
	app.Get("/img/:name", controllers.GetImg)

	// client area
	app.Post("/signup", controllers.SignUp)
	app.Post("/login", controllers.Login)
	// app.Post("/reset-password", controllers)
	app.Get("/search", controllers.SearchItem)
	app.Post("/cart", middleware.RequireAuth, middleware.CheckRole(config.CUSTOMER), controllers.CheckChart, controllers.InsertToChart)

	// payment
	app.Post("/payment", middleware.RequireAuth, controllers.Transaction)
	// callback midtrans
	app.Post("/payment/verify", controllers.VerifyPayment)

	// seller area
	// crud seller items
	app.Get("/seller/items", middleware.RequireAuth, middleware.CheckRole(config.SELLER), controllers.ListItem)
	app.Get("/seller/items/:id", middleware.RequireAuth, middleware.CheckRole(config.SELLER), controllers.DetailItem)
	app.Put("/seller/items/:id", middleware.RequireAuth, middleware.CheckRole(config.SELLER), controllers.UpdateItem)
	app.Post("/seller/items", middleware.RequireAuth, middleware.CheckRole(config.SELLER), controllers.CreateNewItem)
	app.Delete("/seller/items/:id", middleware.RequireAuth, middleware.CheckRole(config.SELLER), controllers.RemoveItem)

	// admin area
	app.Post("/signup/seller", middleware.RequireAuth, middleware.CheckRole(config.ADMIN), controllers.RegisterSeller)

	// project
	app.Get("/projects", controllers.ListProject)

	// get items by merchant
	app.Get("/:merchant", controllers.ListItemFromMerchant)
	app.Get("/:merchant/:itemSlug", controllers.DetailItemWithSlug)

	app.Listen(":" + os.Getenv("PORT"))
}
