package controllers

import (
	"cineverse/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetShowSeats(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		showID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid show ID"})
			return
		}

		//  Get show info
		var show models.Show
		if err := db.First(&show, showID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Show not found"})
			return
		}

		//  Fetch all booked seats for this show (NOTE: All seats are fetched here regardless of booking status for seat map display)
		var bookedSeats []models.BookingSeat
		if err := db.Where("show_id = ?", showID).Find(&bookedSeats).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch booked seats"})
			return
		}

		//  Build booked seat map for fast lookup
		bookedMap := make(map[string]bool)
		for _, s := range bookedSeats {
			bookedMap[s.SeatCode] = true
		}

		// Build seat layout dynamically: UNIFIED LOGIC
		// Use a consistent, large naming convention (A-J, 1-12 max) and stop at show.SeatsTotal.
		rows := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
		maxColsPerRow := 12
		layout := []map[string]interface{}{}

		seatIndex := 0
		seatsToGenerate := show.SeatsTotal // Use the actual total seats from the show model

		for _, row := range rows {
			if seatIndex >= seatsToGenerate {
				break // Stop generating rows if capacity is reached
			}
			rowSeats := []map[string]interface{}{}
			for i := 1; i <= maxColsPerRow; i++ {
				if seatIndex >= seatsToGenerate {
					break // Stop generating seats in this row if capacity is reached
				}

				// Generate seat code using the new convention (e.g., A1, A2, B1...)
				code := row + strconv.Itoa(i)
				status := "available"
				if bookedMap[code] {
					status = "booked"
				}

				rowSeats = append(rowSeats, gin.H{
					"seat_code": code,
					"status":    status,
					"price":     show.Price,
				})
				seatIndex++ // Increment the counter for every seat generated
			}
			layout = append(layout, gin.H{"row": row, "seats": rowSeats})
		}

		// Return the layout
		c.JSON(http.StatusOK, gin.H{
			"show_id":     show.ID,
			"price":       show.Price,
			"seat_layout": layout,
		})
	}
}

// func GetShowSeats(db *gorm.DB) gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		showID, err := strconv.Atoi(c.Param("id"))
// 		if err != nil {
// 			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid show ID"})
// 			return
// 		}

// 		//  Get show info
// 		var show models.Show
// 		if err := db.First(&show, showID).Error; err != nil {
// 			c.JSON(http.StatusNotFound, gin.H{"error": "Show not found"})
// 			return
// 		}

// 		//  Fetch all booked seats for this show
// 		var bookedSeats []models.BookingSeat
// 		if err := db.Where("show_id = ?", showID).Find(&bookedSeats).Error; err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch booked seats"})
// 			return
// 		}

// 		//  Build booked seat map for fast lookup
// 		bookedMap := make(map[string]bool)
// 		for _, s := range bookedSeats {
// 			bookedMap[s.SeatCode] = true
// 		}

// 		// Build seat layout dynamically (rows A–E, 10 seats each)
// 		rows := []string{"A", "B", "C", "D", "E" ,"F"}
// 		totalSeats := 10
// 		layout := []map[string]interface{}{}

// 		for _, row := range rows {
// 			rowSeats := []map[string]interface{}{}
// 			for i := 1; i <= totalSeats; i++ {
// 				code := row + strconv.Itoa(i)
// 				status := "available"
// 				if bookedMap[code] {
// 					status = "booked"
// 				}

// 				rowSeats = append(rowSeats, gin.H{
// 					"seat_code": code,
// 					"status":    status,
// 					"price":     show.Price,
// 				})
// 			}
// 			layout = append(layout, gin.H{"row": row, "seats": rowSeats})
// 		}

// 		// Return the layout
// 		c.JSON(http.StatusOK, gin.H{
// 			"show_id": show.ID,
// 			// "hall":        show.Hall,
// 			"price":       show.Price,
// 			"seat_layout": layout,
// 		})
// 	}
// }
