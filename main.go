package main

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
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

	// Initialize default config
	// app.Use(cors.New())

	// Or extend your config for customization
	app.Use(cors.New(cors.Config{
		AllowOrigins: "http://127.0.0.1:8080, http://localhost:8080",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	// testing
	app.Get("/validate", middleware.RequireAuth, controllers.Validate)
	app.Post("/profile", middleware.RequireAuth, controllers.UserProfile)

	app.Get("/event", controllers.CreateEvent)

	// get image
	app.Get("/img/:name", controllers.GetImg)

	// client area
	app.Post("/signup", controllers.SignUp)
	app.Post("/login", controllers.Login)
	app.Post("/login/check", middleware.RequireAuth, controllers.CheckLogin)
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
	// listSeller
	app.Get("/seller", middleware.RequireAuth, middleware.CheckRole(config.ADMINISTRATOR), controllers.ListSeller)
	app.Get("/seller/:id", middleware.RequireAuth, middleware.CheckRole(config.ADMINISTRATOR), controllers.DetailSeller)

	// admin area
	app.Post("/seller/reset/:id", middleware.RequireAuth, middleware.CheckRole(config.ADMINISTRATOR), controllers.ResetPassword)
	app.Post("/seller", middleware.RequireAuth, middleware.CheckRole(config.ADMINISTRATOR), controllers.RegisterSeller)
	app.Put("/seller", middleware.RequireAuth, middleware.CheckRole(config.ADMINISTRATOR), controllers.UpdateSeller)

	// project
	app.Get("/projects", controllers.ListProject)

	// get items by merchant
	app.Get("/:merchant", controllers.ListItemFromMerchant)
	app.Get("/:merchant/:itemSlug", controllers.DetailItemWithSlug)

	app.Listen(":" + os.Getenv("PORT"))
}
