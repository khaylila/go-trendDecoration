package models

type ProjectItem struct {
	ID        uint `gorm:"column:id"`
	ProjectID uint `gorm:"column:project_id"`
	Project   Project
	ItemID    uint `gorm:"column:item_id"`
	Item      Items
	Qty       uint `gorm:"column:qty"`
	Price     uint `gorm:"column:price"`
}
