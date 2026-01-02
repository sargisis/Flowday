package db

import (
	"context"
	"log"
	"os"
	"time"

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
)

func Connect() {
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal("Failed to ping MongoDB:", err)
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

	log.Println("Connected to MongoDB")
}
