package controllers

import (
	"cineverse/models"
	"cineverse/utils"
	"net/http"
	"time"

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
			query = query.Where("LOWER(full_name) LIKE LOWER(?) OR LOWER(email) LIKE LOWER(?)", "%"+search+"%", "%"+search+"%")
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

func AddUser(db *gorm.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var input models.User

		if err := ctx.ShouldBindJSON(&input); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var existing models.User
		if err := db.Where("email = ?", input.Email).First(&existing).Error; err == nil {
			ctx.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
			return
		}

		hashedPassword, err := utils.HashPassword(input.Password)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}

		user := models.User{
			FullName:  input.FullName,
			Email:     input.Email,
			Password:  hashedPassword,
			Blocked:   false,
			Deleted:   false,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := db.Create(&user).Error; err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add user"})
			return
		}

		ctx.JSON(http.StatusCreated, gin.H{
			"status":  "success",
			"message": "User added successfully",
			"user": gin.H{
				"id":       user.ID,
				"fullName": user.FullName,
				"email":    user.Email,
			},
		})
	}

}
