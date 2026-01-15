package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/smtp"
	"os"
	"time"

	"flowday/internal/db"
	appErrors "flowday/internal/errors"
	"flowday/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func Register(name, email, password string) (*models.User, error) {
	hashed, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	// Check if user exists
	ctx := context.Background()
	var existing models.User
	err = db.Users.FindOne(ctx, bson.M{"email": email}).Decode(&existing)
	if err == nil {
		return nil, appErrors.ErrUserExists
	}
	if err != mongo.ErrNoDocuments {
		return nil, err
	}

	user := models.User{
		ID:                 primitive.NewObjectID(),
		Name:               name,
		Email:              email,
		Password:           hashed,
		XP:                 0,
		Level:              1,
		EmailNotifications: false, // Default: disabled
		SlackWebhookURL:    "",     // Default: empty
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	_, err = db.Users.InsertOne(ctx, user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func Login(email, password string) (string, string, error) {
	ctx := context.Background()
	var user models.User

	err := db.Users.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "", "", appErrors.ErrNotFound
		}
		return "", "", err
	}

	if !CheckPassword(user.Password, password) {
		return "", "", errors.New("invalid credentials")
	}

	accessToken, refreshToken, err := GenerateTokens(user.ID)
	if err != nil {
		return "", "", err
	}

	// Store refresh token in DB
	rawToken := models.RefreshToken{
		ID:        primitive.NewObjectID(),
		UserID:    user.ID,
		Token:     refreshToken,
		ExpiresAt: time.Now().Add(time.Hour * 24 * 7),
		CreatedAt: time.Now(),
		Revoked:   false,
	}

	// Assuming db.RefreshTokens exists (we need to initialize it in db/db.go)
	_, err = db.RefreshTokens.InsertOne(ctx, rawToken)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func RequestPasswordReset(email string) error {
	ctx := context.Background()
	var user models.User
	err := db.Users.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		// Silent fail for security
		return nil
	}

	// Generate 6-digit random code
	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	code := fmt.Sprintf("%06d", n.Int64())

	reset := models.PasswordReset{
		ID:        primitive.NewObjectID(),
		Email:     email,
		Code:      code,
		ExpiresAt: time.Now().Add(15 * time.Minute),
		CreatedAt: time.Now(),
		Used:      false,
	}

	_, err = db.PasswordResets.InsertOne(ctx, reset)
	if err != nil {
		log.Println("Error saving reset code:", err)
		return err
	}

	// Send email
	go func() {
		if err := sendEmail(email, code); err != nil {
			log.Printf("Failed to send email to %s: %v", email, err)
		} else {
			log.Printf("Email sent to %s", email)
		}
	}()

	return nil
}

func ResetPassword(email, code, newPassword string) error {
	ctx := context.Background()
	var reset models.PasswordReset

	// Find valid code that hasn't been used
	err := db.PasswordResets.FindOne(ctx, bson.M{
		"email":      email,
		"code":       code,
		"used":       false, // ✅ Prevent code reuse
		"expires_at": bson.M{"$gt": time.Now()},
	}).Decode(&reset)

	if err != nil {
		return errors.New("invalid or expired code")
	}

	// ✅ Mark code as used immediately
	_, err = db.PasswordResets.UpdateOne(ctx,
		bson.M{"_id": reset.ID},
		bson.M{"$set": bson.M{"used": true}},
	)
	if err != nil {
		return err
	}

	// Hash new password
	hashed, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	// Update user
	_, err = db.Users.UpdateOne(ctx,
		bson.M{"email": email},
		bson.M{"$set": bson.M{
			"password":   hashed,
			"updated_at": time.Now(),
		}},
	)
	if err != nil {
		return err
	}

	// Delete used codes
	db.PasswordResets.DeleteMany(ctx, bson.M{"email": email})

	return nil
}

func sendEmail(to string, code string) error {
	from := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASSWORD")
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")

	if from == "" || password == "" || host == "" || port == "" {
		return errors.New("SMTP configuration missing in .env")
	}

	addr := host + ":" + port
	subject := "Subject: Flowday Password Reset\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body := fmt.Sprintf("<html><body><h3>Password Reset</h3><p>Your verification code is: <b>%s</b></p><p>This code expires in 15 minutes.</p></body></html>", code)
	msg := []byte(subject + mime + body)

	auth := smtp.PlainAuth("", from, password, host)
	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}

func GetUserByID(idStr string) (*models.User, error) {
	ctx := context.Background()
	objID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		return nil, err
	}

	var user models.User
	err = db.Users.FindOne(ctx, bson.M{"_id": objID}).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func Refresh(tokenStr string) (string, error) {
	// 1. Validate signature and expiration
	userIDStr, err := ValidateRefreshToken(tokenStr)
	if err != nil {
		return "", err
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return "", errors.New("invalid user id in token")
	}

	// 2. Check DB logic
	ctx := context.Background()
	var storedToken models.RefreshToken
	err = db.RefreshTokens.FindOne(ctx, bson.M{
		"token": tokenStr,
	}).Decode(&storedToken)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "", errors.New("refresh token not found or revoked")
		}
		return "", err
	}

	if storedToken.Revoked {
		return "", errors.New("refresh token has been revoked")
	}

	// 3. Generate New Access Token
	accessToken, err := GenerateAccessToken(userID)
	if err != nil {
		return "", err
	}

	return accessToken, nil
}

func Logout(refreshToken string) error {
	ctx := context.Background()

	// Delete the refresh token from database
	result, err := db.RefreshTokens.DeleteOne(ctx, bson.M{
		"token": refreshToken,
	})

	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		// Token not found - might have already been deleted
		return nil // Don't error, user is logging out anyway
	}

	return nil
}
