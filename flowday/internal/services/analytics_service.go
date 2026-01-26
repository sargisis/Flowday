package services

import (
	"context"
	"errors"
	"time"

	"flowday/internal/cache"
	"flowday/internal/db"
	"flowday/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TaskAnalytics represents comprehensive task analytics
type TaskAnalytics struct {
	// Basic stats
	TotalTasks      int64 `json:"total_tasks"`
	DoneTasks       int64 `json:"done_tasks"`
	InProgressTasks int64 `json:"in_progress_tasks"`
	BlockedTasks    int64 `json:"blocked_tasks"`
	OverdueTasks    int64 `json:"overdue_tasks"`

	// Performance metrics
	AverageCompletionTime float64 `json:"average_completion_time_hours"` // Average time from creation to completion
	CompletionRate        float64 `json:"completion_rate"`               // Percentage of tasks completed
	Velocity              float64 `json:"velocity"`                      // Tasks completed per day

	// Priority distribution
	HighPriorityTasks   int64 `json:"high_priority_tasks"`
	MediumPriorityTasks int64 `json:"medium_priority_tasks"`
	LowPriorityTasks    int64 `json:"low_priority_tasks"`

	// Time-based metrics
	TasksByDayOfWeek map[string]int64 `json:"tasks_by_day_of_week"` // Tasks completed by day of week
	TaskTrends       []TaskTrend      `json:"task_trends"`          // Daily performance trends
	TasksByDate      []DateTaskCount  `json:"tasks_by_date"`        // Legacy field (deprecated)

	// Top tasks
	TopPriorityTasks []TaskSummary `json:"top_priority_tasks"` // Top 10 high priority tasks
}

type TaskTrend struct {
	Date      string `json:"date"`
	Created   int    `json:"created"`
	Completed int    `json:"completed"`
}

type DateTaskCount struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

type TaskSummary struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Priority    string `json:"priority"`
	Status      string `json:"status"`
	DueDate     string `json:"due_date,omitempty"`
	ProjectName string `json:"project_name,omitempty"`
}

