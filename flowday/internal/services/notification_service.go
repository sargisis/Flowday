package services

import (
	"context"
	"fmt"
	"time"

	"flowday/internal/db"
	"flowday/internal/dto"
	"flowday/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CreateNotification inserts a new notification for a user
func CreateNotification(userID primitive.ObjectID, taskID *primitive.ObjectID, title, message string, notifType models.NotificationType, link string) (*models.Notification, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	notification := models.Notification{
		ID:        primitive.NewObjectID(),
		UserID:    userID,
		TaskID:    taskID,
		Title:     title,
		Message:   message,
		Read:      false,
		Type:      notifType,
		Link:      link,
		CreatedAt: time.Now(),
	}

	_, err := db.Notifications.InsertOne(ctx, notification)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}

	return &notification, nil
}

// GetNotifications fetches all notifications for a user, sorted by newest first
func GetNotifications(userID primitive.ObjectID) ([]models.Notification, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Lazy check: Verify due dates before fetching
	_ = CheckDueDates(userID)

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := db.Notifications.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch notifications: %w", err)
	}
	defer cursor.Close(ctx)

	var notifications []models.Notification
	if err = cursor.All(ctx, &notifications); err != nil {
		return nil, fmt.Errorf("failed to decode notifications: %w", err)
	}

	return notifications, nil
}

// GetNotificationsPaginated returns paginated notifications for a user
func GetNotificationsPaginated(userID primitive.ObjectID, pagination dto.PaginationQuery) ([]models.Notification, dto.PaginationMeta, error) {
	// Validate and set defaults
	pagination.ValidateAndSetDefaults()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Lazy check: Verify due dates before fetching
	_ = CheckDueDates(userID)

	filter := bson.M{"user_id": userID}

	// Get total count
	total, err := db.Notifications.CountDocuments(ctx, filter)
	if err != nil {
		return nil, dto.PaginationMeta{}, fmt.Errorf("failed to count notifications: %w", err)
	}

	// Build sort order
	sortOrder := -1 // desc (newest first by default)
	if pagination.Order == "asc" {
		sortOrder = 1
	}
	sortField := pagination.Sort
	if sortField == "" {
		sortField = "created_at"
	}

	// Build find options
	findOptions := options.Find().
		SetLimit(int64(pagination.Limit)).
		SetSkip(int64(pagination.GetOffset())).
		SetSort(bson.D{{Key: sortField, Value: sortOrder}})

	cursor, err := db.Notifications.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, dto.PaginationMeta{}, fmt.Errorf("failed to fetch notifications: %w", err)
	}
	defer cursor.Close(ctx)

	var notifications []models.Notification
	if err = cursor.All(ctx, &notifications); err != nil {
		return nil, dto.PaginationMeta{}, fmt.Errorf("failed to decode notifications: %w", err)
	}

	meta := dto.NewPaginationMeta(pagination, total)

	return notifications, meta, nil
}

// MarkNotificationRead marks a specific notification as read
func MarkNotificationRead(id primitive.ObjectID, userID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := db.Notifications.UpdateOne(
		ctx,
		bson.M{"_id": id, "user_id": userID},
		bson.M{"$set": bson.M{"read": true}},
	)

	if err != nil {
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("notification not found or unauthorized")
	}

	return nil
}

// CheckDueDates checks for tasks due within the next 24 hours and creates notifications
// This should be called periodically (e.g., via a cron job or a background ticker)
// CheckDueDates checks for tasks due within the next 24 hours and creates notifications
func CheckDueDates(userID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Find tasks that are NOT done and have a due date
	filter := bson.M{
		"user_id": userID, // TODO: For now only checking personal tasks or tasks assigned to user. Project permissions might need simpler check.
		"status":  bson.M{"$ne": "done"},
		"due_date": bson.M{
			"$exists": true,
			"$ne":     nil,
		},
	}

	cursor, err := db.Tasks.Find(ctx, filter)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	var tasks []models.Task
	if err = cursor.All(ctx, &tasks); err != nil {
		return err
	}

	now := time.Now()

	for _, task := range tasks {
		if task.DueDate == nil {
			continue
		}

		hoursDiff := task.DueDate.Sub(now).Hours()
		var notifType models.NotificationType
		var title, message string

		// Logic for Overdue vs Due Soon
		if hoursDiff < 0 {
			// Overdue
			notifType = models.NotificationError // Warning/Error
			title = "Task Overdue"
			message = fmt.Sprintf("Task '%s' is overdue.", task.Title)
		} else if hoursDiff < 24 {
			// Due Soon
			notifType = models.NotificationWarning
			title = "Task Due Soon"
			message = fmt.Sprintf("Task '%s' is due in less than 24 hours.", task.Title)
		} else {
			continue
		}

		// 2. Check if notification already exists for this task + type to avoid spam
		// We avoid creating duplicate alerts for the same condition
		// (e.g. if we alerted "Due Soon", we might still want to alert "Overdue" later, so we check existence slightly vaguely or specifically)
		// For simplicity: Check if an unread notification exists for this task.
		// A better approach: Check if we routed a notification for this specific state.
		// Let's check overlap:
		notifFilter := bson.M{
			"user_id": userID,
			"task_id": task.ID,
			"title":   title, // Simple dedup by title
			"read":    false, // If user read it, we might remind them? Or just let it be. Let's not spam if they haven't read the old one.
		}

		count, _ := db.Notifications.CountDocuments(ctx, notifFilter)
		if count > 0 {
			continue // Already notified
		}

		// Create
		link := "" // Could link to task
		_, _ = CreateNotification(userID, &task.ID, title, message, notifType, link)
	}

	return nil
}
