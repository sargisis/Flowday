package handlers

import (
	"log"
	"net/http"

	"flowday/internal/models"
	"flowday/internal/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// DecomposeTask breaks a task into multiple new task cards
func DecomposeTask(c *gin.Context) {
	taskID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id format"})
		return
	}

	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	task, err := services.GetTask(userID.(primitive.ObjectID), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	subtaskTitles, err := services.DecomposeTask(c.Request.Context(), task.Title, task.Description)
	if err != nil {
		log.Printf("[AI] Decompose error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI decomposition failed: " + err.Error()})
		return
	}

	createdSubtasks := []models.Task{}
	for _, title := range subtaskTitles {
		newSubtask := models.Task{
			Title:     title,
			Status:    "todo",
			Priority:  task.Priority,
			ProjectID: task.ProjectID,
		}
		if err := services.CreateTask(userID.(primitive.ObjectID), &newSubtask); err != nil {
			log.Printf("[AI] Failed to create subtask: %v", err)
			continue
		}
		createdSubtasks = append(createdSubtasks, newSubtask)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Task decomposed successfully",
		"subtasks": createdSubtasks,
	})
}

// EnrichTask generates a detailed plan and updates the task description
func EnrichTask(c *gin.Context) {
	taskID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id format"})
		return
	}

	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	task, err := services.GetTask(userID.(primitive.ObjectID), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	plan, err := services.GenerateTaskPlan(c.Request.Context(), task.Title)
	if err != nil {
		log.Printf("[AI] Enrich error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI planning failed: " + err.Error()})
		return
	}

	if err := services.UpdateTask(userID.(primitive.ObjectID), taskID, map[string]interface{}{
		"description": plan,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update task description"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Task enriched successfully",
		"description": plan,
	})
}

// GetHealthAdvice generates personalized productivity advice based on task stats
func GetHealthAdvice(c *gin.Context) {
	var context services.AnalysisContext
	if err := c.ShouldBindJSON(&context); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid context format"})
		return
	}

	advice, err := services.GetHealthAdvice(c.Request.Context(), context)
	if err != nil {
		log.Printf("[AI] Health advice error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate intelligent advice"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"advice": advice,
	})
}