// GetTaskAnalytics returns comprehensive analytics for tasks
func GetTaskAnalytics(userID primitive.ObjectID, projectID *primitive.ObjectID, startDate, endDate *time.Time) (*TaskAnalytics, error) {
	ctx := context.Background()

	// Build cache key
	cacheKey := "task_analytics:" + userID.Hex()
	if projectID != nil {
		cacheKey += ":" + projectID.Hex()
	}
	if startDate != nil {
		cacheKey += ":" + startDate.Format("2006-01-02")
	}
	if endDate != nil {
		cacheKey += ":" + endDate.Format("2006-01-02")
	}

	// Try cache first
	if cached, ok := cache.Get(cacheKey); ok {
		if analytics, ok := cached.(*TaskAnalytics); ok {
			return analytics, nil
		}
	}

	// Get user's projects
	projects, err := GetProjects(userID)
	if err != nil {
		return nil, err
	}

	projectIDs := make([]primitive.ObjectID, len(projects))
	for i, p := range projects {
		projectIDs[i] = p.ID
	}

	// Filter by project if specified
	if projectID != nil {
		// Verify access
		hasAccess := false
		for _, pid := range projectIDs {
			if pid == *projectID {
				hasAccess = true
				break
			}
		}
		if !hasAccess {
			return nil, errors.New("access denied")
		}
		projectIDs = []primitive.ObjectID{*projectID}
	}

	if len(projectIDs) == 0 {
		return &TaskAnalytics{}, nil
	}

	// Build match filter
	matchFilter := bson.M{"project_id": bson.M{"$in": projectIDs}}
	if startDate != nil || endDate != nil {
		matchFilter["created_at"] = bson.M{}
		if startDate != nil {
			matchFilter["created_at"].(bson.M)["$gte"] = *startDate
		}
		if endDate != nil {
			matchFilter["created_at"].(bson.M)["$lte"] = *endDate
		}
	}

	now := time.Now()

	// Main aggregation pipeline for stats
	statsPipeline := mongo.Pipeline{
		{{"$match", matchFilter}},
		{{"$group", bson.D{
			{"_id", nil},
			{"total", bson.D{{"$sum", 1}}},
			{"done", bson.D{
				{"$sum", bson.D{
					{"$cond", bson.A{
						bson.D{{"$eq", bson.A{"$status", "Done"}}},
						1, 0,
					}},
				}},
			}},
			{"in_progress", bson.D{
				{"$sum", bson.D{
					{"$cond", bson.A{
						bson.D{{"$eq", bson.A{"$status", "In_Progress"}}},
						1, 0,
					}},
				}},
			}},
			{"blocked", bson.D{
				{"$sum", bson.D{
					{"$cond", bson.A{
						bson.D{{"$eq", bson.A{"$status", "Blocked"}}},
						1, 0,
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
						1, 0,
					}},
				}},
			}},
			{"high_priority", bson.D{
				{"$sum", bson.D{
					{"$cond", bson.A{
						bson.D{{"$eq", bson.A{"$priority", "high"}}},
						1, 0,
					}},
				}},
			}},
			{"medium_priority", bson.D{
				{"$sum", bson.D{
					{"$cond", bson.A{
						bson.D{{"$eq", bson.A{"$priority", "medium"}}},
						1, 0,
					}},
				}},
			}},
			{"low_priority", bson.D{
				{"$sum", bson.D{
					{"$cond", bson.A{
						bson.D{{"$eq", bson.A{"$priority", "low"}}},
						1, 0,
					}},
				}},
			}},
		}}},
	}

	cursor, err := db.Tasks.Aggregate(ctx, statsPipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	analytics := &TaskAnalytics{
		TasksByDayOfWeek: make(map[string]int64),
		TasksByDate:      []DateTaskCount{},
		TopPriorityTasks: []TaskSummary{},
	}

	if cursor.Next(ctx) {
		var result bson.M
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}

		if total, ok := result["total"].(int32); ok {
			analytics.TotalTasks = int64(total)
		}
		if done, ok := result["done"].(int32); ok {
			analytics.DoneTasks = int64(done)
		}
		if inProgress, ok := result["in_progress"].(int32); ok {
			analytics.InProgressTasks = int64(inProgress)
		}
		if blocked, ok := result["blocked"].(int32); ok {
			analytics.BlockedTasks = int64(blocked)
		}
		if overdue, ok := result["overdue"].(int32); ok {
			analytics.OverdueTasks = int64(overdue)
		}
		if high, ok := result["high_priority"].(int32); ok {
			analytics.HighPriorityTasks = int64(high)
		}
		if medium, ok := result["medium_priority"].(int32); ok {
			analytics.MediumPriorityTasks = int64(medium)
		}
		if low, ok := result["low_priority"].(int32); ok {
			analytics.LowPriorityTasks = int64(low)
		}
	}

	// Calculate completion rate
	if analytics.TotalTasks > 0 {
		analytics.CompletionRate = float64(analytics.DoneTasks) / float64(analytics.TotalTasks) * 100
	}

	// Get average completion time (for completed tasks)
	avgTimePipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"project_id": bson.M{"$in": projectIDs},
			"status":     "Done",
			"created_at": bson.M{"$exists": true},
			"updated_at": bson.M{"$exists": true},
		}}},
		{{"$project", bson.D{
			{"duration_hours", bson.D{
				{"$divide", bson.A{
					bson.D{{"$subtract", bson.A{"$updated_at", "$created_at"}}},
					3600000, // milliseconds to hours
				}},
			}},
		}}},
		{{"$group", bson.D{
			{"_id", nil},
			{"avg_hours", bson.D{{"$avg", "$duration_hours"}}},
		}}},
	}

	avgCursor, err := db.Tasks.Aggregate(ctx, avgTimePipeline)
	if err == nil {
		defer avgCursor.Close(ctx)
		if avgCursor.Next(ctx) {
			var result bson.M
			if err := avgCursor.Decode(&result); err == nil {
				if avg, ok := result["avg_hours"].(float64); ok {
					analytics.AverageCompletionTime = avg
				}
			}
		}
	}

	// Calculate velocity (tasks completed per day)
	if startDate != nil && endDate != nil {
		days := endDate.Sub(*startDate).Hours() / 24
		if days > 0 {
			analytics.Velocity = float64(analytics.DoneTasks) / days
		}
	}

	// Get tasks by day of week (for completed tasks)
	dayOfWeekPipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"project_id": bson.M{"$in": projectIDs},
			"status":     "Done",
		}}},
		{{"$group", bson.D{
			{"_id", bson.D{{"$dayOfWeek", "$updated_at"}}},
			{"count", bson.D{{"$sum", 1}}},
		}}},
	}

	dayCursor, err := db.Tasks.Aggregate(ctx, dayOfWeekPipeline)
	if err == nil {
		defer dayCursor.Close(ctx)
		dayNames := map[int]string{
			1: "Sunday", 2: "Monday", 3: "Tuesday", 4: "Wednesday",
			5: "Thursday", 6: "Friday", 7: "Saturday",
		}
		for dayCursor.Next(ctx) {
			var result bson.M
			if err := dayCursor.Decode(&result); err == nil {
				if dayNum, ok := result["_id"].(int32); ok {
					if count, ok := result["count"].(int32); ok {
						dayName := dayNames[int(dayNum)]
						analytics.TasksByDayOfWeek[dayName] = int64(count)
					}
				}
			}
		}
	}

	// Get daily trends (Created vs Completed)
	trendsPipeline := mongo.Pipeline{
		{{"$match", bson.M{"project_id": bson.M{"$in": projectIDs}}}},
		{{"$group", bson.D{
			{"_id", bson.D{{"$dateToString", bson.D{{"format", "%Y-%m-%d"}, {"date", "$created_at"}}}}},
			{"created", bson.D{{"$sum", 1}}},
			{"completed", bson.D{{"$sum", bson.D{{"$cond", bson.A{bson.D{{"$eq", bson.A{"$status", "Done"}}}, 1, 0}}}}}},
		}}},
		{{"$sort", bson.D{{"_id", 1}}}},
		{{"$limit", 31}}, // Last month approximately
	}

	trendCursor, err := db.Tasks.Aggregate(ctx, trendsPipeline)
	if err == nil {
		defer trendCursor.Close(ctx)
		for trendCursor.Next(ctx) {
			var result struct {
				ID        string `bson:"_id"`
				Created   int    `bson:"created"`
				Completed int    `bson:"completed"`
			}
			if err := trendCursor.Decode(&result); err == nil {
				analytics.TaskTrends = append(analytics.TaskTrends, TaskTrend{
					Date:      result.ID,
					Created:   result.Created,
					Completed: result.Completed,
				})
			}
		}
	}

	// Get top priority tasks
	topTasksCursor, err := db.Tasks.Find(ctx, bson.M{
		"project_id": bson.M{"$in": projectIDs},
		"priority":   "high",
		"status":     bson.M{"$ne": "Done"},
	}, options.Find().SetLimit(10).SetSort(bson.M{"created_at": -1}))
	if err == nil {
		defer topTasksCursor.Close(ctx)
		for topTasksCursor.Next(ctx) {
			var task models.Task
			if err := topTasksCursor.Decode(&task); err == nil {
				analytics.TopPriorityTasks = append(analytics.TopPriorityTasks, TaskSummary{
					ID:       task.ID.Hex(),
					Title:    task.Title,
					Priority: task.Priority,
					Status:   task.Status,
				})
			}
		}
	}

	// Cache for 5 minutes
	cache.Set(cacheKey, analytics, 5*time.Minute)

	return analytics, nil
}

