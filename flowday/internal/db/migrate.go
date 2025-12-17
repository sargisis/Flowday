package db

import "flowday/internal/models"

func Migrate() {
	DB.AutoMigrate(&models.User{}, &models.Project{}, &models.Task{})
}