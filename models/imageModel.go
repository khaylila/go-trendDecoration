package models

type Image struct {
	ItemsID uint
	Title   string `gorm:"size:128"`
}