// GetActivityData returns task activity counts over time
func GetActivityData(userID primitive.ObjectID, projectID *primitive.ObjectID, days int) ([]DateTaskCount, error) {
	ctx := context.Background()

	// Get user's projects
	projects, err := GetProjects(userID)
	if err != nil {
		return nil, err
	}

	projectIDs := make([]primitive.ObjectID, len(projects))
	for i, p := range projects {
		projectIDs[i] = p.ID
	}

	// Filter by project if specified
	if projectID != nil {
		hasAccess := false
		for _, pid := range projectIDs {
			if pid == *projectID {
				hasAccess = true
				break
			}
		}
		if hasAccess {
			projectIDs = []primitive.ObjectID{*projectID}
		}
	}

	if len(projectIDs) == 0 {
		return []DateTaskCount{}, nil
	}

	// Calculate start date
	startTime := time.Now().AddDate(0, 0, -days)

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"project_id": bson.M{"$in": projectIDs},
			"status":     "Done",
			"updated_at": bson.M{"$gte": startTime},
		}}},
		{{"$group", bson.D{
			{"_id", bson.D{{"$dateToString", bson.D{{"format", "%Y-%m-%d"}, {"date", "$updated_at"}}}}},
			{"count", bson.D{{"$sum", 1}}},
		}}},
		{{"$sort", bson.D{{"_id", 1}}}},
	}

	cursor, err := db.Tasks.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []DateTaskCount
	for cursor.Next(ctx) {
		var res struct {
			Date  string `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err := cursor.Decode(&res); err == nil {
			results = append(results, DateTaskCount{
				Date:  res.Date,
				Count: res.Count,
			})
		}
	}

	return results, nil
}
