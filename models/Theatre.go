package models

import "time"

type Theatre struct {
	ID                  uint     `gorm:"primaryKey" json:"id"`
	Name                string   `gorm:"not null" json:"name"`
	Location            string   `gorm:"not null" json:"location"`
	Screens             []Screen `gorm:"foreignKey:TheatreID" json:"screens"`
	ParkingAvailable    bool     `json:"parking_available" gorm:"default:false"`
	CarParkingFee       float64  `json:"car_parking_fee" gorm:"type:decimal(10,2);default:0.0"`
	BikeParkingFee      float64  `json:"bike_parking_fee" gorm:"type:decimal(10,2);default:0.0"`
	CarParkingCapacity  int      `json:"car_parking_capacity" gorm:"default:0"`
	BikeParkingCapacity int      `json:"bike_parking_capacity" gorm:"default:0"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
