package models

type Payment struct {
	ID      uint   `gorm:"column:id"`
	UserId  uint   `gorm:"column:user_id"`
	Amount  uint   `gorm:"column:amount"`
	Status  int8   `gorm:"column:status"`
	SnapURL string `gorm:"column:snap_url"`
}
