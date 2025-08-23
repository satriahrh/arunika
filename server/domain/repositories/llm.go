package repositories

import "context"

// LargeLanguageModel abstracts any chat/LLM provider
type LargeLanguageModel interface {
	// Generate takes a user prompt and returns the model's reply
	Generate(prompt string) (string, error)
	// GenerateChat creates a chat session with history
	GenerateChat(ctx context.Context, history []ChatMessage) (ChatSession, error)
}

// ChatSession represents an ongoing conversation session
type ChatSession interface {
	SendMessage(ctx context.Context, message ChatMessage) (ChatMessage, error)
	History() ([]ChatMessage, error)
}

// ChatMessage represents a single message in a conversation
type ChatMessage struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

// Role defines the type of message sender
type Role string

const (
	UserRole   Role = "user"
	DollRole   Role = "doll"
	SystemRole Role = "system"
)
