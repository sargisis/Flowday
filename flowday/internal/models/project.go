package models

import "time"

type Project struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `json:"name"`
	UserID    uint      `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}
