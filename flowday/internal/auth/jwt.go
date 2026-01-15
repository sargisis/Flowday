package auth

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func getSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// ✅ SECURITY: In production, JWT_SECRET is required
		if os.Getenv("GIN_MODE") == "release" {
			log.Fatal("[CRITICAL] JWT_SECRET is required in production mode. Please set JWT_SECRET environment variable.")
		}
		log.Fatal("[CRITICAL] JWT_SECRET is required. Please set JWT_SECRET environment variable.")
	}
	
	// ✅ SECURITY: Validate secret strength (minimum 32 characters)
	if len(secret) < 32 {
		log.Fatal("[CRITICAL] JWT_SECRET must be at least 32 characters long for security.")
	}
	
	return []byte(secret)
}

func getRefreshSecret() []byte {
	secret := os.Getenv("JWT_REFRESH_SECRET")
	if secret == "" {
		// ✅ SECURITY: In production, JWT_REFRESH_SECRET is required
		if os.Getenv("GIN_MODE") == "release" {
			log.Fatal("[CRITICAL] JWT_REFRESH_SECRET is required in production mode. Please set JWT_REFRESH_SECRET environment variable.")
		}
		log.Fatal("[CRITICAL] JWT_REFRESH_SECRET is required. Please set JWT_REFRESH_SECRET environment variable.")
	}
	
	// ✅ SECURITY: Validate secret strength (minimum 32 characters)
	if len(secret) < 32 {
		log.Fatal("[CRITICAL] JWT_REFRESH_SECRET must be at least 32 characters long for security.")
	}
	
	return []byte(secret)
}

// GenerateAccessToken creates a short-lived access token
func GenerateAccessToken(userID primitive.ObjectID) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID.Hex(),
		"exp":     time.Now().Add(time.Minute * 15).Unix(),
		"type":    "access",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getSecret())
}

// GenerateRefreshToken creates a long-lived refresh token
func GenerateRefreshToken(userID primitive.ObjectID) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID.Hex(),
		"exp":     time.Now().Add(time.Hour * 24 * 7).Unix(),
		"type":    "refresh",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getRefreshSecret())
}

// GenerateTokens returns access token (short lived) and refresh token (long lived)
func GenerateTokens(userID primitive.ObjectID) (string, string, error) {
	accessToken, err := GenerateAccessToken(userID)
	if err != nil {
		return "", "", err
	}

	refreshToken, err := GenerateRefreshToken(userID)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func ValidateRefreshToken(tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		// ✅ SECURITY: Validate signing method to prevent algorithm confusion attacks
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return getRefreshSecret(), nil
	})

	if err != nil || !token.Valid {
		return "", err
	}

	// ✅ SECURITY: Safe type assertion with error handling
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", jwt.ErrTokenInvalidClaims
	}

	// Check token type
	if claims["type"] != "refresh" {
		return "", jwt.ErrTokenInvalidClaims
	}

	// ✅ SECURITY: Safe type assertion for user_id
	userIDStr, ok := claims["user_id"].(string)
	if !ok {
		return "", jwt.ErrTokenInvalidClaims
	}

	return userIDStr, nil
}
