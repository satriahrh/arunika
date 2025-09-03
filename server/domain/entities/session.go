package entities

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SessionStatus represents the status of a session
type SessionStatus string

const (
	SessionStatusActive     SessionStatus = "active"
	SessionStatusExpired    SessionStatus = "expired"
	SessionStatusTerminated SessionStatus = "terminated"
)

// MessageRole represents the role of a message sender
type MessageRole string

const (
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
)

// SessionMessage represents a message within a session
type SessionMessage struct {
	Timestamp              time.Time              `json:"timestamp" bson:"timestamp"`
	Role                   MessageRole            `json:"role" bson:"role"`
	Content                string                 `json:"content" bson:"content"`
	DurationMs             int                    `json:"duration_ms" bson:"duration_ms"`
	Metadata               SessionMessageMetadata `json:"metadata" bson:"metadata"`
}

// SessionMessageMetadata contains additional metadata for a message
type SessionMessageMetadata struct {
	TranscriptionConfidence *float64 `json:"transcription_confidence,omitempty" bson:"transcription_confidence,omitempty"`
	Emotion                 *string  `json:"emotion,omitempty" bson:"emotion,omitempty"`
}

// SessionMetadata contains session-level metadata
type SessionMetadata struct {
	Language        string                 `json:"language" bson:"language"`
	UserPreferences map[string]interface{} `json:"user_preferences" bson:"user_preferences"`
}

// Session represents a conversation session between a device and the system
type Session struct {
	ID            primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	DeviceID      string             `json:"device_id" bson:"device_id"`
	CreatedAt     time.Time          `json:"created_at" bson:"created_at"`
	LastActiveAt  time.Time          `json:"last_active_at" bson:"last_active_at"`
	LastMessageAt *time.Time         `json:"last_message_at" bson:"last_message_at"`
	ExpiresAt     time.Time          `json:"expires_at" bson:"expires_at"`
	Status        SessionStatus      `json:"status" bson:"status"`
	Messages      []SessionMessage   `json:"messages" bson:"messages"`
	Metadata      SessionMetadata    `json:"metadata" bson:"metadata"`
}

// NewSession creates a new session for a device
func NewSession(deviceID string) *Session {
	now := time.Now()
	return &Session{
		ID:           primitive.NewObjectID(),
		DeviceID:     deviceID,
		CreatedAt:    now,
		LastActiveAt: now,
		ExpiresAt:    now.Add(24 * time.Hour), // Default 24 hour expiration
		Status:       SessionStatusActive,
		Messages:     make([]SessionMessage, 0),
		Metadata: SessionMetadata{
			Language:        "id-ID", // Default to Indonesian
			UserPreferences: make(map[string]interface{}),
		},
	}
}

// AddMessage adds a new message to the session
func (s *Session) AddMessage(role MessageRole, content string, durationMs int, metadata SessionMessageMetadata) {
	now := time.Now()
	message := SessionMessage{
		Timestamp:  now,
		Role:       role,
		Content:    content,
		DurationMs: durationMs,
		Metadata:   metadata,
	}
	
	s.Messages = append(s.Messages, message)
	s.LastMessageAt = &now
	s.UpdateLastActive()
}

// UpdateLastActive updates the last active timestamp and extends expiration
func (s *Session) UpdateLastActive() {
	s.LastActiveAt = time.Now()
	// Extend expiration by 24 hours from last activity
	s.ExpiresAt = s.LastActiveAt.Add(24 * time.Hour)
}

// IsExpired checks if the session has expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt) || s.Status != SessionStatusActive
}

// ShouldCreateNewSession checks if a new session should be created based on the 30-minute rule
func (s *Session) ShouldCreateNewSession() bool {
	if s.LastMessageAt == nil {
		return false // No messages yet, can continue this session
	}
	
	// Create new session if last message was more than 30 minutes ago
	return time.Since(*s.LastMessageAt) > 30*time.Minute
}

// Terminate marks the session as terminated
func (s *Session) Terminate() {
	s.Status = SessionStatusTerminated
	s.UpdateLastActive()
}

// Expire marks the session as expired
func (s *Session) Expire() {
	s.Status = SessionStatusExpired
}

// GetConversationHistory returns the conversation messages for LLM context
func (s *Session) GetConversationHistory() []SessionMessage {
	return s.Messages
}

// Validate validates the session data
func (s *Session) Validate() error {
	if s.DeviceID == "" {
		return errors.New("device_id is required")
	}
	
	if s.Status != SessionStatusActive && s.Status != SessionStatusExpired && s.Status != SessionStatusTerminated {
		return errors.New("invalid session status")
	}
	
	return nil
}