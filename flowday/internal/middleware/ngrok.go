package middleware

import (
	"github.com/gin-gonic/gin"
)

// NgrokSkipWarning middleware adds ngrok-skip-browser-warning header to skip ngrok warning page
func NgrokSkipWarning() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set header to skip ngrok browser warning
		c.Header("ngrok-skip-browser-warning", "true")
		c.Next()
	}
}
