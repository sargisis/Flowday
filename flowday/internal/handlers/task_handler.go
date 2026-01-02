package handlers

import (
	"log"
	"net/http"
	"time"

	"flowday/internal/dto"
	"flowday/internal/models"
	"flowday/internal/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func CreateTask(c *gin.Context) {
	var req dto.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	projectID, err := req.GetProjectObjectID()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project_id format"})
		return
	}

	// Default due date to "Today" if not provided
	var dueDate *time.Time
	if req.DueDate != nil {
		dueDate = req.DueDate
	} else {
		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		dueDate = &today
	}

	userID, _ := c.Get("user_id")
	task := models.Task{
		Title:       req.Title,
		Description: req.Description,
		Priority:    req.Priority,
		DueDate:     dueDate,
		ProjectID:   projectID,
		Status:      "todo",
	}

	if err := services.CreateTask(userID.(primitive.ObjectID), &task); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	services.LogActivity(userID.(primitive.ObjectID), models.ActivityTaskCreated, "Created task: "+task.Title, map[string]string{"task_id": task.ID.Hex()})

	c.JSON(http.StatusCreated, task)
}

func GetTasks(c *gin.Context) {
	projectIDStr := c.Query("project_id")
	if projectIDStr == "" {
		c.JSON(400, gin.H{"error": "project_id is required"})
		return
	}

	projectID, err := primitive.ObjectIDFromHex(projectIDStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid project_id format"})
		return
	}

	userID, _ := c.Get("user_id")
	tasks, err := services.GetTasksByProject(userID.(primitive.ObjectID), projectID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, tasks)
}

func UpdateTask(c *gin.Context) {
	taskID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id format"})
		return
	}

	var req dto.UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.DueDate != nil {
		updates["due_date"] = req.DueDate
	}

	userID, _ := c.Get("user_id")
	if err := services.UpdateTask(userID.(primitive.ObjectID), taskID, updates); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	if req.Status != nil && *req.Status == "done" {
		task, _ := services.GetTask(userID.(primitive.ObjectID), taskID)
		if task != nil {
			services.LogActivity(userID.(primitive.ObjectID), models.ActivityTaskCompleted, "Completed task: "+task.Title, map[string]string{"task_id": task.ID.Hex()})

			// Trigger achievement check
			go services.CheckAndAwardAchievements(userID.(primitive.ObjectID), "task_completed", map[string]interface{}{
				"task_id":      task.ID.Hex(),
				"completed_at": time.Now(),
			})
		}
	}

	c.Status(http.StatusNoContent)
}

func DeleteTask(c *gin.Context) {
	taskID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id format"})
		return
	}

	userID, _ := c.Get("user_id")
	if err := services.DeleteTask(userID.(primitive.ObjectID), taskID); err != nil {
		log.Printf("[DeleteTask] Error: %v", err)
		if err.Error() == "task not found" || err.Error() == "project not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		}
		return
	}

	c.Status(http.StatusNoContent)
}

func GetTask(c *gin.Context) {
	taskID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id format"})
		return
	}

	userID, _ := c.Get("user_id")
	task, err := services.GetTask(userID.(primitive.ObjectID), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	c.JSON(http.StatusOK, task)
}

func GetAllTasks(c *gin.Context) {
	userID, _ := c.Get("user_id")
	tasks, err := services.GetAllTasks(userID.(primitive.ObjectID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tasks)
}
