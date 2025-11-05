package controllers

import (
	"cineverse/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CreateBooking(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			ShowID        uint     `json:"show_id"`
			SeatCodes     []string `json:"seat_codes"`
			PaymentMethod string   `json:"payment_method"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid booking data"})
			return
		}

		// Get logged-in user ID
		userIDRaw, exists := c.Get("userId")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		userID := userIDRaw.(uint)

		// Validate show
		var show models.Show
		if err := db.First(&show, req.ShowID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Show not found"})
			return
		}

		// Start transaction
		tx := db.Begin()
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()

		// Fetch already booked seats for this show
		var bookedSeats []models.BookingSeat
		if err := tx.Where("show_id = ?", show.ID).Find(&bookedSeats).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch booked seats"})
			return
		}

		// Map booked seat codes
		bookedMap := make(map[string]bool)
		for _, s := range bookedSeats {
			bookedMap[s.SeatCode] = true
		}

		// Validate seat codes
		var bookingSeats []models.BookingSeat
		for _, code := range req.SeatCodes {
			if bookedMap[code] {
				tx.Rollback()
				c.JSON(http.StatusConflict, gin.H{"error": "Seat " + code + " already booked"})
				return
			}

			bookingSeats = append(bookingSeats, models.BookingSeat{
				ShowID:    show.ID,
				SeatCode:  code,
				Price:     show.Price,
				CreatedAt: time.Now(),
			})
		}

		// Calculate total
		totalAmount := float64(len(req.SeatCodes)) * show.Price

		// Create booking
		booking := models.Booking{
			UserID:        userID,
			ShowID:        show.ID,
			SeatsCount:    len(req.SeatCodes),
			TotalAmount:   totalAmount,
			Status:        "pending",
			PaymentMethod: req.PaymentMethod,
			CreatedAt:     time.Now(),
		}

		if err := tx.Create(&booking).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create booking"})
			return
		}

		// Assign booking ID to seats
		for i := range bookingSeats {
			bookingSeats[i].BookingID = booking.ID
		}

		if err := tx.Create(&bookingSeats).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save booking seats"})
			return
		}

		// Update seats booked count
		show.SeatsBooked += len(req.SeatCodes)
		if err := tx.Save(&show).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update seat count"})
			return
		}

		tx.Commit()

		// Fetch the booking with related fields
		var fullBooking models.Booking
		if err := db.Preload("User").
			Preload("Show").
			Preload("Show.Movie").
			Preload("Seats").
			First(&fullBooking, booking.ID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch booking details"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message":  "Booking created successfully",
			"booking":  fullBooking,
			"seats":    bookingSeats,
			"subtotal": totalAmount,
		})
	}
}

func GetBookingDetailsUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		id := c.Param("id")

		var booking models.Booking
		if err := db.Preload("Seats").Preload("Show.Movie").First(&booking, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
			return
		}

		c.JSON(http.StatusOK, booking)
	}
}

func GetUserBookings(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		userIDRaw, exists := c.Get("userId")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		userID := userIDRaw.(uint)

		var bookings []models.Booking
		if err := db.Preload("Show.Movie").Where("user_id = ?", userID).Order("created_at desc").Find(&bookings).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user bookings"})
			return
		}

		c.JSON(http.StatusOK, bookings)
	}
}
