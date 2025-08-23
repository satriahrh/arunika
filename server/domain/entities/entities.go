package entities

import (
	"errors"
	"time"
)

// Device represents a doll device
type Device struct {
	ID           string    `json:"id" db:"id"`
	SerialNumber string    `json:"serial_number" db:"serial_number"`
	Model        string    `json:"model" db:"model"`
	OwnerID      *string   `json:"owner_id" db:"owner_id"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// User represents a parent/user account
type User struct {
	ID        string    `json:"id" db:"id"`
	Email     string    `json:"email" db:"email"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Conversation represents a conversation session
type Conversation struct {
	ID        string     `json:"id" db:"id"`
	DeviceID  string     `json:"device_id" db:"device_id"`
	UserID    string     `json:"user_id" db:"user_id"`
	StartedAt time.Time  `json:"started_at" db:"started_at"`
	EndedAt   *time.Time `json:"ended_at" db:"ended_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

// Message represents a single message in a conversation
type Message struct {
	ID             string    `json:"id" db:"id"`
	ConversationID string    `json:"conversation_id" db:"conversation_id"`
	Type           string    `json:"type" db:"type"` // "user" or "assistant"
	Content        string    `json:"content" db:"content"`
	Timestamp      time.Time `json:"timestamp" db:"timestamp"`
}

// Domain validation methods
func (u *User) Validate() error {
	if u.Email == "" {
		return errors.New("email is required")
	}
	if u.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

func (d *Device) Validate() error {
	if d.SerialNumber == "" {
		return errors.New("serial number is required")
	}
	if d.Model == "" {
		return errors.New("model is required")
	}
	return nil
}
