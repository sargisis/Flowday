package handlers

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// ServePublicFile serves uploaded files (avatars, task attachments, etc.)
func ServePublicFile(c *gin.Context) {
	filePath := c.Param("filepath")

	// Remove leading slash if present
	filePath = strings.TrimPrefix(filePath, "/")

	// Security: Prevent path traversal
	if strings.Contains(filePath, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file path"})
		return
	}

	// Construct full path
	fullPath := filepath.Join("uploads", filePath)

	// Serve file
	c.File(fullPath)
}
