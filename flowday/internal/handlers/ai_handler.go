package handlers

import (
	"io"
	"log"
	"net/http"

	"flowday/internal/db"
	"flowday/internal/models"
	"flowday/internal/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// DecomposeTask breaks a task into multiple new task cards
func DecomposeTask(c *gin.Context) {
	taskID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id format"})
		return
	}

	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	task, err := services.GetTask(userID.(primitive.ObjectID), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	subtaskTitles, err := services.DecomposeTask(c.Request.Context(), task.Title, task.Description)
	if err != nil {
		log.Printf("[AI] Decompose error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI decomposition failed: " + err.Error()})
		return
	}

	createdSubtasks := []models.Task{}
	var subtaskIDs []primitive.ObjectID

	for _, title := range subtaskTitles {
		newSubtask := models.Task{
			Title:     title,
			Status:    "todo",
			Priority:  task.Priority,
			ProjectID: task.ProjectID,
			// New subtasks block the original task
			Blocks: []primitive.ObjectID{task.ID},
		}
		if err := services.CreateTask(userID.(primitive.ObjectID), &newSubtask); err != nil {
			log.Printf("[AI] Failed to create subtask: %v", err)
			continue
		}
		createdSubtasks = append(createdSubtasks, newSubtask)
		subtaskIDs = append(subtaskIDs, newSubtask.ID)
	}

	// Also update original task to depend on these new subtasks
	if len(subtaskIDs) > 0 {
		updates := map[string]interface{}{
			"depends_on": append(task.DependsOn, subtaskIDs...),
		}
		_ = services.UpdateTask(userID.(primitive.ObjectID), task.ID, updates)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Task decomposed successfully",
		"subtasks": createdSubtasks,
	})
}

// EnrichTask generates a detailed plan and updates the task description
func EnrichTask(c *gin.Context) {
	taskID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id format"})
		return
	}

	userID, _ := c.Get("user_id")
	task, err := services.GetTask(userID.(primitive.ObjectID), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	// Call new AI Service
	plan, err := services.GenerateEnrichedPlan(c.Request.Context(), task.Title)
	if err != nil {
		log.Printf("[AI] Enrich plan generation error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate plan"})
		return
	}

	// Convert plan subtasks to model Subtasks
	var subtasks []models.Subtask
	for _, stTitle := range plan.Subtasks {
		subtasks = append(subtasks, models.Subtask{
			ID:        primitive.NewObjectID().Hex(),
			Title:     stTitle,
			Completed: false,
		})
	}

	// Update Task in DB
	updates := map[string]interface{}{
		"description": plan.Description,
		"subtasks":    subtasks,
	}

	if err := services.UpdateTask(userID.(primitive.ObjectID), taskID, updates); err != nil {
		log.Printf("[AI] Failed to save enriched task: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save plan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Task enriched successfully",
		"description": plan.Description,
		"subtasks":    subtasks,
	})
}

// GetHealthAdvice generates personalized productivity advice based on task stats
func GetHealthAdvice(c *gin.Context) {
	var context services.AnalysisContext
	if err := c.ShouldBindJSON(&context); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid context format"})
		return
	}

	advice, err := services.GetHealthAdvice(c.Request.Context(), context)
	if err != nil {
		log.Printf("[AI] Health advice error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate intelligent advice"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"advice": advice,
	})
}

// HandleChat processes a user message and returns the AI response
func HandleChat(c *gin.Context) {
	var body struct {
		Message string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message is required"})
		return
	}

	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	reply, err := services.Chat(c.Request.Context(), userID.(primitive.ObjectID), body.Message)
	if err != nil {
		if err.Error() == "quota_exceeded" {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Daily free quota exceeded. Upgrade to Pro."})
			return
		}
		log.Printf("[AI] Chat error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process message"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"reply": reply,
	})
}

// HandleChatStream handles AI chat with SSE streaming
func HandleChatStream(c *gin.Context) {
	var body struct {
		Message string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message is required"})
		return
	}

	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Set headers for SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	respChan := make(chan string)
	errChan := make(chan error)

	// Run AI service in background
	go services.ChatStream(c.Request.Context(), userID.(primitive.ObjectID), body.Message, respChan, errChan)

	// Stream response to client
	c.Stream(func(w io.Writer) bool {
		select {
		case msg, ok := <-respChan:
			if !ok {
				// Stream closed normally
				c.SSEvent("done", "{}")
				return false
			}
			c.SSEvent("message", gin.H{"content": msg})
			return true
		case err := <-errChan:
			if err != nil {
				if err.Error() == "quota_exceeded" {
					c.SSEvent("error", gin.H{"error": "Daily free quota exceeded. Upgrade to Pro."})
				} else {
					log.Printf("[AI] Stream error: %v", err)
					c.SSEvent("error", gin.H{"error": "Failed to process message"})
				}
			}
			return false
		case <-c.Request.Context().Done():
			// Client disconnected
			return false
		}
	})
}

// HandleGetHistory returns the chat history for the user
func HandleGetHistory(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	history, err := services.GetHistory(c.Request.Context(), userID.(primitive.ObjectID))
	if err != nil {
		log.Printf("[AI] History error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"history": history,
	})
}

// HandleGetQuota returns the current quota usage
func HandleGetQuota(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	allowed, remaining, err := services.CheckQuota(c.Request.Context(), userID.(primitive.ObjectID))
	if err != nil {
		log.Printf("[AI] Quota check error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check quota"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"allowed":   allowed,
		"remaining": remaining, // -1 means unlimited
	})
}

// HandleDeleteHistory clears the AI chat history for the user
func HandleDeleteHistory(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	coll := db.Database.Collection("ai_conversations")
	_, err := coll.DeleteOne(c.Request.Context(), bson.M{"user_id": userID.(primitive.ObjectID)})
	if err != nil {
		log.Printf("[AI] Delete history error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Chat history cleared successfully",
	})
}

// HandleGetInsights returns AI-generated insights
func HandleGetInsights(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	opts := options.Find().SetSort(bson.M{"created_at": -1}).SetLimit(10)
	cursor, err := db.Database.Collection("ai_insights").Find(c.Request.Context(), bson.M{"user_id": userID.(primitive.ObjectID)}, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch insights"})
		return
	}
	defer cursor.Close(c.Request.Context())

	var insights []models.Insight
	if err = cursor.All(c.Request.Context(), &insights); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode insights"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"insights": insights,
	})
}
