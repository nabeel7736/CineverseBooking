package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	FullName     string `gorm:"not null" json:"full_name"`
	Email        string `gorm:"uniqueIndex;not null" json:"email"`
	Role         string `gorm:"type:varchar(50);default:user;not null" json:"role"`
	Password     string `gorm:"not null" json:"-"` // stored as bcrypt hash
	IsAdmin      bool   `gorm:"default:false"`
	RefreshToken string `json:"refresh_token"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

type Movie struct {
	ID          uint      `gorm:"primaryKey"`
	Title       string    `gorm:"not null" json:"title"`
	Description string    `gorm:"type:text" json:"description"`
	DurationMin int       `json:"duration_min"` // duration in minutes
	ReleaseDate time.Time `gorm:"type:timestamp" json:"release_date"`
	PosterURL   string    `json:"posterUrl"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
	Shows       []Show
}

// type Show struct {
// 	ID          uint      `gorm:"primaryKey"`
// 	MovieID     int       `gorm:"index;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"movie_id"`
// 	Movie       Movie     `gorm:"foreignKey:MovieID" json:"movie"`
// 	Hall        string    `json:"hall"`
// 	StartTime   time.Time `json:"start_time"`
// 	SeatsTotal  int       `json:"seats_total"`
// 	SeatsBooked int       `json:"seats_booked"`
// 	Price       float64   `json:"price"`
// 	CreatedAt   time.Time
// 	UpdatedAt   time.Time
// 	DeletedAt   gorm.DeletedAt `gorm:"index"`
// }

type Show struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	MovieID     int            `gorm:"index;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"movie_id"`
	Movie       Movie          `gorm:"foreignKey:MovieID" json:"movie"`
	Hall        string         `json:"hall"`
	StartTime   time.Time      `json:"start_time"`
	SeatsTotal  int            `json:"seats_total"`
	SeatsBooked int            `json:"seats_booked"`
	Price       float64        `json:"price"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

type Booking struct {
	ID         uint `gorm:"primaryKey"`
	UserID     uint `gorm:"index;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user_id"`
	User       User
	ShowID     uint `gorm:"index;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"show_id"`
	Show       Show
	SeatsCount int     `json:"seats_count"`
	TotalPrice float64 `json:"total_price"`
	Status     string  `gorm:"type:varchar(20);default:'pending'" json:"status"` // e.g., "pending", "confirmed", "cancelled"
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
