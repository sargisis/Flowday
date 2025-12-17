package handlers

import (
	"net/http"
	"strconv"
	"time"

	"flowday/internal/dto"
	"flowday/internal/models"
	"flowday/internal/services"

	"github.com/gin-gonic/gin"
)

func CreateTask(c *gin.Context) {
	var req dto.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	task := models.Task{
		Title:     req.Title,
		Priority:  req.Priority,
		DueDate:   dueDate,
		ProjectID: req.ProjectID,
		Status:    "todo",
	}

	if err := services.CreateTask(c.GetUint("user_id"), &task); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, task)
}

func GetTasks(c *gin.Context) {
	projectIDStr := c.Query("project_id")
	if projectIDStr == "" {
		c.JSON(400, gin.H{"error": "project_id is required"})
		return
	}

	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid project_id"})
		return
	}

	var q dto.PaginationQuery
	_ = c.ShouldBindQuery(&q)

	tasks, err := services.GetTasksByProjectPaginated(
		c.GetUint("user_id"),
		uint(projectID),
		q.Limit,
		q.Offset,
		q.Order,
		q.Dir,
	)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, tasks)
}


func UpdateTask(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
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
	if req.DueDate != nil {
		updates["due_date"] = req.DueDate
	}

	if err := services.UpdateTask(c.GetUint("user_id"), uint(id), updates); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func DeleteTask(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	if err := services.DeleteTask(c.GetUint("user_id"), uint(id)); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
