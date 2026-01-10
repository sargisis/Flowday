package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ChatMessage struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	SenderID       primitive.ObjectID `bson:"sender_id" json:"sender_id"`
	ReceiverID     primitive.ObjectID `bson:"receiver_id" json:"receiver_id"`
	Content        string             `bson:"content" json:"content"`
	AttachmentURL  string             `bson:"attachment_url,omitempty" json:"attachment_url,omitempty"`
	AttachmentType string             `bson:"attachment_type,omitempty" json:"attachment_type,omitempty"`
	CreatedAt      time.Time          `bson:"created_at" json:"created_at"`
	IsRead         bool               `bson:"is_read" json:"is_read"`
}
