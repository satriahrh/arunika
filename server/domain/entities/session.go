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
	Role       string          `bson:"role" json:"role"`
	Content    string          `bson:"content" json:"content"`
	DurationMs int             `bson:"duration_ms" json:"duration_ms"`
	Metadata   MessageMetadata `bson:"metadata" json:"metadata"`
}

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
