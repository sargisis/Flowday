package services

import (
	"context"
	"encoding/json"
	"time"

	"flowday/internal/cache"
	"flowday/internal/db"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type TaskStats struct {
	Total   int64 `json:"total"`
	Done    int64 `json:"done"`
	Overdue int64 `json:"overdue"`
	Today   int64 `json:"today"`
}

// GetTaskStats optimized version using aggregation pipeline (1 query instead of 4) + caching
func GetTaskStats(userID primitive.ObjectID) (*TaskStats, error) {
	// ✅ OPTIMIZATION: Cache stats for 2 minutes
	cacheKey := "task_stats:" + userID.Hex()
	if cached, ok := cache.Get(cacheKey); ok {
		if stats, ok := cached.(*TaskStats); ok {
			return stats, nil
		}
		// If cached value is JSON string, unmarshal it
		if jsonStr, ok := cached.(string); ok {
			var stats TaskStats
			if err := json.Unmarshal([]byte(jsonStr), &stats); err == nil {
				return &stats, nil
			}
		}
	}

	ctx := context.Background()
	now := time.Now()
	startToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endToday := startToday.Add(24 * time.Hour)

	// Get user's projects
	projects, err := GetProjects(userID)
	if err != nil {
		return nil, err
	}

	if len(projects) == 0 {
		stats := &TaskStats{}
		cache.Set(cacheKey, stats, 2*time.Minute)
		return stats, nil
	}

	projectIDs := make([]primitive.ObjectID, len(projects))
	for i, p := range projects {
		projectIDs[i] = p.ID
	}

	// ✅ OPTIMIZATION: Use aggregation pipeline to get all stats in one query
	pipeline := mongo.Pipeline{
		// Stage 1: Match tasks in user's projects
		{{"$match", bson.D{
			{"project_id", bson.D{{"$in", projectIDs}}},
		}}},
		// Stage 2: Group and calculate all stats at once
		{{"$group", bson.D{
			{"_id", nil},
			{"total", bson.D{{"$sum", 1}}},
			{"done", bson.D{
				{"$sum", bson.D{
					{"$cond", bson.A{
						bson.D{{"$eq", bson.A{"$status", "Done"}}},
						1,
						0,
					}},
				}},
			}},
			{"overdue", bson.D{
				{"$sum", bson.D{
					{"$cond", bson.A{
						bson.D{
							{"$and", bson.A{
								bson.D{{"$ne", bson.A{"$due_date", nil}}},
								bson.D{{"$lt", bson.A{"$due_date", now}}},
								bson.D{{"$ne", bson.A{"$status", "Done"}}},
							}},
						},
						1,
						0,
					}},
				}},
			}},
			{"today", bson.D{
				{"$sum", bson.D{
					{"$cond", bson.A{
						bson.D{
							{"$and", bson.A{
								bson.D{{"$ne", bson.A{"$due_date", nil}}},
								bson.D{{"$gte", bson.A{"$due_date", startToday}}},
								bson.D{{"$lt", bson.A{"$due_date", endToday}}},
							}},
						},
						1,
						0,
					}},
				}},
			}},
		}}},
	}

	cursor, err := db.Tasks.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	stats := TaskStats{}
	if cursor.Next(ctx) {
		var result bson.M
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		
		if total, ok := result["total"].(int32); ok {
			stats.Total = int64(total)
		}
		if done, ok := result["done"].(int32); ok {
			stats.Done = int64(done)
		}
		if overdue, ok := result["overdue"].(int32); ok {
			stats.Overdue = int64(overdue)
		}
		if today, ok := result["today"].(int32); ok {
			stats.Today = int64(today)
		}
	}

	// ✅ OPTIMIZATION: Cache stats for 2 minutes
	cache.Set(cacheKey, &stats, 2*time.Minute)

	return &stats, nil
}
