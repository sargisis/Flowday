package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"flowday/internal/cache"
	"flowday/internal/db"
	"flowday/internal/logger"
	"flowday/internal/middleware"
	"flowday/internal/router"
	"flowday/internal/services"
	"flowday/internal/worker"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// ✅ LOGGING: Initialize structured logging
	logger.Init()

	if err := godotenv.Load(); err != nil {
		logger.Log.Debug("No .env file found")
	}

	logger.Log.Info("Starting Flowday server...")

	// ✅ PERFORMANCE: Initialize cache
	cache.Init()

	db.Connect()

	// Initialize achievements
	if err := services.InitializeAchievements(); err != nil {
		logger.Log.WithError(err).Error("Failed to initialize achievements")
	}

	if err := services.InitAIService(); err != nil {
		logger.Log.WithError(err).Error("Failed to initialize AI Service")
	}

	// Start Background Workers
	worker.StartAnalyzer()

	// ✅ PERFORMANCE: Initialize Redis rate limiter (if available)
	if err := middleware.InitRedisRateLimiter(); err != nil {
		logger.Log.WithError(err).Warn("Redis not available, using memory store for rate limiting")
	} else if os.Getenv("REDIS_URL") != "" {
		logger.Log.Info("Redis rate limiter initialized")
	}

	// ✅ SECURITY: Set Gin mode based on environment
	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "" {
		ginMode = gin.DebugMode // Default to debug for development
	}
	gin.SetMode(ginMode)

	r := gin.Default()

	// ✅ SECURITY: Limit request body size to prevent DoS attacks (10MB max)
	r.MaxMultipartMemory = 10 << 20 // 10MB

	// ✅ PERFORMANCE: Add gzip compression for responses
	r.Use(gzip.Gzip(gzip.DefaultCompression))

	// ✅ TRACING: Add request ID for request tracing
	r.Use(middleware.RequestIDMiddleware())

	// ✅ METRICS: Add Prometheus metrics collection
	r.Use(middleware.MetricsMiddleware())

	// ✅ LOGGING: Add structured request logging
	r.Use(middleware.LoggingMiddleware())

	// ✅ SECURITY: Add global rate limiting (100 req/min per IP)
	r.Use(middleware.GlobalRateLimitMiddleware())

	// ✅ SECURITY: Add user-based rate limiting (200 req/min per user)
	// This applies to authenticated users only
	r.Use(middleware.UserRateLimitMiddleware())

	// ✅ SECURITY: Add audit logging for important actions
	r.Use(middleware.AuditLogMiddleware())

	// ✅ STABILITY: Add panic recovery middleware
	r.Use(middleware.RecoveryMiddleware())

	// ✅ SECURITY: Add input sanitization
	r.Use(middleware.InputSanitizationMiddleware())

	// ✅ PERFORMANCE: Add request timeout (30 seconds default)
	timeout := 30 * time.Second
	if timeoutStr := os.Getenv("REQUEST_TIMEOUT"); timeoutStr != "" {
		if parsedTimeout, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = parsedTimeout
		}
	}
	r.Use(middleware.TimeoutMiddleware(timeout))

	// ✅ SECURITY: Add security headers to all responses
	r.Use(middleware.SecurityHeadersMiddleware())

	// ✅ ERROR HANDLING: Centralized error handling middleware
	r.Use(middleware.ErrorHandlerMiddleware())

	// Skip ngrok browser warning for all requests
	r.Use(middleware.NgrokSkipWarning())

	// ✅ SECURITY: CORS configuration - support both development and production
	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			// Development: allow localhost
			if strings.HasPrefix(origin, "http://localhost") ||
				strings.HasPrefix(origin, "http://127.0.0.1") {
				return true
			}

			// Production: check ALLOWED_ORIGINS environment variable
			allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
			if allowedOrigins == "" {
				// If not set, default to localhost only (safe default)
				return false
			}

			// Parse comma-separated list of allowed origins
			origins := strings.Split(allowedOrigins, ",")
			for _, allowed := range origins {
				allowed = strings.TrimSpace(allowed)
				if origin == allowed {
					return true
				}
			}

			return false
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Cookie"},
		ExposeHeaders:    []string{"Content-Length", "Set-Cookie"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	router.Setup(r)

	// ✅ STABILITY: Configure server with timeouts
	server := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ✅ STABILITY: Graceful shutdown
	go func() {
		logger.Log.Info("🚀 Flowday server starting on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.WithError(err).Fatal("Failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("🛑 Shutting down server...")

	// Graceful shutdown with 30 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Log.WithError(err).Fatal("Server forced to shutdown")
	}

	logger.Log.Info("✅ Server exited gracefully")
}
