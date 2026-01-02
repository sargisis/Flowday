package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AchievementType represents the category of achievement
type AchievementType string

const (
	AchievementTypeStreak    AchievementType = "streak"
	AchievementTypeTask      AchievementType = "task"
	AchievementTypeFocus     AchievementType = "focus"
	AchievementTypeMilestone AchievementType = "milestone"
)

// AchievementRarity represents how rare/valuable an achievement is
type AchievementRarity string

const (
	RarityCommon    AchievementRarity = "common"
	RarityUncommon  AchievementRarity = "uncommon"
	RarityRare      AchievementRarity = "rare"
	RarityEpic      AchievementRarity = "epic"
	RarityLegendary AchievementRarity = "legendary"
)

// Achievement defines an achievement that can be unlocked
type Achievement struct {
	ID          string            `json:"id" bson:"_id"`
	Title       string            `json:"title" bson:"title"`
	Description string            `json:"description" bson:"description"`
	Icon        string            `json:"icon" bson:"icon"` // emoji or icon name
	Type        AchievementType   `json:"type" bson:"type"`
	Rarity      AchievementRarity `json:"rarity" bson:"rarity"`
	XPReward    int               `json:"xp_reward" bson:"xp_reward"`
	Requirement int               `json:"requirement" bson:"requirement"` // e.g., 10 tasks, 7 days streak
	CreatedAt   time.Time         `json:"created_at" bson:"created_at"`
}

// UserAchievement tracks which achievements a user has unlocked
type UserAchievement struct {
	ID            primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID        primitive.ObjectID `json:"user_id" bson:"user_id"`
	AchievementID string             `json:"achievement_id" bson:"achievement_id"`
	Progress      int                `json:"progress" bson:"progress"`       // current progress towards unlock
	IsUnlocked    bool               `json:"is_unlocked" bson:"is_unlocked"` // whether unlocked
	UnlockedAt    *time.Time         `json:"unlocked_at" bson:"unlocked_at"` // when unlocked
	CreatedAt     time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at" bson:"updated_at"`
}

// UserStreak tracks daily activity streaks
type UserStreak struct {
	ID             primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID         primitive.ObjectID `json:"user_id" bson:"user_id"`
	CurrentStreak  int                `json:"current_streak" bson:"current_streak"`
	LongestStreak  int                `json:"longest_streak" bson:"longest_streak"`
	LastActiveDate time.Time          `json:"last_active_date" bson:"last_active_date"`
	UpdatedAt      time.Time          `json:"updated_at" bson:"updated_at"`
}
