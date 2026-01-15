package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type SlackMessage struct {
	Text        string            `json:"text,omitempty"`
	Username    string            `json:"username,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	IconURL     string            `json:"icon_url,omitempty"`
	Channel     string            `json:"channel,omitempty"`
	Blocks      []SlackBlock      `json:"blocks,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
}

type SlackBlock struct {
	Type string     `json:"type"`
	Text *SlackText `json:"text,omitempty"`
}

type SlackText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type SlackAttachment struct {
	Color     string       `json:"color,omitempty"`
	Title     string       `json:"title,omitempty"`
	Text      string       `json:"text,omitempty"`
	Fields    []SlackField `json:"fields,omitempty"`
	Footer    string       `json:"footer,omitempty"`
	Timestamp int64        `json:"ts,omitempty"`
}

type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// SendSlackNotification sends a notification to Slack using webhook URL
func SendSlackNotification(webhookURL string, message SlackMessage) error {
	if webhookURL == "" {
		return errors.New("slack webhook URL is required")
	}

	if !isValidSlackWebhookURL(webhookURL) {
		return errors.New("invalid slack webhook URL format")
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal slack message: %w", err)
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send slack notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack API returned status code: %d", resp.StatusCode)
	}

	return nil
}

// SendTaskNotification sends a formatted task notification to Slack
func SendTaskNotification(webhookURL string, taskTitle string, action string, projectName string, priority string, status string) error {
	// Determine color based on priority
	color := "#36a64f" // green (default)
	switch priority {
	case "high":
		color = "#ff0000" // red
	case "medium":
		color = "#ff9900" // orange
	case "low":
		color = "#36a64f" // green
	}

	// Determine emoji based on action
	emoji := "📝"
	switch action {
	case "created":
		emoji = "✨"
	case "updated":
		emoji = "🔄"
	case "completed":
		emoji = "✅"
	case "deleted":
		emoji = "🗑️"
	}

	message := SlackMessage{
		Username:  "Flowday",
		IconEmoji: ":rocket:",
		Attachments: []SlackAttachment{
			{
				Color: color,
				Title: fmt.Sprintf("%s Task %s", emoji, action),
				Text:  taskTitle,
				Fields: []SlackField{
					{
						Title: "Project",
						Value: projectName,
						Short: true,
					},
					{
						Title: "Priority",
						Value: priority,
						Short: true,
					},
					{
						Title: "Status",
						Value: status,
						Short: true,
					},
				},
				Footer: "Flowday Task Management",
			},
		},
	}

	return SendSlackNotification(webhookURL, message)
}

func isValidSlackWebhookURL(url string) bool {
	return len(url) > 0 &&
		strings.HasPrefix(url, "https://hooks.slack.com/services/")
}

// TestSlackWebhook sends a test message to verify webhook is working
func TestSlackWebhook(webhookURL string) error {
	testMessage := SlackMessage{
		Text:      "🎉 Flowday Slack integration is working!",
		Username:  "Flowday",
		IconEmoji: ":rocket:",
		Attachments: []SlackAttachment{
			{
				Color:  "#36a64f",
				Title:  "✅ Test Notification",
				Text:   "If you see this message, your Slack webhook is configured correctly!",
				Footer: "Flowday Task Management",
			},
		},
	}
	return SendSlackNotification(webhookURL, testMessage)
}

// ========== OAuth API Functions ==========

type SlackChannel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type SlackChannelsResponse struct {
	OK       bool           `json:"ok"`
	Channels []SlackChannel `json:"channels,omitempty"`
	Error    string         `json:"error,omitempty"`
}

type SlackPostMessageResponse struct {
	OK    bool   `json:"ok"`
	TS    string `json:"ts,omitempty"`
	Error string `json:"error,omitempty"`
}

