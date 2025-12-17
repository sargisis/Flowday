package main

import (
	"log"

	"flowday/internal/db"
	"flowday/internal/router"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	db.Init()
	db.Migrate()

	r := gin.Default()
	router.Setup(r)

	log.Println("Flowday running on :8080")
	r.Run(":8080")
}
