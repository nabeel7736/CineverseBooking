package controllers

import (
	"cineverse/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetAvailableSeats returns seats with booking status for a show
func GetAvailableSeats(db *gorm.DB) gin.HandlerFunc {
	// db := c.MustGet("db").(*gorm.DB)
	return func(c *gin.Context) {

		showID := c.Param("showId")

		var seats []models.Seat
		err := db.Raw(`
			SELECT s.*, 
				   CASE WHEN bs.id IS NULL THEN false ELSE true END AS booked
			FROM seats s
			LEFT JOIN booking_seats bs 
				ON bs.seat_id = s.id 
				AND bs.show_id = ? 
				AND bs.deleted_at IS NULL
		`, showID).Scan(&seats).Error

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"show_id": showID, "seats": seats})
	}
}
