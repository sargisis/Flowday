package auth

import (
	appErrors "flowday/internal/errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

func RegisterHandler(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := Register(req.Name, req.Email, req.Password)
	if err != nil {
		if err == appErrors.ErrUserExists {
			c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":    user.ID,
		"name":  user.Name,
		"email": user.Email,
	})
}

func LoginHandler(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accessToken, refreshToken, err := Login(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Set Refresh Token in HTTP-Only Cookie
	// Path: /api/auth/refresh (limit scope)
	// MaxAge: 7 days
	c.SetCookie("refresh_token", refreshToken, int((time.Hour * 24 * 7).Seconds()), "/api/v1/auth", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"token": accessToken,
	})
}

func ForgotPasswordHandler(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := RequestPasswordReset(req.Email); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "If email exists, code sent"})
}

func ResetPasswordHandler(c *gin.Context) {
	var req struct {
		Email       string `json:"email" binding:"required,email"`
		Code        string `json:"code" binding:"required,len=6"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ResetPassword(req.Email, req.Code, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password updated successfully"})
}

func GetMeHandler(c *gin.Context) {
	val, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID := val.(primitive.ObjectID)
	user, err := GetUserByID(userID.Hex())
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if user.Level == 0 {
		user.Level = 1
	}

	c.JSON(http.StatusOK, gin.H{
		"id":             user.ID,
		"name":           user.Name,
		"email":          user.Email,
		"bio":            user.Bio,
		"avatar_url":     user.AvatarURL,
		"workspace_name": user.WorkspaceName,
		"status":         user.Status,
		"velocity":       user.Velocity,
		"xp":             user.XP,
		"level":          user.Level,
	})
}

func RefreshHandler(c *gin.Context) {
	// Get refresh token from cookie
	cookie, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token required"})
		return
	}

	accessToken, err := Refresh(cookie)
	if err != nil {
		// Clear invalid cookie
		c.SetCookie("refresh_token", "", -1, "/api/v1/auth", "", false, true)
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": accessToken,
	})
}
