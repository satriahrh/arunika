package llm

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/genai"

	"github.com/satriahrh/arunika/server/domain/repositories"
)

// GeminiChatSession implements the ChatSession interface
type GeminiChatSession struct {
	client  *genai.Client
	model   string
	history []*genai.Content
}

// NewGeminiChatSession creates a new chat session with history
func NewGeminiChatSession(client *genai.Client, model string, history []repositories.ChatMessage) (*GeminiChatSession, error) {
	// Convert repository format to Gemini format
	geminiHistory := convertRepositoryToGeminiFormat(history)

	return &GeminiChatSession{
		client:  client,
		model:   model,
		history: geminiHistory,
	}, nil
}

// SendMessage sends a message and gets a response, updating the history
func (s *GeminiChatSession) SendMessage(ctx context.Context, message repositories.ChatMessage) (repositories.ChatMessage, error) {
	// Prepare contents for API call (system prompt + history + current message)
	var contents []*genai.Content

	// Add system instruction as the first message
	systemPrompt := `You are a friendly, caring AI companion for children. Your responses should be:
- Safe, appropriate, and educational for children ages 4-12
- Warm, encouraging, and supportive
- Simple to understand but engaging
- Never scary, violent, or inappropriate
- Helpful in learning and development
- Always maintain a positive, nurturing tone

Remember to keep responses conversational and age-appropriate.`

	contents = append(contents, genai.NewContentFromText(systemPrompt, genai.RoleUser))

	// Add existing history (already in Gemini format)
	contents = append(contents, s.history...)

	// Add the current user message to the contents for this API call
	userContent := genai.NewContentFromText(message.Content, genai.RoleUser)
	contents = append(contents, userContent)

	// Configure settings for child-friendly responses using the correct API
	config := &genai.GenerateContentConfig{
		SafetySettings: []*genai.SafetySetting{
			{
				Category:  "HARM_CATEGORY_HARASSMENT",
				Threshold: "BLOCK_MEDIUM_AND_ABOVE",
			},
			{
				Category:  "HARM_CATEGORY_HATE_SPEECH",
				Threshold: "BLOCK_MEDIUM_AND_ABOVE",
			},
			{
				Category:  "HARM_CATEGORY_SEXUALLY_EXPLICIT",
				Threshold: "BLOCK_LOW_AND_ABOVE",
			},
			{
				Category:  "HARM_CATEGORY_DANGEROUS_CONTENT",
				Threshold: "BLOCK_LOW_AND_ABOVE",
			},
		},
		Temperature:     genai.Ptr(float32(0.7)), // Slightly creative but controlled
		TopP:            genai.Ptr(float32(0.8)),
		TopK:            genai.Ptr(float32(40)),
		MaxOutputTokens: 500, // Reasonable response length
	}

	// Add timeout to context if not already set
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Add retry logic
	var response *genai.GenerateContentResponse
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		response, err = s.client.Models.GenerateContent(ctx, s.model, contents, config)
		if err == nil {
			break
		}

		s.logger.Warn("Failed to generate content, retrying",
			zap.Int("attempt", attempt+1),
			zap.Error(err))

		if attempt < 2 {
			time.Sleep(time.Duration(attempt+1) * time.Second)
		}
	}

	if err != nil {
		s.logger.Error("Failed to send message in chat session", zap.Error(err))
		return s.createFallbackResponse(), nil // Return fallback instead of error
	}

	if len(response.Candidates) == 0 || len(response.Candidates[0].Content.Parts) == 0 {
		s.logger.Warn("No content generated in chat session")
		return s.createFallbackResponse(), nil
	}

	// Extract text from the response
	var responseText string
	for _, part := range response.Candidates[0].Content.Parts {
		if part.Text != "" {
			responseText += part.Text
		}
	}

	if responseText == "" {
		s.logger.Warn("Empty response in chat session")
		return s.createFallbackResponse(), nil
	}

	// Create response message and add both user message and response to history
	responseContent := genai.NewContentFromText(responseText, genai.RoleModel)

	// Add both messages to history
	s.history = append(s.history, userContent, responseContent)

	responseMessage := repositories.ChatMessage{
		Role:    repositories.DollRole,
		Content: responseText,
	}

	s.logger.Info("Chat session message processed",
		zap.String("user_message", message.Content[:min(50, len(message.Content))]),
		zap.String("response_preview", responseText[:min(50, len(responseText))]),
		zap.Int("history_length", len(s.history)))

	return responseMessage, nil
}

// History returns the current conversation history
func (s *GeminiChatSession) History() ([]repositories.ChatMessage, error) {
	return convertGeminiToRepositoryFormat(s.history), nil
}

// createFallbackResponse creates a fallback response message
func (s *GeminiChatSession) createFallbackResponse() repositories.ChatMessage {
	fallbacks := []string{
		"I'm thinking really hard about that, can you ask me again?",
		"My brain needs a little rest, let's try talking about something else!",
		"I'm having trouble understanding right now, but I'm still here with you!",
		"Let me think about that... maybe you can help me by asking in a different way?",
		"I'm learning new things every day! Can you tell me more about what you're thinking?",
	}

	// Simple pseudo-random selection based on current time
	index := int(time.Now().UnixNano()) % len(fallbacks)

	fallbackMessage := repositories.ChatMessage{
		Role:    repositories.DollRole,
		Content: fallbacks[index],
	}

	// Add fallback to history as Gemini content
	fallbackContent := genai.NewContentFromText(fallbacks[index], genai.RoleModel)
	s.history = append(s.history, fallbackContent)

	return fallbackMessage
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// convertRepositoryToGeminiFormat converts repository messages to Gemini format
func convertRepositoryToGeminiFormat(messages []repositories.ChatMessage) []*genai.Content {
	var contents []*genai.Content

	for _, msg := range messages {
		var role genai.Role
		switch msg.Role {
		case repositories.UserRole:
			role = genai.RoleUser
		case repositories.DollRole:
			role = genai.RoleModel
		case repositories.SystemRole:
			role = genai.RoleUser // Treat system messages as user messages in Gemini
		default:
			role = genai.RoleUser // Default to user role
		}

		contents = append(contents, genai.NewContentFromText(msg.Content, role))
	}

	return contents
}

// convertGeminiToRepositoryFormat converts Gemini content to repository messages
func convertGeminiToRepositoryFormat(contents []*genai.Content) []repositories.ChatMessage {
	var messages []repositories.ChatMessage

	for _, content := range contents {
		var role repositories.Role
		switch content.Role {
		case genai.RoleUser:
			role = repositories.UserRole
		case genai.RoleModel:
			role = repositories.DollRole
		default:
			role = repositories.UserRole // Default to user role
		}

		// Extract text from parts (limiting to text only as specified)
		var text string
		for _, part := range content.Parts {
			if part.Text != "" {
				text += part.Text
			}
		}

		if text != "" {
			messages = append(messages, repositories.ChatMessage{
				Role:    role,
				Content: text,
			})
		}
	}

	return messages
}
