package models

import "time"

type Payment struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	BookingID  uint      `gorm:"index;not null;unique" json:"booking_id"`
	Method     string    `gorm:"size:50" json:"method"`
	ProviderTx string    `gorm:"size:200" json:"provider_tx,omitempty"` // e.g., payment gateway reference
	Amount     float64   `gorm:"type:decimal(10,2)" json:"amount"`
	Status     string    `gorm:"size:50;default:'completed'" json:"status"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
