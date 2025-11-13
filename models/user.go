package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	FullName      string         `gorm:"not null" json:"full_name"`
	Email         string         `gorm:"uniqueIndex;not null" json:"email"`
	Password      string         `gorm:"not null" json:"-"` // bcrypt hash
	RefreshToken  string         `json:"refresh_token"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	Blocked       bool           `gorm:"default:false" json:"blocked"`
	Deleted       bool           `gorm:"default:false" json:"deleted"`
	Bookings      []Booking      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	BookingsCount int64          `gorm:"-" json:"bookings_count,omitempty"`
	Wishlist      []Wishlist     `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}
