package entities

import (
	"testing"
	"time"
)

func TestSessionCreation(t *testing.T) {
	deviceID := "test-device-123"
	session := NewSession(deviceID)

	if session.DeviceID != deviceID {
		t.Errorf("Expected device ID %s, got %s", deviceID, session.DeviceID)
	}

	if session.Status != SessionStatusActive {
		t.Errorf("Expected status %s, got %s", SessionStatusActive, session.Status)
	}

	if len(session.Messages) != 0 {
		t.Errorf("Expected empty messages, got %d messages", len(session.Messages))
	}

	if session.Metadata.Language != "id-ID" {
		t.Errorf("Expected language id-ID, got %s", session.Metadata.Language)
	}
}

func TestAddMessage(t *testing.T) {
	session := NewSession("test-device")
	
	// Add user message
	userContent := "Hello, how are you?"
	session.AddMessage(MessageRoleUser, userContent, 1500, SessionMessageMetadata{})

	if len(session.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(session.Messages))
	}

	if session.Messages[0].Role != MessageRoleUser {
		t.Errorf("Expected user role, got %s", session.Messages[0].Role)
	}

	if session.Messages[0].Content != userContent {
		t.Errorf("Expected content %s, got %s", userContent, session.Messages[0].Content)
	}

	if session.LastMessageAt == nil {
		t.Error("Expected LastMessageAt to be set")
	}

	// Add assistant message
	assistantContent := "I'm doing well, thank you!"
	session.AddMessage(MessageRoleAssistant, assistantContent, 2000, SessionMessageMetadata{})

	if len(session.Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(session.Messages))
	}

	if session.Messages[1].Role != MessageRoleAssistant {
		t.Errorf("Expected assistant role, got %s", session.Messages[1].Role)
	}
}

func TestSessionExpiration(t *testing.T) {
	session := NewSession("test-device")
	
	// Should not be expired initially
	if session.IsExpired() {
		t.Error("Session should not be expired initially")
	}

	// Manually set expiration to past
	session.ExpiresAt = time.Now().Add(-1 * time.Hour)
	if !session.IsExpired() {
		t.Error("Session should be expired when ExpiresAt is in the past")
	}

	// Test with terminated status
	session.ExpiresAt = time.Now().Add(1 * time.Hour)
	session.Status = SessionStatusTerminated
	if !session.IsExpired() {
		t.Error("Session should be expired when status is terminated")
	}
}

func TestShouldCreateNewSession(t *testing.T) {
	session := NewSession("test-device")
	
	// Should not create new session when no messages
	if session.ShouldCreateNewSession() {
		t.Error("Should not create new session when no messages exist")
	}

	// Add recent message (within 30 minutes)
	session.AddMessage(MessageRoleUser, "Hello", 1000, SessionMessageMetadata{})
	if session.ShouldCreateNewSession() {
		t.Error("Should not create new session when last message is recent")
	}

	// Simulate old message (more than 30 minutes ago)
	oldTime := time.Now().Add(-31 * time.Minute)
	session.LastMessageAt = &oldTime
	if !session.ShouldCreateNewSession() {
		t.Error("Should create new session when last message is old")
	}
}

func TestSessionValidation(t *testing.T) {
	// Valid session
	session := NewSession("test-device")
	if err := session.Validate(); err != nil {
		t.Errorf("Valid session should not have validation errors, got: %v", err)
	}

	// Invalid device ID
	session.DeviceID = ""
	if err := session.Validate(); err == nil {
		t.Error("Session with empty device ID should have validation error")
	}

	// Invalid status
	session.DeviceID = "test-device"
	session.Status = SessionStatus("invalid")
	if err := session.Validate(); err == nil {
		t.Error("Session with invalid status should have validation error")
	}
}

func TestUpdateLastActive(t *testing.T) {
	session := NewSession("test-device")
	originalLastActive := session.LastActiveAt
	originalExpiresAt := session.ExpiresAt
	
	// Wait a bit to ensure time difference
	time.Sleep(10 * time.Millisecond)
	
	session.UpdateLastActive()
	
	if !session.LastActiveAt.After(originalLastActive) {
		t.Error("LastActiveAt should be updated to a later time")
	}

	if !session.ExpiresAt.After(originalExpiresAt) {
		t.Error("ExpiresAt should be extended")
	}

	// Check that expiration is 24 hours from last active
	expectedExpiration := session.LastActiveAt.Add(24 * time.Hour)
	if session.ExpiresAt.Sub(expectedExpiration).Abs() > time.Second {
		t.Error("ExpiresAt should be 24 hours from LastActiveAt")
	}
}