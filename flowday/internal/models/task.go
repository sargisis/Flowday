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
	
	// ✅ NEW FEATURES
	// Task Dependencies
	DependsOn    []primitive.ObjectID `bson:"depends_on,omitempty" json:"depends_on,omitempty"`     // Tasks this task depends on
	BlockedBy    []primitive.ObjectID `bson:"blocked_by,omitempty" json:"blocked_by,omitempty"`     // Tasks that block this task
	Blocks       []primitive.ObjectID `bson:"blocks,omitempty" json:"blocks,omitempty"`             // Tasks blocked by this task
	
	// Recurring Tasks
	IsRecurring  bool          `bson:"is_recurring,omitempty" json:"is_recurring,omitempty"`
	Recurrence   *Recurrence   `bson:"recurrence,omitempty" json:"recurrence,omitempty"`
	TemplateID   *primitive.ObjectID `bson:"template_id,omitempty" json:"template_id,omitempty"` // If created from template
	
	// Time Tracking
	EstimatedHours *float64           `bson:"estimated_hours,omitempty" json:"estimated_hours,omitempty"`
	TimeEntries    []TimeEntry        `bson:"time_entries,omitempty" json:"time_entries,omitempty"`
	TotalTimeSpent float64            `bson:"total_time_spent,omitempty" json:"total_time_spent,omitempty"` // In hours
	
	// Assignee
	AssigneeID  *primitive.ObjectID `bson:"assignee_id,omitempty" json:"assignee_id,omitempty"`
	
	// Mentions
	MentionedUserIDs []primitive.ObjectID `bson:"mentioned_user_ids,omitempty" json:"mentioned_user_ids,omitempty"`
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

// ✅ NEW MODELS

// Recurrence defines how a task repeats
type Recurrence struct {
	Type      string    `bson:"type" json:"type"` // "daily", "weekly", "monthly", "yearly"
	Interval  int       `bson:"interval" json:"interval"` // Every N days/weeks/months
	EndDate   *time.Time `bson:"end_date,omitempty" json:"end_date,omitempty"` // Optional end date
	Count     *int      `bson:"count,omitempty" json:"count,omitempty"` // Optional max occurrences
	DaysOfWeek []int   `bson:"days_of_week,omitempty" json:"days_of_week,omitempty"` // For weekly: [1,3,5] = Mon, Wed, Fri
	DayOfMonth *int    `bson:"day_of_month,omitempty" json:"day_of_month,omitempty"` // For monthly: day 15
	LastCreated *time.Time `bson:"last_created,omitempty" json:"last_created,omitempty"` // Last time task was auto-created
}

// TimeEntry tracks time spent on a task
type TimeEntry struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID `bson:"user_id" json:"user_id"`
	TaskID    primitive.ObjectID `bson:"task_id" json:"task_id"`
	StartTime time.Time          `bson:"start_time" json:"start_time"`
	EndTime   *time.Time         `bson:"end_time,omitempty" json:"end_time,omitempty"`
	Duration  float64            `bson:"duration" json:"duration"` // In hours
	Notes     string             `bson:"notes,omitempty" json:"notes,omitempty"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

// TaskTemplate for reusable task templates
type TaskTemplate struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name" json:"name"`
	Description string             `bson:"description,omitempty" json:"description,omitempty"`
	Title       string             `bson:"title" json:"title"`
	TaskDescription string         `bson:"task_description,omitempty" json:"task_description,omitempty"`
	Priority    string             `bson:"priority" json:"priority"`
	EstimatedHours *float64        `bson:"estimated_hours,omitempty" json:"estimated_hours,omitempty"`
	Subtasks    []Subtask          `bson:"subtasks,omitempty" json:"subtasks,omitempty"`
	ProjectID   *primitive.ObjectID `bson:"project_id,omitempty" json:"project_id,omitempty"` // Optional: template for specific project
	UserID      primitive.ObjectID `bson:"user_id" json:"user_id"` // Template owner
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
	UsageCount  int                `bson:"usage_count" json:"usage_count"` // How many times used
}

// SavedView for custom task filters and views
type SavedView struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name" json:"name"`
	Description string             `bson:"description,omitempty" json:"description,omitempty"`
	UserID      primitive.ObjectID `bson:"user_id" json:"user_id"`
	ProjectID   *primitive.ObjectID `bson:"project_id,omitempty" json:"project_id,omitempty"`
	Filters     ViewFilters        `bson:"filters" json:"filters"`
	SortBy      string             `bson:"sort_by,omitempty" json:"sort_by,omitempty"` // "created_at", "due_date", "priority", "title"
	SortOrder   string             `bson:"sort_order,omitempty" json:"sort_order,omitempty"` // "asc", "desc"
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

// ViewFilters defines filter criteria for saved views
type ViewFilters struct {
	Status      []string             `bson:"status,omitempty" json:"status,omitempty"`
	Priority    []string             `bson:"priority,omitempty" json:"priority,omitempty"`
	AssigneeID  *primitive.ObjectID `bson:"assignee_id,omitempty" json:"assignee_id,omitempty"`
	ProjectID   *primitive.ObjectID `bson:"project_id,omitempty" json:"project_id,omitempty"`
	DueDateFrom *time.Time          `bson:"due_date_from,omitempty" json:"due_date_from,omitempty"`
	DueDateTo   *time.Time          `bson:"due_date_to,omitempty" json:"due_date_to,omitempty"`
	SearchQuery string              `bson:"search_query,omitempty" json:"search_query,omitempty"`
	HasSubtasks *bool               `bson:"has_subtasks,omitempty" json:"has_subtasks,omitempty"`
	IsRecurring *bool               `bson:"is_recurring,omitempty" json:"is_recurring,omitempty"`
}
