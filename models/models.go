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
type Movie struct {
	ID          uint       `gorm:"primaryKey"`
	Title       string     `gorm:"not null" json:"title"`
	Description string     `gorm:"type:text" json:"description"`
	DurationMin int        `json:"duration_min"` // duration in minutes
	ReleaseDate *time.Time `gorm:"type:timestamp" json:"release_date"`
	PosterURL   string     `json:"posterUrl"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
	Shows       []Show
}

type Theatre struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:200;not null" json:"name"`
	Location  string    `gorm:"size:300" json:"location,omitempty"`
	Halls     []Hall    `gorm:"foreignKey:TheatreID" json:"halls,omitempty"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type Hall struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	TheatreID  uint      `gorm:"index;not null" json:"theatre_id"`
	Name       string    `gorm:"size:100;not null" json:"name"`
	TotalSeats int       `json:"total_seats"`
	LayoutJSON string    `gorm:"type:json" json:"layout_json,omitempty"` // optional: store seat layout meta
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type Show struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	MovieID     int            `gorm:"index;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"movie_id"`
	Movie       Movie          `gorm:"foreignKey:MovieID" json:"movie"`
	HallID      uint           `gorm:"index;not null" json:"hall_id"`
	Hall        string         `json:"hall"`
	StartTime   time.Time      `gorm:"not null" json:"start_time"`
	SeatsTotal  int            `json:"seats_total"`
	SeatsBooked int            `json:"seats_booked"`
	Language    string         `gorm:"size:50" json:"language,omitempty"`
	Price       float64        `gorm:"type:decimal(10,2)" json:"price"`
	Seats       []BookingSeat  `gorm:"-" json:"-"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

type Booking struct {
	ID            uint `gorm:"primaryKey"`
	UserID        uint `gorm:"index;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user_id"`
	ShowID        uint `gorm:"index;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"show_id"`
	Show          Show
	SeatsCount    int           `json:"seats_count"`
	TotalAmount   float64       `gorm:"type:decimal(10,2)" json:"total_amount"`
	Status        string        `gorm:"type:varchar(20);default:'pending'" json:"status"` // e.g., "pending", "confirmed", "cancelled"
	PaymentMethod string        `gorm:"size:50" json:"payment_method"`
	CreatedAt     time.Time     `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time     `gorm:"autoUpdateTime" json:"updated_at"`
	Seats         []BookingSeat `gorm:"foreignKey:BookingID" json:"seats,omitempty"`
	Payment       *Payment      `gorm:"foreignKey:BookingID" json:"payment,omitempty"`
}

type BookingSeat struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	BookingID uint      `gorm:"not null" json:"booking_id"`
	ShowID    uint      `gorm:"not null" json:"show_id"`
	SeatCode  string    `gorm:"size:10;not null" json:"seat_code"`
	Price     float64   `json:"price"`
	CreatedAt time.Time `json:"created_at"`
}

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

type Wishlist struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"index;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user_id"`
	User      User      `gorm:"foreignKey:UserID" json:"user"`
	MovieID   uint      `gorm:"index;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"movie_id"`
	Movie     Movie     `gorm:"foreignKey:MovieID" json:"movie"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}
