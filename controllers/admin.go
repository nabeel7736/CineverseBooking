package controllers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"cineverse/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Helper functions to get totals from DB
func GetTotalUsersFromDB(db *gorm.DB) int64 {
	var count int64
	db.Model(&models.User{}).Count(&count)
	return count
}

func GetTotalMoviesFromDB(db *gorm.DB) int64 {
	var count int64
	db.Model(&models.Movie{}).Count(&count)
	return count
}

func GetTotalBookingsFromDB(db *gorm.DB) int64 {
	var count int64
	db.Model(&models.Booking{}).Count(&count)
	return count
}

// Admin: Add Movie
func AdminAddMovie(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var m models.Movie
		if err := c.ShouldBind(&m); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if strings.TrimSpace(m.Title) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
			return
		}
		if err := db.Create(&m).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"movie": m})
	}
}

// Admin: List all bookings (with optional status filter)
func GetAllBookings(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var bookings []models.Booking

		search := c.Query("search")
		status := c.Query("status")

		query := db.Preload("User").Preload("Seats").Preload("Payment").
			Preload("Show.Movie").Preload("Show.Screen.Theatre")

		if status != "" {
			query = query.Where("status = ?", status)
		}

		if search != "" {
			query = query.Joins("JOIN users u ON u.id = bookings.user_id").
				Where("LOWER(u.full_name) LIKE LOWER(?) OR LOWER(u.email) LIKE LOWER(?)", "%"+search+"%", "%"+search+"%")
		}

		if err := query.Order("bookings.created_at DESC").Find(&bookings).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"bookings": bookings})
	}
}

