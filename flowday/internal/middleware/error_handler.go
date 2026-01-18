package middleware

import (
	"net/http"

	"flowday/internal/logger"

	"github.com/gin-gonic/gin"
)

// ErrorHandlerMiddleware provides centralized error handling for the application
// It catches errors set via c.Error() or c.AbortWithError() and returns standardized error responses
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check if there are any errors in the context
		if len(c.Errors) > 0 {
			// Get the last error (most recent)
			err := c.Errors.Last()

			// Log the error with context
			errorTypeStr := "unknown"
			switch err.Type {
			case gin.ErrorTypeBind:
				errorTypeStr = "bind"
			case gin.ErrorTypePublic:
				errorTypeStr = "public"
			case gin.ErrorTypePrivate:
				errorTypeStr = "private"
			}
			
			logger.Log.WithFields(map[string]interface{}{
				"error":  err.Error(),
				"path":   c.Request.URL.Path,
				"method": c.Request.Method,
				"ip":     c.ClientIP(),
				"type":   errorTypeStr,
			}).Error("Request error")

			// Determine status code based on error type
			statusCode := http.StatusInternalServerError
			switch err.Type {
			case gin.ErrorTypeBind:
				// Binding/validation errors
				statusCode = http.StatusBadRequest
			case gin.ErrorTypePublic:
				// Public errors (already have appropriate status)
				if c.Writer.Status() != 0 {
					statusCode = c.Writer.Status()
				}
			case gin.ErrorTypePrivate:
				// Private errors - don't expose details
				statusCode = http.StatusInternalServerError
			}

			// If status was already written, don't write again
			if !c.Writer.Written() {
				// Return standardized error response
				c.JSON(statusCode, gin.H{
					"error": err.Error(),
				})
			}
		}
	}
}
