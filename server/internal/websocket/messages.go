package websocket

import (
	"encoding/json"
	"fmt"
	"time"
)

// MessageType defines the type of WebSocket message
type MessageType string

// Supported message types
const (
	MessageTypeAudioChunk    MessageType = "audio_chunk"
	MessageTypeAIResponse    MessageType = "ai_response"
	MessageTypePing          MessageType = "ping"
	MessageTypePong          MessageType = "pong"
	MessageTypeError         MessageType = "error"
	MessageTypeDeviceStatus  MessageType = "device_status"
	MessageTypeSystemMessage MessageType = "system_message"
	MessageTypeAuth          MessageType = "auth"
)

// BaseMessage defines the common structure for all WebSocket messages
type BaseMessage struct {
	Type      MessageType `json:"type" validate:"required"`
	Timestamp string      `json:"timestamp"`
	MessageID string      `json:"message_id,omitempty"`
}

// AudioChunkMessage represents an incoming audio chunk from device
type AudioChunkMessage struct {
	BaseMessage
	DeviceID     string `json:"device_id" validate:"required"`
	SessionID    string `json:"session_id" validate:"required"`
	AudioData    string `json:"audio_data" validate:"required"` // base64 encoded
	SampleRate   int    `json:"sample_rate" validate:"required,min=8000,max=48000"`
	Encoding     string `json:"encoding" validate:"required,oneof=pcm wav mp3 opus"`
	ChunkSeq     int    `json:"chunk_sequence" validate:"min=0"`
	IsFinal      bool   `json:"is_final"`
	Duration     int    `json:"duration_ms,omitempty"` // duration in milliseconds
	ContentType  string `json:"content_type,omitempty"`
}

// AIResponseMessage represents a response from the AI system
type AIResponseMessage struct {
	BaseMessage
	SessionID       string `json:"session_id" validate:"required"`
	Text            string `json:"response_text"`
	AudioData       string `json:"audio_data"` // base64 encoded
	Emotion         string `json:"emotion,omitempty"`
	Confidence      float64 `json:"confidence,omitempty"`
	ProcessingTime  int64   `json:"processing_time_ms,omitempty"`
	ConversationID  string  `json:"conversation_id,omitempty"`
}

// PingMessage represents a ping message for connection health check
type PingMessage struct {
	BaseMessage
	Data string `json:"data,omitempty"`
}

// PongMessage represents a pong response
type PongMessage struct {
	BaseMessage
	Data string `json:"data,omitempty"`
}

