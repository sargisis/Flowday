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
	"flowday/internal/websocket"

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

	// Use the new helper method that handles null/empty dates
	dueDate := req.GetDueDateTimePtr()
	
	// Default due date to "Today" if not provided
	if dueDate == nil {
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

	// ✅ NEW: Process mentions in task description
	if task.Description != "" {
		go func() {
			services.ProcessTaskMentions(task.ID, task.Description, userID.(primitive.ObjectID))
		}()
	}

	services.LogActivity(userID.(primitive.ObjectID), models.ActivityTaskCreated, "Created task: "+task.Title, map[string]string{"task_id": task.ID.Hex()})

	// ✅ REAL-TIME: Broadcast task creation via WebSocket
	go func() {
		// Get project owner and members for broadcast
		var project models.Project
		if err := db.Projects.FindOne(context.Background(), bson.M{"_id": projectID}).Decode(&project); err == nil {
			userIDs := []primitive.ObjectID{project.UserID}
			// Get project members
			cursor, _ := db.ProjectMembers.Find(context.Background(), bson.M{
				"project_id": projectID,
				"status":     "accepted",
			})
			var members []models.ProjectMember
			cursor.All(context.Background(), &members)
			for _, member := range members {
				userIDs = append(userIDs, member.UserID)
			}
			// Broadcast to all project members
			taskData := map[string]interface{}{
				"task_id":    task.ID.Hex(),
				"title":      task.Title,
				"project_id": projectID.Hex(),
				"status":     task.Status,
				"priority":   task.Priority,
			}
			websocket.BroadcastTaskCreate(userIDs, taskData)
		}
	}()

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
		dueDate := req.GetDueDateTimePtr()
		updates["due_date"] = dueDate
	}

	userID, _ := c.Get("user_id")
	if err := services.UpdateTask(userID.(primitive.ObjectID), taskID, updates); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// ✅ NEW: Process mentions if description was updated
	if description, ok := updates["description"].(string); ok && description != "" {
		go func() {
			services.ProcessTaskMentions(taskID, description, userID.(primitive.ObjectID))
		}()
	}

	// Get updated task for notifications
	task, _ := services.GetTask(userID.(primitive.ObjectID), taskID)

	// ✅ REAL-TIME: Broadcast task update via WebSocket
	go func() {
		if task != nil {
			// Get project owner and members for broadcast
			var project models.Project
			if err := db.Projects.FindOne(context.Background(), bson.M{"_id": task.ProjectID}).Decode(&project); err == nil {
				userIDs := []primitive.ObjectID{project.UserID}
				// Get project members
				cursor, _ := db.ProjectMembers.Find(context.Background(), bson.M{
					"project_id": task.ProjectID,
					"status":     "accepted",
				})
				var members []models.ProjectMember
				cursor.All(context.Background(), &members)
				for _, member := range members {
					userIDs = append(userIDs, member.UserID)
				}
				// Broadcast to all project members
				taskData := map[string]interface{}{
					"task_id":    task.ID.Hex(),
					"title":      task.Title,
					"project_id": task.ProjectID.Hex(),
					"status":     task.Status,
					"priority":   task.Priority,
					"updates":    updates,
				}
				websocket.BroadcastTaskUpdate(userIDs, taskData)
			}
		}
	}()

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

	// Get task info before deletion for WebSocket broadcast
	var task models.Task
	var projectID primitive.ObjectID
	if err := db.Tasks.FindOne(context.Background(), bson.M{"_id": taskID}).Decode(&task); err == nil {
		projectID = task.ProjectID
	}

	if err := services.DeleteTask(userID.(primitive.ObjectID), taskID); err != nil {
		log.Printf("[DeleteTask] Error: %v", err)
		if err.Error() == "task not found" || err.Error() == "project not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		}
		return
	}

	// ✅ REAL-TIME: Broadcast task deletion via WebSocket
	go func() {
		if !projectID.IsZero() {
			// Get project owner and members for broadcast
			var project models.Project
			if err := db.Projects.FindOne(context.Background(), bson.M{"_id": projectID}).Decode(&project); err == nil {
				userIDs := []primitive.ObjectID{project.UserID}
				// Get project members
				cursor, _ := db.ProjectMembers.Find(context.Background(), bson.M{
					"project_id": projectID,
					"status":     "accepted",
				})
				var members []models.ProjectMember
				cursor.All(context.Background(), &members)
				for _, member := range members {
					userIDs = append(userIDs, member.UserID)
				}
				// Broadcast to all project members
				websocket.BroadcastTaskDelete(userIDs, taskID.Hex())
			}
		}
	}()

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

