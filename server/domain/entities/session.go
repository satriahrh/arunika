package entities

import (
	"time"
)

type MessageMetadata struct {
	TranscriptionConfidence float64 `bson:"transcription_confidence" json:"transcription_confidence"`
	Emotion                 string  `bson:"emotion" json:"emotion"`
}

type Message struct {
	Timestamp  time.Time       `bson:"timestamp" json:"timestamp"`
	Role       Role            `bson:"role" json:"role"`
	Content    string          `bson:"content" json:"content"`
	DurationMs int64           `bson:"duration_ms" json:"duration_ms"`
	Metadata   MessageMetadata `bson:"metadata" json:"metadata"`
}

// Role defines the type of message sender
type Role string

const (
	UserRole   Role = "user"
	DollRole   Role = "doll"
	SystemRole Role = "system"
)

type SessionMetadata struct {
	Language        string                 `bson:"language" json:"language"`
	UserPreferences map[string]interface{} `bson:"user_preferences" json:"user_preferences"`
}

type Session struct {
	ID            string          `bson:"_id,omitempty" json:"id"`
	DeviceID      string          `bson:"device_id" json:"device_id"`
	CreatedAt     time.Time       `bson:"created_at" json:"created_at"`
	LastMessageAt time.Time       `bson:"last_message_at" json:"last_message_at"`
	Messages      []Message       `bson:"messages" json:"messages"`
	Metadata      SessionMetadata `bson:"metadata" json:"metadata"`
}

func (s *Session) AddMessage(saveCommand func(s *Session) error, messages ...Message) error {
	s.Messages = append(s.Messages, messages...)
	s.LastMessageAt = messages[len(messages)-1].Timestamp
	return saveCommand(s)
}

func (s *Session) CanContinueThisSession() bool {
	return time.Since(s.LastMessageAt) < 15*time.Minute
}
