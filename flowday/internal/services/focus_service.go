package services

import (
	"context"
	"flowday/internal/db"
	"flowday/internal/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func SaveFocusSession(userID primitive.ObjectID, taskID *primitive.ObjectID, taskTitle string, duration int) error {
	coll := db.FocusSessions

	// Award XP: 10 XP per minute of focus
	xpEarned := duration * 10

	session := models.FocusSession{
		ID:        primitive.NewObjectID(),
		UserID:    userID,
		TaskID:    taskID,
		TaskTitle: taskTitle,
		Duration:  duration,
		XP:        xpEarned,
		CreatedAt: time.Now(),
	}

	_, err := coll.InsertOne(context.Background(), session)
	if err != nil {
		return err
	}

	// Update user XP
	userColl := db.Users
	filter := bson.M{"_id": userID}
	update := bson.M{
		"$inc": bson.M{"xp": xpEarned},
	}
	_, err = userColl.UpdateOne(context.Background(), filter, update)

	return err
}

func GetFocusHistory(userID primitive.ObjectID) ([]models.FocusSession, error) {
	coll := db.FocusSessions
	filter := bson.M{"user_id": userID}
	opts := options.Find().SetSort(bson.M{"created_at": -1}).SetLimit(20)

	cursor, err := coll.Find(context.Background(), filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var sessions = []models.FocusSession{}
	if err = cursor.All(context.Background(), &sessions); err != nil {
		return nil, err
	}
	return sessions, nil
}
