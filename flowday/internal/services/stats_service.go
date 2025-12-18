package services

import (
	"context"
	"time"

	"flowday/internal/db"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TaskStats struct {
	Total   int64 `json:"total"`
	Done    int64 `json:"done"`
	Overdue int64 `json:"overdue"`
	Today   int64 `json:"today"`
}

func GetTaskStats(userID primitive.ObjectID) (*TaskStats, error) {
	ctx := context.Background()
	now := time.Now()
	startToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endToday := startToday.Add(24 * time.Hour)

	// Get user's projects
	projects, err := GetProjects(userID)
	if err != nil {
		return nil, err
	}

	projectIDs := make([]primitive.ObjectID, len(projects))
	for i, p := range projects {
		projectIDs[i] = p.ID
	}

	stats := TaskStats{}

	// Total tasks
	total, err := db.Tasks.CountDocuments(ctx, bson.M{
		"project_id": bson.M{"$in": projectIDs},
	})
	if err != nil {
		return nil, err
	}
	stats.Total = total

	// Done tasks
	done, err := db.Tasks.CountDocuments(ctx, bson.M{
		"project_id": bson.M{"$in": projectIDs},
		"status":     "Done",
	})
	if err != nil {
		return nil, err
	}
	stats.Done = done

	// Overdue tasks
	overdue, err := db.Tasks.CountDocuments(ctx, bson.M{
		"project_id": bson.M{"$in": projectIDs},
		"due_date":   bson.M{"$lt": now, "$ne": nil},
		"status":     bson.M{"$ne": "Done"},
	})
	if err != nil {
		return nil, err
	}
	stats.Overdue = overdue

	// Today's tasks
	today, err := db.Tasks.CountDocuments(ctx, bson.M{
		"project_id": bson.M{"$in": projectIDs},
		"due_date":   bson.M{"$gte": startToday, "$lt": endToday},
	})
	if err != nil {
		return nil, err
	}
	stats.Today = today

	return &stats, nil
}
