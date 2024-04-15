package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	FirstName string `gorm:"size:128" json:"first_name"`
	LastName  string `gorm:"size:128" json:"last_name"`
	Email     string `gorm:"unique;size:256" json:"email"`
	Password  string `gorm:"size:256" json:"password"`
	IsActive  bool   `json:"is_active"`
	IsBanned  bool   `json:"is_banned"`
	Message   string `gorm:"size:128" json:"message"`
	Role      []Role `gorm:"many2many:user_role;foreignKey:ID;references:ID" json:"role"`
	Avatar    string
}
