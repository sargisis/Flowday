package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AuditLog represents an audit log entry for tracking important actions
type AuditLog struct {
	ID         primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	UserID     primitive.ObjectID   `bson:"user_id" json:"user_id"`
	Action     string               `bson:"action" json:"action"`         // "task_created", "task_deleted", "bulk_update", etc.
	Resource   string               `bson:"resource" json:"resource"`   // "task", "project", "user", etc.
	ResourceID *primitive.ObjectID  `bson:"resource_id,omitempty" json:"resource_id,omitempty"`
	Details    map[string]interface{} `bson:"details,omitempty" json:"details,omitempty"` // Additional context
	IP         string               `bson:"ip" json:"ip"`
	UserAgent  string               `bson:"user_agent" json:"user_agent"`
	Timestamp  time.Time            `bson:"timestamp" json:"timestamp"`
	Success    bool                 `bson:"success" json:"success"` // Whether the action succeeded
	Error      string               `bson:"error,omitempty" json:"error,omitempty"` // Error message if failed
}

// AuditAction constants
const (
	// Task actions
	AuditActionTaskCreated      = "task_created"
	AuditActionTaskUpdated      = "task_updated"
	AuditActionTaskDeleted      = "task_deleted"
	AuditActionTaskBulkUpdate   = "task_bulk_update"
	AuditActionTaskBulkDelete   = "task_bulk_delete"
	
	// Project actions
	AuditActionProjectCreated   = "project_created"
	AuditActionProjectUpdated   = "project_updated"
	AuditActionProjectDeleted   = "project_deleted"
	
	// User actions
	AuditActionUserLogin        = "user_login"
	AuditActionUserLogout       = "user_logout"
	AuditActionUserRegistered   = "user_registered"
	AuditActionPasswordChanged  = "password_changed"
	
	// Member actions
	AuditActionMemberAdded      = "member_added"
	AuditActionMemberRemoved    = "member_removed"
	AuditActionMemberUpdated    = "member_updated"
)

// Resource types
const (
	ResourceTypeTask    = "task"
	ResourceTypeProject = "project"
	ResourceTypeUser    = "user"
	ResourceTypeMember  = "member"
)
