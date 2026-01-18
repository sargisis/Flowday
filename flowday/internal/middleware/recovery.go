package middleware

import (
	"net/http"
	"runtime/debug"

	"flowday/internal/logger"

	"github.com/gin-gonic/gin"
)

// RecoveryMiddleware recovers from panics and logs them using structured logging
// It prevents the server from crashing and returns a proper error response
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic with stack trace
				logger.Log.WithFields(map[string]interface{}{
					"error":   err,
					"path":    c.Request.URL.Path,
					"method":  c.Request.Method,
					"ip":      c.ClientIP(),
					"stack":   string(debug.Stack()),
				}).Error("Panic recovered")

				// Return a generic error response to avoid exposing internal details
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error",
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}
