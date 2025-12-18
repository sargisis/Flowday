package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/smtp"
	"os"
	"time"

	"flowday/internal/db"
	"flowday/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// InviteMember creates a pending invitation for a user to join a project
func InviteMember(projectID, ownerID primitive.ObjectID, inviteeEmail string) error {
	ctx := context.Background()

	// Verify owner owns the project
	var project models.Project
	err := db.Projects.FindOne(ctx, bson.M{
		"_id":     projectID,
		"user_id": ownerID,
	}).Decode(&project)
	if err != nil {
		return errors.New("project not found or access denied")
	}

	// Find invitee by email
	var invitee models.User
	err = db.Users.FindOne(ctx, bson.M{"email": inviteeEmail}).Decode(&invitee)
	if err != nil {
		return errors.New("user not found with that email")
	}

	// Check if already a member or invited
	var existing models.ProjectMember
	err = db.ProjectMembers.FindOne(ctx, bson.M{
		"project_id": projectID,
		"user_id":    invitee.ID,
	}).Decode(&existing)
	if err == nil {
		if existing.Status == "accepted" {
			return errors.New("user is already a member")
		}
		return errors.New("invitation already sent")
	}
	if err != mongo.ErrNoDocuments {
		return err
	}

	// Generate unique token
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return err
	}
	token := hex.EncodeToString(b)

	// Create invitation
	member := models.ProjectMember{
		ID:        primitive.NewObjectID(),
		ProjectID: projectID,
		UserID:    invitee.ID,
		Role:      "member",
		Status:    "pending",
		Token:     token,
		InvitedAt: time.Now(),
	}

	_, err = db.ProjectMembers.InsertOne(ctx, member)
	if err != nil {
		return err
	}

	// Send invitation email
	go func() {
		if err := sendInvitationEmail(inviteeEmail, project.Name, projectID, token); err != nil {
			log.Printf("Failed to send invitation email to %s: %v", inviteeEmail, err)
		} else {
			log.Printf("Invitation email sent to %s for project %s with token", inviteeEmail, project.Name)
		}
	}()

	return nil
}

// AcceptInvitation marks an invitation as accepted
func AcceptInvitation(userID, projectID primitive.ObjectID) error {
	ctx := context.Background()

	var member models.ProjectMember
	err := db.ProjectMembers.FindOne(ctx, bson.M{
		"project_id": projectID,
		"user_id":    userID,
		"status":     "pending",
	}).Decode(&member)
	if err != nil {
		return errors.New("invitation not found")
	}

	now := time.Now()
	_, err = db.ProjectMembers.UpdateOne(ctx,
		bson.M{"_id": member.ID},
		bson.M{"$set": bson.M{
			"status":      "accepted",
			"accepted_at": now,
		}},
	)

	return err
}

