package models

import "gorm.io/gorm"

type Items struct {
	gorm.Model
	Name        string `gorm:"size:128" json:"name"`
	Slug        string `gorm:"type:text;" json:"slug"`
	Description string `gorm:"type:text;not null" json:"description"`
	Qty         uint   `json:"qty"`
	Price       uint   `json:"price"`
	OnGoing     uint   `json:"on_going"`
	Closed      uint   `json:"closed"`

	// merchantID
	MerchantID uint
	Merchant   Merchant `gorm:"foreignKey:MerchantID"`

	// img
	Image []Image `gorm:"foreignKey:ItemsID;constraint:OnUpdate:NO ACTION,OnDelete:CASCADE;" json:"images"`
}
