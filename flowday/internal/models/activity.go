package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ActivityType string

const (
	ActivityTaskCreated    ActivityType = "task_created"
	ActivityTaskCompleted  ActivityType = "task_completed"
	ActivityFocusStarted   ActivityType = "focus_started"
	ActivityFocusCompleted ActivityType = "focus_completed"
	ActivityProjectCreated ActivityType = "project_created"
)

type Activity struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID      primitive.ObjectID `bson:"user_id" json:"user_id"`
	Type        ActivityType       `bson:"type" json:"type"`
	Description string             `bson:"description" json:"description"`
	MetaData    map[string]string  `bson:"metadata,omitempty" json:"metadata,omitempty"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
}