func BulkUpdateTasksStatus(c *gin.Context) {
	var req dto.BulkUpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.TaskIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task_ids cannot be empty"})
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
	updatedCount, err := services.BulkUpdateTasksStatus(userID.(primitive.ObjectID), objectIDs, req.Status)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// ✅ REAL-TIME: Broadcast bulk update via WebSocket
	go func() {
		// Get all updated tasks to broadcast
		var tasks []models.Task
		ctx := context.Background()
		cursor, _ := db.Tasks.Find(ctx, bson.M{"_id": bson.M{"$in": objectIDs}})
		cursor.All(ctx, &tasks)

		// Collect unique project IDs
		projectIDs := make(map[primitive.ObjectID]bool)
		for _, task := range tasks {
			projectIDs[task.ProjectID] = true
		}

		// Broadcast to all project members
		for projectID := range projectIDs {
			var project models.Project
			if err := db.Projects.FindOne(ctx, bson.M{"_id": projectID}).Decode(&project); err == nil {
				userIDs := []primitive.ObjectID{project.UserID}
				cursor, _ := db.ProjectMembers.Find(ctx, bson.M{
					"project_id": projectID,
					"status":     "accepted",
				})
				var members []models.ProjectMember
				cursor.All(ctx, &members)
				for _, member := range members {
					userIDs = append(userIDs, member.UserID)
				}

				// Broadcast update for each task
				for _, task := range tasks {
					if task.ProjectID == projectID {
						taskData := map[string]interface{}{
							"task_id":    task.ID.Hex(),
							"title":      task.Title,
							"project_id": task.ProjectID.Hex(),
							"status":     task.Status,
							"priority":   task.Priority,
							"bulk_update": true,
						}
						websocket.BroadcastTaskUpdate(userIDs, taskData)
					}
				}
			}
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"message":       "Tasks updated successfully",
		"updated_count": updatedCount,
	})
}