// GetSlackChannels fetches list of channels user can post to
func GetSlackChannels(accessToken string) ([]SlackChannel, error) {
	req, err := http.NewRequest("GET", "https://slack.com/api/conversations.list?types=public_channel,private_channel&exclude_archived=true", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch channels: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var channelsResp SlackChannelsResponse
	if err := json.Unmarshal(body, &channelsResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !channelsResp.OK {
		return nil, fmt.Errorf("slack API error: %s", channelsResp.Error)
	}

	return channelsResp.Channels, nil
}

// JoinSlackChannel adds the bot to a channel (required before sending messages)
func JoinSlackChannel(accessToken string, channelID string) error {
	// If channelID starts with # or C, clean it
	channelID = strings.TrimPrefix(channelID, "#")

	payload := map[string]interface{}{
		"channel": channelID,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal join request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://slack.com/api/conversations.join", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to join channel: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var joinResp struct {
		OK    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(body, &joinResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Ignore "already_in_channel" error - that's fine
	if !joinResp.OK && joinResp.Error != "already_in_channel" {
		return fmt.Errorf("slack API error: %s", joinResp.Error)
	}

	return nil
}

// OpenDMChannel opens or gets a direct message channel with a user
func OpenDMChannel(accessToken string, userID string) (string, error) {
	payload := map[string]interface{}{
		"users": userID,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal DM request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://slack.com/api/conversations.open", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to open DM: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var dmResp struct {
		OK      bool `json:"ok"`
		Channel struct {
			ID string `json:"id"`
		} `json:"channel,omitempty"`
		Error string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(body, &dmResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if !dmResp.OK {
		return "", fmt.Errorf("slack API error: %s", dmResp.Error)
	}

	return dmResp.Channel.ID, nil
}

// SendSlackMessageOAuth sends a message to Slack using OAuth token
func SendSlackMessageOAuth(accessToken string, channelID string, text string, username string) error {
	if accessToken == "" {
		return errors.New("slack access token is required")
	}

	// If channelID starts with #, remove it
	channelID = strings.TrimPrefix(channelID, "#")

	// Try to join channel first (will fail silently if already in channel)
	JoinSlackChannel(accessToken, channelID)

	payload := map[string]interface{}{
		"channel": channelID,
		"text":    text,
	}

	if username != "" {
		payload["username"] = username
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequest("POST", "https://slack.com/api/chat.postMessage", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var postResp SlackPostMessageResponse
	if err := json.Unmarshal(body, &postResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !postResp.OK {
		// Provide user-friendly error messages
		errorMsg := postResp.Error
		switch postResp.Error {
		case "not_in_channel":
			errorMsg = "Bot is not a member of this channel. Please add the bot to the channel in Slack, or try a different channel."
		case "channel_not_found":
			errorMsg = "Channel not found. Please check the channel ID and try again."
		case "invalid_auth":
			errorMsg = "Invalid authentication. Please reconnect your Slack account."
		case "missing_scope":
			errorMsg = "Missing required permissions. Please reconnect your Slack account to grant all permissions."
		}
		return fmt.Errorf("slack API error: %s", errorMsg)
	}

	return nil
}

// SendTaskNotificationOAuth sends a formatted task notification using OAuth
// If userID is provided, sends as DM to user. Otherwise sends to channel.
// taskID is optional - if provided, adds interactive buttons
func SendTaskNotificationOAuth(accessToken string, channelID string, taskTitle string, action string, projectName string, priority string, status string, userID ...string) error {
	// Extract taskID if provided as last parameter
	var taskID string
	var slackUserID string
	if len(userID) > 0 {
		if len(userID) > 1 {
			taskID = userID[1]
			slackUserID = userID[0]
		} else {
			slackUserID = userID[0]
		}
	}
	// If slackUserID is provided, send as DM instead of channel
	if slackUserID != "" {
		dmChannelID, err := OpenDMChannel(accessToken, slackUserID)
		if err != nil {
			return fmt.Errorf("failed to open DM channel: %w", err)
		}
		channelID = dmChannelID
	} else {
		// If channelID starts with #, remove it
		channelID = strings.TrimPrefix(channelID, "#")
		// Try to join channel first (will fail silently if already in channel)
		JoinSlackChannel(accessToken, channelID)
	}

	// Determine emoji based on action
	emoji := "📝"
	switch action {
	case "created":
		emoji = "✨"
	case "updated":
		emoji = "🔄"
	case "completed":
		emoji = "✅"
	case "deleted":
		emoji = "🗑️"
	}

	// Build message with blocks (Slack's Block Kit)
	// Add user mention if sending DM
	headerText := fmt.Sprintf("*%s Task %s*\n%s", emoji, action, taskTitle)
	if slackUserID != "" && action != "completed" {
		headerText = fmt.Sprintf("<@%s> *%s Task %s*\n%s", slackUserID, emoji, action, taskTitle)
	}

	blocks := []map[string]interface{}{
		{
			"type": "section",
			"text": map[string]interface{}{
				"type": "mrkdwn",
				"text": headerText,
			},
		},
		{
			"type": "section",
			"fields": []map[string]interface{}{
				{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*Project:*\n%s", projectName),
				},
				{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*Priority:*\n%s", priority),
				},
				{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*Status:*\n%s", status),
				},
			},
		},
	}

	// Add interactive buttons if taskID is provided and task is not completed
	if taskID != "" && status != "done" && status != "completed" {
		frontendURL := os.Getenv("FRONTEND_URL")
		if frontendURL == "" {
			frontendURL = "http://localhost:5173"
		}
		taskURL := fmt.Sprintf("%s/app/v1/tasks", frontendURL)

		blocks = append(blocks, map[string]interface{}{
			"type": "actions",
			"elements": []map[string]interface{}{
				{
					"type": "button",
					"text": map[string]interface{}{
						"type": "plain_text",
						"text": "✅ Mark as Done",
					},
					"style":     "primary",
					"value":     taskID,
					"action_id": "task_mark_done",
					// No URL - handled via Interactivity URL
				},
				{
					"type": "button",
					"text": map[string]interface{}{
						"type": "plain_text",
						"text": "👁️ View Task",
					},
					"value":     taskID,
					"action_id": "task_view",
					"url":       taskURL, // Opens in browser
				},
			},
		})
	}

	payload := map[string]interface{}{
		"channel":    channelID,
		"blocks":     blocks,
		"username":   "Flowday",
		"icon_emoji": ":rocket:",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequest("POST", "https://slack.com/api/chat.postMessage", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var postResp SlackPostMessageResponse
	if err := json.Unmarshal(body, &postResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !postResp.OK {
		// Provide user-friendly error messages
		errorMsg := postResp.Error
		switch postResp.Error {
		case "not_in_channel":
			errorMsg = "Bot is not a member of this channel. Please add the bot to the channel in Slack, or try a different channel."
		case "channel_not_found":
			errorMsg = "Channel not found. Please check the channel ID and try again."
		case "invalid_auth":
			errorMsg = "Invalid authentication. Please reconnect your Slack account."
		case "missing_scope":
			errorMsg = "Missing required permissions. Please reconnect your Slack account to grant all permissions."
		}
		return fmt.Errorf("slack API error: %s", errorMsg)
	}

	return nil
}
