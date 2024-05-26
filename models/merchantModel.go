package models

import "gorm.io/gorm"

type Merchant struct {
	gorm.Model
	Name        string  `gorm:"size:128,column:name" json:"name"`
	Slug        string  `gorm:"size:256,column:slug" json:"slug"`
	Address     string  `gorm:"size:128,column:addres" json:"address"`
	PhoneNumber string  `gorm:"size:20,column:phone_number" json:"phone_number"`
	Avatar      string  `gorm:"column:avatar" json:"avatar"`
	Rating      float32 `json:"rating"`

	// userID
	UserID uint `json:"user_id"`
	User   User `gorm:"foreignKey:UserID" json:"user"`
}
