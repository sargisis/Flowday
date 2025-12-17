package services

import (
	"flowday/internal/db"
	"flowday/internal/models"
)

func CreateProject(userID uint, name string) (*models.Project, error) {
	project := models.Project{
		Name:   name,
		UserID: userID,
	}

	if err := db.DB.Create(&project).Error; err != nil {
		return nil, err
	}

	return &project, nil
}

func GetProjects(userID uint) ([]models.Project, error) {
	var projects []models.Project
	err := db.DB.Where("user_id = ?", userID).Find(&projects).Error
	return projects, err
}

func DeleteProject(userID, projectID uint) error {
	return db.DB.
		Where("id = ? AND user_id = ?", projectID, userID).
		Delete(&models.Project{}).Error
}
