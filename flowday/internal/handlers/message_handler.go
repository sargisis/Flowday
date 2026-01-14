package handlers

import (
	"flowday/internal/dto"
	"flowday/internal/services"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func SendMessage(c *gin.Context) {
	var req dto.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Content == "" && req.AttachmentURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message content or attachment is required"})
		return
	}

	senderID, _ := c.Get("user_id")
	receiverID, err := primitive.ObjectIDFromHex(req.ReceiverID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid receiver_id format"})
		return
	}

	msg, err := services.SendMessage(senderID.(primitive.ObjectID), receiverID, req.Content, req.AttachmentURL, req.AttachmentType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, msg)
}

func UploadAttachment(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	// ✅ SECURITY: Validate file size (max 10MB for attachments)
	const maxAttachmentSize = 10 * 1024 * 1024 // 10MB
	if file.Size > maxAttachmentSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File size exceeds maximum allowed size (10MB)"})
		return
	}

	// ✅ SECURITY: Sanitize file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	// Remove any path traversal attempts
	ext = strings.TrimPrefix(ext, ".")
	ext = strings.Trim(ext, "/\\")
	
	// Basic extension validation - prevent executable files
	dangerousExts := map[string]bool{
		"exe": true, "bat": true, "cmd": true, "com": true, "pif": true,
		"scr": true, "vbs": true, "js": true, "jar": true, "sh": true,
	}
	if dangerousExts[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File type not allowed"})
		return
	}

	// Create uploads/messages directory if not exists
	uploadDir := filepath.Join("uploads", "messages")
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.MkdirAll(uploadDir, 0755)
	}

	// ✅ SECURITY: Generate safe filename (no user input in path)
	var safeExt string
	if ext != "" {
		safeExt = "." + ext
	}
	filename := fmt.Sprintf("msg_%d%s", time.Now().UnixNano(), safeExt)
	filePath := filepath.Join(uploadDir, filename)

	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	attachmentURL := fmt.Sprintf("/api/v1/uploads/messages/%s", filename)

	// Determine type based on extension
	attachmentType := "file"
	imgExts := map[string]bool{"jpg": true, "jpeg": true, "png": true, "gif": true, "webp": true}
	if imgExts[ext] {
		attachmentType = "image"
	}

	c.JSON(http.StatusOK, gin.H{
		"attachment_url":  attachmentURL,
		"attachment_type": attachmentType,
	})
}

func GetConversations(c *gin.Context) {
	userID, _ := c.Get("user_id")
	summaries, err := services.GetConversationsForUser(userID.(primitive.ObjectID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summaries)
}

func GetChatHistory(c *gin.Context) {
	partnerID, err := primitive.ObjectIDFromHex(c.Param("partnerID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid partner id format"})
		return
	}

	userID, _ := c.Get("user_id")
	messages, err := services.GetMessagesBetweenUsers(userID.(primitive.ObjectID), partnerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, messages)
}
