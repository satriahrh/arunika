package llm

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"
	"google.golang.org/genai"

	"github.com/satriahrh/arunika/server/domain/repositories"
)

// GeminiLLM implements the LargeLanguageModel interface using Google's Gemini API
type GeminiLLM struct {
	client *genai.Client
	logger *zap.Logger
	model  string
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

	return &GeminiLLM{
		client: client,
		logger: logger,
		model:  "gemini-2.0-flash", // Using Flash model as requested
	}, nil
}

// GenerateChat creates a chat session with history
func (g *GeminiLLM) GenerateChat(ctx context.Context, history []repositories.ChatMessage) (repositories.ChatSession, error) {
	return NewGeminiChatSession(g.client, g.logger, g.model, history)
}
