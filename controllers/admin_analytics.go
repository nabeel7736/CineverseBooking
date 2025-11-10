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

type TheatreRevenueBreakdown struct {
	TheatreID      uint    `json:"theatre_id"`
	TheatreName    string  `json:"theatre_name"`
	ScreenID       uint    `json:"screen_id,omitempty"`
	ScreenName     string  `json:"screen_name,omitempty"`
	TotalRevenue   float64 `json:"total_revenue"`
	ParkingRevenue float64 `json:"parking_revenue"`
	TicketRevenue  float64 `json:"ticket_revenue"`
}

func (ac *AnalyticsController) GetTheatreRevenueAnalytics(c *gin.Context) {
	var rawResults []TheatreRevenueBreakdown

	// Fetch revenue grouped by Screen (which includes Theatre info via joins)
	err := ac.DB.
		Table("bookings").
		Select("t.id AS theatre_id, t.name AS theatre_name, s.id AS screen_id, s.name AS screen_name, SUM(bookings.total_amount) AS total_revenue, SUM(bookings.parking_fee) AS parking_revenue").
		Joins("JOIN shows sh ON sh.id = bookings.show_id").
		Joins("JOIN screens s ON s.id = sh.screen_id").
		Joins("JOIN theatres t ON t.id = s.theatre_id").
		Where("bookings.status = ?", "confirmed"). // Only count confirmed bookings
		Group("t.id, t.name, s.id, s.name").
		Order("t.id ASC, s.id ASC").
		Scan(&rawResults).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch theatre revenue analytics", "details": err.Error()})
		return
	}

	// Post-process to calculate TicketRevenue and group by Theatre
	type TheatreAggregate struct {
		TheatreID      uint                       `json:"theatre_id"`
		TheatreName    string                     `json:"theatre_name"`
		TotalRevenue   float64                    `json:"total_revenue"`
		ParkingRevenue float64                    `json:"parking_revenue"`
		TicketRevenue  float64                    `json:"ticket_revenue"`
		Screens        []*TheatreRevenueBreakdown `json:"screens"`
	}

	theatreMap := make(map[uint]*TheatreAggregate)

	for _, result := range rawResults {
		// Calculate Ticket Revenue for the screen
		result.TicketRevenue = result.TotalRevenue - result.ParkingRevenue

		// Initialize theatre entry if it doesn't exist
		if _, ok := theatreMap[result.TheatreID]; !ok {
			theatreMap[result.TheatreID] = &TheatreAggregate{
				TheatreID:   result.TheatreID,
				TheatreName: result.TheatreName,
				Screens:     []*TheatreRevenueBreakdown{},
			}
		}

		// Update theatre totals
		theatreMap[result.TheatreID].TotalRevenue += result.TotalRevenue
		theatreMap[result.TheatreID].ParkingRevenue += result.ParkingRevenue
		theatreMap[result.TheatreID].TicketRevenue += result.TicketRevenue

		// Add screen breakdown to the theatre's list (pass a copy)
		screenData := result
		theatreMap[result.TheatreID].Screens = append(theatreMap[result.TheatreID].Screens, &screenData)
	}

	// Convert map to slice for final JSON output
	var finalResults []*TheatreAggregate
	for _, tr := range theatreMap {
		finalResults = append(finalResults, tr)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    finalResults,
	})
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
