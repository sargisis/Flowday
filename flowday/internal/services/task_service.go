package services

import (
	"context"
	"errors"
	"log"
	"regexp"
	"strings"
	"time"

	"flowday/internal/db"
	"flowday/internal/dto"
	"flowday/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func CreateTask(userID primitive.ObjectID, task *models.Task) error {
	ctx := context.Background()

	// Verify access
	if err := verifyProjectAccess(ctx, userID, task.ProjectID); err != nil {
		return err
	}

	task.ID = primitive.NewObjectID()
	task.ID = primitive.NewObjectID()
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()

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

// GetTasksByProjectPaginated returns paginated tasks for a project
func GetTasksByProjectPaginated(userID, projectID primitive.ObjectID, pagination dto.PaginationQuery) ([]models.Task, dto.PaginationMeta, error) {
	ctx := context.Background()

	// Validate and set defaults
	pagination.ValidateAndSetDefaults()

	// Verify project exists and user has access (owner OR accepted member)
	var project models.Project
	err := db.Projects.FindOne(ctx, bson.M{"_id": projectID}).Decode(&project)
	if err != nil {
		return nil, dto.PaginationMeta{}, errors.New("project not found")
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
			return nil, dto.PaginationMeta{}, errors.New("access denied")
		}
	}

	filter := bson.M{"project_id": projectID}

	// Get total count
	total, err := db.Tasks.CountDocuments(ctx, filter)
	if err != nil {
		return nil, dto.PaginationMeta{}, err
	}

	// Build sort order
	sortOrder := 1 // asc
	if pagination.Order == "desc" {
		sortOrder = -1 // desc
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

	cursor, err := db.Tasks.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, dto.PaginationMeta{}, err
	}
	defer cursor.Close(ctx)

	var tasks []models.Task
	if err = cursor.All(ctx, &tasks); err != nil {
		return nil, dto.PaginationMeta{}, err
	}

	meta := dto.NewPaginationMeta(pagination, total)

	return tasks, meta, nil
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
	updates["updated_at"] = time.Now()
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

func BulkDeleteTasks(userID primitive.ObjectID, taskIDs []primitive.ObjectID) error {
	for _, id := range taskIDs {
		// We ignore "task not found" errors to ensure we delete as many as possible
		// or if it was already deleted.
		if err := DeleteTask(userID, id); err != nil {
			if err.Error() != "task not found" {
				return err
			}
		}
	}
	return nil
}

func BulkUpdateTasksStatus(userID primitive.ObjectID, taskIDs []primitive.ObjectID, status string) (int, error) {
	ctx := context.Background()
	updatedCount := 0

	// First, verify access to all tasks and collect their project IDs
	var tasks []models.Task
	cursor, err := db.Tasks.Find(ctx, bson.M{"_id": bson.M{"$in": taskIDs}})
	if err != nil {
		return 0, err
	}
	if err = cursor.All(ctx, &tasks); err != nil {
		return 0, err
	}

	// Verify access to all tasks
	for _, task := range tasks {
		if err := verifyProjectAccess(ctx, userID, task.ProjectID); err != nil {
			return 0, errors.New("access denied to task: " + task.ID.Hex())
		}
	}

	// Update all tasks
	updateResult, err := db.Tasks.UpdateMany(ctx,
		bson.M{"_id": bson.M{"$in": taskIDs}},
		bson.M{
			"$set": bson.M{
				"status":     status,
				"updated_at": time.Now(),
			},
		},
	)
	if err != nil {
		return 0, err
	}

	updatedCount = int(updateResult.ModifiedCount)

	// Award XP for completed tasks if status is "done"
	if strings.ToLower(status) == "done" {
		// Get tasks that were updated from non-done to done
		var updatedTasks []models.Task
		cursor, _ := db.Tasks.Find(ctx, bson.M{"_id": bson.M{"$in": taskIDs}})
		cursor.All(ctx, &updatedTasks)

		// Award XP for each task that was completed
		xpToAdd := 10 * updatedCount
		if xpToAdd > 0 {
			_, err = db.Users.UpdateOne(ctx,
				bson.M{"_id": userID},
				bson.M{
					"$inc": bson.M{"xp": xpToAdd},
				},
			)
			if err == nil {
				// Check for level up
				var updatedUser models.User
				if err := db.Users.FindOne(ctx, bson.M{"_id": userID}).Decode(&updatedUser); err == nil {
					if updatedUser.Level == 0 {
						updatedUser.Level = 1
						db.Users.UpdateOne(ctx,
							bson.M{"_id": userID},
							bson.M{"$set": bson.M{"level": 1}},
						)
					}
					newLevel := (updatedUser.XP / 100) + 1
					if newLevel > updatedUser.Level {
						db.Users.UpdateOne(ctx,
							bson.M{"_id": userID},
							bson.M{"$set": bson.M{"level": newLevel}},
						)
					}
				}
			}
		}
	}

	return updatedCount, nil
}

func BulkUpdateTasksPriority(userID primitive.ObjectID, taskIDs []primitive.ObjectID, priority string) (int, error) {
	ctx := context.Background()

	// First, verify access to all tasks
	var tasks []models.Task
	cursor, err := db.Tasks.Find(ctx, bson.M{"_id": bson.M{"$in": taskIDs}})
	if err != nil {
		return 0, err
	}
	if err = cursor.All(ctx, &tasks); err != nil {
		return 0, err
	}

	// Verify access to all tasks
	for _, task := range tasks {
		if err := verifyProjectAccess(ctx, userID, task.ProjectID); err != nil {
			return 0, errors.New("access denied to task: " + task.ID.Hex())
		}
	}

	// Update all tasks
	updateResult, err := db.Tasks.UpdateMany(ctx,
		bson.M{"_id": bson.M{"$in": taskIDs}},
		bson.M{
			"$set": bson.M{
				"priority":   priority,
				"updated_at": time.Now(),
			},
		},
	)
	if err != nil {
		return 0, err
	}

	return int(updateResult.ModifiedCount), nil
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

func GetAllTasks(userID primitive.ObjectID) ([]models.Task, error) {
	ctx := context.Background()

	// Get user's projects (User owns these OR is a member)
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

// GetUserTasks is an alias for GetAllTasks, used by the background worker
func GetUserTasks(userID primitive.ObjectID) ([]models.Task, error) {
	return GetAllTasks(userID)
}

// SearchTasks searches and filters tasks across all user's projects
func SearchTasks(userID primitive.ObjectID, searchQuery dto.SearchTasksQuery) ([]models.Task, dto.PaginationMeta, error) {
	ctx := context.Background()

	// Validate and set defaults
	searchQuery.ValidateAndSetDefaults()

	// Get user's projects (User owns these OR is a member)
	projects, err := GetProjects(userID)
	if err != nil {
		return nil, dto.PaginationMeta{}, err
	}

	projectIDs := make([]primitive.ObjectID, len(projects))
	for i, p := range projects {
		projectIDs[i] = p.ID
	}

	// Build filter
	filter := bson.M{
		"project_id": bson.M{"$in": projectIDs},
	}

	// Filter by project_id if specified
	if searchQuery.ProjectID != "" {
		projectID, err := primitive.ObjectIDFromHex(searchQuery.ProjectID)
		if err == nil {
			// Verify user has access to this project
			if err := verifyProjectAccess(ctx, userID, projectID); err != nil {
				return nil, dto.PaginationMeta{}, errors.New("access denied")
			}
			filter["project_id"] = projectID
		}
	}

	// Filter by status
	if searchQuery.Status != "" {
		filter["status"] = searchQuery.Status
	}

	// Filter by priority
	if searchQuery.Priority != "" {
		filter["priority"] = searchQuery.Priority
	}

	// Filter by due date presence
	if searchQuery.HasDueDate != nil {
		if *searchQuery.HasDueDate {
			filter["due_date"] = bson.M{"$ne": nil}
		} else {
			filter["due_date"] = nil
		}
	}

	// Full-text search in title and description
	if searchQuery.Query != "" {
		filter["$or"] = []bson.M{
			{"title": bson.M{"$regex": searchQuery.Query, "$options": "i"}},
			{"description": bson.M{"$regex": searchQuery.Query, "$options": "i"}},
		}
	}

	// Get total count
	total, err := db.Tasks.CountDocuments(ctx, filter)
	if err != nil {
		return nil, dto.PaginationMeta{}, err
	}

	// Build sort order
	sortOrder := 1 // asc
	if searchQuery.Order == "desc" {
		sortOrder = -1 // desc
	}
	sortField := searchQuery.Sort
	if sortField == "" {
		sortField = "created_at"
	}

	// Build find options
	findOptions := options.Find().
		SetLimit(int64(searchQuery.Limit)).
		SetSkip(int64(searchQuery.GetOffset())).
		SetSort(bson.M{sortField: sortOrder})

	cursor, err := db.Tasks.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, dto.PaginationMeta{}, err
	}
	defer cursor.Close(ctx)

	var tasks []models.Task
	if err = cursor.All(ctx, &tasks); err != nil {
		return nil, dto.PaginationMeta{}, err
	}

	meta := dto.NewPaginationMeta(searchQuery.PaginationQuery, total)

	return tasks, meta, nil
}

// AddTaskAttachment adds an attachment to a task
func AddTaskAttachment(userID, taskID primitive.ObjectID, attachment models.Attachment) error {
	ctx := context.Background()

	// Verify task access
	task, err := GetTask(userID, taskID)
	if err != nil {
		return err
	}

	// Initialize attachments array if nil
	if task.Attachments == nil {
		task.Attachments = []models.Attachment{}
	}

	// Add attachment to array
	task.Attachments = append(task.Attachments, attachment)

	// Update task
	update := bson.M{
		"$set": bson.M{
			"attachments": task.Attachments,
			"updated_at":  time.Now(),
		},
	}

	_, err = db.Tasks.UpdateOne(ctx, bson.M{"_id": taskID}, update)
	return err
}

// GetTaskAttachments returns all attachments for a task
func GetTaskAttachments(userID, taskID primitive.ObjectID) ([]models.Attachment, error) {
	// Verify task access
	task, err := GetTask(userID, taskID)
	if err != nil {
		return nil, err
	}

	if task.Attachments == nil {
		return []models.Attachment{}, nil
	}

	return task.Attachments, nil
}

// DeleteTaskAttachment removes an attachment from a task
func DeleteTaskAttachment(userID, taskID primitive.ObjectID, attachmentID string) (*models.Attachment, error) {
	ctx := context.Background()

	// Verify task access
	task, err := GetTask(userID, taskID)
	if err != nil {
		return nil, err
	}

	// Find attachment to delete
	var attachmentToDelete *models.Attachment
	var updatedAttachments []models.Attachment

	for _, att := range task.Attachments {
		if att.ID == attachmentID {
			attachmentToDelete = &att
		} else {
			updatedAttachments = append(updatedAttachments, att)
		}
	}

	if attachmentToDelete == nil {
		return nil, errors.New("attachment not found")
	}

	// Update task
	update := bson.M{
		"$set": bson.M{
			"attachments": updatedAttachments,
			"updated_at":  time.Now(),
		},
	}

	_, err = db.Tasks.UpdateOne(ctx, bson.M{"_id": taskID}, update)
	if err != nil {
		return nil, err
	}

	return attachmentToDelete, nil
}

// ProcessTaskMentions processes @mentions in a task description and sends notifications
func ProcessTaskMentions(taskID primitive.ObjectID, description string, authorID primitive.ObjectID) {
	ctx := context.Background()

	// Get task to find project
	var task models.Task
	err := db.Tasks.FindOne(ctx, bson.M{"_id": taskID}).Decode(&task)
	if err != nil {
		return // Task not found, skip processing
	}

	// Get project
	var project models.Project
	err = db.Projects.FindOne(ctx, bson.M{"_id": task.ProjectID}).Decode(&project)
	if err != nil {
		return // Project not found, skip processing
	}

	// Extract mentions from description (format: @username)
	mentionRegex := regexp.MustCompile(`@(\w+)`)
	matches := mentionRegex.FindAllStringSubmatch(description, -1)
	if len(matches) == 0 {
		return // No mentions found
	}

	// Get all project members (owner + accepted members)
	members := []models.ProjectMember{}
	
	// Add owner as a member
	ownerMember := models.ProjectMember{
		UserID: project.UserID,
	}
	var owner models.User
	if err := db.Users.FindOne(ctx, bson.M{"_id": project.UserID}).Decode(&owner); err == nil {
		ownerMember.User = &owner
		members = append(members, ownerMember)
	}

	// Get accepted project members
	cursor, err := db.ProjectMembers.Find(ctx, bson.M{
		"project_id": task.ProjectID,
		"status":     "accepted",
	})
	if err == nil {
		defer cursor.Close(ctx)
		var projectMembers []models.ProjectMember
		cursor.All(ctx, &projectMembers)
		
		// Populate user data for each member
		for _, m := range projectMembers {
			var user models.User
			if err := db.Users.FindOne(ctx, bson.M{"_id": m.UserID}).Decode(&user); err == nil {
				m.User = &user
				members = append(members, m)
			}
		}
	}

	// Create a map of username/email to user ID for quick lookup
	userMap := make(map[string]primitive.ObjectID)
	for _, member := range members {
		if member.User != nil {
			// Map by name (case-insensitive)
			if member.User.Name != "" {
				userMap[strings.ToLower(member.User.Name)] = member.User.ID
			}
			// Map by email (case-insensitive)
			if member.User.Email != "" {
				userMap[strings.ToLower(member.User.Email)] = member.User.ID
			}
		}
	}

	// Process each mention and send notifications
	notifiedUsers := make(map[primitive.ObjectID]bool)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		mentionedName := strings.ToLower(match[1])
		
		// Find user by name or email
		mentionedUserID, found := userMap[mentionedName]
		if !found {
			continue // User not found in project members
		}

		// Don't notify the author
		if mentionedUserID == authorID {
			continue
		}

		// Don't notify the same user twice
		if notifiedUsers[mentionedUserID] {
			continue
		}

		// Send notification
		_, _ = CreateNotification(
			mentionedUserID,
			&taskID,
			"You were mentioned in a task",
			"You were mentioned in task: "+task.Title,
			models.NotificationInfo,
			"",
		)
		notifiedUsers[mentionedUserID] = true
	}
}

// AddTaskDependency adds a dependency to a task
func AddTaskDependency(userID, taskID, dependsOnTaskID primitive.ObjectID) error {
	ctx := context.Background()

	// Verify task access
	_, err := GetTask(userID, taskID)
	if err != nil {
		return err
	}

	// Verify depends on task exists and user has access
	_, err = GetTask(userID, dependsOnTaskID)
	if err != nil {
		return errors.New("depends on task not found or access denied")
	}

	// Prevent circular dependency
	if taskID == dependsOnTaskID {
		return errors.New("task cannot depend on itself")
	}

	// Get current task
	var task models.Task
	err = db.Tasks.FindOne(ctx, bson.M{"_id": taskID}).Decode(&task)
	if err != nil {
		return errors.New("task not found")
	}

	// Check if dependency already exists
	for _, dep := range task.DependsOn {
		if dep == dependsOnTaskID {
			return errors.New("dependency already exists")
		}
	}

	// Add dependency
	task.DependsOn = append(task.DependsOn, dependsOnTaskID)

	// Update task
	_, err = db.Tasks.UpdateOne(ctx, bson.M{"_id": taskID}, bson.M{
		"$set": bson.M{
			"depends_on": task.DependsOn,
			"updated_at": time.Now(),
		},
	})

	// Also update the depends on task to add this task to its blocks array
	var dependsOnTask models.Task
	err = db.Tasks.FindOne(ctx, bson.M{"_id": dependsOnTaskID}).Decode(&dependsOnTask)
	if err == nil {
		dependsOnTask.Blocks = append(dependsOnTask.Blocks, taskID)
		db.Tasks.UpdateOne(ctx, bson.M{"_id": dependsOnTaskID}, bson.M{
			"$set": bson.M{
				"blocks":     dependsOnTask.Blocks,
				"updated_at": time.Now(),
			},
		})
	}

	return err
}

// RemoveTaskDependency removes a dependency from a task
func RemoveTaskDependency(userID, taskID, dependsOnTaskID primitive.ObjectID) error {
	ctx := context.Background()

	// Verify task access
	_, err := GetTask(userID, taskID)
	if err != nil {
		return err
	}

	// Get current task
	var task models.Task
	err = db.Tasks.FindOne(ctx, bson.M{"_id": taskID}).Decode(&task)
	if err != nil {
		return errors.New("task not found")
	}

	// Remove dependency
	newDependsOn := []primitive.ObjectID{}
	for _, dep := range task.DependsOn {
		if dep != dependsOnTaskID {
			newDependsOn = append(newDependsOn, dep)
		}
	}

	// Update task
	_, err = db.Tasks.UpdateOne(ctx, bson.M{"_id": taskID}, bson.M{
		"$set": bson.M{
			"depends_on": newDependsOn,
			"updated_at": time.Now(),
		},
	})

	// Also update the depends on task to remove this task from its blocks array
	var dependsOnTask models.Task
	err = db.Tasks.FindOne(ctx, bson.M{"_id": dependsOnTaskID}).Decode(&dependsOnTask)
	if err == nil {
		newBlocks := []primitive.ObjectID{}
		for _, block := range dependsOnTask.Blocks {
			if block != taskID {
				newBlocks = append(newBlocks, block)
			}
		}
		db.Tasks.UpdateOne(ctx, bson.M{"_id": dependsOnTaskID}, bson.M{
			"$set": bson.M{
				"blocks":     newBlocks,
				"updated_at": time.Now(),
			},
		})
	}

	return err
}

// GetTaskDependencies returns all dependencies for a task
func GetTaskDependencies(userID, taskID primitive.ObjectID) (map[string]interface{}, error) {
	// Verify task access
	task, err := GetTask(userID, taskID)
	if err != nil {
		return nil, err
	}

	// Get depends on tasks
	dependsOnTasks := []models.Task{}
	if len(task.DependsOn) > 0 {
		ctx := context.Background()
		cursor, err := db.Tasks.Find(ctx, bson.M{
			"_id": bson.M{"$in": task.DependsOn},
		})
		if err == nil {
			defer cursor.Close(ctx)
			cursor.All(ctx, &dependsOnTasks)
		}
	}

	// Get blocks tasks
	blocksTasks := []models.Task{}
	if len(task.Blocks) > 0 {
		ctx := context.Background()
		cursor, err := db.Tasks.Find(ctx, bson.M{
			"_id": bson.M{"$in": task.Blocks},
		})
		if err == nil {
			defer cursor.Close(ctx)
			cursor.All(ctx, &blocksTasks)
		}
	}

	// Get blocked by tasks (tasks that have this task in their depends_on)
	ctx := context.Background()
	cursor, err := db.Tasks.Find(ctx, bson.M{
		"depends_on": taskID,
	})
	blockedByTasks := []models.Task{}
	if err == nil {
		defer cursor.Close(ctx)
		cursor.All(ctx, &blockedByTasks)
	}

	return map[string]interface{}{
		"depends_on": dependsOnTasks,
		"blocks":      blocksTasks,
		"blocked_by":  blockedByTasks,
	}, nil
}

// CreateTaskTemplate creates a new task template
func CreateTaskTemplate(userID primitive.ObjectID, req dto.CreateTaskTemplateRequest) (*models.TaskTemplate, error) {
	ctx := context.Background()

	template := models.TaskTemplate{
		ID:             primitive.NewObjectID(),
		UserID:         userID,
		Name:           req.Name,
		Description:    req.Description,
		Title:          req.Title,
		TaskDescription: req.TaskDescription,
		Priority:       req.Priority,
		EstimatedHours: req.EstimatedHours,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if req.ProjectID != "" {
		projectID, err := primitive.ObjectIDFromHex(req.ProjectID)
		if err == nil {
			template.ProjectID = &projectID
		}
	}

	// Convert subtasks
	if len(req.Subtasks) > 0 {
		template.Subtasks = make([]models.Subtask, len(req.Subtasks))
		for i, st := range req.Subtasks {
			template.Subtasks[i] = models.Subtask{
				ID:        st.ID,
				Title:     st.Title,
				Completed: st.Completed,
			}
		}
	}

	_, err := db.TaskTemplates.InsertOne(ctx, template)
	if err != nil {
		return nil, err
	}

	return &template, nil
}

// GetTaskTemplates returns all task templates for a user
func GetTaskTemplates(userID primitive.ObjectID) ([]models.TaskTemplate, error) {
	ctx := context.Background()

	cursor, err := db.TaskTemplates.Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var templates []models.TaskTemplate
	if err = cursor.All(ctx, &templates); err != nil {
		return nil, err
	}

	return templates, nil
}

// GetTaskTemplate returns a specific task template
func GetTaskTemplate(userID, templateID primitive.ObjectID) (*models.TaskTemplate, error) {
	ctx := context.Background()

	var template models.TaskTemplate
	err := db.TaskTemplates.FindOne(ctx, bson.M{
		"_id":     templateID,
		"user_id": userID,
	}).Decode(&template)
	if err != nil {
		return nil, errors.New("template not found")
	}

	return &template, nil
}

// UpdateTaskTemplate updates a task template
func UpdateTaskTemplate(userID, templateID primitive.ObjectID, req dto.UpdateTaskTemplateRequest) (*models.TaskTemplate, error) {
	ctx := context.Background()

	update := bson.M{}
	if req.Name != "" {
		update["name"] = req.Name
	}
	if req.Description != "" {
		update["description"] = req.Description
	}
	if req.Title != "" {
		update["title"] = req.Title
	}
	if req.TaskDescription != "" {
		update["task_description"] = req.TaskDescription
	}
	if req.Priority != "" {
		update["priority"] = req.Priority
	}
	if req.EstimatedHours != nil {
		update["estimated_hours"] = *req.EstimatedHours
	}
	if req.ProjectID != "" {
		projectID, err := primitive.ObjectIDFromHex(req.ProjectID)
		if err == nil {
			update["project_id"] = projectID
		}
	}
	if len(req.Subtasks) > 0 {
		subtasks := make([]models.Subtask, len(req.Subtasks))
		for i, st := range req.Subtasks {
			subtasks[i] = models.Subtask{
				ID:        st.ID,
				Title:     st.Title,
				Completed: st.Completed,
			}
		}
		update["subtasks"] = subtasks
	}
	update["updated_at"] = time.Now()

	result := db.TaskTemplates.FindOneAndUpdate(
		ctx,
		bson.M{"_id": templateID, "user_id": userID},
		bson.M{"$set": update},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	var template models.TaskTemplate
	if err := result.Decode(&template); err != nil {
		return nil, errors.New("template not found")
	}

	return &template, nil
}

// DeleteTaskTemplate deletes a task template
func DeleteTaskTemplate(userID, templateID primitive.ObjectID) error {
	ctx := context.Background()

	result, err := db.TaskTemplates.DeleteOne(ctx, bson.M{
		"_id":     templateID,
		"user_id": userID,
	})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return errors.New("template not found")
	}

	return nil
}

// CreateTaskFromTemplate creates a task from a template
func CreateTaskFromTemplate(userID, templateID primitive.ObjectID, req dto.CreateTaskFromTemplateRequest) (*models.Task, error) {
	ctx := context.Background()

	// Get the template
	template, err := GetTaskTemplate(userID, templateID)
	if err != nil {
		return nil, err
	}

	// Parse project ID
	projectID, err := primitive.ObjectIDFromHex(req.ProjectID)
	if err != nil {
		return nil, errors.New("invalid project_id format")
	}

	// Create task from template
	task := models.Task{
		ID:          primitive.NewObjectID(),
		Title:       template.Title,
		Description: template.TaskDescription,
		Priority:    template.Priority,
		Status:      "todo",
		ProjectID:   projectID,
		Subtasks:    template.Subtasks,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if template.EstimatedHours != nil {
		task.EstimatedHours = template.EstimatedHours
	}

	_, err = db.Tasks.InsertOne(ctx, task)
	if err != nil {
		return nil, err
	}

	return &task, nil
}
