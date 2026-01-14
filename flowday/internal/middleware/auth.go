package middleware

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	appErrors "flowday/internal/errors"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func getSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Println("[WARNING] JWT_SECRET not set, using default key (INSECURE - for development only)")
		return []byte("super-secret-key")
	}
	return []byte(secret)
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": appErrors.ErrUnauthorized.Error(),
			})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": appErrors.ErrUnauthorized.Error(),
			})
			return
		}

		tokenStr := parts[1]

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			// ✅ SECURITY: Validate signing method to prevent algorithm confusion attacks
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return getSecret(), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": appErrors.ErrUnauthorized.Error(),
			})
			return
		}

		// ✅ SECURITY: Safe type assertion with error handling
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token claims",
			})
			return
		}

		// ✅ SECURITY: Safe type assertion for user_id
		userIDStr, ok := claims["user_id"].(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid user ID in token",
			})
			return
		}

		userID, err := primitive.ObjectIDFromHex(userIDStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid user ID",
			})
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}
