package middleware

import (
	"net/http"

	"flowday/internal/errors"
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

			// Try to extract AppError
			appErr := errors.GetAppError(err.Err)
			
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
			
			logFields := map[string]interface{}{
				"error":  err.Error(),
				"path":   c.Request.URL.Path,
				"method": c.Request.Method,
				"ip":     c.ClientIP(),
				"type":   errorTypeStr,
			}
			
			if appErr != nil {
				logFields["error_code"] = appErr.Code
			}
			
			logger.Log.WithFields(logFields).Error("Request error")

			// Determine status code based on error type
			statusCode := http.StatusInternalServerError
			errorResponse := gin.H{}

			if appErr != nil {
				// Use structured error response
				errorResponse = gin.H{
					"error": gin.H{
						"code":    appErr.Code,
						"message": appErr.Message,
					},
				}
				if appErr.Details != "" {
					errorResponse["error"].(gin.H)["details"] = appErr.Details
				}

				// Map error codes to HTTP status codes
				switch appErr.Code {
				case errors.CodeUnauthorized:
					statusCode = http.StatusUnauthorized
				case errors.CodeForbidden:
					statusCode = http.StatusForbidden
				case errors.CodeNotFound:
					statusCode = http.StatusNotFound
				case errors.CodeInvalidInput, errors.CodeValidationError, errors.CodeBadRequest:
					statusCode = http.StatusBadRequest
				case errors.CodeRateLimitExceeded:
					statusCode = http.StatusTooManyRequests
				case errors.CodeUserExists:
					statusCode = http.StatusConflict
				default:
					statusCode = http.StatusInternalServerError
				}
			} else {
				// Fallback to old behavior for non-AppError errors
				switch err.Type {
				case gin.ErrorTypeBind:
					statusCode = http.StatusBadRequest
					errorResponse = gin.H{
						"error": gin.H{
							"code":    errors.CodeValidationError,
							"message": "Validation failed",
							"details": err.Error(),
						},
					}
				case gin.ErrorTypePublic:
					if c.Writer.Status() != 0 {
						statusCode = c.Writer.Status()
					}
					errorResponse = gin.H{
						"error": gin.H{
							"code":    errors.CodeInternalError,
							"message": err.Error(),
						},
					}
				case gin.ErrorTypePrivate:
					statusCode = http.StatusInternalServerError
					errorResponse = gin.H{
						"error": gin.H{
							"code":    errors.CodeInternalError,
							"message": "Internal server error",
						},
					}
				}
			}

			// If status was already written, don't write again
			if !c.Writer.Written() {
				c.JSON(statusCode, errorResponse)
			}
		}
	}
}
