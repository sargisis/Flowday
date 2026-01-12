package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"flowday/internal/db"
	"flowday/internal/models"
)

var (
	groqClient       *openai.Client
	FreeQuotaLimit   = 10
	QuotaResetPeriod = 24 * time.Hour
)

func InitAIService() error {
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		log.Println("[AI] WARNING: GROQ_API_KEY is not set. AI features will be disabled.")
		return nil
	}

	config := openai.DefaultConfig(apiKey)
	config.BaseURL = "https://api.groq.com/openai/v1" // Groq API endpoint

	groqClient = openai.NewClientWithConfig(config)
	log.Println("[AI] Service initialized successfully with Groq")
	return nil
}

// EnrichedPlan holds the AI generated content
type EnrichedPlan struct {
	Description string   `json:"description"`
	Subtasks    []string `json:"subtasks"`
}

func GenerateEnrichedPlan(ctx context.Context, title string) (*EnrichedPlan, error) {
	if groqClient == nil {
		return nil, fmt.Errorf("AI Service not initialized or API configuration missing")
	}

	prompt := fmt.Sprintf(`
You are a world-class productivity coach.
**Task:** %s

Output a JSON object with:
1. "description": A motivating, markdown-formatted 2-3 sentence description. Use emojis.
2. "subtasks": An array of 3-5 strings, each being a short actionable step.

Example JSON:
{
  "description": "🚀 Let's crush this! ...",
  "subtasks": ["Step 1", "Step 2"]
}
Return ONLY the JSON.
`, title)

	resp, err := groqClient.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: "llama-3.3-70b-versatile",
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			MaxTokens: 500,
			ResponseFormat: &openai.ChatCompletionResponseFormat{
				Type: openai.ChatCompletionResponseFormatTypeJSONObject,
			},
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to generate plan with Groq: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from Groq")
	}

	var plan EnrichedPlan
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &plan); err != nil {
		// Fallback for non-JSON response (rare with llama-3, but possible)
		return &EnrichedPlan{
			Description: resp.Choices[0].Message.Content,
			Subtasks:    []string{},
		}, nil
	}

	return &plan, nil
}

func DecomposeTask(ctx context.Context, title, description string) ([]string, error) {
	if groqClient == nil {
		return nil, fmt.Errorf("AI Service not initialized or API key missing (GROQ_API_KEY)")
	}

	prompt := fmt.Sprintf(`
You are a productivity expert. Break down the following task into 3-5 small, actionable sub-tasks.
Each sub-task should be a short sentence (max 10 words).
Task Title: %s
Task Description: %s

Output only the list of sub-tasks, one per line, without numbers, bullets, or symbols.
`, title, description)

	resp, err := groqClient.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: "llama-3.3-70b-versatile", // High quality, fast model
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			MaxTokens: 200,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to generate subtasks with Groq: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from Groq")
	}

	rawText := resp.Choices[0].Message.Content
	var subtasks []string
	lines := strings.Split(rawText, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Remove common prefixes
		trimmed = strings.TrimPrefix(trimmed, "- ")
		trimmed = strings.TrimPrefix(trimmed, "* ")
		if trimmed != "" {
			subtasks = append(subtasks, trimmed)
		}
	}

	// Limit to 5 tasks
	if len(subtasks) > 5 {
		subtasks = subtasks[:5]
	}

	return subtasks, nil
}

// AnalysisContext holds the rich data needed for deep AI analysis
type AnalysisContext struct {
	Stats        map[string]int `json:"stats"`
	StaleTasks   []string       `json:"stale_tasks"`   // Tasks in progress > 3 days
	BlockedTasks []string       `json:"blocked_tasks"` // Titles of blocked tasks
	Velocity     int            `json:"velocity"`      // Tasks completed in last 7 days
	OverdueCount int            `json:"overdue_count"`
}

