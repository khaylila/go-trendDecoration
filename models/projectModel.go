package models

import (
	"time"

	"github.com/khaylila/go-trendDecoration/config"
)

type Project struct {
	ID         uint             `gorm:"column:id" json:"id"`
	Invoice    string           `gorm:"column:invoice" json:"invoice"`
	RangeDate  config.DateRange `gorm:"column:range_date" json:"range_date"`
	Address    string           `gorm:"column:address" json:"address"`
	Status     string           `gorm:"column:status" json:"status"`
	Confirm    string           `gorm:"column:confirm" json:"confirm"`
	CreatedAt  time.Time        `gorm:"column:created_at" json:"created_at"`
	UpdatedAt  time.Time        `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt  time.Time        `gorm:"column:deleted_at" json:"deleted_at"`
	MerchantID uint             `gorm:"column:merchant_id" json:"merchant_id"`
	Merchant   Merchant         `json:"merchant"`
	UserID     uint             `gorm:"column:user_id" json:"user_id"`
	User       User             `json:"user"`
}