func BulkUpdateTasksPriority(c *gin.Context) {
	var req dto.BulkUpdatePriorityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.TaskIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task_ids cannot be empty"})
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
	updatedCount, err := services.BulkUpdateTasksPriority(userID.(primitive.ObjectID), objectIDs, req.Priority)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// ✅ REAL-TIME: Broadcast bulk update via WebSocket
	go func() {
		// Get all updated tasks to broadcast
		var tasks []models.Task
		ctx := context.Background()
		cursor, _ := db.Tasks.Find(ctx, bson.M{"_id": bson.M{"$in": objectIDs}})
		cursor.All(ctx, &tasks)

		// Collect unique project IDs
		projectIDs := make(map[primitive.ObjectID]bool)
		for _, task := range tasks {
			projectIDs[task.ProjectID] = true
		}

		// Broadcast to all project members
		for projectID := range projectIDs {
			var project models.Project
			if err := db.Projects.FindOne(ctx, bson.M{"_id": projectID}).Decode(&project); err == nil {
				userIDs := []primitive.ObjectID{project.UserID}
				cursor, _ := db.ProjectMembers.Find(ctx, bson.M{
					"project_id": projectID,
					"status":     "accepted",
				})
				var members []models.ProjectMember
				cursor.All(ctx, &members)
				for _, member := range members {
					userIDs = append(userIDs, member.UserID)
				}

				// Broadcast update for each task
				for _, task := range tasks {
					if task.ProjectID == projectID {
						taskData := map[string]interface{}{
							"task_id":    task.ID.Hex(),
							"title":      task.Title,
							"project_id": task.ProjectID.Hex(),
							"status":     task.Status,
							"priority":   task.Priority,
							"bulk_update": true,
						}
						websocket.BroadcastTaskUpdate(userIDs, taskData)
					}
				}
			}
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"message":       "Tasks updated successfully",
		"updated_count": updatedCount,
	})
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

	// ✅ SECURITY: Validate MIME type from Content-Type header
	contentType := file.Header.Get("Content-Type")
	// Allow common file types for task attachments (more permissive than avatars)
	allowedMimeTypes := map[string]bool{
		// Images
		"image/jpeg":    true,
		"image/jpg":     true,
		"image/png":     true,
		"image/gif":     true,
		"image/webp":    true,
		"image/svg+xml": true,
		// Documents
		"application/pdf":    true,
		"application/msword": true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
		"application/vnd.ms-excel": true,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
		"text/plain": true,
		"text/csv":   true,
		// Archives
		"application/zip":              true,
		"application/x-zip-compressed": true,
	}

	// If Content-Type is provided, validate it
	if contentType != "" && !allowedMimeTypes[contentType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File type not allowed. Only images, documents, and archives are permitted"})
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
		"php": true, "asp": true, "aspx": true, "jsp": true,
	}
	if dangerousExts[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File type not allowed. Executable files are prohibited"})
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

// AddTaskDependency handles POST /tasks/:id/dependencies
func AddTaskDependency(c *gin.Context) {
	taskID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id format"})
		return
	}

	var req dto.AddTaskDependencyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dependsOnTaskID, err := primitive.ObjectIDFromHex(req.DependsOnTaskID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid depends_on_task_id format"})
		return
	}

	userID, _ := c.Get("user_id")
	if err := services.AddTaskDependency(userID.(primitive.ObjectID), taskID, dependsOnTaskID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// RemoveTaskDependency handles DELETE /tasks/:id/dependencies
func RemoveTaskDependency(c *gin.Context) {
	taskID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id format"})
		return
	}

	var req dto.RemoveTaskDependencyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dependsOnTaskID, err := primitive.ObjectIDFromHex(req.DependsOnTaskID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid depends_on_task_id format"})
		return
	}

	userID, _ := c.Get("user_id")
	if err := services.RemoveTaskDependency(userID.(primitive.ObjectID), taskID, dependsOnTaskID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetTaskDependencies handles GET /tasks/:id/dependencies
func GetTaskDependencies(c *gin.Context) {
	taskID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id format"})
		return
	}

	userID, _ := c.Get("user_id")
	dependencies, err := services.GetTaskDependencies(userID.(primitive.ObjectID), taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dependencies)
}

// CreateTaskTemplate handles POST /templates
func CreateTaskTemplate(c *gin.Context) {
	var req dto.CreateTaskTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	template, err := services.CreateTaskTemplate(userID.(primitive.ObjectID), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, template)
}

// GetTaskTemplates handles GET /templates
func GetTaskTemplates(c *gin.Context) {
	userID, _ := c.Get("user_id")
	templates, err := services.GetTaskTemplates(userID.(primitive.ObjectID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, templates)
}

// GetTaskTemplate handles GET /templates/:id
func GetTaskTemplate(c *gin.Context) {
	templateID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template id format"})
		return
	}

	userID, _ := c.Get("user_id")
	template, err := services.GetTaskTemplate(userID.(primitive.ObjectID), templateID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, template)
}

// UpdateTaskTemplate handles PATCH /templates/:id
func UpdateTaskTemplate(c *gin.Context) {
	templateID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template id format"})
		return
	}

	var req dto.UpdateTaskTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	template, err := services.UpdateTaskTemplate(userID.(primitive.ObjectID), templateID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, template)
}

// DeleteTaskTemplate handles DELETE /templates/:id
func DeleteTaskTemplate(c *gin.Context) {
	templateID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template id format"})
		return
	}

	userID, _ := c.Get("user_id")
	if err := services.DeleteTaskTemplate(userID.(primitive.ObjectID), templateID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// CreateTaskFromTemplate handles POST /templates/:id/create-task
func CreateTaskFromTemplate(c *gin.Context) {
	templateID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template id format"})
		return
	}

	var req dto.CreateTaskFromTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	task, err := services.CreateTaskFromTemplate(userID.(primitive.ObjectID), templateID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, task)
}
