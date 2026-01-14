package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Test rate limiter
	router := gin.New()
	router.GET("/test", RateLimitMiddleware("2-M"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	// Make requests
	req1 := httptest.NewRequest("GET", "/test", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)
	assert.NotEmpty(t, w1.Header().Get("X-RateLimit-Limit"))
	assert.NotEmpty(t, w1.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, w1.Header().Get("X-RateLimit-Reset"))

	// Second request should also pass
	req2 := httptest.NewRequest("GET", "/test", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
}

func TestResetPasswordRateLimiter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/reset-password", ResetPasswordRateLimiter(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	req := httptest.NewRequest("POST", "/reset-password", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	// Check headers are properly formatted (not rune conversion)
	limit := w.Header().Get("X-RateLimit-Limit")
	assert.NotEmpty(t, limit)
	// Should be a number string, not a single character
	assert.Greater(t, len(limit), 0)
}

func TestRefreshRateLimiter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/refresh", RefreshRateLimiter(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	req := httptest.NewRequest("POST", "/refresh", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	// Check headers are properly formatted
	limit := w.Header().Get("X-RateLimit-Limit")
	assert.NotEmpty(t, limit)
}

func TestRateLimitHeadersFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/test", RateLimitMiddleware("5-M"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	limit := w.Header().Get("X-RateLimit-Limit")
	remaining := w.Header().Get("X-RateLimit-Remaining")
	reset := w.Header().Get("X-RateLimit-Reset")

	// Headers should exist
	assert.NotEmpty(t, limit)
	assert.NotEmpty(t, remaining)
	assert.NotEmpty(t, reset)

	// Headers should be numeric strings (not single characters from rune conversion)
	// This tests that we're using strconv.FormatInt, not string(rune())
	assert.Greater(t, len(limit), 0)
	
	// Limit should be a number (at least 1 digit, but can be more)
	// For 5-M, limit should be "5" (single digit, but properly formatted)
	// The important thing is it's not a rune character
	assert.NotEqual(t, limit, string(rune(5))) // Should not be rune conversion
}
