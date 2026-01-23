package handlers

import (
	"net/http"
	"time"

	"flowday/internal/dto"
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

// GetTaskAnalytics handles GET /api/v1/tasks/analytics
func GetTaskAnalytics(c *gin.Context) {
	var req dto.TaskAnalyticsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "BAD_REQUEST",
				"message": "Invalid query parameters",
			},
		})
		return
	}

	userID, _ := c.Get("user_id")

	// Parse optional project ID
	var projectID *primitive.ObjectID
	if req.ProjectID != "" {
		pid, err := primitive.ObjectIDFromHex(req.ProjectID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":    "BAD_REQUEST",
					"message": "Invalid project_id format",
				},
			})
			return
		}
		projectID = &pid
	}

	// Parse optional date range
	var startDate, endDate *time.Time
	if req.StartDate != "" {
		if t, err := time.Parse("2006-01-02", req.StartDate); err == nil {
			startDate = &t
		}
	}
	if req.EndDate != "" {
		if t, err := time.Parse("2006-01-02", req.EndDate); err == nil {
			// Set to end of day
			endOfDay := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, t.Location())
			endDate = &endOfDay
		}
	}

	analytics, err := services.GetTaskAnalytics(
		userID.(primitive.ObjectID),
		projectID,
		startDate,
		endDate,
	)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error": gin.H{
				"code":    "FORBIDDEN",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, analytics)
}
