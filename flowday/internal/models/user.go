package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name          string             `bson:"name" json:"name"`
	Email         string             `bson:"email" json:"email"`
	Password      string             `bson:"password" json:"-"`
	Bio           string             `bson:"bio" json:"bio"`
	AvatarURL     string             `bson:"avatar_url" json:"avatar_url"`
	WorkspaceName string             `bson:"workspace_name" json:"workspace_name"`
	Status        string             `bson:"status" json:"status"`
	Velocity      int                `bson:"velocity" json:"velocity"`
	XP            int                `bson:"xp" json:"xp"`
	Level         int                `bson:"level" json:"level"`

	// AI Quota & Plan
	AIQuotaUsed    int       `bson:"ai_quota_used" json:"ai_quota_used"`
	LastQuotaReset time.Time `bson:"last_quota_reset" json:"last_quota_reset"`
	Plan           string    `bson:"plan" json:"plan"` // "free", "pro"

	// Notification Preferences
	EmailNotifications bool   `bson:"email_notifications" json:"email_notifications"`
	SlackWebhookURL    string `bson:"slack_webhook_url" json:"slack_webhook_url,omitempty"` // Legacy: kept for backward compatibility
	
	// Slack OAuth Integration
	SlackAccessToken  string `bson:"slack_access_token,omitempty" json:"-"` // Never expose in JSON
	SlackRefreshToken string `bson:"slack_refresh_token,omitempty" json:"-"` // Never expose in JSON
	SlackTeamID       string `bson:"slack_team_id,omitempty" json:"slack_team_id,omitempty"`
	SlackUserID       string `bson:"slack_user_id,omitempty" json:"slack_user_id,omitempty"`
	SlackTeamName     string `bson:"slack_team_name,omitempty" json:"slack_team_name,omitempty"`
	SlackChannelID     string `bson:"slack_channel_id,omitempty" json:"slack_channel_id,omitempty"` // Default channel for notifications

	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}
