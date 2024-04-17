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
	//
	Rent       uint    `json:"rent"`
	Rating     float32 `json:"rating"`
	UserRating uint    `json:"user_rating"`

	// merchantID
	MerchantID uint
	Merchant   Merchant `gorm:"foreignKey:MerchantID" json:"merchant"`

	// img
	Image []Image `gorm:"foreignKey:ItemsID;constraint:OnUpdate:NO ACTION,OnDelete:CASCADE;" json:"images"`
}
