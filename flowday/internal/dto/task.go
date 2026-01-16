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

// ✅ NEW DTOs

// Task Template DTOs
type CreateTaskTemplateRequest struct {
	Name            string    `json:"name" binding:"required"`
	Description     string    `json:"description"`
	Title           string    `json:"title" binding:"required"`
	TaskDescription string    `json:"task_description"`
	Priority        string    `json:"priority"`
	EstimatedHours  *float64  `json:"estimated_hours"`
	Subtasks        []SubtaskDTO `json:"subtasks"`
	ProjectID       string    `json:"project_id"`
}

type UpdateTaskTemplateRequest struct {
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Title           string    `json:"title"`
	TaskDescription string    `json:"task_description"`
	Priority        string    `json:"priority"`
	EstimatedHours  *float64  `json:"estimated_hours"`
	Subtasks        []SubtaskDTO `json:"subtasks"`
	ProjectID       string    `json:"project_id"`
}

type SubtaskDTO struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

type CreateTaskFromTemplateRequest struct {
	ProjectID string `json:"project_id" binding:"required"`
}

// Saved View DTOs
type CreateSavedViewRequest struct {
	Name        string                  `json:"name" binding:"required"`
	Description string                  `json:"description"`
	ProjectID   string                  `json:"project_id"`
	Filters     ViewFiltersDTO          `json:"filters"`
	SortBy      string                  `json:"sort_by"`
	SortOrder   string                  `json:"sort_order"`
}

type UpdateSavedViewRequest struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Filters     ViewFiltersDTO `json:"filters"`
	SortBy      string         `json:"sort_by"`
	SortOrder   string         `json:"sort_order"`
}

type ViewFiltersDTO struct {
	Status      []string `json:"status"`
	Priority    []string `json:"priority"`
	AssigneeID  string   `json:"assignee_id"`
	ProjectID   string   `json:"project_id"`
	DueDateFrom *time.Time `json:"due_date_from"`
	DueDateTo   *time.Time `json:"due_date_to"`
	SearchQuery string    `json:"search_query"`
	HasSubtasks *bool     `json:"has_subtasks"`
	IsRecurring *bool     `json:"is_recurring"`
}

// Time Tracking DTOs
type StartTimeEntryRequest struct {
	TaskID string `json:"task_id" binding:"required"`
}

type StopTimeEntryRequest struct {
	EntryID string `json:"entry_id" binding:"required"`
}

type UpdateTimeEntryRequest struct {
	Notes string `json:"notes"`
}

// Task Dependencies DTOs
type AddTaskDependencyRequest struct {
	DependsOnTaskID string `json:"depends_on_task_id" binding:"required"`
}

type RemoveTaskDependencyRequest struct {
	DependsOnTaskID string `json:"depends_on_task_id" binding:"required"`
}

// Recurring Task DTOs
type SetRecurrenceRequest struct {
	Type       string     `json:"type" binding:"required"` // "daily", "weekly", "monthly", "yearly"
	Interval   int        `json:"interval" binding:"required"`
	EndDate    *time.Time `json:"end_date"`
	Count      *int       `json:"count"`
	DaysOfWeek []int      `json:"days_of_week"` // For weekly: [1,3,5] = Mon, Wed, Fri
	DayOfMonth *int       `json:"day_of_month"` // For monthly
}
