package models

import "time"

type Booking struct {
	ID     uint `gorm:"primaryKey" json:"id"`
	UserID uint `gorm:"index;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user_id"`
	User   User `gorm:"foreignKey:UserID" json:"user"`

	ShowID uint `gorm:"index;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"show_id"`
	Show   Show `gorm:"foreignKey:ShowID" json:"show"`
	// StartTime     time.Time     `gorm:"not null" json:"start_time"`
	SeatsCount    int           `json:"seats_count"`
	TotalAmount   float64       `gorm:"type:decimal(10,2)" json:"total_amount"`
	Status        string        `gorm:"type:varchar(20);default:'pending';index" json:"status"` // e.g., "pending", "confirmed", "cancelled"
	PaymentMethod string        `gorm:"size:50" json:"payment_method"`
	HasParking    bool          `json:"has_parking" gorm:"default:false"`
	VehicleType   string        `json:"vehicle_type" gorm:"size:20"` // "Car" or "Bike"
	ParkingFee    float64       `json:"parking_fee" gorm:"type:decimal(10,2);default:0.0"`
	CreatedAt     time.Time     `gorm:"autoCreateTime;index" json:"created_at"`
	UpdatedAt     time.Time     `gorm:"autoUpdateTime" json:"updated_at"`
	Seats         []BookingSeat `gorm:"foreignKey:BookingID" json:"seats"`
	Payment       *Payment      `gorm:"foreignKey:BookingID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"payment"`
}
