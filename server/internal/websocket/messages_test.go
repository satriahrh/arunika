package websocket

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestMessageValidator_ValidateAudioChunk(t *testing.T) {
	validator := NewMessageValidator()

	tests := []struct {
		name    string
		message string
		wantErr bool
	}{
		{
			name: "valid audio chunk",
			message: `{
				"type": "audio_chunk",
				"device_id": "test-device-1",
				"session_id": "session-123",
				"audio_data": "SGVsbG8gV29ybGQ=",
				"sample_rate": 16000,
				"encoding": "pcm",
				"chunk_sequence": 1,
				"is_final": false
			}`,
			wantErr: false,
		},
		{
			name: "missing device_id",
			message: `{
				"type": "audio_chunk",
				"session_id": "session-123",
				"audio_data": "SGVsbG8gV29ybGQ=",
				"sample_rate": 16000,
				"encoding": "pcm"
			}`,
			wantErr: true,
		},
		{
			name: "invalid sample rate",
			message: `{
				"type": "audio_chunk",
				"device_id": "test-device-1",
				"session_id": "session-123",
				"audio_data": "SGVsbG8gV29ybGQ=",
				"sample_rate": 100000,
				"encoding": "pcm"
			}`,
			wantErr: true,
		},
		{
			name: "invalid encoding",
			message: `{
				"type": "audio_chunk",
				"device_id": "test-device-1",
				"session_id": "session-123",
				"audio_data": "SGVsbG8gV29ybGQ=",
				"sample_rate": 16000,
				"encoding": "invalid"
			}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validator.ValidateMessage([]byte(tt.message))
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMessageValidator_ValidatePing(t *testing.T) {
	validator := NewMessageValidator()

	message := `{
		"type": "ping",
		"data": "test-ping"
	}`

	result, err := validator.ValidateMessage([]byte(message))
	if err != nil {
		t.Errorf("ValidateMessage() error = %v", err)
	}

	pingMsg, ok := result.(*PingMessage)
	if !ok {
		t.Errorf("Expected *PingMessage, got %T", result)
	}

	if pingMsg.Data != "test-ping" {
		t.Errorf("Expected data 'test-ping', got '%s'", pingMsg.Data)
	}
}

func TestMessageValidator_ValidateDeviceStatus(t *testing.T) {
	validator := NewMessageValidator()

	tests := []struct {
		name    string
		message string
		wantErr bool
	}{
		{
			name: "valid device status",
			message: `{
				"type": "device_status",
				"device_id": "test-device-1",
				"status": "online",
				"battery_level": 85
			}`,
			wantErr: false,
		},
		{
			name: "invalid battery level",
			message: `{
				"type": "device_status",
				"device_id": "test-device-1",
				"status": "online",
				"battery_level": 150
			}`,
			wantErr: true,
		},
		{
			name: "invalid status",
			message: `{
				"type": "device_status",
				"device_id": "test-device-1",
				"status": "invalid_status"
			}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validator.ValidateMessage([]byte(tt.message))
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreateErrorMessage(t *testing.T) {
	code := "TEST_ERROR"
	message := "Test error message"
	details := "Test error details"

	errorMsg := CreateErrorMessage(code, message, details)

	if errorMsg.Type != MessageTypeError {
		t.Errorf("Expected type %s, got %s", MessageTypeError, errorMsg.Type)
	}
	if errorMsg.Code != code {
		t.Errorf("Expected code %s, got %s", code, errorMsg.Code)
	}
	if errorMsg.Message != message {
		t.Errorf("Expected message %s, got %s", message, errorMsg.Message)
	}
	if errorMsg.Details != details {
		t.Errorf("Expected details %s, got %s", details, errorMsg.Details)
	}

	// Verify timestamp is recent
	timestamp, err := time.Parse(time.RFC3339, errorMsg.Timestamp)
	if err != nil {
		t.Errorf("Invalid timestamp format: %v", err)
	}
	if time.Since(timestamp) > time.Second {
		t.Errorf("Timestamp is not recent: %s", errorMsg.Timestamp)
	}
}

func TestCreatePongMessage(t *testing.T) {
	data := "test-pong-data"
	pongMsg := CreatePongMessage(data)

	if pongMsg.Type != MessageTypePong {
		t.Errorf("Expected type %s, got %s", MessageTypePong, pongMsg.Type)
	}
	if pongMsg.Data != data {
		t.Errorf("Expected data %s, got %s", data, pongMsg.Data)
	}

	// Verify timestamp is recent
	timestamp, err := time.Parse(time.RFC3339, pongMsg.Timestamp)
	if err != nil {
		t.Errorf("Invalid timestamp format: %v", err)
	}
	if time.Since(timestamp) > time.Second {
		t.Errorf("Timestamp is not recent: %s", pongMsg.Timestamp)
	}
}

func TestMessageSerialization(t *testing.T) {
	// Test that all message types can be properly serialized and deserialized
	tests := []struct {
		name    string
		message interface{}
	}{
		{
			name: "AudioChunkMessage",
			message: &AudioChunkMessage{
				BaseMessage: BaseMessage{
					Type:      MessageTypeAudioChunk,
					Timestamp: time.Now().Format(time.RFC3339),
				},
				DeviceID:   "test-device",
				SessionID:  "test-session",
				AudioData:  "SGVsbG8=",
				SampleRate: 16000,
				Encoding:   "pcm",
				ChunkSeq:   1,
				IsFinal:    false,
			},
		},
		{
			name: "AIResponseMessage",
			message: &AIResponseMessage{
				BaseMessage: BaseMessage{
					Type:      MessageTypeAIResponse,
					Timestamp: time.Now().Format(time.RFC3339),
				},
				SessionID:      "test-session",
				Text:           "Hello World",
				AudioData:      "SGVsbG8=",
				Emotion:        "happy",
				Confidence:     0.95,
				ProcessingTime: 150,
			},
		},
		{
			name: "ErrorMessage",
			message: CreateErrorMessage("TEST_ERROR", "Test message", "Test details"),
		},
		{
			name: "SystemMessage",
			message: CreateSystemMessage("normal", "Test Title", "Test Content"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize
			data, err := json.Marshal(tt.message)
			if err != nil {
				t.Errorf("Failed to marshal message: %v", err)
				return
			}

			// Deserialize back to map to verify JSON structure
			var result map[string]interface{}
			if err := json.Unmarshal(data, &result); err != nil {
				t.Errorf("Failed to unmarshal message: %v", err)
				return
			}

			// Verify basic structure
			if _, exists := result["type"]; !exists {
				t.Errorf("Message missing 'type' field")
			}
			if _, exists := result["timestamp"]; !exists {
				t.Errorf("Message missing 'timestamp' field")
			}
		})
	}
}

func TestMessageValidator_InvalidJSON(t *testing.T) {
	validator := NewMessageValidator()

	invalidMessages := []string{
		`{invalid json}`,
		`{"type": "audio_chunk", "device_id":}`,
		``,
		`null`,
		`{"type": }`,
	}

	for i, msg := range invalidMessages {
		t.Run(fmt.Sprintf("invalid_json_%d", i), func(t *testing.T) {
			_, err := validator.ValidateMessage([]byte(msg))
			if err == nil {
				t.Errorf("Expected error for invalid JSON, got nil")
			}
		})
	}
}

func TestMessageValidator_UnsupportedMessageType(t *testing.T) {
	validator := NewMessageValidator()

	message := `{
		"type": "unsupported_type",
		"data": "some data"
	}`

	_, err := validator.ValidateMessage([]byte(message))
	if err == nil {
		t.Errorf("Expected error for unsupported message type, got nil")
	}
}