package initializers

import "github.com/khaylila/go-trendDecoration/models"

func SyncDatabase() {
	DB.AutoMigrate(&models.Merchant{}, &models.User{}, &models.Role{}, &models.Items{}, &models.Image{}, &models.Carts{})
}
