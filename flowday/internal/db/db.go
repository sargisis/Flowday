package db

import (
	"context"
	"os"
	"time"

	"flowday/internal/logger"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	Client   *mongo.Client
	Database *mongo.Database

	// Collections
	Users               *mongo.Collection
	Projects            *mongo.Collection
	Tasks               *mongo.Collection
	PasswordResets      *mongo.Collection
	ProjectMembers      *mongo.Collection
	Notifications       *mongo.Collection
	EmailChangeRequests *mongo.Collection
	FocusSessions       *mongo.Collection
	Activities          *mongo.Collection
	Achievements        *mongo.Collection
	UserAchievements    *mongo.Collection
	UserStreaks         *mongo.Collection
	Messages            *mongo.Collection
	RefreshTokens       *mongo.Collection
	Comments            *mongo.Collection
)

func Connect() {
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// ✅ PERFORMANCE: Configure connection pooling for better performance
	clientOptions := options.Client().ApplyURI(mongoURI).
		SetMaxPoolSize(100).                  // Maximum number of connections in the pool
		SetMinPoolSize(10).                   // Minimum number of connections in the pool
		SetMaxConnIdleTime(30 * time.Minute). // Close connections after 30 minutes of inactivity
		SetConnectTimeout(10 * time.Second).  // Timeout for initial connection
		SetSocketTimeout(30 * time.Second)    // Timeout for socket operations

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		logger.Log.WithError(err).Fatal("Failed to connect to MongoDB")
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		logger.Log.WithError(err).Fatal("Failed to ping MongoDB")
	}

	Client = client
	Database = client.Database("flowday")

	// Initialize collections
	Users = Database.Collection("users")
	Projects = Database.Collection("projects")
	Tasks = Database.Collection("tasks")
	PasswordResets = Database.Collection("password_resets")
	ProjectMembers = Database.Collection("project_members")
	Notifications = Database.Collection("notifications")
	EmailChangeRequests = Database.Collection("email_change_requests")
	FocusSessions = Database.Collection("focus_sessions")
	Activities = Database.Collection("activities")
	Achievements = Database.Collection("achievements")
	UserAchievements = Database.Collection("user_achievements")
	UserStreaks = Database.Collection("user_streaks")
	Messages = Database.Collection("messages")
	RefreshTokens = Database.Collection("refresh_tokens")
	Comments = Database.Collection("comments")

	// ✅ PERFORMANCE: Create indexes for faster queries
	if err := createIndexes(ctx); err != nil {
		logger.Log.WithError(err).Warn("Failed to create some indexes")
		// Don't fail startup if indexes fail, but log the warning
	}

	logger.Log.Info("Connected to MongoDB with connection pooling and indexes")
}

