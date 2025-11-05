package middlewares

import (
	"cineverse/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	ContextUserID   = "userId"
	ContextUserRole = "userRole"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing"})
			return
		}

		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header"})
			return
		}

		tokenStr := parts[1]
		claims, err := utils.ValidateJWT(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token", "details": err.Error()})
			return
		}

		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextUserRole, claims.Role)
		c.Next()
	}
}

func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing"})
			return
		}

		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		claims, err := utils.ValidateJWT(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		if claims.Role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Admins only"})
			return
		}

		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextUserRole, claims.Role)
		c.Next()
	}
}

func UserMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing"})
			return
		}

		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		claims, err := utils.ValidateJWT(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		if claims.Role != "user" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Users only"})
			return
		}

		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextUserRole, claims.Role)
		c.Next()
	}
}

func PreventLoginWhenAuthenticated() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")

		if auth != "" {
			tokenStr := strings.TrimPrefix(auth, "Bearer ")
			claims, err := utils.ValidateJWT(tokenStr)
			if err == nil && claims != nil {
				switch claims.Role {
				case "admin":
					c.Redirect(http.StatusFound, "/admin/dashboard")
					c.Abort()
					return
				case "user":
					c.Redirect(http.StatusFound, "/")
					c.Abort()
					return
				}
			}
		}
		c.Next()
	}
}