// Admin: Update booking status
func UpdateBookingStatus(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		id := c.Param("id")

		var body struct {
			Status string `json:"status"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.Status == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		var booking models.Booking
		if err := db.First(&booking, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
			return
		}

		oldStatus := booking.Status

		tx := db.Begin()
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()

		booking.Status = body.Status
		if err := tx.Save(&booking).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update booking"})
			return
		}

		if oldStatus != "confirmed" && body.Status == "confirmed" {
			// Transition to confirmed: Increment count
			var show models.Show
			if err := tx.First(&show, booking.ShowID).Error; err == nil {
				show.SeatsBooked += booking.SeatsCount
				tx.Save(&show)
			}
		} else if oldStatus == "confirmed" && body.Status != "confirmed" {
			// Transition from confirmed: Decrement count
			var show models.Show
			if err := tx.First(&show, booking.ShowID).Error; err == nil {
				if show.SeatsBooked >= booking.SeatsCount { // Prevent negative count
					show.SeatsBooked -= booking.SeatsCount
					tx.Save(&show)
				}
			}
		}

		if err := tx.Commit().Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Booking status updated successfully", "booking": booking})
	}
}

// DeleteBooking — Admin: delete booking
func DeleteBooking(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idParam := c.Param("id")
		id, err := strconv.Atoi(idParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid booking ID"})
			return
		}

		tx := db.Begin()
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Unexpected server error"})
			}
		}()

		var booking models.Booking
		if err := tx.Preload("Payment").First(&booking, id).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
			return
		}

		// Restrict deletion for completed or paid bookings
		status := strings.ToLower(booking.Status)
		paymentStatus := ""
		if booking.Payment != nil {
			paymentStatus = strings.ToLower(booking.Payment.Status)
		}

		if status == "confirmed" || paymentStatus == "completed" {
			tx.Rollback()
			c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete a confirmed or paid booking"})
			return
		}

		// Delete associated seats
		if err := tx.Where("booking_id = ?", id).Delete(&models.BookingSeat{}).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete booking seats"})
			return
		}

		// Delete the booking itself
		if err := tx.Delete(&models.Booking{}, id).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete booking"})
			return
		}

		if err := tx.Commit().Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Booking deleted successfully"})
	}
}

func GetBookingDetails(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var booking models.Booking

		// Fetch booking with all related data
		if err := db.Preload("User").
			Preload("Seats").
			Preload("Payment").
			Preload("Show.Movie").
			Preload("Show.Screen.Theatre").
			First(&booking, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"booking": booking,
		})
	}
}

// Admin: List Movies
func AdminListMovies(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var movies []models.Movie
		if err := db.Find(&movies).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"movies": movies})
	}
}

// Admin: Delete Movie
func AdminDeleteMovie(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Movie ID"})
			return
		}

		var movie models.Movie
		if err := db.First(&movie, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Movie not Found"})
			return
		}

		if err := db.Where("movie_id = ?", id).Delete(&models.Show{}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete related shows"})
			return
		}

		if err := db.Unscoped().Delete(&movie, id).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		var movies []models.Movie
		db.Find(&movies)
		c.JSON(http.StatusOK, gin.H{"message": "movie deleted", "movies": movies})
	}
}

func AdminAddShow(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Define a temporary struct for parsing JSON
		var payload struct {
			MovieID    uint    `json:"movie_id"`
			TheatreID  uint    `json:"theatre_id"`
			ScreenID   uint    `json:"screen_id"`
			StartTime  string  `json:"start_time"` // receive as string
			Language   string  `json:"language"`
			Price      float64 `json:"price"`
			SeatsTotal int     `json:"seats_total"`
		}

		// Bind the payload
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate Movie
		if payload.MovieID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Movie ID is required"})
			return
		}
		var movie models.Movie
		if err := db.First(&movie, payload.MovieID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid movie ID"})
			return
		}

		// Validate Screen
		if payload.ScreenID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Screen ID is required"})
			return
		}
		var screen models.Screen
		if err := db.Preload("Theatre").First(&screen, payload.ScreenID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid screen ID"})
			return
		}

		// Parse Start Time
		startTime, err := time.Parse(time.RFC3339, payload.StartTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_time format. Must be RFC3339."})
			return
		}

		// Create the show
		show := models.Show{
			MovieID:     payload.MovieID,
			ScreenID:    payload.ScreenID,
			StartTime:   startTime,
			Language:    payload.Language,
			Price:       payload.Price,
			SeatsTotal:  payload.SeatsTotal,
			SeatsBooked: 0,
		}

		// Save to DB
		if err := db.Create(&show).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Reload with relations
		if err := db.Preload("Movie").
			Preload("Screen.Theatre").
			First(&show, show.ID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load created show"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "Show added successfully",
			"show":    show,
		})
	}
}

func AdminListShows(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var shows []models.Show
		if err := db.Preload("Movie").Preload("Screen.Theatre").Find(&shows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		var formatted []gin.H
		for _, s := range shows {
			// Fetch the specific booked seat codes, but only for CONFIRMED bookings
			var confirmedSeats []models.BookingSeat
			db.Table("booking_seats").
				Select("booking_seats.seat_code").
				Joins("JOIN bookings ON bookings.id = booking_seats.booking_id").
				Where("booking_seats.show_id = ? AND bookings.status = ?", s.ID, "confirmed").
				Scan(&confirmedSeats) // Scan only the seat_code

			var bookedSeatCodes []string
			for _, seat := range confirmedSeats {
				bookedSeatCodes = append(bookedSeatCodes, seat.SeatCode)
			}

			formatted = append(formatted, gin.H{
				"id":                s.ID,
				"movie_title":       s.Movie.Title,
				"theatre":           s.Screen.Theatre.Name,
				"screen":            s.Screen.Name,
				"date":              s.StartTime,
				"language":          s.Language,
				"seats_total":       s.SeatsTotal,
				"seats_booked":      s.SeatsBooked, // Note: This field may count pending bookings, but the visual preview below won't.
				"time":              s.StartTime,
				"available_seats":   s.SeatsTotal - s.SeatsBooked,
				"price":             s.Price,
				"booked_seat_codes": bookedSeatCodes, // <<-- NOW CONTAINS ONLY CONFIRMED SEATS
			})
		}
		c.JSON(http.StatusOK, gin.H{"shows": formatted})
	}
}

// Admin: Delete Show
func AdminDeleteShow(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, _ := strconv.Atoi(idStr)
		var show models.Show
		if err := db.First(&show, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "show not found"})
			return
		}
		if err := db.Delete(&models.Show{}, id).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "show deleted"})
	}
}

// Admin: Edit Show
func AdminEditShow(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid show ID"})
			return
		}

		var payload struct {
			MovieID   uint      `json:"movie_id" form:"movie_id"`
			StartTime time.Time `json:"start_time" form:"start_time"`
			Seats     int       `json:"seats_total" form:"seats_total"`
			Price     float64   `json:"price" form:"price"`
		}

		if err := c.ShouldBind(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var show models.Show
		if err := db.First(&show, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Show not found"})
			return
		}

		// ✅ Update only provided fields
		if payload.MovieID != 0 {
			show.MovieID = payload.MovieID
		}
		if !payload.StartTime.IsZero() {
			show.StartTime = payload.StartTime
		}
		if payload.Seats != 0 {
			show.SeatsTotal = payload.Seats
		}
		if payload.Price != 0 {
			show.Price = payload.Price
		}

		if err := db.Save(&show).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Show updated successfully",
			"show":    show,
		})
	}
}

// Admin Dashboard
func AdminDashboard(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		totalUsers := GetTotalUsersFromDB(db)
		totalMovies := GetTotalMoviesFromDB(db)
		totalBookings := GetTotalBookingsFromDB(db)

		token := c.Query("token")
		if token == "" {
			token = c.GetHeader("Authorization")
			if strings.HasPrefix(token, "Bearer ") {
				token = strings.TrimPrefix(token, "Bearer ")
			}
		}

		c.HTML(http.StatusOK, "admin_dashboard.html", gin.H{
			"TotalUsers":    totalUsers,
			"TotalMovies":   totalMovies,
			"TotalBookings": totalBookings,
			"Token":         token,
		})
	}
}

// Fetch all theatres
func GetAllTheatres(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var theatres []models.Theatre
		if err := db.Preload("Screens").Find(&theatres).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch theatres"})
			return
		}
		c.JSON(http.StatusOK, theatres)
	}
}

// Fetch screens by theatre ID
func GetScreensByTheatre(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var screens []models.Screen
		if err := db.Where("theatre_id = ?", id).Find(&screens).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch screens"})
			return
		}
		c.JSON(http.StatusOK, screens)
	}
}
