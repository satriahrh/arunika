package llm

import (
	"context"
	"fmt"

	"github.com/satriahrh/arunika/server/domain/repositories"
)

// MockGeminiClient is a placeholder implementation for Gemini LLM
type MockGeminiClient struct{}

// NewMockGeminiClient creates a new mock Gemini client
func NewMockGeminiClient() repositories.LargeLanguageModel {
	return &MockGeminiClient{}
}

// Generate implements repositories.LargeLanguageModel
func (g *MockGeminiClient) Generate(prompt string) (string, error) {
	// Mock response for testing
	return "Halo! Saya adalah boneka pintar Arunika. Bagaimana kabar kamu hari ini?", nil
}

// GenerateChat implements repositories.LargeLanguageModel
func (g *MockGeminiClient) GenerateChat(ctx context.Context, history []repositories.ChatMessage) (repositories.ChatSession, error) {
	return &MockGeminiChatSession{
		history: history,
	}, nil
}

// MockGeminiChatSession implements repositories.ChatSession
type MockGeminiChatSession struct {
	history []repositories.ChatMessage
}

// SendMessage implements repositories.ChatSession
func (g *MockGeminiChatSession) SendMessage(ctx context.Context, message repositories.ChatMessage) (repositories.ChatMessage, error) {
	// Add user message to history
	g.history = append(g.history, message)

	// Generate mock response
	var response string
	switch {
	case len(message.Content) > 0:
		response = fmt.Sprintf("Terima kasih sudah bercerita! Saya senang mendengar '%s'. Apa lagi yang ingin kamu ceritakan?", message.Content)
	default:
		response = "Halo! Saya adalah boneka pintar Arunika. Apa yang ingin kamu ceritakan hari ini?"
	}

	responseMessage := repositories.ChatMessage{
		Role:    repositories.DollRole,
		Content: response,
	}

	// Add response to history
	g.history = append(g.history, responseMessage)

	return responseMessage, nil
}

// History implements repositories.ChatSession
func (g *MockGeminiChatSession) History() ([]repositories.ChatMessage, error) {
	return g.history, nil
}
