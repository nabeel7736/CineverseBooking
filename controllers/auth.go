package controllers

import (
	"cineverse/config"
	"cineverse/models"
	"cineverse/utils"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// UserSignupHandler handles new customer registration
func UserSignupHandler(c *gin.Context) {
	var input struct {
		FullName string `json:"full_name" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user already exists
	var existing models.User
	if err := config.DB.Where("email = ?", input.Email).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
		return
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(input.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	user := models.User{
		FullName:  input.FullName,
		Email:     input.Email,
		Password:  hashedPassword,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := config.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "User registered successfully",
	})
}

// UserLoginHandler handles customer login
func UserLoginHandler(c *gin.Context) {
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

	if user.Blocked {
		c.JSON(http.StatusForbidden, gin.H{"error": "Your account has been blocked"})
		return
	}

	if !utils.CheckPasswordHash(input.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// Create JWT with role=user

	token, err := utils.CreateToken(uint(user.ID), "user")
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
		"status": "success",
		"role":   "user",
		"token":  token,
		"user": gin.H{
			"id":    user.ID,
			"name":  user.FullName,
			"email": user.Email,
		},
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

func AdminRegister(c *gin.Context) {
	var input struct {
		FullName string `json:"full_name" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var existing models.Admin
	if err := config.DB.Where("email = ?", input.Email).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
		return
	}

	hashedPassword, err := utils.HashPassword(input.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	admin := models.Admin{
		FullName:  input.FullName,
		Email:     input.Email,
		Password:  hashedPassword,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := config.DB.Create(&admin).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create admin"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "Admin registered successfully",
	})
}

// AdminLoginHandler handles admin login
func AdminLogin(c *gin.Context) {
	var input struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var admin models.Admin
	if err := config.DB.Where("email = ?", input.Email).First(&admin).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if !utils.CheckPasswordHash(input.Password, admin.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Create JWT with role=admin
	token, err := utils.CreateToken(uint(admin.ID), "admin")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create token"})
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
	if err := utils.SaveRefreshToken(config.DB, admin.ID, hashedToken, expiresAt); err != nil {
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
		"role":         "admin",
		"access_token": token,
		"admin": gin.H{
			"id":    admin.ID,
			"name":  admin.FullName,
			"email": admin.Email,
		},
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
		"password":   hashedPassword,
		"updated_at": time.Now(),
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Password reset successfully",
	})
}

func AdminVerify(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no token provided"})
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		claims, err := utils.ValidateJWT(tokenStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		if claims.Role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "not an admin"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "valid token",
			"role":    claims.Role,
		})
	}
}
