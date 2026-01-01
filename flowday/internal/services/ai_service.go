package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/sashabaranov/go-openai"
)

var (
	groqClient *openai.Client
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

func GenerateTaskPlan(ctx context.Context, title string) (string, error) {
	if groqClient == nil {
		return "", fmt.Errorf("AI Service not initialized or API configuration missing")
	}

	prompt := fmt.Sprintf(`
You are a productivity expert. Create a concise, professional execution plan for the following task:
Task: %s

Formatting:
- Use Markdown.
- Start with a brief overview (1-2 sentences).
- List 3-5 clear bullet points or a short checklist.
- Be concise (max 100 words total).
- Don't use greetings or conversational filler.
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
			MaxTokens: 400,
		},
	)

	if err != nil {
		return "", fmt.Errorf("failed to generate plan with Groq: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from Groq")
	}

	return resp.Choices[0].Message.Content, nil
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
