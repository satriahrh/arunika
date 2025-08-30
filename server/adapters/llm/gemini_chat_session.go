package llm

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"google.golang.org/genai"

	"github.com/satriahrh/arunika/server/domain/repositories"
)

// GeminiChatSession implements the ChatSession interface
type GeminiChatSession struct {
	client          *genai.Client
	logger          *zap.Logger
	model           string
	temperature     float32
	topP            float32
	topK            float32
	maxOutputTokens int
	timeoutSeconds  int
	safetySettings  []*genai.SafetySetting
	systemPrompt    string
	history         []*genai.Content
}

// ValidateGeminiConfig validates the GeminiConfig
func ValidateGeminiConfig(config GeminiConfig) error {
	if config.APIKey == "" {
		return fmt.Errorf("Google AI API key is required")
	}

	// Validate temperature is in the valid range
	if config.Temperature != 0 && (config.Temperature < 0 || config.Temperature > 1) {
		return fmt.Errorf("temperature must be between 0 and 1, got %f", config.Temperature)
	}

	// Validate topP is in the valid range
	if config.TopP != 0 && (config.TopP < 0 || config.TopP > 1) {
		return fmt.Errorf("topP must be between 0 and 1, got %f", config.TopP)
	}

	// Validate topK is positive if specified
	if config.TopK < 0 {
		return fmt.Errorf("topK must be positive, got %f", config.TopK)
	}

	// Validate timeout is reasonable if specified
	if config.TimeoutSeconds < 0 {
		return fmt.Errorf("timeout must be positive, got %d", config.TimeoutSeconds)
	}

	return nil
}

// NewGeminiChatSession creates a new chat session with config and history
func NewGeminiChatSession(client *genai.Client, config GeminiConfig, logger *zap.Logger, history []repositories.ChatMessage) (*GeminiChatSession, error) {
	// Validate required configuration
	if err := ValidateGeminiConfig(config); err != nil {
		return nil, err
	}

	// Convert repository format to Gemini format
	geminiHistory := convertRepositoryToGeminiFormat(history)

	// Apply defaults where needed
	model := config.Model
	if model == "" {
		model = defaultModel
		logger.Info("Using default model", zap.String("model", model))
	}

	temperature := config.Temperature
	if temperature == 0 {
		temperature = float32(defaultTemperature)
		logger.Info("Using default temperature", zap.Float32("temperature", temperature))
	}

	topP := config.TopP
	if topP == 0 {
		topP = float32(defaultTopP)
		logger.Info("Using default topP", zap.Float32("topP", topP))
	}

	topK := config.TopK
	if topK == 0 {
		topK = float32(defaultTopK)
		logger.Info("Using default topK", zap.Float32("topK", topK))
	}

	maxOutputTokens := config.MaxOutputTokens
	if maxOutputTokens == 0 {
		maxOutputTokens = defaultMaxTokens
		logger.Info("Using default maxOutputTokens", zap.Int("maxOutputTokens", maxOutputTokens))
	}

	timeoutSeconds := config.TimeoutSeconds
	if timeoutSeconds == 0 {
		timeoutSeconds = defaultTimeoutSeconds
		logger.Info("Using default timeoutSeconds", zap.Int("timeoutSeconds", timeoutSeconds))
	}

	// Use the hardcoded safety settings
	logger.Info("Using hardcoded safety settings and system prompt")

	return &GeminiChatSession{
		client:          client,
		logger:          logger,
		model:           model,
		temperature:     temperature,
		topP:            topP,
		topK:            topK,
		maxOutputTokens: maxOutputTokens,
		timeoutSeconds:  timeoutSeconds,
		safetySettings:  GeminiHardcodedConfig.SafetySettings,
		systemPrompt:    GeminiHardcodedConfig.SystemPrompt,
		history:         geminiHistory,
	}, nil
}

// SendMessage sends a message and gets a response, updating the history
func (s *GeminiChatSession) SendMessage(ctx context.Context, message repositories.ChatMessage) (repositories.ChatMessage, error) {
	// Prepare contents for API call (system prompt + history + current message)
	var contents []*genai.Content

	// Add system instruction as the first message
	contents = append(contents, genai.NewContentFromText(s.systemPrompt, genai.RoleUser))

	// Add existing history (already in Gemini format)
	contents = append(contents, s.history...)

	// Add the current user message to the contents for this API call
	userContent := genai.NewContentFromText(message.Content, genai.RoleUser)
	contents = append(contents, userContent)

	// Configure settings using the session's configuration
	config := &genai.GenerateContentConfig{
		SafetySettings:  s.safetySettings,
		Temperature:     genai.Ptr(s.temperature),
		TopP:            genai.Ptr(s.topP),
		TopK:            genai.Ptr(s.topK),
		MaxOutputTokens: int32(s.maxOutputTokens),
	}

	// Add timeout to context if not already set
	ctx, cancel := context.WithTimeout(ctx, time.Duration(s.timeoutSeconds)*time.Second)
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
	// Simple pseudo-random selection based on current time
	fallbacks := GeminiHardcodedConfig.Fallbacks
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
