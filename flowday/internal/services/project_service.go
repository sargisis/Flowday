package services

import (
	"context"
	"time"

	"flowday/internal/db"
	"flowday/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func CreateProject(userID primitive.ObjectID, name string) (*models.Project, error) {
	project := models.Project{
		ID:        primitive.NewObjectID(),
		Name:      name,
		UserID:    userID,
		CreatedAt: time.Now(),
	}

	ctx := context.Background()
	_, err := db.Projects.InsertOne(ctx, project)
	if err != nil {
		return nil, err
	}

	return &project, nil
}

func GetProjects(userID primitive.ObjectID) ([]models.Project, error) {
	ctx := context.Background()

	// Get projects owned by user
	cursor, err := db.Projects.Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var ownedProjects []models.Project
	if err = cursor.All(ctx, &ownedProjects); err != nil {
		return nil, err
	}

	// Get projects where user is an accepted member
	memberCursor, err := db.ProjectMembers.Find(ctx, bson.M{
		"user_id": userID,
		"status":  "accepted",
	})
	if err != nil {
		return ownedProjects, nil // Return owned projects even if member query fails
	}
	defer memberCursor.Close(ctx)

	var memberships []models.ProjectMember
	if err = memberCursor.All(ctx, &memberships); err != nil {
		return ownedProjects, nil
	}

	// Get project details for each membership
	projectMap := make(map[primitive.ObjectID]bool)
	for _, p := range ownedProjects {
		projectMap[p.ID] = true
	}

	for _, membership := range memberships {
		// Skip if already in owned projects
		if projectMap[membership.ProjectID] {
			continue
		}

		var project models.Project
		err := db.Projects.FindOne(ctx, bson.M{"_id": membership.ProjectID}).Decode(&project)
		if err == nil {
			ownedProjects = append(ownedProjects, project)
			projectMap[project.ID] = true
		}
	}

	return ownedProjects, nil
}

func DeleteProject(userID, projectID primitive.ObjectID) error {
	ctx := context.Background()
	_, err := db.Projects.DeleteOne(ctx, bson.M{
		"_id":     projectID,
		"user_id": userID,
	})
	return err
}
