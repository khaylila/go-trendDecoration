package models

import "gorm.io/gorm"

type Merchant struct {
	gorm.Model
	Name        string `gorm:"size:128"`
	Slug        string `gorm:"size:256"`
	Address     string `gorm:"size:128"`
	PhoneNumber string `gorm:"size:20"`

	// userID
	UserID uint
	User   User `gorm:"foreignKey:UserID"`
}
