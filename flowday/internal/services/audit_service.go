package services

import (
	"context"
	"time"

	"flowday/internal/db"
	"flowday/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// LogAuditEvent manually logs an audit event (for cases where middleware doesn't catch it)
func LogAuditEvent(userID primitive.ObjectID, action, resource string, resourceID *primitive.ObjectID, details map[string]interface{}, success bool, errorMsg string) {
	auditLog := models.AuditLog{
		ID:         primitive.NewObjectID(),
		UserID:     userID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Timestamp:  time.Now(),
		Success:    success,
		Error:      errorMsg,
		Details:    details,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.AuditLogs.InsertOne(ctx, auditLog)
	if err != nil {
		// Silently fail - audit logging should not break the application
		_ = err
	}
}

// GetAuditLogs retrieves audit logs for a user
func GetAuditLogs(userID primitive.ObjectID, limit int) ([]models.AuditLog, error) {
	ctx := context.Background()

	findOptions := options.Find().SetLimit(int64(limit)).SetSort(bson.M{"timestamp": -1})
	cursor, err := db.AuditLogs.Find(ctx, bson.M{"user_id": userID}, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var logs []models.AuditLog
	if err = cursor.All(ctx, &logs); err != nil {
		return nil, err
	}

	return logs, nil
}
