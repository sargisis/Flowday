package handlers

import (
	"context"
	"crypto/rand"
	"flowday/internal/auth"
	"flowday/internal/db"
	"flowday/internal/models"
	"flowday/internal/utils"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type UpdateProfileRequest struct {
	Name          string `json:"name"`
	Bio           string `json:"bio"`
	WorkspaceName string `json:"workspace_name"`
}

func UpdateProfile(c *gin.Context) {
	val, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := val.(primitive.ObjectID)

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	update := bson.M{
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	}

	if req.Name != "" {
		update["$set"].(bson.M)["name"] = req.Name
	}
	update["$set"].(bson.M)["bio"] = req.Bio
	update["$set"].(bson.M)["workspace_name"] = req.WorkspaceName

	_, err := db.Users.UpdateOne(context.Background(), bson.M{"_id": userID}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully"})
}

func UploadAvatar(c *gin.Context) {
	val, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := val.(primitive.ObjectID)

	file, err := c.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	// Create uploads directory if not exists
	uploadDir := "uploads"
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.Mkdir(uploadDir, 0755)
	}

	// Generate unique filename
	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%s_%d%s", userID.Hex(), time.Now().Unix(), ext)
	filePath := filepath.Join(uploadDir, filename)

	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	// Update user's avatar URL
	avatarURL := fmt.Sprintf("/api/v1/uploads/%s", filename)
	_, err = db.Users.UpdateOne(context.Background(), bson.M{"_id": userID}, bson.M{
		"$set": bson.M{
			"avatar_url": avatarURL,
			"updated_at": time.Now(),
		},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user avatar"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Avatar uploaded successfully",
		"avatar_url": avatarURL,
	})
}

type RequestEmailChangeRequest struct {
	CurrentPassword string `json:"current_password"`
	NewEmail        string `json:"new_email"`
}

func RequestEmailChange(c *gin.Context) {
	val, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := val.(primitive.ObjectID)

	var req RequestEmailChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. Verify User Password
	var user models.User
	err := db.Users.FindOne(context.Background(), bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if !auth.CheckPassword(user.Password, req.CurrentPassword) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid current password"})
		return
	}

	// 2. Check if new email is in use
	var existingUser models.User
	err = db.Users.FindOne(context.Background(), bson.M{"email": req.NewEmail}).Decode(&existingUser)
	if err != mongo.ErrNoDocuments {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email already in use"})
		return
	}

	// 3. Generate 6-digit code
	codeInt, _ := rand.Int(rand.Reader, big.NewInt(900000))
	code := fmt.Sprintf("%06d", codeInt.Int64()+100000)

	// 4. Save Request
	changeReq := models.EmailChangeRequest{
		ID:        primitive.NewObjectID(),
		UserID:    userID,
		NewEmail:  req.NewEmail,
		Code:      code,
		ExpiresAt: time.Now().Add(15 * time.Minute),
		CreatedAt: time.Now(),
	}

	_, err = db.EmailChangeRequests.InsertOne(context.Background(), changeReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	// 5. Send Email
	err = utils.SendVerificationCode(req.NewEmail, code, "Flowday Email Change Verification", "Confirm Your New Email")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send verification email"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Verification code sent to your new email"})
}

type ConfirmEmailChangeRequest struct {
	NewEmail string `json:"new_email"`
	Code     string `json:"code"`
}

func ConfirmEmailChange(c *gin.Context) {
	val, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := val.(primitive.ObjectID)

	var req ConfirmEmailChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. Find the request
	var changeReq models.EmailChangeRequest
	err := db.EmailChangeRequests.FindOne(context.Background(), bson.M{
		"user_id":    userID,
		"new_email":  req.NewEmail,
		"code":       req.Code,
		"expires_at": bson.M{"$gt": time.Now()},
	}).Decode(&changeReq)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired code"})
		return
	}

	// 2. Update user email
	_, err = db.Users.UpdateOne(context.Background(), bson.M{"_id": userID}, bson.M{
		"$set": bson.M{
			"email":      req.NewEmail,
			"updated_at": time.Now(),
		},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update email"})
		return
	}

	// 3. Delete the request
	db.EmailChangeRequests.DeleteOne(context.Background(), bson.M{"_id": changeReq.ID})

	c.JSON(http.StatusOK, gin.H{"message": "Email updated successfully"})
}

type UpdateStatusRequest struct {
	Status string `json:"status"`
}

func UpdateStatus(c *gin.Context) {
	val, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := val.(primitive.ObjectID)

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := db.Users.UpdateOne(context.Background(), bson.M{"_id": userID}, bson.M{
		"$set": bson.M{
			"status":     req.Status,
			"updated_at": time.Now(),
		},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Status updated successfully"})
}