// ErrorMessage represents an error response
type ErrorMessage struct {
	BaseMessage
	Code    string `json:"error_code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// DeviceStatusMessage represents device status updates
type DeviceStatusMessage struct {
	BaseMessage
	DeviceID     string                 `json:"device_id" validate:"required"`
	Status       string                 `json:"status" validate:"required,oneof=online offline sleeping error"`
	BatteryLevel int                    `json:"battery_level,omitempty" validate:"min=0,max=100"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// SystemMessage represents system-wide messages and announcements
type SystemMessage struct {
	BaseMessage
	Priority string `json:"priority" validate:"required,oneof=low normal high critical"`
	Title    string `json:"title" validate:"required"`
	Content  string `json:"content" validate:"required"`
	Actions  []MessageAction `json:"actions,omitempty"`
}

// MessageAction represents an action button in system messages
type MessageAction struct {
	ID    string `json:"id" validate:"required"`
	Label string `json:"label" validate:"required"`
	URL   string `json:"url,omitempty"`
	Type  string `json:"type" validate:"required,oneof=button link dismiss"`
}

// AuthMessage represents authentication-related messages
type AuthMessage struct {
	BaseMessage
	Action    string `json:"action" validate:"required,oneof=authenticate refresh logout"`
	Token     string `json:"token,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

// MessageValidator provides validation for WebSocket messages
type MessageValidator struct{}

// NewMessageValidator creates a new message validator
func NewMessageValidator() *MessageValidator {
	return &MessageValidator{}
}

// ValidateMessage validates an incoming message
func (v *MessageValidator) ValidateMessage(messageBytes []byte) (interface{}, error) {
	// First parse as base message to get type
	var base BaseMessage
	if err := json.Unmarshal(messageBytes, &base); err != nil {
		return nil, fmt.Errorf("invalid JSON format: %w", err)
	}

	// Add timestamp if missing
	if base.Timestamp == "" {
		base.Timestamp = time.Now().Format(time.RFC3339)
	}

	// Validate specific message type
	switch base.Type {
	case MessageTypeAudioChunk:
		var msg AudioChunkMessage
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			return nil, fmt.Errorf("invalid audio chunk message: %w", err)
		}
		if err := v.validateAudioChunk(&msg); err != nil {
			return nil, err
		}
		return &msg, nil

	case MessageTypePing:
		var msg PingMessage
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			return nil, fmt.Errorf("invalid ping message: %w", err)
		}
		return &msg, nil

	case MessageTypeDeviceStatus:
		var msg DeviceStatusMessage
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			return nil, fmt.Errorf("invalid device status message: %w", err)
		}
		if err := v.validateDeviceStatus(&msg); err != nil {
			return nil, err
		}
		return &msg, nil

	case MessageTypeAuth:
		var msg AuthMessage
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			return nil, fmt.Errorf("invalid auth message: %w", err)
		}
		return &msg, nil

	default:
		return nil, fmt.Errorf("unsupported message type: %s", base.Type)
	}
}

// validateAudioChunk validates audio chunk message fields
func (v *MessageValidator) validateAudioChunk(msg *AudioChunkMessage) error {
	if msg.DeviceID == "" {
		return fmt.Errorf("device_id is required")
	}
	if msg.SessionID == "" {
		return fmt.Errorf("session_id is required")
	}
	if msg.AudioData == "" {
		return fmt.Errorf("audio_data is required")
	}
	if msg.SampleRate < 8000 || msg.SampleRate > 48000 {
		return fmt.Errorf("sample_rate must be between 8000 and 48000")
	}
	if msg.Encoding == "" {
		return fmt.Errorf("encoding is required")
	}
	
	// Validate encoding values
	validEncodings := map[string]bool{
		"pcm": true, "wav": true, "mp3": true, "opus": true,
	}
	if !validEncodings[msg.Encoding] {
		return fmt.Errorf("encoding must be one of: pcm, wav, mp3, opus")
	}
	
	return nil
}

// validateDeviceStatus validates device status message fields
func (v *MessageValidator) validateDeviceStatus(msg *DeviceStatusMessage) error {
	if msg.DeviceID == "" {
		return fmt.Errorf("device_id is required")
	}
	if msg.Status == "" {
		return fmt.Errorf("status is required")
	}
	
	// Validate status values
	validStatuses := map[string]bool{
		"online": true, "offline": true, "sleeping": true, "error": true,
	}
	if !validStatuses[msg.Status] {
		return fmt.Errorf("status must be one of: online, offline, sleeping, error")
	}
	
	if msg.BatteryLevel < 0 || msg.BatteryLevel > 100 {
		return fmt.Errorf("battery_level must be between 0 and 100")
	}
	
	return nil
}

// CreateErrorMessage creates a standardized error message
func CreateErrorMessage(code, message, details string) *ErrorMessage {
	return &ErrorMessage{
		BaseMessage: BaseMessage{
			Type:      MessageTypeError,
			Timestamp: time.Now().Format(time.RFC3339),
		},
		Code:    code,
		Message: message,
		Details: details,
	}
}

// CreatePongMessage creates a pong response message
func CreatePongMessage(data string) *PongMessage {
	return &PongMessage{
		BaseMessage: BaseMessage{
			Type:      MessageTypePong,
			Timestamp: time.Now().Format(time.RFC3339),
		},
		Data: data,
	}
}

// CreateSystemMessage creates a system message
func CreateSystemMessage(priority, title, content string) *SystemMessage {
	return &SystemMessage{
		BaseMessage: BaseMessage{
			Type:      MessageTypeSystemMessage,
			Timestamp: time.Now().Format(time.RFC3339),
		},
		Priority: priority,
		Title:    title,
		Content:  content,
	}
}