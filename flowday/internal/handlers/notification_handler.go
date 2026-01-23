package handlers

import (
	"fmt"
	"net/http"

	"flowday/internal/dto"
	"flowday/internal/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetNotificationsHandler handles GET /api/v1/notifications
func GetNotificationsHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Parse pagination query
	var pagination dto.PaginationQuery
	if err := c.ShouldBindQuery(&pagination); err != nil {
		// If pagination params are not provided, use defaults
		pagination = dto.PaginationQuery{}
	}

	notifications, meta, err := services.GetNotificationsPaginated(userID.(primitive.ObjectID), pagination)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notifications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": notifications,
		"meta": meta,
	})
}

// MarkNotificationReadHandler handles PATCH /api/v1/notifications/:id/read
func MarkNotificationReadHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	notifIDStr := c.Param("id")
	notifID, err := primitive.ObjectIDFromHex(notifIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	err = services.MarkNotificationRead(notifID, userID.(primitive.ObjectID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification marked as read"})
}

// ✅ ENHANCED: Batch notification handlers

// MarkNotificationsReadHandler handles POST /api/v1/notifications/batch-read
func MarkNotificationsReadHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		NotificationIDs []string `json:"notification_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Convert string IDs to ObjectIDs
	objectIDs := make([]primitive.ObjectID, 0, len(req.NotificationIDs))
	for _, idStr := range req.NotificationIDs {
		oid, err := primitive.ObjectIDFromHex(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID: " + idStr})
			return
		}
		objectIDs = append(objectIDs, oid)
	}

	modifiedCount, err := services.MarkNotificationsRead(objectIDs, userID.(primitive.ObjectID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Notifications marked as read",
		"modified_count": modifiedCount,
	})
}

// MarkAllNotificationsReadHandler handles POST /api/v1/notifications/mark-all-read
func MarkAllNotificationsReadHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	modifiedCount, err := services.MarkAllNotificationsRead(userID.(primitive.ObjectID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "All notifications marked as read",
		"modified_count": modifiedCount,
	})
}

// GetUnreadCountHandler handles GET /api/v1/notifications/unread-count
func GetUnreadCountHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	count, err := services.GetUnreadCount(userID.(primitive.ObjectID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

// DeleteOldNotificationsHandler handles DELETE /api/v1/notifications/old?days=30
func DeleteOldNotificationsHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	daysStr := c.DefaultQuery("days", "30")
	var days int
	if _, err := fmt.Sscanf(daysStr, "%d", &days); err != nil || days < 1 {
		days = 30 // Default to 30 days
	}

	deletedCount, err := services.DeleteOldNotifications(userID.(primitive.ObjectID), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Old notifications deleted",
		"deleted_count": deletedCount,
	})
}
