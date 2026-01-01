package services

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"flowday/internal/db"
	"flowday/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func CreateTask(userID primitive.ObjectID, task *models.Task) error {
	ctx := context.Background()

	// Verify access
	if err := verifyProjectAccess(ctx, userID, task.ProjectID); err != nil {
		return err
	}

	task.ID = primitive.NewObjectID()
	task.CreatedAt = time.Now()

	_, err := db.Tasks.InsertOne(ctx, task)
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

	// Verify access
	if err := verifyProjectAccess(ctx, userID, task.ProjectID); err != nil {
		return err
	}

	// Update task
	updateResult, err := db.Tasks.UpdateOne(ctx,
		bson.M{"_id": taskID},
		bson.M{"$set": updates},
	)
	if err != nil {
		return err
	}

	// XP & Leveling Logic
	newStatus, ok := updates["status"].(string)
	if ok && strings.ToLower(newStatus) == "done" && strings.ToLower(task.Status) != "done" {
		// Award XP
		xpToAdd := 10
		_, err = db.Users.UpdateOne(ctx,
			bson.M{"_id": userID},
			bson.M{
				"$inc": bson.M{"xp": xpToAdd},
			},
		)
		if err == nil {
			// Check for Level up (Level = (XP / 100) + 1)
			var updatedUser models.User
			if err := db.Users.FindOne(ctx, bson.M{"_id": userID}).Decode(&updatedUser); err == nil {
				// Initialize level if it's 0
				if updatedUser.Level == 0 {
					updatedUser.Level = 1
				}
				newLevel := (updatedUser.XP / 100) + 1
				if newLevel > updatedUser.Level {
					db.Users.UpdateOne(ctx,
						bson.M{"_id": userID},
						bson.M{"$set": bson.M{"level": newLevel}},
					)
				} else if updatedUser.Level == 1 && updatedUser.XP < 100 {
					// Ensure level 1 is saved even if XP is low
					db.Users.UpdateOne(ctx,
						bson.M{"_id": userID},
						bson.M{"$set": bson.M{"level": 1}},
					)
				}
			}
		}
	}

	_ = updateResult
	return nil
}

func DeleteTask(userID, taskID primitive.ObjectID) error {
	ctx := context.Background()

	// First get the task
	var task models.Task
	err := db.Tasks.FindOne(ctx, bson.M{"_id": taskID}).Decode(&task)
	if err != nil {
		return errors.New("task not found")
	}

	// Verify access
	if err := verifyProjectAccess(ctx, userID, task.ProjectID); err != nil {
		return err
	}

	_, err = db.Tasks.DeleteOne(ctx, bson.M{"_id": taskID})
	return err
}

func verifyProjectAccess(ctx context.Context, userID, projectID primitive.ObjectID) error {
	log.Printf("[AccessCheck] User: %s, Project: %s", userID.Hex(), projectID.Hex())

	// Check if owner
	var project models.Project
	err := db.Projects.FindOne(ctx, bson.M{"_id": projectID}).Decode(&project)
	if err != nil {
		log.Printf("[AccessCheck] Project Not Found: %v", err)
		return errors.New("project not found")
	}

	if project.UserID == userID {
		log.Printf("[AccessCheck] Access Granted: Owner")
		return nil
	}

	log.Printf("[AccessCheck] Not Owner (Project Owner: %s)", project.UserID.Hex())

	// Check if accepted member
	var member models.ProjectMember
	err = db.ProjectMembers.FindOne(ctx, bson.M{
		"project_id": projectID,
		"user_id":    userID,
		"status":     "accepted",
	}).Decode(&member)

	if err == nil {
		log.Printf("[AccessCheck] Access Granted: Accepted Member")
		return nil
	}

	log.Printf("[AccessCheck] Access Denied: Member not found or not accepted (%v)", err)
	return errors.New("access denied")
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
	if err := verifyProjectAccess(ctx, userID, task.ProjectID); err != nil {
		return nil, err
	}

	return &task, nil
}
