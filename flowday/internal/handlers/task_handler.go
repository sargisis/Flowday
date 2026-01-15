package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"flowday/internal/db"
	"flowday/internal/dto"
	"flowday/internal/models"
	"flowday/internal/services"
	"flowday/internal/utils"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
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

	// Send Slack notification (OAuth preferred, fallback to webhook)
	go func() {
		var user models.User
		if err := db.Users.FindOne(context.Background(), bson.M{"_id": userID}).Decode(&user); err == nil {
			var project models.Project
			if err := db.Projects.FindOne(context.Background(), bson.M{"_id": projectID}).Decode(&project); err == nil {
				// Try OAuth first
				if user.SlackAccessToken != "" {
					// Send to channel if configured
					if user.SlackChannelID != "" {
						utils.SendTaskNotificationOAuth(user.SlackAccessToken, user.SlackChannelID, task.Title, "created", project.Name, task.Priority, task.Status, "", task.ID.Hex())
					}
					// Also send DM to user if Slack User ID is available
					if user.SlackUserID != "" {
						utils.SendTaskNotificationOAuth(user.SlackAccessToken, "", task.Title, "created", project.Name, task.Priority, task.Status, user.SlackUserID, task.ID.Hex())
					}
				} else if user.SlackWebhookURL != "" {
					// Fallback to webhook
					utils.SendTaskNotification(user.SlackWebhookURL, task.Title, "created", project.Name, task.Priority, task.Status)
				}
			}
		}
	}() // This fixed the "expression in go must be function call" lint error by calling the goroutine properly.

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

	// Parse pagination query
	var pagination dto.PaginationQuery
	if err := c.ShouldBindQuery(&pagination); err != nil {
		// If pagination params are not provided, use defaults
		pagination = dto.PaginationQuery{}
	}

	userID, _ := c.Get("user_id")
	tasks, meta, err := services.GetTasksByProjectPaginated(userID.(primitive.ObjectID), projectID, pagination)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"data": tasks,
		"meta": meta,
	})
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

	// Get updated task for notifications
	task, _ := services.GetTask(userID.(primitive.ObjectID), taskID)

	// Trigger achievement/streak check on any update (activity)
	go services.CheckAndAwardAchievements(userID.(primitive.ObjectID), "task_updated", map[string]interface{}{
		"updated_at": time.Now(),
	})

	// Send Slack notification (OAuth preferred, fallback to webhook)
	if task != nil {
		go func() {
			var user models.User
			if err := db.Users.FindOne(context.Background(), bson.M{"_id": userID}).Decode(&user); err == nil {
				var project models.Project
				if err := db.Projects.FindOne(context.Background(), bson.M{"_id": task.ProjectID}).Decode(&project); err == nil {
					action := "updated"
					if req.Status != nil && strings.ToLower(*req.Status) == "done" {
						action = "completed"
					}
					// Try OAuth first
					if user.SlackAccessToken != "" {
						// Send to channel if configured
						if user.SlackChannelID != "" {
							utils.SendTaskNotificationOAuth(user.SlackAccessToken, user.SlackChannelID, task.Title, action, project.Name, task.Priority, task.Status, "", task.ID.Hex())
						}
						// Also send DM to user if Slack User ID is available
						if user.SlackUserID != "" {
							utils.SendTaskNotificationOAuth(user.SlackAccessToken, "", task.Title, action, project.Name, task.Priority, task.Status, user.SlackUserID, task.ID.Hex())
						}
					} else if user.SlackWebhookURL != "" {
						// Fallback to webhook
						utils.SendTaskNotification(user.SlackWebhookURL, task.Title, action, project.Name, task.Priority, task.Status)
					}
				}
			}
		}()
	}

	if req.Status != nil && (strings.ToLower(*req.Status) == "done") {
		if task != nil {
			services.LogActivity(userID.(primitive.ObjectID), models.ActivityTaskCompleted, "Completed task: "+task.Title, map[string]string{"task_id": task.ID.Hex()})

			// Trigger specific achievement check for completion
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

func BulkDeleteTasks(c *gin.Context) {
	var req struct {
		TaskIDs []string `json:"task_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if len(req.TaskIDs) == 0 {
		c.Status(http.StatusNoContent)
		return
	}

	var objectIDs []primitive.ObjectID
	for _, id := range req.TaskIDs {
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id format: " + id})
			return
		}
		objectIDs = append(objectIDs, oid)
	}

	userID, _ := c.Get("user_id")
	if err := services.BulkDeleteTasks(userID.(primitive.ObjectID), objectIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

func SearchTasks(c *gin.Context) {
	userID, _ := c.Get("user_id")

	// Parse search query
	var searchQuery dto.SearchTasksQuery
	if err := c.ShouldBindQuery(&searchQuery); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters"})
		return
	}

	tasks, meta, err := services.SearchTasks(userID.(primitive.ObjectID), searchQuery)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": tasks,
		"meta": meta,
	})
}

// UploadTaskAttachment handles file uploads for tasks
func UploadTaskAttachment(c *gin.Context) {
	taskID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id format"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	// ✅ SECURITY: Validate file size (max 10MB for attachments)
	const maxAttachmentSize = 10 * 1024 * 1024 // 10MB
	if file.Size > maxAttachmentSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File size exceeds maximum allowed size (10MB)"})
		return
	}

	// ✅ SECURITY: Sanitize file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	ext = strings.TrimPrefix(ext, ".")
	ext = strings.Trim(ext, "/\\")

	// Basic extension validation - prevent executable files
	dangerousExts := map[string]bool{
		"exe": true, "bat": true, "cmd": true, "com": true, "pif": true,
		"scr": true, "vbs": true, "js": true, "jar": true, "sh": true,
	}
	if dangerousExts[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File type not allowed"})
		return
	}

	// Verify task access (will be checked in AddTaskAttachment)
	userID, _ := c.Get("user_id")

	// Create uploads/tasks directory if not exists
	uploadDir := filepath.Join("uploads", "tasks")
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.MkdirAll(uploadDir, 0755)
	}

	// ✅ SECURITY: Generate safe filename (no user input in path)
	var safeExt string
	if ext != "" {
		safeExt = "." + ext
	}
	filename := fmt.Sprintf("task_%s_%d%s", taskID.Hex(), time.Now().UnixNano(), safeExt)
	filePath := filepath.Join(uploadDir, filename)

	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	attachmentURL := fmt.Sprintf("/api/v1/uploads/tasks/%s", filename)

	// Determine type based on extension
	attachmentType := "file"
	imgExts := map[string]bool{"jpg": true, "jpeg": true, "png": true, "gif": true, "webp": true, "svg": true}
	if imgExts[ext] {
		attachmentType = "image"
	}

	// Create attachment object
	attachment := models.Attachment{
		ID:         primitive.NewObjectID().Hex(),
		URL:        attachmentURL,
		Type:       attachmentType,
		Filename:   file.Filename,
		Size:       file.Size,
		UploadedAt: time.Now(),
	}

	// Add attachment to task
	if err := services.AddTaskAttachment(userID.(primitive.ObjectID), taskID, attachment); err != nil {
		// Clean up uploaded file if database operation fails
		os.Remove(filePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save attachment"})
		return
	}

	c.JSON(http.StatusOK, attachment)
}

// GetTaskAttachments returns all attachments for a task
func GetTaskAttachments(c *gin.Context) {
	taskID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id format"})
		return
	}

	userID, _ := c.Get("user_id")
	attachments, err := services.GetTaskAttachments(userID.(primitive.ObjectID), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	c.JSON(http.StatusOK, attachments)
}

// DeleteTaskAttachment removes an attachment from a task
func DeleteTaskAttachment(c *gin.Context) {
	taskID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id format"})
		return
	}

	attachmentID := c.Param("attachment_id")
	if attachmentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "attachment_id is required"})
		return
	}

	userID, _ := c.Get("user_id")
	attachment, err := services.DeleteTaskAttachment(userID.(primitive.ObjectID), taskID, attachmentID)
	if err != nil {
		if err.Error() == "task not found" || err.Error() == "attachment not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		}
		return
	}

	// Delete physical file
	if attachment.URL != "" {
		// Extract filename from URL
		filename := filepath.Base(attachment.URL)
		filePath := filepath.Join("uploads", "tasks", filename)
		os.Remove(filePath) // Ignore error if file doesn't exist
	}

	c.Status(http.StatusNoContent)
}
