package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PasswordReset struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Email     string             `bson:"email" json:"email"`
	Code      string             `bson:"code" json:"code"`
	ExpiresAt time.Time          `bson:"expires_at" json:"expires_at"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}
