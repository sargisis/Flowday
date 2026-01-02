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

func LogActivity(userID primitive.ObjectID, activityType models.ActivityType, description string, metadata map[string]string) error {
	coll := db.Activities

	activity := models.Activity{
		ID:          primitive.NewObjectID(),
		UserID:      userID,
		Type:        activityType,
		Description: description,
		MetaData:    metadata,
		CreatedAt:   time.Now(),
	}

	_, err := coll.InsertOne(context.Background(), activity)
	return err
}

func GetUserActivities(userID primitive.ObjectID) ([]models.Activity, error) {
	coll := db.Activities
	filter := bson.M{"user_id": userID}
	opts := options.Find().SetSort(bson.M{"created_at": -1}).SetLimit(30)

	cursor, err := coll.Find(context.Background(), filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var activities = []models.Activity{}
	if err = cursor.All(context.Background(), &activities); err != nil {
		return nil, err
	}
	return activities, nil
}
