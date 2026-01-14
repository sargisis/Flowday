package handlers

import (
	"flowday/internal/dto"
	"flowday/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetActivityFeed(c *gin.Context) {
	u, _ := c.Get("user_id")
	userID := u.(primitive.ObjectID)

	// Parse pagination query
	var pagination dto.PaginationQuery
	if err := c.ShouldBindQuery(&pagination); err != nil {
		// If pagination params are not provided, use defaults
		pagination = dto.PaginationQuery{}
	}

	activities, meta, err := services.GetUserActivitiesPaginated(userID, pagination)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch activity feed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": activities,
		"meta": meta,
	})
}
