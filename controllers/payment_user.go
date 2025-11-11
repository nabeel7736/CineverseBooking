package controllers

import (
	"cineverse/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 🔹 POST /api/payments/initiate
// Step 1 – user initiates mock payment
func InitiatePayment(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		var req struct {
			BookingID uint    `json:"booking_id"`
			Amount    float64 `json:"amount"`
			Method    string  `json:"method"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payment data"})
			return
		}

		var booking models.Booking
		if err := db.First(&booking, req.BookingID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
			return
		}

		if booking.TotalAmount != req.Amount {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Payment amount mismatch with booking total"})
			return
		}

		// mock “gateway” processing delay or token generation
		payment := models.Payment{
			BookingID: req.BookingID,
			Amount:    req.Amount,
			Method:    req.Method,
			Status:    "initiated",
			CreatedAt: time.Now(),
		}

		if err := db.Create(&payment).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate payment"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":       "Payment initiated",
			"payment_id":    payment.ID,
			"mock_redirect": "/api/payments/mock/confirm/" + strconv.Itoa(int(payment.ID)),
		})
	}

}

// Step 2 – simulate Razorpay/Stripe webhook
func MockConfirmPayment(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		id := c.Param("id")

		tx := db.Begin()
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()

		var payment models.Payment
		if err := tx.First(&payment, id).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
			return
		}

		var booking models.Booking
		if err := tx.First(&booking, payment.BookingID).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
			return
		}

		if booking.Status == "confirmed" {
			tx.Commit()
			c.JSON(http.StatusOK, gin.H{
				"message": "Payment already confirmed",
				"booking": booking.ID,
				"status":  booking.Status,
			})
			return
		}

		wasConfirmed := booking.Status == "confirmed"

		// mock successful payment (90 % success rate)
		payment.Status = "completed"
		payment.UpdatedAt = time.Now()
		booking.Status = "confirmed"

		if err := tx.Save(&payment).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update payment status"})
			return
		}
		if err := tx.Save(&booking).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update booking status"})
			return
		}

		if !wasConfirmed {
			var show models.Show
			if err := tx.First(&show, booking.ShowID).Error; err == nil {
				show.SeatsBooked += booking.SeatsCount
				if err := tx.Save(&show).Error; err != nil {
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update seats count"})
					return
				}
			}
		}

		if err := tx.Commit().Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":  "Payment successful (mock)",
			"booking":  booking.ID,
			"status":   booking.Status,
			"amount":   payment.Amount,
			"method":   payment.Method,
			"datetime": payment.UpdatedAt,
		})
	}
}

// 🔹 GET /api/payments/user
func GetUserPayments(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		userIDRaw, exists := c.Get("userId")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		userID := userIDRaw.(uint)

		var payments []models.Payment
		if err := db.Joins("JOIN bookings ON bookings.id = payments.booking_id").
			Where("bookings.user_id = ?", userID).
			Order("payments.created_at desc").
			Find(&payments).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch payments"})
			return
		}

		c.JSON(http.StatusOK, payments)
	}
}
