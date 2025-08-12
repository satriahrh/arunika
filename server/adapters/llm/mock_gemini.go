package llm

import (
	"context"
	"fmt"

	"github.com/satriahrh/arunika/server/repository"
)

// MockGeminiClient is a placeholder implementation for Gemini LLM
type MockGeminiClient struct{}

// NewMockGeminiClient creates a new mock Gemini client
func NewMockGeminiClient() repository.Llm {
	return &MockGeminiClient{}
}

// Generate implements repository.Llm
func (g *MockGeminiClient) Generate(prompt string) (string, error) {
	// Mock response for testing
	return "Halo! Saya adalah boneka pintar Arunika. Bagaimana kabar kamu hari ini?", nil
}

// GenerateChat implements repository.Llm
func (g *MockGeminiClient) GenerateChat(ctx context.Context, history []repository.ChatMessage) (repository.ChatSession, error) {
	return &MockGeminiChatSession{
		history: history,
	}, nil
}

// MockGeminiChatSession implements repository.ChatSession
type MockGeminiChatSession struct {
	history []repository.ChatMessage
}

// SendMessage implements repository.ChatSession
func (g *MockGeminiChatSession) SendMessage(ctx context.Context, message repository.ChatMessage) (repository.ChatMessage, error) {
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

	responseMessage := repository.ChatMessage{
		Role:    repository.DollRole,
		Content: response,
	}

	// Add response to history
	g.history = append(g.history, responseMessage)

	return responseMessage, nil
}

// History implements repository.ChatSession
func (g *MockGeminiChatSession) History() ([]repository.ChatMessage, error) {
	return g.history, nil
}
