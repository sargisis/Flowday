package dto

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CreateTaskRequest struct {
	Title       string     `json:"title" binding:"required"`
	Description string     `json:"description"`
	Priority    string     `json:"priority"`
	DueDate     *time.Time `json:"due_date"`
	ProjectID   string     `json:"project_id" binding:"required"`
}

type UpdateTaskRequest struct {
	Status      *string    `json:"status"`
	Priority    *string    `json:"priority"`
	Description *string    `json:"description"`
	DueDate     *time.Time `json:"due_date"`
}

// Helper to convert ProjectID string to ObjectID
func (r *CreateTaskRequest) GetProjectObjectID() (primitive.ObjectID, error) {
	return primitive.ObjectIDFromHex(r.ProjectID)
}
