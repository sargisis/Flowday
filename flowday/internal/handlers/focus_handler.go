package handlers

import (
	"flowday/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FocusSessionRequest struct {
	TaskID    string `json:"task_id"`
	TaskTitle string `json:"task_title"`
	Duration  int    `json:"duration"`
}

func CreateFocusSession(c *gin.Context) {
	u, _ := c.Get("user_id")
	userID := u.(primitive.ObjectID)

	var req FocusSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var taskID *primitive.ObjectID
	if req.TaskID != "" {
		tid, err := primitive.ObjectIDFromHex(req.TaskID)
		if err == nil {
			taskID = &tid
		}
	}

	err := services.SaveFocusSession(userID, taskID, req.TaskTitle, req.Duration)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save focus session"})
		return
	}

	services.LogActivity(userID, "focus_completed", "Completed focus session: "+req.TaskTitle, map[string]string{
		"duration": string(rune(req.Duration)), // simplify for log
		"task_id":  req.TaskID,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Focus session saved successfully"})
}

func GetFocusSessions(c *gin.Context) {
	u, _ := c.Get("user_id")
	userID := u.(primitive.ObjectID)

	sessions, err := services.GetFocusHistory(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch focus history"})
		return
	}

	c.JSON(http.StatusOK, sessions)
}
