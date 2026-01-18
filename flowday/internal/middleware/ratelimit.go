package middleware

import (
	"net/http"
	"os"
	"strconv"

	"flowday/internal/logger"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
	redisstore "github.com/ulule/limiter/v3/drivers/store/redis"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	globalRateLimiter *limiter.Limiter
	userRateLimiter   *limiter.Limiter
	redisStore        limiter.Store
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

		// ✅ SECURITY: Fix rate limit headers - use proper string conversion
		c.Header("X-RateLimit-Limit", strconv.FormatInt(context.Limit, 10))
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(context.Remaining, 10))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(context.Reset, 10))

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

// ResetPasswordRateLimiter returns a rate limiter for password reset confirmation
func ResetPasswordRateLimiter() gin.HandlerFunc {
	return RateLimitMiddleware("5-M") // 5 requests per minute (allow more attempts than forgot-password)
}

// RefreshRateLimiter returns a rate limiter for token refresh
func RefreshRateLimiter() gin.HandlerFunc {
	return RateLimitMiddleware("10-M") // 10 requests per minute (less restrictive for refresh)
}

// InitRedisRateLimiter initializes Redis store for rate limiting if available
func InitRedisRateLimiter() error {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		return nil // Redis not configured, will use memory store
	}

	// Create Redis client
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return err
	}

	rdb := redis.NewClient(opt)

	// Create Redis store with options
	store, err := redisstore.NewStoreWithOptions(rdb, limiter.StoreOptions{
		Prefix:   "flowday:ratelimit:",
		MaxRetry: 3,
	})
	if err != nil {
		return err
	}

	redisStore = store
	logger.Log.Info("Redis rate limiter store initialized")
	return nil
}

// getStore returns Redis store if available, otherwise memory store
func getStore() limiter.Store {
	if redisStore != nil {
		return redisStore
	}
	return memory.NewStore()
}

// GlobalRateLimitMiddleware provides global rate limiting (100 req/min per IP)
// Excludes WebSocket and health check endpoints
func GlobalRateLimitMiddleware() gin.HandlerFunc {
	limitRate, err := limiter.NewRateFromFormatted("100-M")
	if err != nil {
		panic(err)
	}

	store := getStore()
	globalRateLimiter = limiter.New(store, limitRate)

	return func(c *gin.Context) {
		// Skip rate limiting for WebSocket and health check endpoints
		path := c.Request.URL.Path
		if path == "/ws/connect" || path == "/health" || path == "/ready" || path == "/live" || path == "/metrics" {
			c.Next()
			return
		}

		key := c.ClientIP()

		context, err := globalRateLimiter.Get(c.Request.Context(), key)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Rate limiter error"})
			c.Abort()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.FormatInt(context.Limit, 10))
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(context.Remaining, 10))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(context.Reset, 10))

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

// UserRateLimitMiddleware provides user-based rate limiting (200 req/min per user)
// Excludes WebSocket and health check endpoints
func UserRateLimitMiddleware() gin.HandlerFunc {
	limitRate, err := limiter.NewRateFromFormatted("200-M")
	if err != nil {
		panic(err)
	}

	store := getStore()
	userRateLimiter = limiter.New(store, limitRate)

	return func(c *gin.Context) {
		// Skip rate limiting for WebSocket and health check endpoints
		path := c.Request.URL.Path
		if path == "/ws/connect" || path == "/health" || path == "/ready" || path == "/live" || path == "/metrics" {
			c.Next()
			return
		}

		// Only apply to authenticated users
		userID, exists := c.Get("user_id")
		if !exists {
			// Not authenticated, skip user rate limiting
			c.Next()
			return
		}

		// Use user ID as key (convert ObjectID to string)
		key := "user:" + userID.(primitive.ObjectID).Hex()

		context, err := userRateLimiter.Get(c.Request.Context(), key)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Rate limiter error"})
			c.Abort()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.FormatInt(context.Limit, 10))
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(context.Remaining, 10))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(context.Reset, 10))

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
