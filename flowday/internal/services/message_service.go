package services

import (
	"context"
	"flowday/internal/db"
	"flowday/internal/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func SendMessage(senderID, receiverID primitive.ObjectID, content string, attachmentURL, attachmentType string) (*models.ChatMessage, error) {
	ctx := context.Background()

	msg := &models.ChatMessage{
		ID:             primitive.NewObjectID(),
		SenderID:       senderID,
		ReceiverID:     receiverID,
		Content:        content,
		AttachmentURL:  attachmentURL,
		AttachmentType: attachmentType,
		CreatedAt:      time.Now(),
		IsRead:         false,
	}

	_, err := db.Messages.InsertOne(ctx, msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func GetMessagesBetweenUsers(userA, userB primitive.ObjectID) ([]models.ChatMessage, error) {
	ctx := context.Background()

	filter := bson.M{
		"$or": []bson.M{
			{"sender_id": userA, "receiver_id": userB},
			{"sender_id": userB, "receiver_id": userA},
		},
	}

	opts := options.Find().SetSort(bson.M{"created_at": 1})
	cursor, err := db.Messages.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []models.ChatMessage
	if err := cursor.All(ctx, &messages); err != nil {
		return nil, err
	}

	// Mark received messages as read
	_, _ = db.Messages.UpdateMany(ctx, bson.M{
		"sender_id":   userB,
		"receiver_id": userA,
		"is_read":     false,
	}, bson.M{"$set": bson.M{"is_read": true}})

	return messages, nil
}

func GetConversationsForUser(userID primitive.ObjectID) ([]models.ChatConversationSummary, error) {
	ctx := context.Background()

	// Aggregate to find unique chat partners
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"$or": []bson.M{
					{"sender_id": userID},
					{"receiver_id": userID},
				},
			},
		},
		{
			"$sort": bson.M{"created_at": -1},
		},
		{
			"$group": bson.M{
				"_id": bson.M{
					"$cond": []interface{}{
						bson.M{"$eq": []interface{}{"$sender_id", userID}},
						"$receiver_id",
						"$sender_id",
					},
				},
				"last_message":      bson.M{"$first": "$content"},
				"last_message_time": bson.M{"$first": "$created_at"},
				"unread_count": bson.M{
					"$sum": bson.M{
						"$cond": []interface{}{
							bson.M{
								"$and": []interface{}{
									bson.M{"$eq": []interface{}{"$receiver_id", userID}},
									bson.M{"$eq": []interface{}{"$is_read", false}},
								},
							},
							1,
							0,
						},
					},
				},
			},
		},
	}

	cursor, err := db.Messages.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var summaries []models.ChatConversationSummary
	for cursor.Next(ctx) {
		var result struct {
			ID              primitive.ObjectID `bson:"_id"`
			LastMessage     string             `bson:"last_message"`
			LastMessageTime time.Time          `bson:"last_message_time"`
			UnreadCount     int                `bson:"unread_count"`
		}
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}

		// Fetch user details
		var user models.User
		_ = db.Users.FindOne(ctx, bson.M{"_id": result.ID}).Decode(&user)

		summaries = append(summaries, models.ChatConversationSummary{
			UserID:          result.ID.Hex(),
			UserName:        user.Name,
			UserAvatar:      user.AvatarURL,
			LastMessage:     result.LastMessage,
			LastMessageTime: result.LastMessageTime,
			UnreadCount:     result.UnreadCount,
			Status:          user.Status,
		})
	}

	return summaries, nil
}
