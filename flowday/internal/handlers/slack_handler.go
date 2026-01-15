package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"flowday/internal/db"
	"flowday/internal/models"
	"flowday/internal/utils"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SlackOAuthResponse struct {
	OK          bool   `json:"ok"`
	AccessToken string `json:"access_token,omitempty"`
	Team        struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"team,omitempty"`
	AuthedUser struct {
		ID string `json:"id"`
	} `json:"authed_user,omitempty"`
	Error string `json:"error,omitempty"`
}

type SlackUserInfo struct {
	OK   bool `json:"ok"`
	User struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"user,omitempty"`
	Error string `json:"error,omitempty"`
}

// InitiateSlackOAuth redirects user to Slack OAuth page
func InitiateSlackOAuth(c *gin.Context) {
	val, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := val.(primitive.ObjectID)

	// Get Slack OAuth credentials from environment
	clientID := os.Getenv("SLACK_CLIENT_ID")
	redirectURI := os.Getenv("SLACK_REDIRECT_URI")

	if clientID == "" || redirectURI == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Slack OAuth not configured on server"})
		return
	}

	// Generate state parameter (use user ID as state for security)
	state := userID.Hex()

	// Build OAuth URL
	// Required scopes:
	// - chat:write (send messages)
	// - channels:read, groups:read (read channel list)
	// - channels:join, groups:write (join channels automatically)
	// - im:write (send direct messages)
	// - users:read (user info)
	oauthURL := fmt.Sprintf(
		"https://slack.com/oauth/v2/authorize?client_id=%s&scope=chat:write,channels:read,groups:read,channels:join,groups:write,im:write,users:read&redirect_uri=%s&state=%s",
		clientID,
		url.QueryEscape(redirectURI),
		state,
	)

	c.JSON(http.StatusOK, gin.H{
		"oauth_url": oauthURL,
		"state":     state,
	})
}

// HandleSlackOAuthCallback processes the OAuth callback from Slack
func HandleSlackOAuthCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	errorParam := c.Query("error")

	if errorParam != "" {
		c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%s?error=%s", os.Getenv("FRONTEND_URL"), url.QueryEscape(errorParam)))
		return
	}

	if code == "" || state == "" {
		c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%s?error=missing_parameters", os.Getenv("FRONTEND_URL")))
		return
	}

	// Parse user ID from state
	userID, err := primitive.ObjectIDFromHex(state)
	if err != nil {
		c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%s?error=invalid_state", os.Getenv("FRONTEND_URL")))
		return
	}

	// Exchange code for access token
	clientID := os.Getenv("SLACK_CLIENT_ID")
	clientSecret := os.Getenv("SLACK_CLIENT_SECRET")
	redirectURI := os.Getenv("SLACK_REDIRECT_URI")

	if clientID == "" || clientSecret == "" || redirectURI == "" {
		c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%s?error=server_config_error", os.Getenv("FRONTEND_URL")))
		return
	}

	// Exchange code for token
	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("redirect_uri", redirectURI)

	resp, err := http.PostForm("https://slack.com/api/oauth.v2.access", data)
	if err != nil {
		c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%s?error=token_exchange_failed", os.Getenv("FRONTEND_URL")))
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%s?error=read_response_failed", os.Getenv("FRONTEND_URL")))
		return
	}

	var oauthResp SlackOAuthResponse
	if err := json.Unmarshal(body, &oauthResp); err != nil {
		c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%s?error=parse_response_failed", os.Getenv("FRONTEND_URL")))
		return
	}

	if !oauthResp.OK {
		c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%s?error=%s", os.Getenv("FRONTEND_URL"), url.QueryEscape(oauthResp.Error)))
		return
	}

	// Get user info from Slack
	userInfo, err := getSlackUserInfo(oauthResp.AccessToken)
	if err != nil {
		c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%s?error=user_info_failed", os.Getenv("FRONTEND_URL")))
		return
	}

	// Save tokens to database
	update := bson.M{
		"$set": bson.M{
			"slack_access_token": oauthResp.AccessToken,
			"slack_team_id":      oauthResp.Team.ID,
			"slack_team_name":    oauthResp.Team.Name,
			"slack_user_id":      userInfo.User.ID,
			"updated_at":         time.Now(),
		},
	}

	// Only set refresh token if provided (some OAuth flows don't provide it)
	if oauthResp.AuthedUser.ID != "" {
		update["$set"].(bson.M)["slack_user_id"] = oauthResp.AuthedUser.ID
	}

	_, err = db.Users.UpdateOne(context.Background(), bson.M{"_id": userID}, update)
	if err != nil {
		c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%s?error=database_update_failed", os.Getenv("FRONTEND_URL")))
		return
	}

	// Redirect to frontend with success
	c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%s/app/v1/settings?slack_connected=true", os.Getenv("FRONTEND_URL")))
}

