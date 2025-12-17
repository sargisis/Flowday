package services

import (
	"errors"

	"flowday/internal/db"
	"flowday/internal/models"
)

func CreateTask(userID uint, task *models.Task) error {
	var project models.Project
	if err := db.DB.
		Where("id = ? AND user_id = ?", task.ProjectID, userID).
		First(&project).Error; err != nil {
		return errors.New("project not found")
	}

	return db.DB.Create(task).Error
}

func GetTasksByProjectPaginated(
	userID uint,
	projectID uint,
	limit int,
	offset int,
	order string,
	dir string,
) ([]models.Task, error) {

	// 1) defaults
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	// 2) whitelist order (ВОТ ТУТ твой allowedOrder)
	allowedOrder := map[string]bool{
		"created_at": true,
		"due_date":   true,
		"priority":   true,
		"status":     true,
	}
	if !allowedOrder[order] {
		order = "created_at"
	}

	// disambiguate common columns
	if order == "created_at" {
		order = "tasks.created_at"
	}

	// 3) dir validation
	if dir != "asc" && dir != "desc" {
		dir = "desc"
	}

	// 4) query
	var tasks []models.Task
	err := db.DB.
		Joins("JOIN projects ON projects.id = tasks.project_id").
		Where("projects.user_id = ? AND tasks.project_id = ?", userID, projectID).
		Preload("Project").
		Order(order + " " + dir).
		Limit(limit).
		Offset(offset).
		Find(&tasks).Error

	return tasks, err
}

func GetTasksByProject(userID, projectID uint) ([]models.Task, error) {
	var tasks []models.Task

	err := db.DB.
		Joins("JOIN projects ON projects.id = tasks.project_id").
		Where("projects.user_id = ? AND tasks.project_id = ?", userID, projectID).
		Preload("Project").
		Find(&tasks).Error

	return tasks, err
}

func UpdateTask(userID, taskID uint, updates map[string]interface{}) error {
	return db.DB.
		Joins("JOIN projects ON projects.id = tasks.project_id").
		Where("tasks.id = ? AND projects.user_id = ?", taskID, userID).
		Model(&models.Task{}).
		Updates(updates).Error
}

func DeleteTask(userID, taskID uint) error {
	return db.DB.
		Joins("JOIN projects ON projects.id = tasks.project_id").
		Where("tasks.id = ? AND projects.user_id = ?", taskID, userID).
		Delete(&models.Task{}).Error
}
