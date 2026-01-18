package worker

import (
	"context"
	"log"
	"time"

	"flowday/internal/db"
	"flowday/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StartRecurringTasksProcessor starts the background worker that processes recurring tasks
func StartRecurringTasksProcessor() {
	ticker := time.NewTicker(1 * time.Hour) // Run every hour
	go func() {
		for {
			select {
			case <-ticker.C:
				processRecurringTasks()
			}
		}
	}()
	log.Println("[Worker] Recurring Tasks Processor started")
}

func processRecurringTasks() {
	log.Println("[Worker] Processing recurring tasks...")
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
		log.Printf("[Worker] Failed to fetch recurring tasks: %v", err)
		return
	}
	defer cursor.Close(ctx)

	now := time.Now()
	for cursor.Next(ctx) {
		var task models.Task
		if err := cursor.Decode(&task); err != nil {
			continue
		}

		if task.Recurrence == nil {
			continue
		}

		// Check if it's time to create a new instance
		if shouldCreateRecurringTask(task, now) {
			createNextRecurringTask(ctx, task)
		}
	}
}

func shouldCreateRecurringTask(task models.Task, now time.Time) bool {
	if task.Recurrence.LastCreated == nil {
		// First time processing - create if task was completed recently
		return true
	}

	lastCreated := *task.Recurrence.LastCreated
	recurrence := task.Recurrence

	switch recurrence.Type {
	case "daily":
		return now.Sub(lastCreated).Hours() >= float64(24*recurrence.Interval)
	case "weekly":
		daysSince := int(now.Sub(lastCreated).Hours() / 24)
		return daysSince >= 7*recurrence.Interval
	case "monthly":
		monthsSince := int(now.Sub(lastCreated).Hours() / (24 * 30))
		return monthsSince >= recurrence.Interval
	case "yearly":
		yearsSince := int(now.Sub(lastCreated).Hours() / (24 * 365))
		return yearsSince >= recurrence.Interval
	default:
		return false
	}
}

func createNextRecurringTask(ctx context.Context, originalTask models.Task) {
	// Create a new task instance based on the recurring task
	newTask := models.Task{
		ID:          primitive.NewObjectID(),
		Title:       originalTask.Title,
		Description: originalTask.Description,
		Status:      "todo",
		Priority:    originalTask.Priority,
		ProjectID:   originalTask.ProjectID,
		IsRecurring: true,
		Recurrence:  originalTask.Recurrence,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Update the recurrence's last created time
	now := time.Now()
	newTask.Recurrence.LastCreated = &now

	// Insert the new task
	_, err := db.Tasks.InsertOne(ctx, newTask)
	if err != nil {
		log.Printf("[Worker] Failed to create recurring task: %v", err)
		return
	}

	// Update the original task's recurrence last created time
	update := bson.M{
		"$set": bson.M{
			"recurrence.last_created": now,
		},
	}
	_, err = db.Tasks.UpdateOne(ctx, bson.M{"_id": originalTask.ID}, update)
	if err != nil {
		log.Printf("[Worker] Failed to update recurring task: %v", err)
	}

	log.Printf("[Worker] Created recurring task: %s", newTask.Title)
}
