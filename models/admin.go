package models

import (
	"time"

	"gorm.io/gorm"
)

type Admin struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	FullName     string         `gorm:"not null" json:"full_name"`
	Email        string         `gorm:"uniqueIndex;not null" json:"email"`
	Password     string         `gorm:"not null" json:"-"` // bcrypt hash
	Role         string         `gorm:"type:varchar(50);default:'admin'" json:"role"`
	RefreshToken string         `json:"refresh_token"`
	Blocked      bool           `gorm:"default:false" json:"blocked"`
	Deleted      bool           `gorm:"default:false" json:"deleted"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}
