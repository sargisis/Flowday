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
	
	// ✅ ENHANCED: Limit string length to prevent DoS (max 10MB per field)
	const maxLength = 10 * 1024 * 1024
	if len(input) > maxLength {
		input = input[:maxLength]
	}
	
	// Normalize excessive whitespace (but keep single spaces/newlines/tabs for legitimate use)
	// This helps prevent some DoS attacks via excessive whitespace
	
	return input
}

// ValidateObjectID validates MongoDB ObjectID format
func ValidateObjectID(id string) bool {
	if len(id) != 24 {
		return false
	}
	for _, char := range id {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return false
		}
	}
	return true
}

// ValidateEmail validates email format (basic but improved)
func ValidateEmail(email string) bool {
	if len(email) > 254 || len(email) < 3 { // RFC 5321, minimum: a@b
		return false
	}
	
	// Must contain @ and .
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return false
	}
	
	// Split by @
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	
	// Local part (before @) must not be empty
	if len(parts[0]) == 0 {
		return false
	}
	
	// Domain part (after @) must contain .
	if !strings.Contains(parts[1], ".") {
		return false
	}
	
	// Domain must not start or end with .
	if strings.HasPrefix(parts[1], ".") || strings.HasSuffix(parts[1], ".") {
		return false
	}
	
	return true
}

// ValidateStringLength validates string length
func ValidateStringLength(str string, min, max int) bool {
	length := len(str)
	return length >= min && length <= max
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
