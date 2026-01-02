package main

import (
	"log"
	"time"

	"flowday/internal/db"
	"flowday/internal/router"
	"flowday/internal/services"

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

	r := gin.Default()

	// ✅ CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:5174"}, // Vite
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	router.Setup(r)

	log.Println("Flowday running on :8080")
	r.Run(":8080")
}
