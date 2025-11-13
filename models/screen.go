package models

import "time"

type Screen struct {
	ID         uint    `gorm:"primaryKey"`
	Name       string  `gorm:"not null" json:"name"`
	SeatsTotal int     `json:"seats_total"`
	TheatreID  uint    `gorm:"index;not null" json:"theatre_id"`
	Theatre    Theatre `gorm:"foreignKey:TheatreID" json:"theatre"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
