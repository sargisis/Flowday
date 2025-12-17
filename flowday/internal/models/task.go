package models

import "time"

type Task struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Title     string     `json:"title"`
	Status    string     `gorm:"index" json:"status"`
	Priority  string     `gorm:"index" json:"priority"`
	DueDate   *time.Time `gorm:"index" json:"due_date"`
	ProjectID uint       `gorm:"index" json:"project_id"`
	Project   *Project   `json:"project,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}