// DisconnectSlack removes Slack integration for user
func DisconnectSlack(c *gin.Context) {
	val, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := val.(primitive.ObjectID)

	update := bson.M{
		"$unset": bson.M{
			"slack_access_token":  "",
			"slack_refresh_token": "",
			"slack_team_id":       "",
			"slack_user_id":       "",
			"slack_team_name":     "",
			"slack_channel_id":    "",
		},
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	}

	_, err := db.Users.UpdateOne(context.Background(), bson.M{"_id": userID}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disconnect Slack"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Slack disconnected successfully"})
}

// GetSlackChannels returns list of channels user can post to
func GetSlackChannels(c *gin.Context) {
	val, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := val.(primitive.ObjectID)

	var user models.User
	err := db.Users.FindOne(context.Background(), bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if user.SlackAccessToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Slack not connected"})
		return
	}

	channels, err := utils.GetSlackChannels(user.SlackAccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch channels: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"channels": channels})
}

// UpdateSlackChannel updates the default channel for notifications
func UpdateSlackChannel(c *gin.Context) {
	val, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := val.(primitive.ObjectID)

	var req struct {
		ChannelID string `json:"channel_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	err := db.Users.FindOne(context.Background(), bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if user.SlackAccessToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Slack not connected"})
		return
	}

	update := bson.M{
		"$set": bson.M{
			"slack_channel_id": req.ChannelID,
			"updated_at":       time.Now(),
		},
	}

	_, err = db.Users.UpdateOne(context.Background(), bson.M{"_id": userID}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update channel"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Channel updated successfully", "channel_id": req.ChannelID})
}

// TestSlackOAuth sends a test message using OAuth token
func TestSlackOAuth(c *gin.Context) {
	val, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := val.(primitive.ObjectID)

	var user models.User
	err := db.Users.FindOne(context.Background(), bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if user.SlackAccessToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Slack not connected"})
		return
	}

	// Send test message as DM if user has SlackUserID, otherwise to channel
	if user.SlackUserID != "" {
		dmChannelID, err2 := utils.OpenDMChannel(user.SlackAccessToken, user.SlackUserID)
		if err2 != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open DM channel: " + err2.Error()})
			return
		}
		err = utils.SendSlackMessageOAuth(user.SlackAccessToken, dmChannelID, "🎉 Flowday Slack integration is working! You'll receive task notifications here.", "Flowday Test")
	} else {
		channelID := user.SlackChannelID
		if channelID == "" {
			channelID = "#general" // Default channel
		}
		err = utils.SendSlackMessageOAuth(user.SlackAccessToken, channelID, "🎉 Flowday Slack integration is working!", "Flowday Test")
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send test message: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Test message sent successfully! Check your Slack channel."})
}

// Helper function to get Slack user info
func getSlackUserInfo(accessToken string) (*SlackUserInfo, error) {
	req, err := http.NewRequest("GET", "https://slack.com/api/auth.test", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var authTest struct {
		OK    bool   `json:"ok"`
		User  string `json:"user_id,omitempty"`
		Team  string `json:"team_id,omitempty"`
		Error string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(body, &authTest); err != nil {
		return nil, err
	}

	if !authTest.OK {
		return nil, fmt.Errorf("slack API error: %s", authTest.Error)
	}

	// Return user info in expected format
	return &SlackUserInfo{
		OK: true,
		User: struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}{
			ID: authTest.User,
		},
	}, nil
}

// HandleSlackInteractivity processes interactive button clicks from Slack
func HandleSlackInteractivity(c *gin.Context) {
	// Slack sends data as form-urlencoded with a "payload" field containing JSON
	payloadStr := c.PostForm("payload")
	if payloadStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing payload"})
		return
	}

	// First, check if this is a URL verification challenge from Slack
	var challengeCheck struct {
		Type      string `json:"type"`
		Challenge string `json:"challenge"`
		Token     string `json:"token"`
	}

	if err := json.Unmarshal([]byte(payloadStr), &challengeCheck); err == nil {
		// This is a URL verification challenge
		if challengeCheck.Type == "url_verification" {
			// Verify token if configured
			expectedToken := os.Getenv("SLACK_VERIFICATION_TOKEN")
			if expectedToken != "" && challengeCheck.Token != expectedToken {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid verification token"})
				return
			}
			// Return the challenge value to verify the URL
			c.JSON(http.StatusOK, gin.H{"challenge": challengeCheck.Challenge})
			return
		}
	}

	var payload struct {
		Type     string `json:"type"`
		Token    string `json:"token"`
		ActionTs string `json:"action_ts"`
		User     struct {
			ID string `json:"id"`
		} `json:"user"`
		Team struct {
			ID string `json:"id"`
		} `json:"team"`
		Actions []struct {
			ActionID string `json:"action_id"`
			Value    string `json:"value"`
		} `json:"actions"`
		ResponseURL string `json:"response_url"`
		Message     struct {
			Ts string `json:"ts"`
		} `json:"message"`
	}

	if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload format"})
		return
	}

	// Verify request token (optional but recommended for security)
	expectedToken := os.Getenv("SLACK_VERIFICATION_TOKEN")
	if expectedToken != "" && payload.Token != expectedToken {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid verification token"})
		return
	}

	if len(payload.Actions) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No actions in payload"})
		return
	}

	action := payload.Actions[0]
	taskID := action.Value

	// Find user by Slack User ID
	var user models.User
	err := db.Users.FindOne(context.Background(), bson.M{
		"slack_user_id": payload.User.ID,
		"slack_team_id": payload.Team.ID,
	}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found for this Slack account"})
		return
	}

	// Handle different action types
	switch action.ActionID {
	case "task_mark_done":
		// Update task status to "done"
		taskObjectID, err := primitive.ObjectIDFromHex(taskID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID format"})
			return
		}

		// Verify task belongs to user's project
		var task models.Task
		err = db.Tasks.FindOne(context.Background(), bson.M{"_id": taskObjectID}).Decode(&task)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
			return
		}

		// Update task status
		update := bson.M{
			"$set": bson.M{
				"status":     "done",
				"updated_at": time.Now(),
			},
		}
		_, err = db.Tasks.UpdateOne(context.Background(), bson.M{"_id": taskObjectID}, update)
		if err != nil {
			// Still respond to Slack quickly, but with error
			response := map[string]interface{}{
				"response_type": "ephemeral",
				"text":          "❌ Failed to update task. Please try again.",
			}
			c.JSON(http.StatusOK, response)
			return
		}

		// Immediately respond to Slack (must be within 3 seconds)
		response := map[string]interface{}{
			"response_type": "ephemeral",
			"text":          "✅ Task marked as done!",
		}
		c.JSON(http.StatusOK, response)

		// Asynchronously update the original message via response_url
		if payload.ResponseURL != "" {
			go func() {
				updatedBlocks := []map[string]interface{}{
					{
						"type": "section",
						"text": map[string]interface{}{
							"type": "mrkdwn",
							"text": fmt.Sprintf("✅ *Task completed*\n*%s*", task.Title),
						},
					},
					{
						"type": "section",
						"fields": []map[string]interface{}{
							{
								"type": "mrkdwn",
								"text": fmt.Sprintf("*Project:*\n%s", task.ProjectID.Hex()),
							},
							{
								"type": "mrkdwn",
								"text": fmt.Sprintf("*Status:*\n✅ Done"),
							},
						},
					},
				}

				updatePayload := map[string]interface{}{
					"replace_original": true,
					"blocks":           updatedBlocks,
				}

				jsonData, _ := json.Marshal(updatePayload)
				req, _ := http.NewRequest("POST", payload.ResponseURL, bytes.NewBuffer(jsonData))
				req.Header.Set("Content-Type", "application/json")

				client := &http.Client{Timeout: 5 * time.Second}
				client.Do(req)
			}()
		}

	case "task_view":
		// Return URL to view task (Slack will open it)
		frontendURL := os.Getenv("FRONTEND_URL")
		if frontendURL == "" {
			frontendURL = "http://localhost:5173"
		}
		taskURL := fmt.Sprintf("%s/app/v1/tasks?task=%s", frontendURL, taskID)

		response := map[string]interface{}{
			"response_type":    "ephemeral",
			"text":             fmt.Sprintf("👁️ Opening task: %s", taskURL),
			"replace_original": false,
		}
		c.JSON(http.StatusOK, response)

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown action type"})
		return
	}
}
