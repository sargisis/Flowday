package handlers

import (
	"context"
	"net/http"
	"time"

	"flowday/internal/db"
	"flowday/internal/dto"
	"flowday/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// StartTimeEntry handles POST /api/v1/time/start
func StartTimeEntry(c *gin.Context) {
	var req dto.StartTimeEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	taskID, err := primitive.ObjectIDFromHex(req.TaskID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task_id format"})
		return
	}

	userID, _ := c.Get("user_id")

	// Check if user has an active time entry
	var existingEntry models.TimeEntry
	filter := bson.M{
		"user_id":   userID.(primitive.ObjectID),
		"end_time":  nil,
	}
	err = db.TimeEntries.FindOne(context.Background(), filter).Decode(&existingEntry)
	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "you already have an active time entry"})
		return
	}

	// Create new time entry
	entry := models.TimeEntry{
		ID:        primitive.NewObjectID(),
		UserID:    userID.(primitive.ObjectID),
		TaskID:    taskID,
		StartTime: time.Now(),
		Duration:  0,
		CreatedAt: time.Now(),
	}

	_, err = db.TimeEntries.InsertOne(context.Background(), entry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start time entry"})
		return
	}

	c.JSON(http.StatusCreated, entry)
}

// StopTimeEntry handles POST /api/v1/time/stop
func StopTimeEntry(c *gin.Context) {
	var req dto.StopTimeEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entryID, err := primitive.ObjectIDFromHex(req.EntryID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid entry_id format"})
		return
	}

	userID, _ := c.Get("user_id")

	// Get the time entry
	var entry models.TimeEntry
	filter := bson.M{
		"_id":     entryID,
		"user_id": userID.(primitive.ObjectID),
	}
	err = db.TimeEntries.FindOne(context.Background(), filter).Decode(&entry)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "time entry not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve time entry"})
		return
	}

	// Calculate duration
	endTime := time.Now()
	duration := endTime.Sub(entry.StartTime).Hours()

	// Update the time entry
	update := bson.M{
		"$set": bson.M{
			"end_time": endTime,
			"duration": duration,
		},
	}
	_, err = db.TimeEntries.UpdateOne(context.Background(), filter, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to stop time entry"})
		return
	}

	// Update entry for response
	entry.EndTime = &endTime
	entry.Duration = duration

	c.JSON(http.StatusOK, entry)
}

// GetTimeEntries handles GET /api/v1/time/entries
func GetTimeEntries(c *gin.Context) {
	userID, _ := c.Get("user_id")
	taskIDStr := c.Query("task_id")

	filter := bson.M{"user_id": userID.(primitive.ObjectID)}
	if taskIDStr != "" {
		taskID, err := primitive.ObjectIDFromHex(taskIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task_id format"})
			return
		}
		filter["task_id"] = taskID
	}

	opts := options.Find().SetSort(bson.M{"created_at": -1})
	cursor, err := db.TimeEntries.Find(context.Background(), filter, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve time entries"})
		return
	}
	defer cursor.Close(context.Background())

	var entries []models.TimeEntry
	if err = cursor.All(context.Background(), &entries); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode time entries"})
		return
	}

	c.JSON(http.StatusOK, entries)
}

// DeleteTimeEntry handles DELETE /api/v1/time/entries/:id
func DeleteTimeEntry(c *gin.Context) {
	entryIDStr := c.Param("id")
	entryID, err := primitive.ObjectIDFromHex(entryIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid entry id format"})
		return
	}

	userID, _ := c.Get("user_id")

	// Verify the time entry belongs to the user
	filter := bson.M{
		"_id":     entryID,
		"user_id": userID.(primitive.ObjectID),
	}

	result, err := db.TimeEntries.DeleteOne(context.Background(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete time entry"})
		return
	}

	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "time entry not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetTimeReport handles GET /api/v1/time/report
func GetTimeReport(c *gin.Context) {
	userID, _ := c.Get("user_id")

	// Get all time entries for the user
	filter := bson.M{"user_id": userID.(primitive.ObjectID)}
	cursor, err := db.TimeEntries.Find(context.Background(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve time entries"})
		return
	}
	defer cursor.Close(context.Background())

	var entries []models.TimeEntry
	if err = cursor.All(context.Background(), &entries); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode time entries"})
		return
	}

	// Calculate totals
	totalHours := 0.0
	totalEntries := len(entries)
	byTask := make(map[string]float64)

	for _, entry := range entries {
		totalHours += entry.Duration
		taskID := entry.TaskID.Hex()
		byTask[taskID] += entry.Duration
	}

	report := gin.H{
		"total_hours":  totalHours,
		"total_entries": totalEntries,
		"by_task":      byTask,
		"entries":      entries,
	}

	c.JSON(http.StatusOK, report)
}
