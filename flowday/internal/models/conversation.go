package models

import "time"

type ChatConversationSummary struct {
	UserID          string    `json:"user_id"`
	UserName        string    `json:"user_name"`
	UserAvatar      string    `json:"user_avatar"`
	LastMessage     string    `json:"last_message"`
	LastMessageTime time.Time `json:"last_message_time"`
	UnreadCount     int       `json:"unread_count"`
	Status          string    `json:"status"`
}
