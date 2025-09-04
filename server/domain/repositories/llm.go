package repositories

import (
	"context"

	"github.com/satriahrh/arunika/server/domain/entities"
)

// LargeLanguageModel abstracts any chat/LLM provider
type LargeLanguageModel interface {
	// GenerateChat creates a chat session with history
	GenerateChat(ctx context.Context, history []entities.Message) (ChatSession, error)
}

// ChatSession represents an ongoing conversation session
type ChatSession interface {
	SendMessage(ctx context.Context, message entities.Message) (entities.Message, error)
	History() ([]entities.Message, error)
}
