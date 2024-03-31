package models

import (
	"github.com/khaylila/go-trendDecoration/config"
)

type Event struct {
	ID   uint
	Name string
	Date config.DateRange `gorm:"type:daterange"`
}
