package services

import (
	"context"
	"time"

	"flowday/internal/db"
	"flowday/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetTasksByDate(userID primitive.ObjectID, date time.Time) ([]models.Task, error) {
	start := time.Date(
		date.Year(),
		date.Month(),
		date.Day(),
		0, 0, 0, 0,
		date.Location(),
	)
	end := start.Add(24 * time.Hour)

	// First, get all user's projects
	projects, err := GetProjects(userID)
	if err != nil {
		return nil, err
	}

	// Extract project IDs
	projectIDs := make([]primitive.ObjectID, len(projects))
	for i, p := range projects {
		projectIDs[i] = p.ID
	}

	// Find tasks in those projects with due date in range
	ctx := context.Background()
	cursor, err := db.Tasks.Find(ctx, bson.M{
		"project_id": bson.M{"$in": projectIDs},
		"due_date":   bson.M{"$gte": start, "$lt": end},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tasks []models.Task
	if err = cursor.All(ctx, &tasks); err != nil {
		return nil, err
	}

	return tasks, nil
}
