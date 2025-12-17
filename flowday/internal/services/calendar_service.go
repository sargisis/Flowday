package services

import (
	"time"

	"flowday/internal/db"
	"flowday/internal/models"
)

func GetTasksByDate(userID uint, date time.Time) ([]models.Task, error) {
	start := time.Date(
		date.Year(),
		date.Month(),
		date.Day(),
		0, 0, 0, 0,
		date.Location(),
	)
	end := start.Add(24 * time.Hour)

	var tasks []models.Task
	err := db.DB.
		Joins("JOIN projects ON projects.id = tasks.project_id").
		Where(
			"projects.user_id = ? AND tasks.due_date IS NOT NULL AND tasks.due_date >= ? AND tasks.due_date < ?",
			userID, start, end,
		).
		Preload("Project").
		Find(&tasks).Error

	return tasks, err
}
