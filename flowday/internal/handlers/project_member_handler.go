package handlers

import (
	"log"
	"net/http"

	"flowday/internal/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// InviteMember handles POST /projects/:id/members
func InviteMember(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var params struct {
		ID string `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	projectID, err := primitive.ObjectIDFromHex(params.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id format"})
		return
	}

	var req struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = services.InviteMember(projectID, userID.(primitive.ObjectID), req.Email)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Invitation sent"})
}

// GetProjectMembers handles GET /projects/:id/members
func GetProjectMembers(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var params struct {
		ID string `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	projectID, err := primitive.ObjectIDFromHex(params.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id format"})
		return
	}

	members, err := services.GetProjectMembers(userID.(primitive.ObjectID), projectID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, members)
}

// GetMyInvitations handles GET /invitations
func GetMyInvitations(c *gin.Context) {
	userID, _ := c.Get("user_id")

	invitations, err := services.GetPendingInvitations(userID.(primitive.ObjectID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch invitations"})
		return
	}

	c.JSON(http.StatusOK, invitations)
}

// AcceptInvitation handles POST /invitations/:id/accept
func AcceptInvitation(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var params struct {
		ID string `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	projectID, err := primitive.ObjectIDFromHex(params.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id format"})
		return
	}

	err = services.AcceptInvitation(userID.(primitive.ObjectID), projectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Invitation accepted"})
}

// RejectInvitation handles POST /invitations/:id/reject
func RejectInvitation(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var params struct {
		ID string `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	projectID, err := primitive.ObjectIDFromHex(params.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id format"})
		return
	}

	err = services.RejectInvitation(userID.(primitive.ObjectID), projectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Invitation rejected"})
}

// RemoveMember handles DELETE /projects/:id/members/:userID
func RemoveMember(c *gin.Context) {
	ownerID, _ := c.Get("user_id")
	log.Printf("DEBUG: RemoveMember request by ownerID: %v", ownerID)

	var params struct {
		ProjectID string `uri:"id" binding:"required"`
		MemberID  string `uri:"userID" binding:"required"`
	}
	if err := c.ShouldBindUri(&params); err != nil {
		log.Printf("DEBUG: RemoveMember binding failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ids"})
		return
	}

	log.Printf("DEBUG: RemoveMember params: projectID=%s, memberID=%s", params.ProjectID, params.MemberID)

	pID, err := primitive.ObjectIDFromHex(params.ProjectID)
	if err != nil {
		log.Printf("DEBUG: RemoveMember pID hex error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id format"})
		return
	}

	mID, err := primitive.ObjectIDFromHex(params.MemberID)
	if err != nil {
		log.Printf("DEBUG: RemoveMember mID hex error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid member user id format"})
		return
	}

	err = services.RemoveMember(ownerID.(primitive.ObjectID), pID, mID)
	if err != nil {
		log.Printf("DEBUG: RemoveMember service error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member removed successfully"})
}
