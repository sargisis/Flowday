package services

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"flowday/internal/cache"
	"flowday/internal/db"
	"flowday/internal/dto"
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

	// ✅ OPTIMIZATION: Invalidate cache for user's projects
	cache.Delete("projects:" + userID.Hex())

	return &project, nil
}

func GetProjects(userID primitive.ObjectID) ([]models.Project, error) {
	// ✅ OPTIMIZATION: Try to get from cache first
	cacheKey := "projects:" + userID.Hex()
	if cached, ok := cache.Get(cacheKey); ok {
		if projects, ok := cached.([]models.Project); ok {
			return projects, nil
		}
		// If cached value is JSON string, unmarshal it
		if jsonStr, ok := cached.(string); ok {
			var projects []models.Project
			if err := json.Unmarshal([]byte(jsonStr), &projects); err == nil {
				return projects, nil
			}
		}
	}

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

// GetProjectsPaginated returns paginated projects for a user
func GetProjectsPaginated(userID primitive.ObjectID, pagination dto.PaginationQuery) ([]models.Project, dto.PaginationMeta, error) {
	// Validate and set defaults
	pagination.ValidateAndSetDefaults()

	// Get all projects (owned + member)
	allProjects, err := GetProjects(userID)
	if err != nil {
		return nil, dto.PaginationMeta{}, err
	}

	total := int64(len(allProjects))

	// Sort projects
	sortField := pagination.Sort
	if sortField == "" {
		sortField = "created_at"
	}

	sort.Slice(allProjects, func(i, j int) bool {
		switch sortField {
		case "name":
			if pagination.Order == "asc" {
				return allProjects[i].Name < allProjects[j].Name
			}
			return allProjects[i].Name > allProjects[j].Name
		case "created_at":
			if pagination.Order == "asc" {
				return allProjects[i].CreatedAt.Before(allProjects[j].CreatedAt)
			}
			return allProjects[i].CreatedAt.After(allProjects[j].CreatedAt)
		default:
			// Default to created_at desc
			return allProjects[i].CreatedAt.After(allProjects[j].CreatedAt)
		}
	})

	// Apply pagination
	offset := pagination.GetOffset()
	end := offset + pagination.Limit
	if end > len(allProjects) {
		end = len(allProjects)
	}

	if offset >= len(allProjects) {
		return []models.Project{}, dto.NewPaginationMeta(pagination, total), nil
	}

	paginatedProjects := allProjects[offset:end]

	meta := dto.NewPaginationMeta(pagination, total)

	return paginatedProjects, meta, nil
}

func DeleteProject(userID, projectID primitive.ObjectID) error {
	ctx := context.Background()
	_, err := db.Projects.DeleteOne(ctx, bson.M{
		"_id":     projectID,
		"user_id": userID,
	})
	return err
}
