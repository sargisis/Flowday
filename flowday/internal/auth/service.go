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

func Register(email, password string) (*models.User, error) {
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
		ID:        primitive.NewObjectID(),
		Email:     email,
		Password:  hashed,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err = db.Users.InsertOne(ctx, user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func Login(email, password string) (string, error) {
	ctx := context.Background()
	var user models.User

	err := db.Users.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "", appErrors.ErrNotFound
		}
		return "", err
	}

	if !CheckPassword(user.Password, password) {
		return "", errors.New("invalid credentials")
	}

	return GenerateToken(user.ID)
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

	// Find valid code
	err := db.PasswordResets.FindOne(ctx, bson.M{
		"email":      email,
		"code":       code,
		"expires_at": bson.M{"$gt": time.Now()},
	}).Decode(&reset)

	if err != nil {
		return errors.New("invalid or expired code")
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
