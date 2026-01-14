package services

import (
	"context"
	"errors"
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
