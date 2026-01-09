package services

import (
	"context"
	"flowday/internal/db"
	"flowday/internal/models"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// InitializeAchievements creates the default set of achievements in the database
func InitializeAchievements() error {
	achievements := []models.Achievement{
		{
			ID:          "fire_starter",
			Title:       "Fire Starter",
			Description: "Complete your first day of activity",
			Icon:        "🔥",
			Type:        models.AchievementTypeStreak,
			Rarity:      models.RarityCommon,
			XPReward:    50,
			Requirement: 1,
			CreatedAt:   time.Now(),
		},
		{
			ID:          "week_warrior",
			Title:       "Week Warrior",
			Description: "Maintain a 7-day streak",
			Icon:        "🏆",
			Type:        models.AchievementTypeStreak,
			Rarity:      models.RarityRare,
			XPReward:    200,
			Requirement: 7,
			CreatedAt:   time.Now(),
		},
		{
			ID:          "sharpshooter",
			Title:       "Sharpshooter",
			Description: "Complete 10 tasks",
			Icon:        "🎯",
			Type:        models.AchievementTypeTask,
			Rarity:      models.RarityUncommon,
			XPReward:    100,
			Requirement: 10,
			CreatedAt:   time.Now(),
		},
		{
			ID:          "night_owl",
			Title:       "Night Owl",
			Description: "Work after 10 PM five times",
			Icon:        "🌙",
			Type:        models.AchievementTypeFocus,
			Rarity:      models.RarityUncommon,
			XPReward:    150,
			Requirement: 5,
			CreatedAt:   time.Now(),
		},
		{
			ID:          "speed_demon",
			Title:       "Speed Demon",
			Description: "Complete 5 tasks in one focus session",
			Icon:        "⚡",
			Type:        models.AchievementTypeFocus,
			Rarity:      models.RarityEpic,
			XPReward:    250,
			Requirement: 5,
			CreatedAt:   time.Now(),
		},
	}

	for _, achievement := range achievements {
		// Upsert each achievement
		filter := bson.M{"_id": achievement.ID}
		update := bson.M{"$setOnInsert": achievement}
		opts := options.Update().SetUpsert(true)

		_, err := db.Achievements.UpdateOne(context.Background(), filter, update, opts)
		if err != nil {
			return err
		}
	}

	log.Println("Achievements initialized")
	return nil
}

// UpdateStreak updates the user's daily activity streak
func UpdateStreak(userID primitive.ObjectID) error {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	// Find existing streak record
	var streak models.UserStreak
	err := db.UserStreaks.FindOne(context.Background(), bson.M{"user_id": userID}).Decode(&streak)

	if err != nil {
		// Create new streak
		streak = models.UserStreak{
			UserID:         userID,
			CurrentStreak:  1,
			LongestStreak:  1,
			LastActiveDate: today,
			UpdatedAt:      now,
		}
		_, err = db.UserStreaks.InsertOne(context.Background(), streak)
		return err
	}

	// Check if already active today
	lastActiveDay := time.Date(
		streak.LastActiveDate.Year(),
		streak.LastActiveDate.Month(),
		streak.LastActiveDate.Day(),
		0, 0, 0, 0, time.UTC,
	)

	if lastActiveDay.Equal(today) {
		return nil // Already counted today
	}

	// Check if streak continues (yesterday was last active day)
	yesterday := today.AddDate(0, 0, -1)
	if lastActiveDay.Equal(yesterday) {
		// Continue streak
		streak.CurrentStreak++
	} else {
		// Streak broken, restart
		streak.CurrentStreak = 1
	}

	// Update longest streak if needed
	if streak.CurrentStreak > streak.LongestStreak {
		streak.LongestStreak = streak.CurrentStreak
	}

	streak.LastActiveDate = today
	streak.UpdatedAt = now

	// Update in database
	_, err = db.UserStreaks.UpdateOne(
		context.Background(),
		bson.M{"user_id": userID},
		bson.M{"$set": streak},
	)

	if err == nil {
		log.Printf("[Streak] Updated for user %s: current=%d, lastActive=%v", userID.Hex(), streak.CurrentStreak, today)
	} else {
		log.Printf("[Streak] Error updating for user %s: %v", userID.Hex(), err)
	}

	// Check for streak achievements
	go CheckAndAwardAchievements(userID, "streak_updated", map[string]interface{}{
		"current_streak": streak.CurrentStreak,
	})

	return err
}

// CheckAndAwardAchievements evaluates if any achievements should be unlocked
func CheckAndAwardAchievements(userID primitive.ObjectID, eventType string, metadata map[string]interface{}) {
	// Update streak on any activity
	UpdateStreak(userID)

	// Get all achievements
	cursor, err := db.Achievements.Find(context.Background(), bson.M{})
	if err != nil {
		log.Println("Error fetching achievements:", err)
		return
	}
	defer cursor.Close(context.Background())

	var achievements []models.Achievement
	if err = cursor.All(context.Background(), &achievements); err != nil {
		log.Println("Error decoding achievements:", err)
		return
	}

	for _, achievement := range achievements {
		shouldCheck := false

		// Determine if we should check this achievement based on event type
		switch eventType {
		case "task_completed":
			shouldCheck = achievement.Type == models.AchievementTypeTask
		case "focus_completed":
			shouldCheck = achievement.Type == models.AchievementTypeFocus
		case "streak_updated":
			shouldCheck = achievement.Type == models.AchievementTypeStreak
		}

		if !shouldCheck {
			continue
		}

		// Get or create user achievement progress
		var userAch models.UserAchievement
		err := db.UserAchievements.FindOne(
			context.Background(),
			bson.M{"user_id": userID, "achievement_id": achievement.ID},
		).Decode(&userAch)

		if err != nil {
			// Create new progress record
			userAch = models.UserAchievement{
				UserID:        userID,
				AchievementID: achievement.ID,
				Progress:      0,
				IsUnlocked:    false,
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}
		}

		// Skip if already unlocked
		if userAch.IsUnlocked {
			continue
		}

		// Update progress based on achievement type
		shouldUpdate := false
		switch achievement.ID {
		case "sharpshooter": // 10 tasks
			if eventType == "task_completed" {
				userAch.Progress++
				shouldUpdate = true
			}
		case "night_owl": // Work after 10pm 5 times
			if eventType == "focus_completed" {
				if taskTime, ok := metadata["completed_at"].(time.Time); ok {
					if taskTime.Hour() >= 22 || taskTime.Hour() < 6 {
						userAch.Progress++
						shouldUpdate = true
					}
				}
			}
		case "speed_demon": // 5 tasks in one session
			if eventType == "focus_completed" {
				if tasksInSession, ok := metadata["tasks_completed"].(int); ok {
					if tasksInSession >= 5 {
						userAch.Progress = achievement.Requirement
						shouldUpdate = true
					}
				}
			}
		case "fire_starter", "week_warrior": // Streaks
			if eventType == "streak_updated" {
				if currentStreak, ok := metadata["current_streak"].(int); ok {
					userAch.Progress = currentStreak
					shouldUpdate = true
				}
			}
		}

		// Check if unlocked
		if shouldUpdate && userAch.Progress >= achievement.Requirement {
			userAch.IsUnlocked = true
			now := time.Now()
			userAch.UnlockedAt = &now
			userAch.UpdatedAt = now

			// Award XP
			userFilter := bson.M{"_id": userID}
			userUpdate := bson.M{"$inc": bson.M{"xp": achievement.XPReward}}
			db.Users.UpdateOne(context.Background(), userFilter, userUpdate)

			// Log activity
			LogActivity(userID, models.ActivityType("achievement_unlocked"), "Unlocked achievement: "+achievement.Title, map[string]string{
				"achievement_id": achievement.ID,
				"icon":           achievement.Icon,
			})

			log.Printf("User %s unlocked achievement: %s", userID.Hex(), achievement.Title)
		}

		if shouldUpdate {
			userAch.UpdatedAt = time.Now()

			// Upsert user achievement
			filter := bson.M{"user_id": userID, "achievement_id": achievement.ID}
			update := bson.M{"$set": userAch}
			opts := options.Update().SetUpsert(true)
			db.UserAchievements.UpdateOne(context.Background(), filter, update, opts)
		}
	}
}

// GetUserAchievements returns all achievements with user's progress
func GetUserAchievements(userID primitive.ObjectID) ([]map[string]interface{}, error) {
	// Get all achievements
	cursor, err := db.Achievements.Find(context.Background(), bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var achievements []models.Achievement
	if err = cursor.All(context.Background(), &achievements); err != nil {
		return nil, err
	}

	// Get user's progress
	userAchCursor, err := db.UserAchievements.Find(context.Background(), bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	defer userAchCursor.Close(context.Background())

	var userAchs []models.UserAchievement
	if err = userAchCursor.All(context.Background(), &userAchs); err != nil {
		return nil, err
	}

	// Create map for quick lookup
	progressMap := make(map[string]models.UserAchievement)
	for _, ua := range userAchs {
		progressMap[ua.AchievementID] = ua
	}

	// Combine data
	result := make([]map[string]interface{}, 0, len(achievements))
	for _, ach := range achievements {
		item := map[string]interface{}{
			"achievement": ach,
			"progress":    0,
			"is_unlocked": false,
			"unlocked_at": nil,
		}

		if userAch, exists := progressMap[ach.ID]; exists {
			item["progress"] = userAch.Progress
			item["is_unlocked"] = userAch.IsUnlocked
			item["unlocked_at"] = userAch.UnlockedAt
		}

		result = append(result, item)
	}

	return result, nil
}

// GetUserStreak returns the user's current streak information
func GetUserStreak(userID primitive.ObjectID) (*models.UserStreak, error) {
	var streak models.UserStreak
	err := db.UserStreaks.FindOne(context.Background(), bson.M{"user_id": userID}).Decode(&streak)
	if err != nil {
		return nil, err
	}
	return &streak, nil
}
