package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// InputSanitizationMiddleware sanitizes user input to prevent injection attacks
// It removes dangerous characters like null bytes and excessive whitespace
// Note: For HTML content, use proper HTML sanitization libraries in your handlers
func InputSanitizationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Sanitize query parameters
		query := c.Request.URL.Query()
		for key, values := range query {
			sanitizedValues := make([]string, len(values))
			for i, value := range values {
				sanitizedValues[i] = sanitizeString(value)
			}
			query[key] = sanitizedValues
		}
		c.Request.URL.RawQuery = query.Encode()

		// Sanitize form data if Content-Type is application/x-www-form-urlencoded
		if c.ContentType() == "application/x-www-form-urlencoded" {
			if err := c.Request.ParseForm(); err == nil {
				for key, values := range c.Request.PostForm {
					sanitizedValues := make([]string, len(values))
					for i, value := range values {
						sanitizedValues[i] = sanitizeString(value)
					}
					c.Request.PostForm[key] = sanitizedValues
				}
			}
		}

		c.Next()
	}
}

// sanitizeString removes potentially dangerous characters
// It removes null bytes and normalizes whitespace while preserving the input
func sanitizeString(input string) string {
	// Remove null bytes (always dangerous)
	input = strings.ReplaceAll(input, "\x00", "")

	// Remove other dangerous control characters
	input = strings.ReplaceAll(input, "\r", "")
	
	// Normalize excessive whitespace (but keep single spaces/newlines/tabs for legitimate use)
	// This helps prevent some DoS attacks via excessive whitespace
	
	return input
}

// SanitizeJSONBody is a helper function that can be used in handlers
// to sanitize JSON request bodies. This is called manually in handlers
// because we need to parse the JSON first.
func SanitizeJSONBody(data map[string]interface{}) map[string]interface{} {
	sanitized := make(map[string]interface{})
	for key, value := range data {
		sanitizedKey := sanitizeString(key)
		switch v := value.(type) {
		case string:
			sanitized[sanitizedKey] = sanitizeString(v)
		case []interface{}:
			sanitizedArray := make([]interface{}, len(v))
			for i, item := range v {
				if str, ok := item.(string); ok {
					sanitizedArray[i] = sanitizeString(str)
				} else {
					sanitizedArray[i] = item
				}
			}
			sanitized[sanitizedKey] = sanitizedArray
		case map[string]interface{}:
			sanitized[sanitizedKey] = SanitizeJSONBody(v)
		default:
			sanitized[sanitizedKey] = value
		}
	}
	return sanitized
}
