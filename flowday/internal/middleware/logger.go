package middleware

import (
	"time"

	"flowday/internal/logger"

	"github.com/gin-gonic/gin"
)

// Logging provides structured request logging middleware
func Logging() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Build log fields
		fields := map[string]interface{}{
			"status":     c.Writer.Status(),
			"method":     c.Request.Method,
			"path":       path,
			"ip":         c.ClientIP(),
			"latency_ms": latency.Milliseconds(),
			"user_agent": c.Request.UserAgent(),
		}

		if raw != "" {
			fields["query"] = raw
		}

		// Get request ID if available
		if requestID, exists := c.Get("request_id"); exists {
			fields["request_id"] = requestID
		}

		// Get user ID if available
		if userID, exists := c.Get("user_id"); exists {
			fields["user_id"] = userID
		}

		// Log based on status code
		if c.Writer.Status() >= 500 {
			logger.Log.WithFields(fields).Error("Request error")
		} else if c.Writer.Status() >= 400 {
			logger.Log.WithFields(fields).Warn("Request warning")
		} else {
			logger.Log.WithFields(fields).Info("Request completed")
		}
	}
}
