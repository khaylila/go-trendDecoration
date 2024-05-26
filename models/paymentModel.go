package models

import "time"

type Payment struct {
	ID             uint      `gorm:"column:id"`
	Amount         uint      `gorm:"column:amount"`
	Status         int8      `gorm:"column:status"`
	SnapURL        string    `gorm:"column:snap_url"`
	CreatedAt      time.Time `gorm:"column:created_at"`
	ExpiryTime     time.Time `gorm:"column:expiry_time"`
	SettlementTime time.Time `gorm:"column:settlement_time"`
	PaymentType    string    `gorm:"column:payment_type"`
	Bank           string    `gorm:"column:bank"`
	VaNumber       string    `gorm:"column:va_number"`
}
