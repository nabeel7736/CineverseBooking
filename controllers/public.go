package controllers

import (
	"net/http"

	"cineverse/config"
	"cineverse/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetMovies(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var movies []models.Movie
		if err := config.DB.Find(&movies).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch movies"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"movies": movies})
	}
}

func GetMovieDetails(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var movie models.Movie
		if err := db.Preload("Shows").First(&movie, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "movie not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"movie": movie})
	}
}

func GetShowsByMovie(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		id := c.Param("id")
		var movie models.Movie
		var shows []models.Show

		if err := config.DB.First(&movie, id).Error; err != nil {
			c.String(404, "Movie not found")
			return
		}
		config.DB.Where("movie_id = ?", id).Find(&shows)

		c.HTML(http.StatusOK, "public_movie_details.html", gin.H{
			"Movie": movie,
			"Shows": shows,
		})
	}
}

func BookTickets(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var payload struct {
			ShowID uint `json:"show_id" binding:"required"`
			Seats  int  `json:"seats" binding:"required,min=1"`
		}
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// get user id from context (set by AuthMiddleware)
		uidv, _ := c.Get("user_id")
		userID := uidv.(uint)

		var show models.Show
		if err := db.First(&show, payload.ShowID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "show not found"})
			return
		}
		available := show.SeatsTotal - show.SeatsBooked
		if payload.Seats > available {
			c.JSON(http.StatusBadRequest, gin.H{"error": "not enough seats", "available": available})
			return
		}
		// transaction: create booking and update show.SeatsBooked
		err := db.Transaction(func(tx *gorm.DB) error {
			booking := models.Booking{
				UserID:      userID,
				ShowID:      show.ID,
				SeatsCount:  payload.Seats,
				TotalAmount: float64(payload.Seats) * show.Price,
				Status:      "confirmed",
			}
			if err := tx.Create(&booking).Error; err != nil {
				return err
			}
			show.SeatsBooked += payload.Seats
			if err := tx.Save(&show).Error; err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"message": "booking confirmed"})
	}
}

func GetUserBookings(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		uidv, _ := c.Get("user_id")
		userID := uidv.(uint)
		var bookings []models.Booking
		if err := db.Preload("Show").Preload("Show.Movie").Where("user_id = ?", userID).Find(&bookings).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"bookings": bookings})
	}
}