// GetProjectMembers returns all accepted members of a project (including owner)
func GetProjectMembers(userID, projectID primitive.ObjectID) ([]models.ProjectMember, error) {
	ctx := context.Background()

	// Verify user has access to project
	var project models.Project
	err := db.Projects.FindOne(ctx, bson.M{"_id": projectID}).Decode(&project)
	if err != nil {
		return nil, errors.New("project not found")
	}

	// Check if requesting user is owner or accepted member
	isOwner := project.UserID == userID
	if !isOwner {
		var access models.ProjectMember
		err := db.ProjectMembers.FindOne(ctx, bson.M{
			"project_id": projectID,
			"user_id":    userID,
			"status":     "accepted",
		}).Decode(&access)
		if err != nil {
			return nil, errors.New("access denied")
		}
	}

	// Get all accepted members
	cursor, err := db.ProjectMembers.Find(ctx, bson.M{
		"project_id": projectID,
		"status":     "accepted",
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var members []models.ProjectMember
	if err = cursor.All(ctx, &members); err != nil {
		return nil, err
	}

	// Populate User field for each member
	for i := range members {
		var user models.User
		err := db.Users.FindOne(ctx, bson.M{"_id": members[i].UserID}).Decode(&user)
		if err == nil {
			members[i].User = &user
		}
	}

	// Add owner as first member (if not already in the list)
	var owner models.User
	err = db.Users.FindOne(ctx, bson.M{"_id": project.UserID}).Decode(&owner)
	if err == nil {
		// Check if owner is already in members (shouldn't be, but just in case)
		ownerExists := false
		for _, m := range members {
			if m.UserID == project.UserID {
				ownerExists = true
				break
			}
		}

		if !ownerExists {
			ownerMember := models.ProjectMember{
				ID:         primitive.NewObjectID(),
				ProjectID:  projectID,
				UserID:     owner.ID,
				Role:       "owner",
				Status:     "accepted",
				InvitedAt:  project.CreatedAt,
				AcceptedAt: &project.CreatedAt,
				User:       &owner,
			}
			members = append([]models.ProjectMember{ownerMember}, members...)
		}
	}

	return members, nil
}

// GetPendingInvitations returns all pending invitations for a user
func GetPendingInvitations(userID primitive.ObjectID) ([]models.ProjectMember, error) {
	ctx := context.Background()

	cursor, err := db.ProjectMembers.Find(ctx, bson.M{
		"user_id": userID,
		"status":  "pending",
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var invitations []models.ProjectMember
	if err = cursor.All(ctx, &invitations); err != nil {
		return nil, err
	}

	// Populate Project field for each invitation
	for i := range invitations {
		var project models.Project
		err := db.Projects.FindOne(ctx, bson.M{"_id": invitations[i].ProjectID}).Decode(&project)
		if err == nil {
			invitations[i].Project = &project
		}
	}

	return invitations, nil
}

// RemoveMember removes a member from a project (only for project owner)
func RemoveMember(ownerID, projectID, memberUserID primitive.ObjectID) error {
	ctx := context.Background()

	log.Printf("DEBUG: Service.RemoveMember - ownerID: %v, projectID: %v, memberUserID: %v", ownerID, projectID, memberUserID)

	// Verify requester is the owner of the project
	var project models.Project
	err := db.Projects.FindOne(ctx, bson.M{
		"_id":     projectID,
		"user_id": ownerID,
	}).Decode(&project)
	if err != nil {
		log.Printf("DEBUG: Service.RemoveMember - ownership check failed: %v", err)
		return errors.New("only project owner can remove members")
	}

	// Cannot remove yourself (the owner)
	if ownerID == memberUserID {
		log.Printf("DEBUG: Service.RemoveMember - attempt to remove owner")
		return errors.New("cannot remove project owner")
	}

	// Remove from project_members collection
	res, err := db.ProjectMembers.DeleteOne(ctx, bson.M{
		"project_id": projectID,
		"user_id":    memberUserID,
	})

	log.Printf("DEBUG: Service.RemoveMember - delete result: deletedCount=%v, err=%v", res.DeletedCount, err)

	return err
}

// RejectInvitation deletes a pending invitation and notifies the owner
func RejectInvitation(userID, projectID primitive.ObjectID) error {
	ctx := context.Background()

	// Find the invitation
	var member models.ProjectMember
	err := db.ProjectMembers.FindOne(ctx, bson.M{
		"project_id": projectID,
		"user_id":    userID,
		"status":     "pending",
	}).Decode(&member)
	if err != nil {
		return errors.New("invitation not found")
	}

	// Get project and owner info
	var project models.Project
	err = db.Projects.FindOne(ctx, bson.M{"_id": projectID}).Decode(&project)
	if err != nil {
		return errors.New("project not found")
	}

	var owner models.User
	err = db.Users.FindOne(ctx, bson.M{"_id": project.UserID}).Decode(&owner)
	if err != nil {
		return errors.New("project owner not found")
	}

	var rejectingUser models.User
	err = db.Users.FindOne(ctx, bson.M{"_id": userID}).Decode(&rejectingUser)
	if err != nil {
		return errors.New("user not found")
	}

	// Delete the invitation
	_, err = db.ProjectMembers.DeleteOne(ctx, bson.M{"_id": member.ID})
	if err != nil {
		return err
	}

	// Send notification email to owner
	go func() {
		if err := sendRejectionEmailToOwner(owner.Email, project.Name, rejectingUser.Email); err != nil {
			log.Printf("Failed to send rejection notification email to %s: %v", owner.Email, err)
		} else {
			log.Printf("Rejection notification email sent to %s", owner.Email)
		}
	}()

	return nil
}

func sendRejectionEmailToOwner(ownerEmail, projectName, inviteeEmail string) error {
	from := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASSWORD")
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")

	if from == "" || password == "" || host == "" || port == "" {
		return errors.New("SMTP configuration missing in .env")
	}

	addr := host + ":" + port
	subject := "Subject: Invitation Rejected - " + projectName + "\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body := fmt.Sprintf(`
		<html>
		<body>
			<h2>Invitation Rejected</h2>
			<p>Hello,</p>
			<p>We wanted to let you know that <strong>%s</strong> has declined your invitation to join the project: <strong>%s</strong>.</p>
			<br>
			<p style="color: #888; font-size: 0.9em;">This is an automated notification.</p>
		</body>
		</html>
	`, inviteeEmail, projectName)

	msg := []byte(subject + mime + body)

	auth := smtp.PlainAuth("", from, password, host)
	return smtp.SendMail(addr, auth, from, []string{ownerEmail}, msg)
}

func sendInvitationEmail(to string, projectName string, projectID primitive.ObjectID, token string) error {
	from := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASSWORD")
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")

	if from == "" || password == "" || host == "" || port == "" {
		return errors.New("SMTP configuration missing in .env")
	}

	// Frontend URL for accepting invitation
	acceptURL := fmt.Sprintf("http://localhost:5173/app/v1/invitations?token=%s", token)

	addr := host + ":" + port
	subject := "Subject: Project Invitation - " + projectName + "\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body := fmt.Sprintf(`
		<html>
		<body>
			<h2>You've been invited to collaborate!</h2>
			<p>You have been invited to join the project: <strong>%s</strong></p>
			<p>Click the link below to view and accept your invitation:</p>
			<p><a href="%s" style="background: #3498db; color: white; padding: 12px 24px; text-decoration: none; border-radius: 8px; display: inline-block;">View Invitation</a></p>
			<p>Or copy this link: <a href="%s">%s</a></p>
			<br>
			<p style="color: #888; font-size: 0.9em;">If you didn't expect this invitation, you can safely ignore this email.</p>
		</body>
		</html>
	`, projectName, acceptURL, acceptURL, acceptURL)

	msg := []byte(subject + mime + body)

	auth := smtp.PlainAuth("", from, password, host)
	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}
