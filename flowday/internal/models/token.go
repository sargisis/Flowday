package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RefreshToken struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID `bson:"user_id" json:"user_id"`
	Token     string             `bson:"token" json:"token"`
	ExpiresAt time.Time          `bson:"expires_at" json:"expires_at"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	Revoked   bool               `bson:"revoked" json:"revoked"`
	IPAddress string             `bson:"ip_address,omitempty" json:"ip_address,omitempty"`
	UserAgent string             `bson:"user_agent,omitempty" json:"user_agent,omitempty"`
}
