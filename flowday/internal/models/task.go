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
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
	Subtasks    []Subtask          `bson:"subtasks,omitempty" json:"subtasks,omitempty"`
	Attachments []Attachment       `bson:"attachments,omitempty" json:"attachments,omitempty"`
}

type Attachment struct {
	ID         string    `bson:"id" json:"id"`
	URL        string    `bson:"url" json:"url"`
	Type       string    `bson:"type" json:"type"` // "image" or "file"
	Filename   string    `bson:"filename" json:"filename"`
	Size       int64     `bson:"size" json:"size"` // File size in bytes
	UploadedAt time.Time `bson:"uploaded_at" json:"uploaded_at"`
}

type Subtask struct {
	ID        string `bson:"id" json:"id"`
	Title     string `bson:"title" json:"title"`
	Completed bool   `bson:"completed" json:"completed"`
}
