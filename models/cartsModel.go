package models

import (
	"github.com/khaylila/go-trendDecoration/config"
)

type Carts struct {
	ID        uint             `gorm:"primaryKey"`
	RentRange config.DateRange `gorm:"type:daterange"`
	ItemID    uint
	Item      Items `gorm:"foreignKey:ItemID"`
	Qty       uint
	UserID    uint
	User      User `gorm:"foreignKey:UserID"`
}
