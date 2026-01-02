package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FocusSession struct {
	ID        primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID  `bson:"user_id" json:"user_id"`
	TaskID    *primitive.ObjectID `bson:"task_id,omitempty" json:"task_id,omitempty"`
	TaskTitle string              `bson:"task_title" json:"task_title"`
	Duration  int                 `bson:"duration" json:"duration"` // in minutes
	XP        int                 `bson:"xp" json:"xp"`
	CreatedAt time.Time           `bson:"created_at" json:"created_at"`
}
