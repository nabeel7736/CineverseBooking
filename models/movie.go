package models

import (
	"time"

	"gorm.io/gorm"
)

type Movie struct {
	ID          uint       `gorm:"primaryKey"`
	Title       string     `gorm:"not null" json:"title"`
	Description string     `gorm:"type:text" json:"description"`
	DurationMin string     `json:"duration_min"` // duration in minutes
	ReleaseDate *time.Time `gorm:"type:timestamp" json:"release_date"`
	PosterURL   string     `json:"posterUrl"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
	Shows       []Show
}
