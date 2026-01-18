package handlers

import (
	"encoding/csv"
	"fmt"
	"net/http"

	"flowday/internal/db"
	"flowday/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ExportTasksCSV handles GET /export/tasks/csv
func ExportTasksCSV(c *gin.Context) {
	projectID := c.Query("project_id")

	filter := bson.M{}
	if projectID != "" {
		pid, err := primitive.ObjectIDFromHex(projectID)
		if err == nil {
			filter["project_id"] = pid
		}
	}

	cursor, err := db.Tasks.Find(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer cursor.Close(c.Request.Context())

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=tasks_%s.csv", projectID))
	c.Header("Content-Transfer-Encoding", "binary")

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// Write header
	writer.Write([]string{"ID", "Title", "Description", "Status", "Priority", "Due Date", "Project ID", "Created At", "Updated At"})

	// Write rows
	for cursor.Next(c.Request.Context()) {
		var task models.Task
		if err := cursor.Decode(&task); err != nil {
			continue
		}

		var dueDate, createdAt, updatedAt string
		if task.DueDate != nil {
			dueDate = task.DueDate.Format("2006-01-02")
		}
		createdAt = task.CreatedAt.Format("2006-01-02 15:04:05")
		updatedAt = task.UpdatedAt.Format("2006-01-02 15:04:05")

		writer.Write([]string{
			task.ID.Hex(),
			task.Title,
			task.Description,
			task.Status,
			task.Priority,
			dueDate,
			task.ProjectID.Hex(),
			createdAt,
			updatedAt,
		})
	}
}

// ExportTasksJSON handles GET /export/tasks/json
func ExportTasksJSON(c *gin.Context) {
	projectID := c.Query("project_id")

	filter := bson.M{}
	if projectID != "" {
		pid, err := primitive.ObjectIDFromHex(projectID)
		if err == nil {
			filter["project_id"] = pid
		}
	}

	cursor, err := db.Tasks.Find(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer cursor.Close(c.Request.Context())

	var tasks []models.Task
	if err = cursor.All(c.Request.Context(), &tasks); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=tasks_%s.json", projectID))
	c.JSON(http.StatusOK, tasks)
}

// ImportTasksCSV handles POST /import/tasks/csv
func ImportTasksCSV(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	openedFile, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to open file"})
		return
	}
	defer openedFile.Close()

	reader := csv.NewReader(openedFile)
	records, err := reader.ReadAll()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse CSV"})
		return
	}

	if len(records) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CSV must have at least a header and one row"})
		return
	}

	imported := 0
	errors := []string{}

	// Skip header row
	for i := 1; i < len(records); i++ {
		record := records[i]
		if len(record) < 6 {
			errors = append(errors, fmt.Sprintf("Row %d: insufficient columns", i+1))
			continue
		}

		projectID, err := primitive.ObjectIDFromHex(record[6])
		if err != nil {
			errors = append(errors, fmt.Sprintf("Row %d: invalid project_id", i+1))
			continue
		}

		task := models.Task{
			ID:          primitive.NewObjectID(),
			Title:       record[1],
			Description: record[2],
			Status:      record[3],
			Priority:    record[4],
			ProjectID:   projectID,
		}

		_, err = db.Tasks.InsertOne(c.Request.Context(), task)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Row %d: %s", i+1, err.Error()))
			continue
		}

		imported++
	}

	c.JSON(http.StatusOK, gin.H{
		"imported": imported,
		"errors":   errors,
		"message":  fmt.Sprintf("Imported %d tasks", imported),
	})
}

// ImportTasksJSON handles POST /import/tasks/json
func ImportTasksJSON(c *gin.Context) {
	var tasks []map[string]interface{}
	if err := c.ShouldBindJSON(&tasks); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	imported := 0
	errors := []string{}

	for i, taskData := range tasks {
		title, ok := taskData["title"].(string)
		if !ok || title == "" {
			errors = append(errors, fmt.Sprintf("Task %d: title is required", i+1))
			continue
		}

		projectIDStr, ok := taskData["project_id"].(string)
		if !ok {
			errors = append(errors, fmt.Sprintf("Task %d: project_id is required", i+1))
			continue
		}

		projectID, err := primitive.ObjectIDFromHex(projectIDStr)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Task %d: invalid project_id", i+1))
			continue
		}

		task := models.Task{
			ID:        primitive.NewObjectID(),
			Title:     title,
			Status:    "todo",
			ProjectID: projectID,
		}

		if desc, ok := taskData["description"].(string); ok {
			task.Description = desc
		}
		if priority, ok := taskData["priority"].(string); ok {
			task.Priority = priority
		}

		_, err = db.Tasks.InsertOne(c.Request.Context(), task)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Task %d: %s", i+1, err.Error()))
			continue
		}

		imported++
	}

	c.JSON(http.StatusOK, gin.H{
		"imported": imported,
		"errors":   errors,
		"message":  fmt.Sprintf("Imported %d tasks", imported),
	})
}
