package handlers

import (
	"flowday/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetAchievements returns all available achievements with user progress
func GetAchievements(c *gin.Context) {
	u, _ := c.Get("user_id")
	userID := u.(primitive.ObjectID)

	achievements, err := services.GetUserAchievements(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch achievements"})
		return
	}

	c.JSON(http.StatusOK, achievements)
}

// GetStreak returns the user's current streak information
func GetStreak(c *gin.Context) {
	u, _ := c.Get("user_id")
	userID := u.(primitive.ObjectID)

	streak, err := services.GetUserStreak(userID)
	if err != nil {
		// Return empty streak if not found
		c.JSON(http.StatusOK, gin.H{
			"current_streak": 0,
			"longest_streak": 0,
		})
		return
	}

	c.JSON(http.StatusOK, streak)
}
