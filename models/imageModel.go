package models

type Image struct {
	ItemsID uint   `json:"item_id"`
	Title   string `gorm:"size:128" json:"title"`
}
