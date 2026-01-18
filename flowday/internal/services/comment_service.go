package services

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"flowday/internal/db"
	"flowday/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CreateComment creates a new comment on a task
func CreateComment(userID, taskID primitive.ObjectID, content string) (*models.Comment, error) {
	ctx := context.Background()

	// Verify task exists and user has access
	task, err := GetTask(userID, taskID)
	if err != nil {
		return nil, errors.New("task not found or access denied")
	}

	// Create comment
	comment := models.Comment{
		ID:        primitive.NewObjectID(),
		TaskID:    taskID,
		UserID:    userID,
		Content:   content,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err = db.Comments.InsertOne(ctx, comment)
	if err != nil {
		return nil, err
	}

	// Log activity
	LogActivity(userID, models.ActivityTaskCommented, "Commented on task: "+task.Title, map[string]string{
		"task_id":    taskID.Hex(),
		"comment_id": comment.ID.Hex(),
	})

	// Get project to find owner
	var project models.Project
	err = db.Projects.FindOne(ctx, bson.M{"_id": task.ProjectID}).Decode(&project)
	if err == nil {
		// Send notification to project owner if not the commenter
		if project.UserID != userID {
			_, _ = CreateNotification(project.UserID, &taskID, "New comment on task", "New comment on task: "+task.Title, models.NotificationInfo, "")
		}

		// Get other commenters on this task (excluding current user)
		cursor, err := db.Comments.Find(ctx, bson.M{
			"task_id": taskID,
			"user_id": bson.M{"$ne": userID},
		})
		if err == nil {
			defer cursor.Close(ctx)
			commenterIDs := make(map[primitive.ObjectID]bool)
			var otherComment models.Comment
			for cursor.Next(ctx) {
				if err := cursor.Decode(&otherComment); err == nil {
					commenterIDs[otherComment.UserID] = true
				}
			}

			// Send notification to other commenters
			for commenterID := range commenterIDs {
				if commenterID != project.UserID { // Don't notify twice if owner already commented
					_, _ = CreateNotification(commenterID, &taskID, "New comment on task", "New comment on task: "+task.Title, models.NotificationInfo, "")
				}
			}
		}
	}

	return &comment, nil
}

// GetTaskComments retrieves all comments for a task
func GetTaskComments(userID, taskID primitive.ObjectID) ([]models.Comment, error) {
	ctx := context.Background()

	// Verify task exists and user has access
	_, err := GetTask(userID, taskID)
	if err != nil {
		return nil, errors.New("task not found or access denied")
	}

	cursor, err := db.Comments.Find(ctx, bson.M{
		"task_id": taskID,
	}, options.Find().SetSort(bson.M{"created_at": 1}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var comments []models.Comment
	if err = cursor.All(ctx, &comments); err != nil {
		return nil, err
	}

	return comments, nil
}

// UpdateComment updates a comment (only by the author)
func UpdateComment(userID, commentID primitive.ObjectID, content string) error {
	ctx := context.Background()

	// Get comment
	var comment models.Comment
	err := db.Comments.FindOne(ctx, bson.M{"_id": commentID}).Decode(&comment)
	if err != nil {
		return errors.New("comment not found")
	}

	// Verify user is the author
	if comment.UserID != userID {
		return errors.New("access denied: only comment author can update")
	}

	// Verify task access
	_, err = GetTask(userID, comment.TaskID)
	if err != nil {
		return errors.New("task not found or access denied")
	}

	// Update comment
	_, err = db.Comments.UpdateOne(ctx,
		bson.M{"_id": commentID},
		bson.M{
			"$set": bson.M{
				"content":   content,
				"updated_at": time.Now(),
			},
		},
	)

	return err
}

// DeleteComment deletes a comment (only by the author or project owner)
func DeleteComment(userID, commentID primitive.ObjectID) error {
	ctx := context.Background()

	// Get comment
	var comment models.Comment
	err := db.Comments.FindOne(ctx, bson.M{"_id": commentID}).Decode(&comment)
	if err != nil {
		return errors.New("comment not found")
	}

	// Get task to check project ownership
	task, err := GetTask(userID, comment.TaskID)
	if err != nil {
		return errors.New("task not found or access denied")
	}

	// Get project to check ownership
	var project models.Project
	err = db.Projects.FindOne(ctx, bson.M{"_id": task.ProjectID}).Decode(&project)
	if err != nil {
		return errors.New("project not found")
	}

	// Verify user is the comment author OR project owner
	isAuthor := comment.UserID == userID
	isProjectOwner := project.UserID == userID

	if !isAuthor && !isProjectOwner {
		return errors.New("access denied: only comment author or project owner can delete")
	}

	// Delete comment
	_, err = db.Comments.DeleteOne(ctx, bson.M{"_id": commentID})
	return err
}

// ProcessCommentMentions processes @mentions in a comment and sends notifications
func ProcessCommentMentions(commentID primitive.ObjectID, content string, taskID, authorID primitive.ObjectID) {
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

	// Extract mentions from content (format: @username)
	mentionRegex := regexp.MustCompile(`@(\w+)`)
	matches := mentionRegex.FindAllStringSubmatch(content, -1)
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
			"You were mentioned in a comment",
			"You were mentioned in a comment on task: "+task.Title,
			models.NotificationInfo,
			"",
		)
		notifiedUsers[mentionedUserID] = true
	}
}
