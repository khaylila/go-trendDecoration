package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	FirstName string `gorm:"size:128"`
	LastName string `gorm:"size:128"`
	Email string `gorm:"unique;size:256"`
	Password string `gorm:"size:256"`
	IsActive bool
	IsBanned bool
	Message string `gorm:"size:128"`
	Role []Role `gorm:"many2many:user_role;foreignKey:ID;references:ID"`
}