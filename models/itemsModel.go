package models

import "gorm.io/gorm"

type Items struct {
	gorm.Model
	Name        string `gorm:"size:128"`
	Slug        string `gorm:"type:text;"`
	Description string `gorm:"type:text;not null"`
	Qty         uint

	// merchantID
	MerchantID uint
	Merchant   Merchant `gorm:"foreignKey:MerchantID"`

	// img
	Image []Image `gorm:"foreignKey:ItemsID;constraint:OnUpdate:NO ACTION,OnDelete:CASCADE;"`
}
