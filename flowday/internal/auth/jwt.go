package auth

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func getSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return []byte("super-secret-key")
	}
	return []byte(secret)
}

func getRefreshSecret() []byte {
	secret := os.Getenv("JWT_REFRESH_SECRET")
	if secret == "" {
		return []byte("super-secret-refresh-key")
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
		return getRefreshSecret(), nil
	})

	if err != nil || !token.Valid {
		return "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", jwt.ErrTokenInvalidClaims
	}

	// Check token type
	if claims["type"] != "refresh" {
		return "", jwt.ErrTokenInvalidClaims
	}

	return claims["user_id"].(string), nil
}
