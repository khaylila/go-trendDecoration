package models

import "gorm.io/gorm"

type Merchant struct {
	gorm.Model
	Name        string `gorm:"size:128" json:"name"`
	Slug        string `gorm:"size:256" json:"slug"`
	Address     string `gorm:"size:128" json:"address"`
	PhoneNumber string `gorm:"size:20" json:"phone_number"`
	Avatar      string `json:"avatar"`

	// userID
	UserID uint `json:"user_id"`
	User   User `gorm:"foreignKey:UserID" json:"user"`
}