func GetHealthAdvice(ctx context.Context, context AnalysisContext) (string, error) {
	if groqClient == nil {
		return "", fmt.Errorf("AI Service not initialized or API configuration missing")
	}

	prompt := fmt.Sprintf(`
You are a senior technical project manager. Analyze the following project state and identify the SINGLE biggest risk or opportunity.
Data:
- Task Counts: Todo: %d, In Progress: %d, Blocked: %d, Done: %d
- Stale Tasks (In Progress > 3 days): %v
- Blocked Tasks: %v
- Overdue Tasks: %d
- Weekly Velocity: %d tasks/week

Heuristics:
1. If there are STALE tasks, they are the biggest risk. Advice: "You've been stuck on '[Task Name]' for a while. Break it down or ask for help."
2. If BLOCKED tasks exist, they kill flow. Advice: "Unblock '[Task Name]' immediately to restore momentum."
3. If In Progress > 3 and no stale tasks, warn about context switching.
4. If Velocity is high (>5) and no issues, praise the momentum.

Output ONE concise, punchy sentence (max 25 words). No intros.
`,
		context.Stats["todo"], context.Stats["in_progress"], context.Stats["blocked"], context.Stats["done"],
		context.StaleTasks,
		context.BlockedTasks,
		context.OverdueCount,
		context.Velocity,
	)

	resp, err := groqClient.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: "llama-3.3-70b-versatile",
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			MaxTokens: 100,
		},
	)

	if err != nil {
		return "", fmt.Errorf("failed to generate advice: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from AI")
	}

	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

// --- Chat & Quota Logic ---

func CheckQuota(ctx context.Context, userID primitive.ObjectID) (bool, int, error) {
	var user models.User
	err := db.Users.FindOne(ctx, bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		return false, 0, err
	}

	// Pro users have unlimited quota (or very high)
	if user.Plan == "pro" {
		return true, -1, nil
	}

	// Reset quota if needed
	if time.Since(user.LastQuotaReset) > QuotaResetPeriod {
		update := bson.M{
			"$set": bson.M{
				"ai_quota_used":    0,
				"last_quota_reset": time.Now(),
			},
		}
		_, err := db.Users.UpdateOne(ctx, bson.M{"_id": userID}, update)
		if err != nil {
			return false, 0, err
		}
		return true, FreeQuotaLimit, nil
	}

	if user.AIQuotaUsed >= FreeQuotaLimit {
		return false, 0, nil
	}

	return true, FreeQuotaLimit - user.AIQuotaUsed, nil
}

func IncrementQuota(ctx context.Context, userID primitive.ObjectID, amount int) error {
	var user models.User
	err := db.Users.FindOne(ctx, bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		return err
	}

	if user.Plan == "pro" {
		return nil
	}

	_, err = db.Users.UpdateOne(ctx, bson.M{"_id": userID}, bson.M{
		"$inc": bson.M{"ai_quota_used": amount},
	})
	return err
}

