package dto

import (
	"encoding/json"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// NullableTime handles null, empty string, and valid time values
type NullableTime struct {
	Time  *time.Time
	Valid bool
}

// UnmarshalJSON custom unmarshaler that handles empty strings and null
func (nt *NullableTime) UnmarshalJSON(data []byte) error {
	// Handle null
	if string(data) == "null" {
		nt.Valid = false
		nt.Time = nil
		return nil
	}

	// Handle empty string
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		if str == "" {
			nt.Valid = false
			nt.Time = nil
			return nil
		}
		// Parse non-empty string
		t, err := time.Parse(time.RFC3339, str)
		if err != nil {
			return err
		}
		nt.Time = &t
		nt.Valid = true
		return nil
	}

	// Try to parse as time object directly
	var t time.Time
	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}
	nt.Time = &t
	nt.Valid = true
	return nil
}

type CreateTaskRequest struct {
	Title       string       `json:"title" binding:"required"`
	Description string       `json:"description"`
	Priority    string       `json:"priority"`
	DueDate     NullableTime `json:"due_date"`
	ProjectID   string       `json:"project_id" binding:"required"`
}

// GetDueDateTimePtr returns the time pointer or nil
func (r *CreateTaskRequest) GetDueDateTimePtr() *time.Time {
	if r.DueDate.Valid {
		return r.DueDate.Time
	}
	return nil
}

type UpdateTaskRequest struct {
	Status      *string      `json:"status"`
	Priority    *string      `json:"priority"`
	Description *string      `json:"description"`
	DueDate     *NullableTime `json:"due_date,omitempty"`
}

// GetDueDateTimePtr returns the time pointer or nil
func (r *UpdateTaskRequest) GetDueDateTimePtr() *time.Time {
	if r.DueDate != nil && r.DueDate.Valid {
		return r.DueDate.Time
	}
	return nil
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

// Bulk Update DTOs
type BulkUpdateStatusRequest struct {
	TaskIDs []string `json:"task_ids" binding:"required"`
	Status  string   `json:"status" binding:"required"`
}

type BulkUpdatePriorityRequest struct {
	TaskIDs []string `json:"task_ids" binding:"required"`
	Priority string   `json:"priority" binding:"required"`
}
