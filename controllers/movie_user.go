package controllers

import (
	"cineverse/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetAllMovies(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		var movies []models.Movie

		if err := db.Preload("Shows").Find(&movies).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch movies"})
			return
		}

		c.JSON(http.StatusOK, movies)
	}
}

func GetMovieWithShows(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		id := c.Param("id")

		var movie models.Movie
		if err := db.Preload("Shows").First(&movie, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Movie not found"})
			return
		}

		c.JSON(http.StatusOK, movie)
	}
}

func GetUpcomingShows(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		var shows []models.Show

		today := time.Now()
		if err := db.Preload("Movie").Where("start_time >= ?", today).Find(&shows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch upcoming shows"})
			return
		}

		c.JSON(http.StatusOK, shows)
	}
}
