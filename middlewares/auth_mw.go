package middlewares

import (
	"cineverse/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const ContextUserID = "userId"
const ContextUserRole = "userRole"

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization required"})
			return
		}
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
			return
		}
		tokenStr := parts[1]
		claims, err := utils.ValidateJWT(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token", "details": err.Error()})
			return
		}
		// save in context
		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextUserRole, claims.Role)
		c.Next()
	}
}

// func AdminMiddleware() gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		role, exists := c.Get(ContextUserRole)
// 		if !exists {
// 			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "role missing"})
// 			return
// 		}
// 		roleStr, ok := role.(string)
// 		if !ok || roleStr != "admin" {
// 			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
// 			return
// 		}
// 		c.Next()
// 	}
// }

func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid Authorization header"})
			c.Abort()
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := utils.ValidateJWT(tokenStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		if claims.Role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			c.Abort()
			return
		}

		c.Set("userId", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}
