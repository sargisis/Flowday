package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type NotificationType string

const (
	NotificationInfo    NotificationType = "info"
	NotificationSuccess NotificationType = "success"
	NotificationWarning NotificationType = "warning"
	NotificationError   NotificationType = "error"
)

type Notification struct {
	ID        primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID  `bson:"user_id" json:"user_id"`
	TaskID    *primitive.ObjectID `bson:"task_id,omitempty" json:"task_id,omitempty"` // Optional link to specific task
	Title     string              `bson:"title" json:"title"`
	Message   string              `bson:"message" json:"message"`
	Read      bool                `bson:"read" json:"read"`
	Type      NotificationType    `bson:"type" json:"type"`
	Link      string              `bson:"link,omitempty" json:"link,omitempty"` // Optional link to task or project
	CreatedAt time.Time           `bson:"created_at" json:"created_at"`
}
