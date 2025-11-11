package controllers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
	db.Model(&models.Booking{}).Where("status = ?", "confirmed").Count(&count)
	return count
}

func AdminAddMovie(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse multipart form (needed for file upload)
		if err := c.Request.ParseMultipartForm(10 << 20); err != nil { // 10 MB max
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse form"})
			return
		}

		title := c.PostForm("title")
		description := c.PostForm("description")
		durationMin := c.PostForm("duration_min")

		if strings.TrimSpace(title) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Title is required"})
			return
		}

		// Handle poster upload
		var posterURL string
		file, err := c.FormFile("poster_file")
		if err == nil {
			uploadPath := "./uploads/posters/"
			if err := os.MkdirAll(uploadPath, os.ModePerm); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
				return
			}

			// Save file with unique name (timestamp to avoid collisions)
			filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(file.Filename))
			filePath := filepath.Join(uploadPath, filename)

			if err := c.SaveUploadedFile(file, filePath); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save poster image"})
				return
			}

			// Public URL for frontend
			posterURL = "/uploads/posters/" + filename
		} else if err != http.ErrMissingFile {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
			return
		}

		movie := models.Movie{
			Title:       title,
			Description: description,
			DurationMin: durationMin,
			PosterURL:   posterURL,
			ReleaseDate: nil,
		}

		if err := db.Create(&movie).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save movie"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "Movie added successfully",
			"movie":   movie,
		})
	}
}

// Admin: List all bookings
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
		id := c.Param("id")

		var movie models.Movie
		if err := db.First(&movie, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Movie not found"})
			return
		}

		// Delete poster file if it exists
		if movie.PosterURL != "" {
			// Convert relative URL (/uploads/posters/filename.jpg) → local file path
			filePath := "." + movie.PosterURL
			if _, err := os.Stat(filePath); err == nil {
				if err := os.Remove(filePath); err != nil {
					log.Printf(" Failed to delete poster file: %v", err)
				}
			}
		}

		// Delete movie record from DB
		if err := db.Delete(&movie).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete movie"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Movie deleted successfully"})
	}
}

func AdminAddShow(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var payload struct {
			MovieID    uint    `json:"movie_id"`
			TheatreID  uint    `json:"theatre_id"`
			ScreenID   uint    `json:"screen_id"`
			StartTime  string  `json:"start_time"`
			Language   string  `json:"language"`
			Price      float64 `json:"price"`
			SeatsTotal int     `json:"seats_total"`
		}

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
			var confirmedSeats []models.BookingSeat
			db.Table("booking_seats").
				Select("booking_seats.seat_code").
				Joins("JOIN bookings ON bookings.id = booking_seats.booking_id").
				Where("booking_seats.show_id = ? AND bookings.status = ?", s.ID, "confirmed").
				Scan(&confirmedSeats)

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
				"seats_booked":      s.SeatsBooked,
				"available_seats":   s.SeatsTotal - s.SeatsBooked,
				"price":             s.Price,
				"booked_seat_codes": bookedSeatCodes,
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

		var activeBookingsCount int64
		err := db.Model(&models.Booking{}).
			Where("show_id = ? AND status IN (?, ?)", id, "confirmed", "pending").
			Count(&activeBookingsCount).Error

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check for active bookings"})
			return
		}

		if activeBookingsCount > 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete show: active bookings exist for this show."})
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
			ScreenID  uint      `json:"screen_id"`
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

		var activeBookingsCount int64
		db.Model(&models.Booking{}).
			Where("show_id = ? AND status IN (?, ?)", id, "confirmed", "pending").
			Count(&activeBookingsCount)

		if activeBookingsCount > 0 {
			if (payload.ScreenID != 0 && payload.ScreenID != show.ScreenID) ||
				(payload.MovieID != 0 && payload.MovieID != show.MovieID) {
				c.JSON(http.StatusForbidden, gin.H{"error": "Cannot change movie or screen: active bookings exist."})
				return
			}

			if payload.Seats != 0 && payload.Seats < show.SeatsBooked {
				c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("Cannot reduce total seats below %d (currently booked).", show.SeatsBooked)})
				return
			}
		}

		if payload.ScreenID != 0 && payload.ScreenID != show.ScreenID {
			var screen models.Screen
			if err := db.First(&screen, payload.ScreenID).Error; err == nil {
				show.ScreenID = payload.ScreenID
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Screen ID provided"})
				return
			}
		}

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

		var adminFullName string = "Admin"
		if userIDRaw, exists := c.Get("userId"); exists {
			if userID, ok := userIDRaw.(uint); ok {
				var admin models.Admin
				if err := db.First(&admin, userID).Error; err == nil {
					adminFullName = admin.FullName
				}
			}
		}

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
			"AdminFullName": adminFullName,
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
