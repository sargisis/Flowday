package handlers

import (
	"net/http"
	"os"

	"flowday/internal/db"
	"flowday/internal/dto"
	"flowday/internal/logger"
	"flowday/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"

	"flowday/internal/websocket"
)

// HandleBMACWebhook processes incoming webhooks from Buy Me a Coffee
func HandleBMACWebhook(c *gin.Context) {
	// Verify Secret (simple check, or HMAC if supported by your BMAC tier)
	// BMAC sends a generic webhook, usually we check a secret in the URL or header
	// For this implementation, we'll assume a shared secret in environment
	secret := os.Getenv("BMAC_WEBHOOK_SECRET")

	// If secret is set, verify it (Optional, depends on how you configure BMAC)
	// Here we check a custom header you set in BMAC dashboard, e.g., X-Webhook-Secret
	if secret != "" {
		requestSecret := c.GetHeader("X-Webhook-Secret")
		if requestSecret != secret {
			// Try checking query param as fallback
			if c.Query("secret") != secret {
				logger.Log.Warn("Unauthorized BMAC webhook attempt")
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid secret"})
				return
			}
		}
	}

	var payload dto.BMACWebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		logger.Log.WithError(err).Error("Failed to parse BMAC webhook")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	// Log the entire payload for debugging
	logger.Log.Infof("Received BMAC webhook RAW: %+v", payload)
	logger.Log.Infof("Received BMAC webhook: Type=%s Email=%s SupportType=%s Amount=%s", payload.Type, payload.SupporterEmail, payload.SupportType, payload.Amount)

	// Logic: If user exists with this email, upgrade them to PRO/ULTRA
	// We update the plan to "pro"

	ctx := c.Request.Context()

	// Find user by email
	var user models.User
	err := db.Users.FindOne(ctx, bson.M{"email": payload.SupporterEmail}).Decode(&user)
	if err != nil {
		logger.Log.Infof("Supporter email %s not found in users, skipping upgrade", payload.SupporterEmail)
		// Return 200 OK so BMAC doesn't retry
		c.JSON(http.StatusOK, gin.H{"status": "skipped", "reason": "user not found"})
		return
	}

	// Determine plan level based on amount or type (customize logic here)
	// For now, any donation > $5 or any subscription = PRO
	newPlan := "pro"

	// Update user plan
	update := bson.M{
		"$set": bson.M{
			"plan":       newPlan,
			"updated_at": user.UpdatedAt, // ideally set to Now, but using existing pattern
		},
	}

	_, err = db.Users.UpdateOne(ctx, bson.M{"_id": user.ID}, update)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to update user plan")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database update failed"})
		return
	}

	logger.Log.Infof(" upgraded user %s (%s) to %s", user.Name, user.Email, newPlan)

	// Broadcast update via WebSocket to the user
	websocket.BroadcastUserUpdate(user.ID, map[string]interface{}{
		"plan": newPlan,
	})

	// Send notification or email here if needed

	c.JSON(http.StatusOK, gin.H{"status": "success", "user_upgraded": true})
}
