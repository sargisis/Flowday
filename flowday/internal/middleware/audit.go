package middleware

import (
	"context"
	"time"

	"flowday/internal/db"
	"flowday/internal/logger"
	"flowday/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AuditLogMiddleware logs important actions to audit_logs collection
// It should be placed after AuthMiddleware to have access to user_id
func AuditLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip audit logging for certain paths
		path := c.Request.URL.Path
		skipPaths := []string{
			"/health",
			"/ready",
			"/live",
			"/metrics",
			"/ws/connect",
		}
		for _, skipPath := range skipPaths {
			if path == skipPath {
				c.Next()
				return
			}
		}

		// Get user ID if authenticated
		userID, exists := c.Get("user_id")
		if !exists {
			c.Next()
			return
		}

		// Start timer for request duration
		start := time.Now()

		// Process request
		c.Next()

		// Log only important actions (POST, PUT, PATCH, DELETE)
		method := c.Request.Method
		if method != "POST" && method != "PUT" && method != "PATCH" && method != "DELETE" {
			return
		}

		// Determine action and resource from path
		action, resource, resourceID := parseAuditInfo(c)

		// Skip if not an important action
		if action == "" {
			return
		}

		// Get status code
		statusCode := c.Writer.Status()
		success := statusCode >= 200 && statusCode < 400

		// Create audit log entry
		auditLog := models.AuditLog{
			ID:        primitive.NewObjectID(),
			UserID:    userID.(primitive.ObjectID),
			Action:    action,
			Resource:  resource,
			ResourceID: resourceID,
			IP:        c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
			Timestamp: time.Now(),
			Success:   success,
			Details: map[string]interface{}{
				"method":      method,
				"path":        path,
				"status_code": statusCode,
				"duration_ms": time.Since(start).Milliseconds(),
			},
		}

		// Add error if failed
		if !success {
			// Try to get error from response
			if err, exists := c.Get("error"); exists {
				if errStr, ok := err.(string); ok {
					auditLog.Error = errStr
				}
			}
		}

		// Save audit log asynchronously (don't block request)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			_, err := db.AuditLogs.InsertOne(ctx, auditLog)
			if err != nil {
				logger.Log.WithError(err).Warn("Failed to save audit log")
			}
		}()
	}
}

// parseAuditInfo extracts action, resource, and resource ID from the request
func parseAuditInfo(c *gin.Context) (action, resource string, resourceID *primitive.ObjectID) {
	path := c.Request.URL.Path
	method := c.Request.Method

	// Parse /api/v1/tasks/:id patterns
	if len(path) > 12 && path[:12] == "/api/v1/tasks" {
		resource = models.ResourceTypeTask
		
		// Extract task ID from path
		if id := c.Param("id"); id != "" {
			if objID, err := primitive.ObjectIDFromHex(id); err == nil {
				resourceID = &objID
			}
		}

		switch method {
		case "POST":
			if path == "/api/v1/tasks" {
				action = models.AuditActionTaskCreated
			} else if path[len(path)-12:] == "/bulk-delete" {
				action = models.AuditActionTaskBulkDelete
			} else if path[len(path)-19:] == "/bulk-update-status" {
				action = models.AuditActionTaskBulkUpdate
			} else if path[len(path)-21:] == "/bulk-update-priority" {
				action = models.AuditActionTaskBulkUpdate
			}
		case "PATCH":
			action = models.AuditActionTaskUpdated
		case "DELETE":
			action = models.AuditActionTaskDeleted
		}
		return
	}

	// Parse /api/v1/projects/:id patterns
	if len(path) > 15 && path[:15] == "/api/v1/projects" {
		resource = models.ResourceTypeProject
		
		if id := c.Param("id"); id != "" {
			if objID, err := primitive.ObjectIDFromHex(id); err == nil {
				resourceID = &objID
			}
		}

		switch method {
		case "POST":
			action = models.AuditActionProjectCreated
		case "PATCH", "PUT":
			action = models.AuditActionProjectUpdated
		case "DELETE":
			action = models.AuditActionProjectDeleted
		}
		return
	}

	// Parse /api/v1/projects/:id/members/:userID patterns
	if len(path) > 23 && path[:23] == "/api/v1/projects" && len(path) > 30 && path[len(path)-8:] == "/members" {
		resource = models.ResourceTypeMember
		
		if id := c.Param("userID"); id != "" {
			if objID, err := primitive.ObjectIDFromHex(id); err == nil {
				resourceID = &objID
			}
		}

		switch method {
		case "POST", "PUT":
			action = models.AuditActionMemberAdded
		case "DELETE":
			action = models.AuditActionMemberRemoved
		}
		return
	}

	// Parse auth endpoints
	if len(path) > 10 && path[:10] == "/api/v1/auth" {
		resource = models.ResourceTypeUser
		
		switch path {
		case "/api/v1/auth/login":
			action = models.AuditActionUserLogin
		case "/api/v1/auth/logout":
			action = models.AuditActionUserLogout
		case "/api/v1/auth/register":
			action = models.AuditActionUserRegistered
		case "/api/v1/auth/reset-password":
			action = models.AuditActionPasswordChanged
		}
		return
	}

	return "", "", nil
}
