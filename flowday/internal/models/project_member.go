package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ProjectMember struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ProjectID  primitive.ObjectID `bson:"project_id" json:"project_id"`
	UserID     primitive.ObjectID `bson:"user_id" json:"user_id"`
	Role       string             `bson:"role" json:"role"`     // owner, member
	Status     string             `bson:"status" json:"status"` // pending, accepted
	Token      string             `bson:"token,omitempty" json:"token,omitempty"`
	InvitedAt  time.Time          `bson:"invited_at" json:"invited_at"`
	AcceptedAt *time.Time         `bson:"accepted_at,omitempty" json:"accepted_at,omitempty"`

	// Relationships (populated manually)
	User    *User    `bson:"-" json:"user,omitempty"`
	Project *Project `bson:"-" json:"project,omitempty"`
}
