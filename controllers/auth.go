package controllers

import (
	"cineverse/config"
	"cineverse/models"
	"cineverse/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// SignupHandler handles new user registration
func SignupHandler(c *gin.Context) {
	var input struct {
		FullName string `json:"full_name" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if email already exists
	var existingUser models.User
	if err := config.DB.Where("email = ?", input.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
		return
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(input.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating account"})
		return
	}

	user := models.User{
		FullName: input.FullName,
		Email:    input.Email,
		Password: hashedPassword,
		// Role:         "user",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := config.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "Account created. Please check your email for the verification code.",
	})
}

// LoginHandler handles user login
func LoginHandler(c *gin.Context) {
	var input struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := config.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// if !user.IsVerified {
	// 	c.JSON(http.StatusUnauthorized, gin.H{"error": "Please verify your email before login"})
	// 	return
	// }

	// if user.IsBlocked {
	// 	c.JSON(http.StatusForbidden, gin.H{"error": "Your account has been blocked. Please contact support."})
	// 	return
	// }

	if !utils.CheckPasswordHash(input.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// Generate JWT access token
	accessToken, err := utils.CreateToken(uint(user.ID), user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating access token"})
		return
	}

	// Generate refresh token
	refreshToken, hashedToken, err := utils.GenerateRefreshToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating refresh token"})
		return
	}

	// Save hashed refresh token in DB
	expiresAt := time.Now().Add(time.Hour * 1) // 1 hour
	if err := utils.SaveRefreshToken(config.DB, user.ID, hashedToken, expiresAt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save refresh token"})
		return
	}

	// Set refresh token as HTTP-only cookie
	c.SetCookie(
		"refresh_token",
		refreshToken,
		int(time.Until(expiresAt).Seconds()),
		"/", // path
		"",
		false,
		true,
	)

	c.JSON(http.StatusOK, gin.H{
		"status":       "success",
		"userId":       user.ID,
		"role":         user.Role,
		"access_token": accessToken,
	})
}

func RefreshTokenHandler(c *gin.Context) {
	// Get refresh token from cookie
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token required"})
		return
	}

	rt, err := utils.ValidateRefreshToken(config.DB, refreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired refresh token"})
		return
	}

	// Generate new access token
	accessToken, err := utils.CreateToken(uint(rt.UserID), "user") // replace "user" with actual role if needed
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating access token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":       "success",
		"access_token": accessToken,
	})
}

func LogoutHandler(c *gin.Context) {
	// Get refresh token from cookie
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Refresh token required"})
		return
	}

	// Delete token from DB
	if err := utils.DeleteRefreshToken(config.DB, refreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not logout"})
		return
	}

	// Clear cookie
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Logged out successfully",
	})
}

// ForgotPasswordHandler
func ForgotPasswordHandler(c *gin.Context) {
	var input struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := config.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success"})
}

// ResetPasswordHandler
func ResetPasswordHandler(c *gin.Context) {
	var input struct {
		Email       string `json:"email" binding:"required,email"`
		OTP         string `json:"otp" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := config.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Hash new password
	hashedPassword, err := utils.HashPassword(input.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash new password"})
		return
	}

	// Update password and updated_at
	if err := config.DB.Model(&user).Updates(map[string]interface{}{
		"password_hash": hashedPassword,
		"updated_at":    time.Now(),
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Password reset successfully",
	})
}
