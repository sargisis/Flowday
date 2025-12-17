package services

import (
	"time"

	"flowday/internal/db"
	"flowday/internal/models"
)

func GetTaskByRange(userId uint, from, to time.Time) ([]models.Task, error) {
	start := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
	end := time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 0, to.Location())

	var tasks []models.Task
	err := db.DB.
		Joins("JOIN projects ON projects.id = tasks.project_id").
		Where(
			"projects.user_id = ? AND tasks.due_date IS NOT NULL AND tasks.due_date >= ? AND tasks.due_date <= ?",
			userId, start, end,
		).
		Preload("Project").
		Find(&tasks).Error

	return tasks, err
}
