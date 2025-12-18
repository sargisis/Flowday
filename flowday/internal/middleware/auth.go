package middleware

import (
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
			return getSecret(), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": appErrors.ErrUnauthorized.Error(),
			})
			return
		}

		claims := token.Claims.(jwt.MapClaims)
		userIDStr := claims["user_id"].(string)

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
