package controllers

import (
	"cineverse/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetAllUsers - Admin fetch all users
func GetAllUsers(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		var users []models.User
		search := c.Query("search")

		query := db.Model(&models.User{}).
			Preload("Bookings").
			Where("deleted = FALSE")

		if search != "" {
			query = query.Where("LOWER(name) LIKE LOWER(?) OR LOWER(email) LIKE LOWER(?)", "%"+search+"%", "%"+search+"%")
		}

		if err := query.Find(&users).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
			return
		}

		// Compute booking count manually
		for i := range users {
			users[i].BookingsCount = int64(len(users[i].Bookings))
		}

		c.JSON(http.StatusOK, gin.H{"users": users})
	}
}

// BlockUser - toggle user status
func BlockUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		id := c.Param("id")
		var user models.User

		if err := db.First(&user, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		user.Blocked = !user.Blocked
		db.Save(&user)
		status := "unblocked"
		if user.Blocked {
			status = "blocked"
		}
		c.JSON(http.StatusOK, gin.H{"message": "User " + status + " successfully"})
	}
}

func DeleteUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var user models.User
		if err := db.First(&user, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// Delete all bookings for this user (with cascade)
		if err := db.Where("user_id = ?", id).Delete(&models.Booking{}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user bookings"})
			return
		}

		if err := db.Delete(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
			return
		}

		var users []models.User
		db.Find(&users)
		c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully", "users": users})
	}
}
