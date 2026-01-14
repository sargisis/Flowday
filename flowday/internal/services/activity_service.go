package services

import (
	"context"
	"flowday/internal/db"
	"flowday/internal/dto"
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

// GetUserActivitiesPaginated returns paginated activities for a user
func GetUserActivitiesPaginated(userID primitive.ObjectID, pagination dto.PaginationQuery) ([]models.Activity, dto.PaginationMeta, error) {
	// Validate and set defaults
	pagination.ValidateAndSetDefaults()

	ctx := context.Background()
	coll := db.Activities
	filter := bson.M{"user_id": userID}

	// Get total count
	total, err := coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, dto.PaginationMeta{}, err
	}

	// Build sort order
	sortOrder := -1 // desc (newest first by default)
	if pagination.Order == "asc" {
		sortOrder = 1
	}
	sortField := pagination.Sort
	if sortField == "" {
		sortField = "created_at"
	}

	// Build find options
	findOptions := options.Find().
		SetLimit(int64(pagination.Limit)).
		SetSkip(int64(pagination.GetOffset())).
		SetSort(bson.M{sortField: sortOrder})

	cursor, err := coll.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, dto.PaginationMeta{}, err
	}
	defer cursor.Close(ctx)

	var activities []models.Activity
	if err = cursor.All(ctx, &activities); err != nil {
		return nil, dto.PaginationMeta{}, err
	}

	meta := dto.NewPaginationMeta(pagination, total)

	return activities, meta, nil
}
