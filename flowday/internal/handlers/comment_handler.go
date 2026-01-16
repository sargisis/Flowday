package handlers

import (
	"net/http"

	"flowday/internal/dto"
	"flowday/internal/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CreateCommentHandler handles POST /api/v1/tasks/:id/comments
func CreateCommentHandler(c *gin.Context) {
	taskIDStr := c.Param("id")
	taskID, err := primitive.ObjectIDFromHex(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id format"})
		return
	}

	var req dto.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	comment, err := services.CreateComment(userID.(primitive.ObjectID), taskID, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// ✅ NEW: Process mentions in comment
	if req.Content != "" {
		go func() {
			services.ProcessCommentMentions(comment.ID, req.Content, taskID, userID.(primitive.ObjectID))
		}()
	}

	c.JSON(http.StatusCreated, comment)
}

// GetTaskCommentsHandler handles GET /api/v1/tasks/:id/comments
func GetTaskCommentsHandler(c *gin.Context) {
	taskIDStr := c.Param("id")
	taskID, err := primitive.ObjectIDFromHex(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id format"})
		return
	}

	userID, _ := c.Get("user_id")
	comments, err := services.GetTaskComments(userID.(primitive.ObjectID), taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, comments)
}

// UpdateCommentHandler handles PUT /api/v1/comments/:comment_id
func UpdateCommentHandler(c *gin.Context) {
	commentIDStr := c.Param("comment_id")
	commentID, err := primitive.ObjectIDFromHex(commentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid comment_id format"})
		return
	}

	var req dto.UpdateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	err = services.UpdateComment(userID.(primitive.ObjectID), commentID, req.Content)
	if err != nil {
		if err.Error() == "comment not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "access denied: only comment author can update" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "comment updated successfully"})
}

// DeleteCommentHandler handles DELETE /api/v1/comments/:comment_id
func DeleteCommentHandler(c *gin.Context) {
	commentIDStr := c.Param("comment_id")
	commentID, err := primitive.ObjectIDFromHex(commentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid comment_id format"})
		return
	}

	userID, _ := c.Get("user_id")
	err = services.DeleteComment(userID.(primitive.ObjectID), commentID)
	if err != nil {
		if err.Error() == "comment not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "access denied: only comment author or project owner can delete" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "comment deleted successfully"})
}
