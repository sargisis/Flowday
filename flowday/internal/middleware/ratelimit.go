package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

// RateLimitMiddleware creates a rate limiter middleware
func RateLimitMiddleware(rate string) gin.HandlerFunc {
	// Parse rate format: "5-M" = 5 requests per minute
	limitRate, err := limiter.NewRateFromFormatted(rate)
	if err != nil {
		panic(err)
	}

	// Use in-memory store (for production, use Redis)
	store := memory.NewStore()
	instance := limiter.New(store, limitRate)

	return func(c *gin.Context) {
		// Use IP as identifier
		key := c.ClientIP()

		context, err := instance.Get(c.Request.Context(), key)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Rate limiter error"})
			c.Abort()
			return
		}

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", string(rune(context.Limit)))
		c.Header("X-RateLimit-Remaining", string(rune(context.Remaining)))
		c.Header("X-RateLimit-Reset", string(rune(context.Reset)))

		if context.Reached {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// AuthRateLimiter returns a stricter rate limiter for auth endpoints
func AuthRateLimiter() gin.HandlerFunc {
	return RateLimitMiddleware("5-M") // 5 requests per minute
}

// RegisterRateLimiter returns a rate limiter for registration
func RegisterRateLimiter() gin.HandlerFunc {
	return RateLimitMiddleware("3-M") // 3 requests per minute
}

// ForgotPasswordRateLimiter returns a rate limiter for password reset
func ForgotPasswordRateLimiter() gin.HandlerFunc {
	return RateLimitMiddleware("3-M") // 3 requests per minute
}
