package services

import (
	"context"
	"time"

	"flowday/internal/db"
	"flowday/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetTaskByRange(userId primitive.ObjectID, from, to time.Time) ([]models.Task, error) {
	ctx := context.Background()
	start := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
	end := time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 0, to.Location())

	// Get user's projects
	projects, err := GetProjects(userId)
	if err != nil {
		return nil, err
	}

	projectIDs := make([]primitive.ObjectID, len(projects))
	for i, p := range projects {
		projectIDs[i] = p.ID
	}

	// Find tasks in date range
	cursor, err := db.Tasks.Find(ctx, bson.M{
		"project_id": bson.M{"$in": projectIDs},
		"due_date":   bson.M{"$gte": start, "$lte": end, "$ne": nil},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tasks []models.Task
	if err = cursor.All(ctx, &tasks); err != nil {
		return nil, err
	}

	// Populate Project field for each task
	for i := range tasks {
		for _, p := range projects {
			if p.ID == tasks[i].ProjectID {
				tasks[i].Project = &p
				break
			}
		}
	}

	return tasks, nil
}
