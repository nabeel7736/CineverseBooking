package models

import (
	"time"

	"gorm.io/gorm"
)

type Show struct {
	ID      uint  `gorm:"primaryKey" json:"id"`
	MovieID uint  `gorm:"index;not null" json:"movie_id"`
	Movie   Movie `gorm:"foreignKey:MovieID"`

	// ScreenID uint   `gorm:"index" json:"screen_id"`
	ScreenID uint   `gorm:"index;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"screen_id"`
	Screen   Screen `gorm:"foreignKey:ScreenID"`

	TheatreID uint    `gorm:"-" json:"theatre_id"` // for frontend convenience
	Theatre   Theatre `gorm:"-" json:"theatre"`

	StartTime   time.Time `form:"start_time" json:"start_time"`
	Language    string    `gorm:"size:50" json:"language"`
	Price       float64   `gorm:"type:decimal(10,2)" json:"price"`
	SeatsTotal  int       `json:"seats_total"`
	SeatsBooked int       `json:"seats_booked"`

	BookingSeat []BookingSeat  `gorm:"-" json:"-"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updates_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}
