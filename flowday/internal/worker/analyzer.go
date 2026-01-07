package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"flowday/internal/db"
	"flowday/internal/models"
	"flowday/internal/services"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StartAnalyzer starts the background analysis loop
func StartAnalyzer() {
	ticker := time.NewTicker(1 * time.Hour) // Run every hour
	go func() {
		for {
			select {
			case <-ticker.C:
				runAnalysis()
			}
		}
	}()
	log.Println("[Worker] Background Analyzer started")
}

func runAnalysis() {
	log.Println("[Worker] Running scheduled analysis...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 1. Fetch all pro users (or all users if we want to upsell)
	// For now, let's analyze everyone but prioritize Pro in production
	cursor, err := db.Users.Find(ctx, bson.M{})
	if err != nil {
		log.Printf("[Worker] Failed to fetch users: %v", err)
		return
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var user models.User
		if err := cursor.Decode(&user); err != nil {
			continue
		}

		// Analyze this user
		if err := analyzeUser(ctx, user); err != nil {
			log.Printf("[Worker] Failed to analyze user %s: %v", user.ID.Hex(), err)
		}
	}
}

func analyzeUser(ctx context.Context, user models.User) error {
	// 2. Gather State
	tasks, err := services.GetUserTasks(user.ID)
	if err != nil {
		return err
	}

	// 3. Simple Heuristics for Insight Generation

	// Check for Stale Tasks
	for _, t := range tasks {
		if t.Status == "in_progress" && time.Since(t.UpdatedAt).Hours() > 72 {
			// Found a stale task
			createInsight(ctx, user.ID, models.InsightTypeStale,
				fmt.Sprintf("Stuck on '%s'?", t.Title),
				"This task has been in progress for over 3 days. Consider breaking it down or moving it back to Todo.")
		}
	}

	// Check Velocity/Burnout
	completedLast24h := 0
	for _, t := range tasks {
		if t.Status == "done" && time.Since(t.UpdatedAt).Hours() < 24 {
			completedLast24h++
		}
	}

	if completedLast24h > 10 {
		createInsight(ctx, user.ID, models.InsightTypeGeneral,
			"On Fire! 🔥",
			fmt.Sprintf("You completed %d tasks in the last 24 hours. Great flow! Don't forget to take a break.", completedLast24h))
	}

	if completedLast24h == 0 && len(tasks) > 5 {
		// Gentle nudge
		createInsight(ctx, user.ID, models.InsightTypeGeneral,
			"Ready to flow?",
			"No tasks completed recently. Pick one small thing start with.")
	}

	return nil
}

func createInsight(ctx context.Context, userID primitive.ObjectID, iType models.InsightType, title, message string) {
	// Check duplicate in last 24h
	count, _ := db.Database.Collection("ai_insights").CountDocuments(ctx, bson.M{
		"user_id":    userID,
		"title":      title,
		"created_at": bson.M{"$gt": time.Now().Add(-24 * time.Hour)},
	})

	if count > 0 {
		return // Don't spam
	}

	insight := models.Insight{
		ID:        primitive.NewObjectID(),
		UserID:    userID,
		Type:      iType,
		Title:     title,
		Message:   message,
		IsRead:    false,
		CreatedAt: time.Now(),
	}

	_, err := db.Database.Collection("ai_insights").InsertOne(ctx, insight)
	if err != nil {
		log.Printf("[Worker] Failed to save insight: %v", err)
	}
}
