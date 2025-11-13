package models

import "time"

type Wishlist struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"index;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user_id"`
	User      User      `gorm:"foreignKey:UserID" json:"user"`
	MovieID   uint      `gorm:"index;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"movie_id"`
	Movie     Movie     `gorm:"foreignKey:MovieID" json:"movie"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}