func Chat(ctx context.Context, userID primitive.ObjectID, message string) (string, error) {
	if groqClient == nil {
		return "", fmt.Errorf("AI Service not initialized")
	}

	// 1. Check Quota
	allowed, remaining, err := CheckQuota(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to check quota: %w", err)
	}
	if !allowed {
		return "", fmt.Errorf("quota_exceeded") // Handler should check for this specific error string
	}

	// 2. Get/Create Conversation
	coll := db.Database.Collection("ai_conversations")
	var conv models.Conversation
	err = coll.FindOne(ctx, bson.M{"user_id": userID}).Decode(&conv)

	if err == mongo.ErrNoDocuments {
		// Create new conversation
		conv = models.Conversation{
			ID:        primitive.NewObjectID(),
			UserID:    userID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Messages:  []models.Message{},
		}
		_, err = coll.InsertOne(ctx, conv)
		if err != nil {
			return "", fmt.Errorf("failed to create conversation: %w", err)
		}
	} else if err != nil {
		return "", fmt.Errorf("failed to fetch conversation: %w", err)
	}

	// 3. Prepare Context (last 10 messages)
	systemPrompt := `You are FlowBot, the AI heart of the Flowday OS. You are helpful, concise, and focused on helping the user stay in "Flow".

IMPORTANT FORMATTING RULES:
- Use **Markdown** for everything (headers, lists, bold).
- Break long text into short, readable paragraphs (max 2-3 sentences).
- Use **bullet_points** for lists.
- If the user asks for code or technical steps, use code blocks.
- If the user asks to create/plan tasks, output strictly a JSON array (wrapped in ` + "```json" + `) of objects with keys: 'title', 'priority' ('high', 'medium', 'low'). Do not output regular text if creating tasks.`

	aiMessages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
	}

	startIdx := 0
	if len(conv.Messages) > 10 {
		startIdx = len(conv.Messages) - 10
	}
	for i := startIdx; i < len(conv.Messages); i++ {
		msg := conv.Messages[i]
		aiMessages = append(aiMessages, openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Add current user message
	aiMessages = append(aiMessages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: message,
	})

	// 4. Call LLM
	resp, err := groqClient.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:    "llama-3.3-70b-versatile",
			Messages: aiMessages,
		},
	)
	if err != nil {
		return "", fmt.Errorf("AI provider error: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("empty response from AI")
	}

	assistantReply := resp.Choices[0].Message.Content
	cost := 1

	// Check for Smart Task Creation JSON
	// Expected format: ```json\n[{"title": "...", "priority": "high", ...}]\n```
	if strings.Contains(assistantReply, "```json") && strings.Contains(assistantReply, "priority") {
		// Extract JSON
		start := strings.Index(assistantReply, "```json") + 7
		end := strings.LastIndex(assistantReply, "```")
		if end > start {
			jsonStr := assistantReply[start:end]
			var tasks []models.Task
			if err := json.Unmarshal([]byte(jsonStr), &tasks); err == nil && len(tasks) > 0 {
				// We have valid tasks!

				// Re-check quota for higher cost
				if allowed, rem, _ := CheckQuota(ctx, userID); !allowed || (rem != -1 && rem < 3) {
					return "I'd love to organize that for you, but Smart Plan requires 3 credits. You're running low!", nil
				}

				createdCount := 0
				for _, t := range tasks {
					t.ProjectID = primitive.NilObjectID // Will be set below

					// Let's quickly fetch a project
					projects, _ := GetProjects(userID)
					if len(projects) > 0 {
						t.ProjectID = projects[0].ID
					} else {
						// Cannot create task without project
						continue
					}

					// Set defaults
					t.Status = "todo"
					if t.Priority == "" {
						t.Priority = "medium"
					}

					// Assuming CreateTask handles the rest (validation etc)
					if err := CreateTask(userID, &t); err == nil {
						createdCount++
					}
				}

				if createdCount > 0 {
					cost = 3
					assistantReply = fmt.Sprintf("✨ done! I've created %d prioritized tasks for you:\n\n", createdCount)
					for _, t := range tasks {
						assistantReply += fmt.Sprintf("- **%s** (%s)\n", t.Title, t.Priority)
					}
				}
			}
		}
	}

	// 5. Update Conversation History
	newMessages := []models.Message{
		{Role: "user", Content: message, Timestamp: time.Now()},
		{Role: "assistant", Content: assistantReply, Timestamp: time.Now()},
	}

	_, err = coll.UpdateOne(
		ctx,
		bson.M{"_id": conv.ID},
		bson.M{
			"$push": bson.M{"messages": bson.M{"$each": newMessages}},
			"$set":  bson.M{"updated_at": time.Now()},
		},
	)
	if err != nil {
		log.Printf("[AI] Failed to save history: %v", err)
		// Don't fail the request, user got the answer
	}

	// 6. Increment Quota
	if remaining != -1 { // -1 means pro/unlimited
		_ = IncrementQuota(ctx, userID, cost)
	}

	return assistantReply, nil
}

func GetHistory(ctx context.Context, userID primitive.ObjectID) ([]models.Message, error) {
	coll := db.Database.Collection("ai_conversations")
	var conv models.Conversation
	err := coll.FindOne(ctx, bson.M{"user_id": userID}).Decode(&conv)
	if err == mongo.ErrNoDocuments {
		return []models.Message{}, nil
	}
	if err != nil {
		return nil, err
	}
	return conv.Messages, nil
}
