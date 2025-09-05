package llm

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"go.uber.org/zap"
	"google.golang.org/genai"

	"github.com/satriahrh/arunika/server/domain/entities"
	"github.com/satriahrh/arunika/server/domain/repositories"
)

const (
	defaultModel          = "gemini-1.0-pro"
	defaultTemperature    = 0.7
	defaultTopP           = 0.8
	defaultTopK           = 40.0
	defaultMaxTokens      = 500
	defaultTimeoutSeconds = 30
)

// GeminiHardcodedConfig contains fixed configuration values that are not meant to be configurable
var GeminiHardcodedConfig = struct {
	// SystemPrompt is the fixed system prompt used for child-friendly interactions
	SystemPrompt string

	// SafetySettings are the fixed safety settings for content generation
	SafetySettings []*genai.SafetySetting

	// Fallbacks are the fixed fallback messages used when generation fails
	Fallbacks []string
}{
	SystemPrompt: `You are a friendly, caring AI companion for children. Your responses should be:
- Safe, appropriate, and educational for children ages 4-12
- Warm, encouraging, and supportive
- SHORT and simple - maximum 1 paragraph (2-3 sentences)
- Easy to understand with simple words
- Never scary, violent, or inappropriate
- Helpful in learning and development
- Always maintain a positive, nurturing tone

KEEP RESPONSES BRIEF: Children have short attention spans. Make every response concise, focused, and engaging.

Example: "That's such a wonderful story! I love how creative you are. Want to hear a secret about dragons?"

Remember: Keep responses short, simple, and age-appropriate.`,

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

	Fallbacks: []string{
		"I'm thinking really hard about that, can you ask me again?",
		"My brain needs a little rest, let's try talking about something else!",
		"I'm having trouble understanding right now, but I'm still here with you!",
		"Let me think about that... maybe you can help me by asking in a different way?",
		"I'm learning new things every day! Can you tell me more about what you're thinking?",
	},
}

// GeminiConfig holds configuration for the GeminiChatSession adapter
// This struct should be used to configure the GeminiChatSession adapter
// Required fields:
// - APIKey: Your Google AI API key
// Optional fields with defaults:
// - Model: The model to use (default: "gemini-1.0-pro")
// - Temperature: Controls randomness between 0 and 1 (default: 0.7)
// - TopP: Nucleus sampling parameter between 0 and 1 (default: 0.8)
// - TopK: Top-k sampling parameter (default: 40)
// - MaxOutputTokens: Maximum tokens in response (default: 500)
// - TimeoutSeconds: Timeout for API calls in seconds (default: 30)
type GeminiConfig struct {
	APIKey          string  // Required: Your Google AI API key
	Model           string  // Optional: The model to use
	Temperature     float32 // Optional: Controls randomness between 0 and 1
	TopP            float32 // Optional: Nucleus sampling parameter between 0 and 1
	TopK            float32 // Optional: Top-k sampling parameter
	MaxOutputTokens int     // Optional: Maximum tokens in response
	TimeoutSeconds  int     // Optional: Timeout for API calls in seconds
}

// GeminiLLM implements the LargeLanguageModel interface using Google's Gemini API
type GeminiLLM struct {
	client *genai.Client
	logger *zap.Logger
	config GeminiConfig
}

// NewGeminiLLM creates a new Gemini LLM instance
func NewGeminiLLM(logger *zap.Logger) (*GeminiLLM, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable is required")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	config := NewGeminiConfigFromEnv()
	// Set default model if not provided through environment
	if config.Model == "" {
		config.Model = "gemini-2.0-flash" // Using Flash model as requested
	}

	return &GeminiLLM{
		client: client,
		logger: logger,
		config: config,
	}, nil
}

// NewGeminiConfigFromEnv creates a new GeminiConfig from environment variables
// This is a helper function to simplify the creation of a properly configured GeminiConfig
func NewGeminiConfigFromEnv() GeminiConfig {
	// Read required API key
	apiKey := os.Getenv("GEMINI_API_KEY") // Using GEMINI_API_KEY to match the key used in NewGeminiLLM

	// Read optional parameters with defaults
	config := GeminiConfig{
		APIKey: apiKey,
		Model:  os.Getenv("GOOGLE_AI_MODEL"),
	}

	// System prompt is now hardcoded and not configurable via environment variables

	// Parse numeric values from environment
	if temperatureStr := os.Getenv("GOOGLE_AI_TEMPERATURE"); temperatureStr != "" {
		if temperature, err := strconv.ParseFloat(temperatureStr, 32); err == nil && temperature >= 0 && temperature <= 1 {
			config.Temperature = float32(temperature)
		}
	}

	if topPStr := os.Getenv("GOOGLE_AI_TOP_P"); topPStr != "" {
		if topP, err := strconv.ParseFloat(topPStr, 32); err == nil && topP >= 0 && topP <= 1 {
			config.TopP = float32(topP)
		}
	}

	if topKStr := os.Getenv("GOOGLE_AI_TOP_K"); topKStr != "" {
		if topK, err := strconv.ParseFloat(topKStr, 32); err == nil {
			config.TopK = float32(topK)
		}
	}

	if maxTokensStr := os.Getenv("GOOGLE_AI_MAX_OUTPUT_TOKENS"); maxTokensStr != "" {
		if maxTokens, err := strconv.Atoi(maxTokensStr); err == nil && maxTokens > 0 {
			config.MaxOutputTokens = maxTokens
		}
	}

	if timeoutStr := os.Getenv("GOOGLE_AI_TIMEOUT_SECONDS"); timeoutStr != "" {
		if timeout, err := strconv.Atoi(timeoutStr); err == nil && timeout > 0 {
			config.TimeoutSeconds = timeout
		}
	}

	return config
}

// GenerateChat creates a chat session with history
func (g *GeminiLLM) GenerateChat(ctx context.Context, history []entities.Message) (repositories.ChatSession, error) {
	return NewGeminiChatSession(g.client, g.config, g.logger, history)
}
