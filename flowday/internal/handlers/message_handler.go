package handlers

import (
	"flowday/internal/dto"
	"flowday/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func SendMessage(c *gin.Context) {
	var req dto.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	senderID, _ := c.Get("user_id")
	receiverID, err := primitive.ObjectIDFromHex(req.ReceiverID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid receiver_id format"})
		return
	}

	msg, err := services.SendMessage(senderID.(primitive.ObjectID), receiverID, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, msg)
}

func GetConversations(c *gin.Context) {
	userID, _ := c.Get("user_id")
	summaries, err := services.GetConversationsForUser(userID.(primitive.ObjectID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summaries)
}

func GetChatHistory(c *gin.Context) {
	partnerID, err := primitive.ObjectIDFromHex(c.Param("partnerID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid partner id format"})
		return
	}

	userID, _ := c.Get("user_id")
	messages, err := services.GetMessagesBetweenUsers(userID.(primitive.ObjectID), partnerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, messages)
}
