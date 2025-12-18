package auth

import (
	appErrors "flowday/internal/errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AuthRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

func RegisterHandler(c *gin.Context) {
	var req AuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := Register(req.Email, req.Password)
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
		"email": user.Email,
	})
}

func LoginHandler(c *gin.Context) {
	var req AuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := Login(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
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
