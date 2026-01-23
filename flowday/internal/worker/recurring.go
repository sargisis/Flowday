package worker

import (
	"context"
	"time"

	"flowday/internal/db"
	"flowday/internal/logger"
	"flowday/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StartRecurringTasksProcessor starts the background worker that processes recurring tasks
func StartRecurringTasksProcessor() {
	ticker := time.NewTicker(1 * time.Hour) // Run every hour
	go func() {
		// Run immediately on startup
		processRecurringTasks()
		
		for {
			select {
			case <-ticker.C:
				processRecurringTasks()
			}
		}
	}()
	logger.Log.Info("✅ Recurring Tasks Processor started")
}

func processRecurringTasks() {
	logger.Log.Info("[Worker] Processing recurring tasks...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Find all recurring tasks that need to be processed
	filter := bson.M{
		"is_recurring": true,
		"status":       "done", // Only process completed recurring tasks
		"recurrence":   bson.M{"$exists": true, "$ne": nil},
	}

	cursor, err := db.Tasks.Find(ctx, filter)
	if err != nil {
		logger.Log.WithError(err).Error("[Worker] Failed to fetch recurring tasks")
		return
	}
	defer cursor.Close(ctx)

	now := time.Now()
	processedCount := 0
	errorCount := 0

	for cursor.Next(ctx) {
		var task models.Task
		if err := cursor.Decode(&task); err != nil {
			logger.Log.WithError(err).Warn("[Worker] Failed to decode recurring task")
			errorCount++
			continue
		}

		if task.Recurrence == nil {
			continue
		}

		// ✅ ENHANCED: Check if recurrence has ended
		if task.Recurrence.EndDate != nil && now.After(*task.Recurrence.EndDate) {
			logger.Log.Debugf("[Worker] Recurring task %s has reached end date, skipping", task.ID.Hex())
			continue
		}

		// ✅ ENHANCED: Check if max count reached
		if task.Recurrence.Count != nil {
			// Count how many times this task has been created
			count, err := db.Tasks.CountDocuments(ctx, bson.M{
				"recurrence.last_created": bson.M{"$exists": true},
				"title":                   task.Title,
				"project_id":              task.ProjectID,
			})
			if err == nil && int64(*task.Recurrence.Count) <= count {
				logger.Log.Debugf("[Worker] Recurring task %s has reached max count (%d), skipping", 
					task.ID.Hex(), *task.Recurrence.Count)
				continue
			}
		}

		// Check if it's time to create a new instance
		if shouldCreateRecurringTask(task, now) {
			if err := createNextRecurringTask(ctx, task); err != nil {
				logger.Log.WithError(err).Errorf("[Worker] Failed to create recurring task: %s", task.ID.Hex())
				errorCount++
			} else {
				processedCount++
			}
		}
	}

	if processedCount > 0 || errorCount > 0 {
		logger.Log.Infof("[Worker] Processed %d recurring tasks, %d errors", processedCount, errorCount)
	}
}

func shouldCreateRecurringTask(task models.Task, now time.Time) bool {
	if task.Recurrence.LastCreated == nil {
		// ✅ ENHANCED: First time processing - check if task was completed recently (within last 24 hours)
		// This prevents creating tasks for old completed tasks
		if task.UpdatedAt.IsZero() {
			return false
		}
		hoursSinceCompletion := now.Sub(task.UpdatedAt).Hours()
		return hoursSinceCompletion <= 24 // Only create if completed within last 24 hours
	}

	lastCreated := *task.Recurrence.LastCreated
	recurrence := task.Recurrence

	// ✅ ENHANCED: Improved time calculations
	switch recurrence.Type {
	case "daily":
		hoursSince := now.Sub(lastCreated).Hours()
		return hoursSince >= float64(24*recurrence.Interval)
	case "weekly":
		daysSince := now.Sub(lastCreated).Hours() / 24
		return daysSince >= float64(7*recurrence.Interval)
	case "monthly":
		// More accurate month calculation
		monthsSince := float64(now.Year()-lastCreated.Year())*12 + float64(now.Month()-lastCreated.Month())
		return monthsSince >= float64(recurrence.Interval)
	case "yearly":
		yearsSince := float64(now.Year() - lastCreated.Year())
		return yearsSince >= float64(recurrence.Interval)
	default:
		logger.Log.Warnf("[Worker] Unknown recurrence type: %s", recurrence.Type)
		return false
	}
}

func createNextRecurringTask(ctx context.Context, originalTask models.Task) error {
	// ✅ ENHANCED: Create a new task instance based on the recurring task
	now := time.Now()
	newTask := models.Task{
		ID:          primitive.NewObjectID(),
		Title:       originalTask.Title,
		Description: originalTask.Description,
		Status:      "todo",
		Priority:    originalTask.Priority,
		ProjectID:   originalTask.ProjectID,
		IsRecurring: true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// ✅ ENHANCED: Copy recurrence settings and update last created time
	if originalTask.Recurrence != nil {
		recurrence := *originalTask.Recurrence
		recurrence.LastCreated = &now
		newTask.Recurrence = &recurrence
	}

	// Insert the new task
	_, err := db.Tasks.InsertOne(ctx, newTask)
	if err != nil {
		logger.Log.WithError(err).Errorf("[Worker] Failed to create recurring task: %s", originalTask.ID.Hex())
		return err
	}

	// ✅ ENHANCED: Update the original task's recurrence last created time
	update := bson.M{
		"$set": bson.M{
			"recurrence.last_created": now,
		},
	}
	_, err = db.Tasks.UpdateOne(ctx, bson.M{"_id": originalTask.ID}, update)
	if err != nil {
		logger.Log.WithError(err).Warnf("[Worker] Failed to update recurring task last_created: %s", originalTask.ID.Hex())
		// Don't return error - task was created successfully
	}

	logger.Log.Infof("[Worker] ✅ Created recurring task: %s (ID: %s)", newTask.Title, newTask.ID.Hex())
	return nil
}
