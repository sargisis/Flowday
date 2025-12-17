package services

import (
	"time"

	"flowday/internal/db"
)

type TaskStats struct {
	Total   int64 `json:"total"`
	Done    int64 `json:"done"`
	Overdue int64 `json:"overdue"`
	Today   int64 `json:"today"`
}

func GetTaskStats(userID uint) (*TaskStats, error) {
	now := time.Now()
	startToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endToday := startToday.Add(24 * time.Hour)

	stats := TaskStats{}

	db.DB.
		Table("tasks").
		Joins("JOIN projects ON projects.id = tasks.project_id").
		Where("projects.user_id = ?", userID).
		Count(&stats.Total)

	db.DB.
		Table("tasks").
		Joins("JOIN projects ON projects.id = tasks.project_id").
		Where("projects.user_id = ? AND tasks.status = ?", userID, "done").
		Count(&stats.Done)

	db.DB.
		Table("tasks").
		Joins("JOIN projects ON projects.id = tasks.project_id").
		Where(
			"projects.user_id = ? AND tasks.due_date IS NOT NULL AND tasks.due_date < ? AND tasks.status != ?",
			userID, now, "done",
		).
		Count(&stats.Overdue)

	db.DB.
		Table("tasks").
		Joins("JOIN projects ON projects.id = tasks.project_id").
		Where(
			"projects.user_id = ? AND tasks.due_date >= ? AND tasks.due_date < ?",
			userID, startToday, endToday,
		).
		Count(&stats.Today)

	return &stats, nil
}
