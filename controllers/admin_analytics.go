package controllers

import (
	"cineverse/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AnalyticsController struct {
	DB *gorm.DB
}

func (ac *AnalyticsController) GetDailyRevenue(c *gin.Context) {
	type RevenueData struct {
		Date    string  `json:"date"`
		Revenue float64 `json:"revenue"`
	}

	var results []RevenueData

	ac.DB.
		Model(&models.Booking{}).
		Select("TO_CHAR(created_at, 'YYYY-MM-DD') as date, SUM(total_amount) as revenue").
		Where("status = ?", "confirmed").
		Group("TO_CHAR(created_at, 'YYYY-MM-DD')").
		Order("TO_CHAR(created_at, 'YYYY-MM-DD') ASC").
		Limit(7).
		Scan(&results)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    results,
	})
}

func (ac *AnalyticsController) GetBookingsPerMovie(c *gin.Context) {
	type MovieData struct {
		Movie    string `json:"movie"`
		Bookings int64  `json:"bookings"`
	}

	var results []MovieData

	ac.DB.Table("bookings").
		Select("movies.title as movie, COUNT(bookings.id) as bookings").
		Joins("JOIN shows ON shows.id = bookings.show_id").
		Joins("JOIN movies ON movies.id = shows.movie_id").
		Group("movies.title").
		Order("bookings DESC").
		Limit(7).
		Scan(&results)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    results,
	})
}

func (ac *AnalyticsController) GetUserActivity(c *gin.Context) {
	type UserData struct {
		User     string `json:"user"`
		Bookings int64  `json:"bookings"`
	}

	var results []UserData

	ac.DB.Table("bookings").
		Select("users.full_name as user, COUNT(bookings.id) as bookings").
		Joins("JOIN users ON users.id = bookings.user_id").
		Group("users.full_name").
		Order("bookings DESC").
		Limit(5).
		Scan(&results)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    results,
	})
}

func (ac *AnalyticsController) GetDashboardStats(c *gin.Context) {
	var userCount, movieCount, bookingCount int64

	ac.DB.Model(&models.User{}).Count(&userCount)
	ac.DB.Model(&models.Movie{}).Count(&movieCount)
	ac.DB.Model(&models.Booking{}).Count(&bookingCount)

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"total_users":    userCount,
		"total_movies":   movieCount,
		"total_bookings": bookingCount,
	})
}
