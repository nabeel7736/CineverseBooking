package models

import "time"

type BookingSeat struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	BookingID uint      `gorm:"not null" json:"booking_id"`
	ShowID    uint      `gorm:"not null" json:"show_id"`
	SeatCode  string    `gorm:"size:10;not null" json:"seat_code"`
	Price     float64   `json:"price"`
	CreatedAt time.Time `json:"created_at"`
}
