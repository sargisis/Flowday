package handlers

import (
	"flowday/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetActivityFeed(c *gin.Context) {
	u, _ := c.Get("user_id")
	userID := u.(primitive.ObjectID)

	activities, err := services.GetUserActivities(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch activity feed"})
		return
	}

	c.JSON(http.StatusOK, activities)
}
