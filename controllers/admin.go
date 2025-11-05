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

// Admin: Create show

// Admin: List all bookings (with optional status filter)
func GetAllBookings(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var bookings []models.Booking

		search := c.Query("search")
		status := c.Query("status")

		query := db.Preload("User").Preload("Show").Preload("Seats").Preload("Payment").Order("created_at desc")

		if status != "" {
			query = query.Where("status = ?", status)
		}

		if search != "" {
			query = query.Joins("JOIN users ON users.id = bookings.user_id").
				Where("LOWER(users.name) LIKE LOWER(?) OR LOWER(users.email) LIKE LOWER(?)", "%"+search+"%", "%"+search+"%")
		}

		if err := query.Order("bookings.created_at DESC").Find(&bookings).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bookings"})
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

		booking.Status = body.Status
		if err := db.Save(&booking).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update booking"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Booking status updated successfully", "booking": booking})
	}
}

// DeleteBooking — Admin: delete booking
func DeleteBooking(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		id := c.Param("id")

		if err := db.Delete(&models.Booking{}, id).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete booking"})
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
			Preload("Show").
			Preload("Show.Theatre").
			Preload("Show.Movie").
			Preload("Seats").
			Preload("Payment").
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
		var payload struct {
			MovieID int     `json:"movie_id" form:"movie_id"`
			Hall    string  `json:"hall" form:"hall"`
			Start   string  `json:"start_time" form:"start_time"` // RFC3339 or custom parse
			Seats   int     `json:"seats_total" form:"seats_total"`
			Price   float64 `json:"price" form:"price"`
		}
		if err := c.ShouldBind(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		var movie models.Movie
		if err := db.First(&movie, payload.MovieID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid movie id"})
			return
		}

		// parse time
		// t, err := time.Parse("2006-01-02T15:04", payload.Start)
		// if err != nil {
		// 	c.JSON(http.StatusBadRequest, gin.H{"error": "invalid datetime format"})
		// 	return
		// }
		t, err := time.Parse(time.RFC3339, payload.Start)
		if err != nil {
			// fallback: handle "2006-01-02T15:04" (without seconds)
			t, err = time.Parse("2006-01-02T15:04", payload.Start)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid datetime format"})
				return
			}
		}

		show := models.Show{
			MovieID:     payload.MovieID,
			Hall:        payload.Hall,
			StartTime:   t,
			SeatsTotal:  payload.Seats,
			SeatsBooked: 0,
			Price:       payload.Price,
		}
		if err := db.Create(&show).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"show": show})
	}
}

// Admin: List Shows
func AdminListShows(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var shows []models.Show
		if err := db.Preload("Movie").Find(&shows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// Format for template
		var formatted []gin.H
		for _, s := range shows {
			formatted = append(formatted, gin.H{
				"id":              s.ID,
				"movie_title":     s.Movie.Title,
				"hall":            s.Hall,
				"date":            s.StartTime.Format("2006-01-02"),
				"time":            s.StartTime.Format("15:04"),
				"available_seats": s.SeatsTotal - s.SeatsBooked,
				"price":           s.Price,
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid show id"})
			return
		}

		var payload struct {
			MovieID int     `json:"movie_id" form:"movie_id"`
			Hall    string  `json:"hall" form:"hall"`
			Start   string  `json:"start_time" form:"start_time"`
			Seats   int     `json:"seats_total" form:"seats_total"`
			Price   float64 `json:"price" form:"price"`
		}
		if err := c.ShouldBind(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var show models.Show
		if err := db.First(&show, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "show not found"})
			return
		}

		// Optional: validate MovieID if changed
		if payload.MovieID != 0 && payload.MovieID != int(show.MovieID) {
			var movie models.Movie
			if err := db.First(&movie, payload.MovieID).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid movie id"})
				return
			}
			show.MovieID = payload.MovieID
		}

		// Parse and update start time if provided
		if payload.Start != "" {
			t, err := time.Parse(time.RFC3339, payload.Start)
			if err != nil {
				// fallback: handle "2006-01-02T15:04"
				t, err = time.Parse("2006-01-02T15:04", payload.Start)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "invalid datetime format"})
					return
				}
			}
			show.StartTime = t
		}

		// Update fields if provided
		if payload.Hall != "" {
			show.Hall = payload.Hall
		}
		if payload.Seats != 0 {
			show.SeatsTotal = payload.Seats
		}
		if payload.Price != 0 {
			show.Price = payload.Price
		}

		// Save updates
		if err := db.Save(&show).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "show updated successfully", "show": show})
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
