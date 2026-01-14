package main

import (
	"log"
	"os"
	"strings"
	"time"

	"flowday/internal/db"
	"flowday/internal/router"
	"flowday/internal/services"
	"flowday/internal/worker"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	db.Connect()

	// Initialize achievements
	if err := services.InitializeAchievements(); err != nil {
		log.Printf("Failed to initialize achievements: %v", err)
	}

	if err := services.InitAIService(); err != nil {
		log.Printf("Failed to initialize AI Service: %v", err)
	}

	// Start Background Workers
	worker.StartAnalyzer()

	r := gin.Default()

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

	log.Println("Flowday running on :8080")
	r.Run(":8080")
}
