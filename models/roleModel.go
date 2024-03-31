package models

type Role struct {
	ID uint
	Role string `gorm:"size:64"`
	Description string `gorm:"size:128"`
}