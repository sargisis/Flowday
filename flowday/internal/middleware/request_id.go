package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestIDMiddleware adds a unique request ID to each request for tracing
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if request ID already exists in header
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			// Generate a new unique ID
			bytes := make([]byte, 16)
			if _, err := rand.Read(bytes); err != nil {
				// Fallback to timestamp-based ID if crypto/rand fails
				requestID = fmt.Sprintf("%d", time.Now().UnixNano())
			} else {
				requestID = hex.EncodeToString(bytes)
			}
		}

		// Set the request ID in the context
		c.Set("request_id", requestID)

		// Add it to the response header
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}
