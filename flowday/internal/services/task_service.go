package services

import (
	"context"
	"errors"
	"time"

	"flowday/internal/db"
	"flowday/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func CreateTask(userID primitive.ObjectID, task *models.Task) error {
	ctx := context.Background()

	// Verify project ownership
	var project models.Project
	err := db.Projects.FindOne(ctx, bson.M{
		"_id":     task.ProjectID,
		"user_id": userID,
	}).Decode(&project)

	if err != nil {
		return errors.New("project not found")
	}

	task.ID = primitive.NewObjectID()
	task.CreatedAt = time.Now()

	_, err = db.Tasks.InsertOne(ctx, task)
	return err
}

func GetTasksByProject(userID, projectID primitive.ObjectID) ([]models.Task, error) {
	ctx := context.Background()

	// Verify project exists and user has access (owner OR accepted member)
	var project models.Project
	err := db.Projects.FindOne(ctx, bson.M{"_id": projectID}).Decode(&project)
	if err != nil {
		return nil, errors.New("project not found")
	}

	// Check if user is owner
	isOwner := project.UserID == userID

	// If not owner, check if user is accepted member
	if !isOwner {
		var membership models.ProjectMember
		err := db.ProjectMembers.FindOne(ctx, bson.M{
			"project_id": projectID,
			"user_id":    userID,
			"status":     "accepted",
		}).Decode(&membership)

		if err != nil {
			return nil, errors.New("access denied")
		}
	}

	cursor, err := db.Tasks.Find(ctx, bson.M{"project_id": projectID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tasks []models.Task
	if err = cursor.All(ctx, &tasks); err != nil {
		return nil, err
	}

	return tasks, nil
}

func UpdateTask(userID, taskID primitive.ObjectID, updates map[string]interface{}) error {
	ctx := context.Background()

	// First get the task
	var task models.Task
	err := db.Tasks.FindOne(ctx, bson.M{"_id": taskID}).Decode(&task)
	if err != nil {
		return errors.New("task not found")
	}

	// Verify user owns the project
	var project models.Project
	err = db.Projects.FindOne(ctx, bson.M{
		"_id":     task.ProjectID,
		"user_id": userID,
	}).Decode(&project)

	if err != nil {
		return errors.New("access denied")
	}

	// Update task
	_, err = db.Tasks.UpdateOne(ctx,
		bson.M{"_id": taskID},
		bson.M{"$set": updates},
	)

	return err
}

func DeleteTask(userID, taskID primitive.ObjectID) error {
	ctx := context.Background()

	// First get the task
	var task models.Task
	err := db.Tasks.FindOne(ctx, bson.M{"_id": taskID}).Decode(&task)
	if err != nil {
		return errors.New("task not found")
	}

	// Verify user owns the project
	var project models.Project
	err = db.Projects.FindOne(ctx, bson.M{
		"_id":     task.ProjectID,
		"user_id": userID,
	}).Decode(&project)

	if err != nil {
		return errors.New("access denied")
	}

	_, err = db.Tasks.DeleteOne(ctx, bson.M{"_id": taskID})
	return err
}

func GetTasksByDateRange(userID primitive.ObjectID, start, end time.Time) ([]models.Task, error) {
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

	cursor, err := db.Tasks.Find(ctx, bson.M{
		"project_id": bson.M{"$in": projectIDs},
		"due_date":   bson.M{"$gte": start, "$lte": end},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tasks []models.Task
	if err = cursor.All(ctx, &tasks); err != nil {
		return nil, err
	}

	return tasks, nil
}

func GetTask(userID, taskID primitive.ObjectID) (*models.Task, error) {
	ctx := context.Background()

	// Get the task
	var task models.Task
	err := db.Tasks.FindOne(ctx, bson.M{"_id": taskID}).Decode(&task)
	if err != nil {
		return nil, errors.New("task not found")
	}

	// Verify access (Project Owner OR Member)
	var project models.Project
	err = db.Projects.FindOne(ctx, bson.M{"_id": task.ProjectID}).Decode(&project)
	if err != nil {
		return nil, errors.New("project not found")
	}

	if project.UserID == userID {
		return &task, nil
	}

	// Check membership if not owner
	var member models.ProjectMember
	err = db.ProjectMembers.FindOne(ctx, bson.M{
		"project_id": task.ProjectID,
		"user_id":    userID,
		"status":     "accepted",
	}).Decode(&member)

	if err == nil {
		return &task, nil
	}

	return nil, errors.New("access denied")
}
