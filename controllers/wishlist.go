package controllers

import (
	"cineverse/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetWishlist(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		userID := c.GetUint("userId")

		var wishlist []models.Wishlist
		if err := db.Preload("Movie").
			Preload("User").
			Where("user_id = ?", userID).
			Find(&wishlist).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch wishlist"})
			return
		}

		c.JSON(http.StatusOK, wishlist)
	}
}

func AddToWishlist(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		userID := c.GetUint("userId")

		var body struct {
			MovieID uint `json:"movie_id"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
			return
		}

		w := models.Wishlist{UserID: userID, MovieID: body.MovieID}
		if err := db.Create(&w).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add to wishlist"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Movie added to wishlist"})
	}
}

func RemoveFromWishlist(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		userID := c.GetUint("userId")
		movieID := c.Param("id")

		if err := db.Where("user_id = ? AND movie_id = ?", userID, movieID).Delete(&models.Wishlist{}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove movie"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Movie removed from wishlist"})
	}
}
