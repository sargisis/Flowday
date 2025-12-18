package handlers

import (
	"net/http"

	"flowday/internal/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetTaskStats(c *gin.Context) {
	userID, _ := c.Get("user_id")
	stats, err := services.GetTaskStats(userID.(primitive.ObjectID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}
