package models

import (
	"time"
)

type ProjectTimeline struct {
	ID          uint        `gorm:"column:id" json:"id"`
	MessageFrom int8        `gorm:"column:from" json:"msg_from"`
	Message     uint        `gorm:"column:message" json:"message"`
	Images      []string    `gorm:"column:images" json:"images"`
	ProjectID   uint        `gorm:"column:project_item_id" json:"project_id"`
	Project     ProjectItem `json:"project"`
	CreatedAt   time.Time   `gorm:"column:created_at" json:"created_at"`
}
