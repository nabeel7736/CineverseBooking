package utils

import (
	"cineverse/models"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type MyClaims struct {
	UserID uint   `json:"userId"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// func CreateToken(userID uint, Role string) (string, error) {
// 	secret := os.Getenv("JWT_SECRET")
// 	claims := jwt.MapClaims{
// 		"userId": userID,
// 		"role":   Role,
// 		"exp":    time.Now().Add(time.Minute * 45).Unix(), // 45 minutes
// 	}

// 	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
// 	return token.SignedString([]byte(secret))
// }

func CreateToken(userID uint, role string) (string, error) {
	claims := MyClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret := os.Getenv("JWT_SECRET")
	return token.SignedString([]byte(secret))
}

// func CreateToken(userID uint, role string) (string, error) {
// 	secret := os.Getenv("JWT_SECRET")
// 	claims := MyClaims{
// 		UserID: userID,
// 		Role:   role,
// 		RegisteredClaims: jwt.RegisteredClaims{
// 			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
// 			IssuedAt:  jwt.NewNumericDate(time.Now()),
// 		},
// 	}
// 	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
// 	return token.SignedString([]byte(secret))
// }

// func ValidateJWT(tokenStr string) (int, string, error) {
// 	secret := os.Getenv("JWT_SECRET")

// 	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
// 		// Validate signing method
// 		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
// 			return nil, fmt.Errorf("unexpected signing method")
// 		}
// 		return []byte(secret), nil
// 	})
// 	if err != nil {
// 		return 0, "", err
// 	}

// 	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
// 		// Extract userId
// 		userIDFloat, ok := claims["userId"].(float64)
// 		if !ok {
// 			return 0, "", fmt.Errorf("invalid userId in token")
// 		}

// 		// Extract role
// 		role, ok := claims["role"].(string)
// 		if !ok {
// 			return 0, "", fmt.Errorf("invalid role in token")
// 		}

// 		return int(userIDFloat), role, nil
// 	}

// 	return 0, "", fmt.Errorf("invalid token")
// }

func ValidateJWT(tokenStr string) (*MyClaims, error) {
	if tokenStr == "" {
		return nil, errors.New("missing token")
	}

	secret := os.Getenv("JWT_SECRET")
	token, err := jwt.ParseWithClaims(tokenStr, &MyClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*MyClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// func ValidateJWT(tokenStr string) (*MyClaims, error) {
// 	secret := os.Getenv("JWT_SECRET")
// 	if tokenStr == "" {
// 		return nil, errors.New("empty token")
// 	}
// 	tkn, err := jwt.ParseWithClaims(tokenStr, &MyClaims{}, func(token *jwt.Token) (interface{}, error) {
// 		return []byte(secret), nil
// 	})
// 	if err != nil {
// 		return nil, err
// 	}
// 	if claims, ok := tkn.Claims.(*MyClaims); ok && tkn.Valid {
// 		return claims, nil
// 	}
// 	return nil, errors.New("invalid token claims")
// }

// func ParseToken(tokenStr string) (*MyClaims, error) {
// 	secret := os.Getenv("JWT_SECRET")
// 	token, err := jwt.ParseWithClaims(tokenStr, &MyClaims{}, func(token *jwt.Token) (interface{}, error) {
// 		return []byte(secret), nil
// 	})
// 	if err != nil {
// 		return nil, err
// 	}
// 	if claims, ok := token.Claims.(*MyClaims); ok && token.Valid {
// 		return claims, nil
// 	}
// 	return nil, jwt.ErrTokenInvalidClaims
// }

func GenerateRefreshToken() (string, string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", "", err
	}
	token := hex.EncodeToString(b)
	hash := sha256.Sum256([]byte(token))
	return token, hex.EncodeToString(hash[:]), nil
}

// func SaveRefreshToken(db *gorm.DB, userID uint, hashedToken string, expiresAt time.Time) error {
// 	rt := models.RefreshToken{
// 		UserID:    userID,
// 		Token:     hashedToken,
// 		ExpiresAt: expiresAt,
// 	}
// 	return db.Create(&rt).Error
// }

func SaveRefreshToken(db *gorm.DB, userID uint, hashedToken string, expiresAt time.Time) error {
	rt := models.RefreshToken{
		UserID:    userID,
		Token:     hashedToken,
		ExpiresAt: expiresAt,
	}

	// If user already has a token, update instead of insert
	var existing models.RefreshToken
	err := db.Where("user_id = ?", userID).First(&existing).Error
	if err == nil {
		// Update existing
		existing.Token = hashedToken
		existing.ExpiresAt = expiresAt
		return db.Save(&existing).Error
	}

	// Otherwise create new
	return db.Create(&rt).Error
}

func ValidateRefreshToken(db *gorm.DB, token string) (*models.RefreshToken, error) {
	hash := sha256.Sum256([]byte(token))
	hashedToken := hex.EncodeToString(hash[:])

	var rt models.RefreshToken
	err := db.Where("token = ? AND expires_at > ?", hashedToken, time.Now()).First(&rt).Error
	if err != nil {
		return nil, errors.New("invalid or expired refresh token")
	}
	return &rt, nil
}

func DeleteRefreshToken(db *gorm.DB, token string) error {
	hash := sha256.Sum256([]byte(token))
	hashedToken := hex.EncodeToString(hash[:])
	return db.Where("token = ?", hashedToken).Delete(&models.RefreshToken{}).Error
}
