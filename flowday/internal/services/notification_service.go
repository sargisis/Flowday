package services

import (
	"context"
	"fmt"
	"time"

	"flowday/internal/db"
	"flowday/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CreateNotification inserts a new notification for a user
func CreateNotification(userID primitive.ObjectID, title, message string, notifType models.NotificationType, link string) (*models.Notification, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	notification := models.Notification{
		ID:        primitive.NewObjectID(),
		UserID:    userID,
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

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
func CheckDueDates(userID primitive.ObjectID) error {
	// TODO: implementing this efficiently requires finding tasks across all projects
	// For now, we'll verify tasks for the logged in user if passed, or could be a system-wide job
	// Implementation deferred to keep this step focused on basic CRUD first.
	return nil
}
