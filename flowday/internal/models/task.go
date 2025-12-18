package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Task struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title       string             `bson:"title" json:"title"`
	Description string             `bson:"description,omitempty" json:"description,omitempty"`
	Status      string             `bson:"status" json:"status"`
	Priority    string             `bson:"priority" json:"priority"`
	DueDate     *time.Time         `bson:"due_date,omitempty" json:"due_date,omitempty"`
	ProjectID   primitive.ObjectID `bson:"project_id" json:"project_id"`
	Project     *Project           `bson:"-" json:"project,omitempty"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
}
