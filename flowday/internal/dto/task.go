package dto

import "time"

type CreateTaskRequest struct {
	Title     string     `json:"title" binding:"required"`
	Priority  string     `json:"priority"`
	DueDate   *time.Time `json:"due_date"`
	ProjectID uint       `json:"project_id" binding:"required"`
}

type UpdateTaskRequest struct {
	Status   *string    `json:"status"`
	Priority *string    `json:"priority"`
	DueDate  *time.Time `json:"due_date"`
}