// createIndexes creates all necessary indexes for optimal query performance
func createIndexes(ctx context.Context) error {
	// Users collection indexes
	usersIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "slack_user_id", Value: 1}}},
		{Keys: bson.D{{Key: "slack_team_id", Value: 1}}},
	}
	if _, err := Users.Indexes().CreateMany(ctx, usersIndexes); err != nil {
		return err
	}

	// Projects collection indexes
	projectsIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "user_id", Value: 1}}},
		{Keys: bson.D{{Key: "created_at", Value: -1}}},
	}
	if _, err := Projects.Indexes().CreateMany(ctx, projectsIndexes); err != nil {
		return err
	}

	// Tasks collection indexes
	tasksIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "project_id", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "priority", Value: 1}}},
		{Keys: bson.D{{Key: "due_date", Value: 1}}},
		{Keys: bson.D{{Key: "created_at", Value: -1}}},
		// Compound index for common query: project_id + status
		{Keys: bson.D{{Key: "project_id", Value: 1}, {Key: "status", Value: 1}}},
		// Compound index for search: project_id + priority + status
		{Keys: bson.D{{Key: "project_id", Value: 1}, {Key: "priority", Value: 1}, {Key: "status", Value: 1}}},
	}
	if _, err := Tasks.Indexes().CreateMany(ctx, tasksIndexes); err != nil {
		return err
	}

	// ProjectMembers collection indexes
	projectMembersIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "user_id", Value: 1}}},
		{Keys: bson.D{{Key: "project_id", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		// Compound index for common query: user_id + status
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "status", Value: 1}}},
		// Compound unique index: project_id + user_id (one user can only be in a project once)
		{Keys: bson.D{{Key: "project_id", Value: 1}, {Key: "user_id", Value: 1}}, Options: options.Index().SetUnique(true)},
	}
	if _, err := ProjectMembers.Indexes().CreateMany(ctx, projectMembersIndexes); err != nil {
		return err
	}

	// Comments collection indexes
	commentsIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "task_id", Value: 1}}},
		{Keys: bson.D{{Key: "user_id", Value: 1}}},
		{Keys: bson.D{{Key: "created_at", Value: -1}}},
		// Compound index for common query: task_id + created_at
		{Keys: bson.D{{Key: "task_id", Value: 1}, {Key: "created_at", Value: -1}}},
	}
	if _, err := Comments.Indexes().CreateMany(ctx, commentsIndexes); err != nil {
		return err
	}

	// Messages collection indexes
	messagesIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "sender_id", Value: 1}}},
		{Keys: bson.D{{Key: "receiver_id", Value: 1}}},
		{Keys: bson.D{{Key: "created_at", Value: -1}}},
		{Keys: bson.D{{Key: "is_read", Value: 1}}},
		// Compound index for conversation query: sender_id + receiver_id + created_at
		{Keys: bson.D{{Key: "sender_id", Value: 1}, {Key: "receiver_id", Value: 1}, {Key: "created_at", Value: -1}}},
		// Compound index for unread messages: receiver_id + is_read + created_at
		{Keys: bson.D{{Key: "receiver_id", Value: 1}, {Key: "is_read", Value: 1}, {Key: "created_at", Value: -1}}},
	}
	if _, err := Messages.Indexes().CreateMany(ctx, messagesIndexes); err != nil {
		return err
	}

	// Notifications collection indexes
	notificationsIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "user_id", Value: 1}}},
		{Keys: bson.D{{Key: "read", Value: 1}}},
		{Keys: bson.D{{Key: "created_at", Value: -1}}},
		// Compound index for common query: user_id + read + created_at
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "read", Value: 1}, {Key: "created_at", Value: -1}}},
	}
	if _, err := Notifications.Indexes().CreateMany(ctx, notificationsIndexes); err != nil {
		return err
	}

	// FocusSessions collection indexes
	focusSessionsIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "user_id", Value: 1}}},
		{Keys: bson.D{{Key: "created_at", Value: -1}}},
		// Compound index for user's focus history: user_id + created_at
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}}},
	}
	if _, err := FocusSessions.Indexes().CreateMany(ctx, focusSessionsIndexes); err != nil {
		return err
	}

	// RefreshTokens collection indexes
	refreshTokensIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "token", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "user_id", Value: 1}}},
		{Keys: bson.D{{Key: "expires_at", Value: 1}}},
	}
	if _, err := RefreshTokens.Indexes().CreateMany(ctx, refreshTokensIndexes); err != nil {
		return err
	}

	// PasswordResets collection indexes
	passwordResetsIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "email", Value: 1}}},
		{Keys: bson.D{{Key: "code", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "expires_at", Value: 1}}},
		// TTL index to auto-delete expired password resets (expires after expires_at)
		{Keys: bson.D{{Key: "expires_at", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
	}
	if _, err := PasswordResets.Indexes().CreateMany(ctx, passwordResetsIndexes); err != nil {
		return err
	}

	// EmailChangeRequests collection indexes
	emailChangeIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "user_id", Value: 1}}},
		{Keys: bson.D{{Key: "code", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "expires_at", Value: 1}}},
		// TTL index to auto-delete expired email change requests
		{Keys: bson.D{{Key: "expires_at", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
	}
	if _, err := EmailChangeRequests.Indexes().CreateMany(ctx, emailChangeIndexes); err != nil {
		return err
	}

	// Activities collection indexes
	activitiesIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "user_id", Value: 1}}},
		{Keys: bson.D{{Key: "created_at", Value: -1}}},
		// Compound index for activity feed: user_id + created_at
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}}},
	}
	if _, err := Activities.Indexes().CreateMany(ctx, activitiesIndexes); err != nil {
		return err
	}

	// UserAchievements collection indexes
	userAchievementsIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "user_id", Value: 1}}},
		{Keys: bson.D{{Key: "achievement_id", Value: 1}}},
		// Compound unique index: user_id + achievement_id (one achievement per user)
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "achievement_id", Value: 1}}, Options: options.Index().SetUnique(true)},
	}
	if _, err := UserAchievements.Indexes().CreateMany(ctx, userAchievementsIndexes); err != nil {
		return err
	}

	// UserStreaks collection indexes
	userStreaksIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "user_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "last_activity_date", Value: 1}}},
	}
	if _, err := UserStreaks.Indexes().CreateMany(ctx, userStreaksIndexes); err != nil {
		return err
	}

	logger.Log.Info("✅ Database indexes created successfully")
	return nil
}
