package auth

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims represents the claims in our JWT token
type JWTClaims struct {
	DeviceID string `json:"device_id"`
	UserID   string `json:"user_id,omitempty"`
	Role     string `json:"role"` // "device" or "user"
	jwt.RegisteredClaims
}

// JWTSecret should be loaded from environment variable
var JWTSecret = []byte(getJWTSecret())

// getJWTSecret loads JWT secret from environment or uses default for development
func getJWTSecret() string {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// Use a default secret for development - should be changed in production
		secret = "arunika-development-secret-key-change-in-production"
	}
	return secret
}

// GenerateDeviceToken generates a JWT token for device authentication
func GenerateDeviceToken(deviceID string) (string, error) {
	claims := &JWTClaims{
		DeviceID: deviceID,
		Role:     "device",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWTSecret)
}

// GenerateUserToken generates a JWT token for user authentication
func GenerateUserToken(userID string) (string, error) {
	claims := &JWTClaims{
		UserID: userID,
		Role:   "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)), // 7 days
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWTSecret)
}

// ValidateToken validates a JWT token and returns the claims
func ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return JWTSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrInvalidKey
}
