package controllers

import (
	"cineverse/models"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var seatRows = []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}

const maxColsPerRow = 12

func isSeatCodeValid(seatCode string, seatsTotal int) bool {
	if len(seatCode) < 2 {
		return false
	}

	rowLetter := strings.ToUpper(seatCode[:1])
	colStr := seatCode[1:]

	col, err := strconv.Atoi(colStr)
	if err != nil || col <= 0 || col > maxColsPerRow {
		return false
	}

	rowIndex := -1
	for i, r := range seatRows {
		if r == rowLetter {
			rowIndex = i
			break
		}
	}

	if rowIndex == -1 {
		return false
	}

	absoluteSeatIndex := (rowIndex * maxColsPerRow) + col

	return absoluteSeatIndex <= seatsTotal
}

func CreateBooking(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			ShowID        uint     `json:"show_id"`
			SeatCodes     []string `json:"seat_codes"`
			PaymentMethod string   `json:"payment_method"`
			HasParking    bool     `json:"has_parking"`
			VehicleType   string   `json:"vehicle_type"`
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

		var show models.Show
		if err := db.First(&show, req.ShowID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Show not found"})
			return
		}
		if err := db.Preload("Screen.Theatre").First(&show, req.ShowID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Show or associated venue data not found"})
			return
		}

		var parkingFee float64 = 0.0
		var vehicleType string = ""
		var capacity int = 0
		var parkingAvailable = show.Screen.Theatre.ParkingAvailable

		if req.HasParking {
			if !parkingAvailable {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Parking not available at this theatre"})
				return
			}

			switch strings.ToLower(req.VehicleType) {
			case "car":
				parkingFee = show.Screen.Theatre.CarParkingFee
				vehicleType = "Car"
				capacity = show.Screen.Theatre.CarParkingCapacity
			case "bike":
				parkingFee = show.Screen.Theatre.BikeParkingFee
				vehicleType = "Bike"
				capacity = show.Screen.Theatre.BikeParkingCapacity
			default:
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid vehicle type for parking"})
				return
			}

			var currentBookedParking int64
			err := db.Model(&models.Booking{}).Where("show_id = ? AND vehicle_type = ? AND has_parking = ? AND status IN (?, ?)",
				show.ID, vehicleType, true, "confirmed", "pending").Count(&currentBookedParking).Error

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check Parking availability"})
				return
			}

			if capacity > 0 && int(currentBookedParking) >= capacity {
				c.JSON(http.StatusConflict, gin.H{"error": "Parking for " + vehicleType + " is full for this show"})
				return
			}

		} else if req.VehicleType != "" {
			req.VehicleType = ""
		}

		for _, code := range req.SeatCodes {
			if !isSeatCodeValid(code, show.SeatsTotal) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid seat code: " + code + ". This seat is not part of the screen layout."})
				return
			}
		}

		// Start transaction
		tx := db.Begin()
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()

		// Fetch already booked seats for this show, only considering confirmed or pending bookings
		var bookedSeats []models.BookingSeat
		if err := tx.Table("booking_seats").
			Select("booking_seats.seat_code").
			Joins("JOIN bookings ON bookings.id = booking_seats.booking_id").
			Where("booking_seats.show_id = ? AND bookings.status IN (?, ?)", show.ID, "confirmed", "pending"). // Filter by confirmed or pending status
			Find(&bookedSeats).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch currently booked seats"})
			return
		}

		// Map booked seat codes
		bookedMap := make(map[string]bool)
		for _, s := range bookedSeats {
			bookedMap[s.SeatCode] = true
		}

		// Validate seat codes against currently booked seats
		var bookingSeats []models.BookingSeat
		for _, code := range req.SeatCodes {
			if bookedMap[code] {
				tx.Rollback()
				c.JSON(http.StatusConflict, gin.H{"error": "Seat " + code + " already booked by another active booking."})
				return
			}

			bookingSeats = append(bookingSeats, models.BookingSeat{
				ShowID:    show.ID,
				SeatCode:  code,
				Price:     show.Price,
				CreatedAt: time.Now(),
			})
		}

		// Calculate total: Seat Subtotal + Parking Fee
		seatSubtotal := float64(len(req.SeatCodes)) * show.Price
		totalAmount := seatSubtotal + parkingFee

		// Create booking
		booking := models.Booking{
			UserID:        userID,
			ShowID:        show.ID,
			SeatsCount:    len(req.SeatCodes),
			TotalAmount:   totalAmount,
			Status:        "pending",
			PaymentMethod: req.PaymentMethod,
			CreatedAt:     time.Now(),
			HasParking:    req.HasParking,
			VehicleType:   vehicleType,
			ParkingFee:    parkingFee,
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

		tx.Commit()

		// Fetch the booking with related fields
		var fullBooking models.Booking
		if err := db.Preload("User").
			Preload("Show").
			Preload("Show.Movie").
			Preload("Show.Screen.Theatre").
			Preload("Seats").
			First(&fullBooking, booking.ID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch booking details"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message":       "Booking created successfully",
			"booking":       fullBooking,
			"seats":         bookingSeats,
			"Total_Amount":  totalAmount,
			"seat_subtotal": seatSubtotal,
			"parking_fee":   parkingFee,
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
