package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/satriahrh/arunika/server/internal/auth"
	"github.com/satriahrh/arunika/server/usecase"
)

// MockConversationService for testing
type MockConversationService struct{}

func (m *MockConversationService) ProcessAudioChunk(ctx context.Context, msg interface{}) (interface{}, error) {
	// Return mock response
	return map[string]interface{}{
		"type":          "ai_response",
		"session_id":    "test-session",
		"response_text": "Hello from AI",
		"audio_data":    "SGVsbG8=",
		"emotion":       "friendly",
		"timestamp":     time.Now().Format(time.RFC3339),
	}, nil
}

func setupTestHub(t testing.TB) (*Hub, *zap.Logger) {
	logger := zap.NewNop() // No-op logger for tests
	
	// We need to cast to the expected interface
	conversationService := &usecase.ConversationService{}
	hub := NewHub(conversationService, logger)
	
	return hub, logger
}

func TestHub_NewHub(t *testing.T) {
	hub, _ := setupTestHub(t)

	if hub == nil {
		t.Fatal("NewHub returned nil")
	}

	if hub.clients == nil {
		t.Error("Hub clients map not initialized")
	}

	if hub.deviceSessions == nil {
		t.Error("Hub deviceSessions map not initialized")
	}

	if hub.register == nil {
		t.Error("Hub register channel not initialized")
	}

	if hub.unregister == nil {
		t.Error("Hub unregister channel not initialized")
	}

	if hub.broadcast == nil {
		t.Error("Hub broadcast channel not initialized")
	}
}

func TestHub_DeviceSessionManagement(t *testing.T) {
	hub, _ := setupTestHub(t)

	deviceID := "test-device-1"
	userID := "test-user-1"

	// Test creating a new session
	hub.updateDeviceSession(deviceID, userID, true)

	session, exists := hub.GetDeviceSession(deviceID)
	if !exists {
		t.Error("Device session should exist after creation")
	}

	if session.DeviceID != deviceID {
		t.Errorf("Expected device ID %s, got %s", deviceID, session.DeviceID)
	}

	if session.UserID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, session.UserID)
	}

	if !session.IsActive {
		t.Error("Session should be active")
	}

	// Test updating existing session to inactive
	hub.updateDeviceSession(deviceID, userID, false)

	session, exists = hub.GetDeviceSession(deviceID)
	if !exists {
		t.Error("Device session should still exist after deactivation")
	}

	if session.IsActive {
		t.Error("Session should be inactive")
	}
}

func TestHub_GetActiveDevices(t *testing.T) {
	hub, logger := setupTestHub(t)

	// Create mock clients
	client1 := &Client{
		hub:      hub,
		deviceID: "device-1",
		userID:   "user-1",
		send:     make(chan []byte, 256),
		logger:   logger,
	}

	client2 := &Client{
		hub:      hub,
		deviceID: "device-2",
		userID:   "user-2",
		send:     make(chan []byte, 256),
		logger:   logger,
	}

	// Add clients to hub
	hub.clients[client1.deviceID] = client1
	hub.clients[client2.deviceID] = client2

	activeDevices := hub.GetActiveDevices()

	if len(activeDevices) != 2 {
		t.Errorf("Expected 2 active devices, got %d", len(activeDevices))
	}

	// Check if both devices are in the list
	deviceMap := make(map[string]bool)
	for _, deviceID := range activeDevices {
		deviceMap[deviceID] = true
	}

	if !deviceMap["device-1"] {
		t.Error("device-1 should be in active devices list")
	}

	if !deviceMap["device-2"] {
		t.Error("device-2 should be in active devices list")
	}
}

func TestHub_SendToDevice(t *testing.T) {
	hub, logger := setupTestHub(t)

	deviceID := "test-device"
	client := &Client{
		hub:      hub,
		deviceID: deviceID,
		userID:   "test-user",
		send:     make(chan []byte, 256),
		logger:   logger,
	}

	hub.clients[deviceID] = client

	message := []byte(`{"type":"test","message":"hello"}`)

	// Test successful message send
	err := hub.SendToDevice(deviceID, message)
	if err != nil {
		t.Errorf("SendToDevice should not return error, got: %v", err)
	}

	// Verify message was received
	select {
	case receivedMsg := <-client.send:
		if string(receivedMsg) != string(message) {
			t.Errorf("Expected message %s, got %s", string(message), string(receivedMsg))
		}
	case <-time.After(time.Second):
		t.Error("Message not received within timeout")
	}

	// Test sending to non-existent device
	err = hub.SendToDevice("non-existent-device", message)
	if err == nil {
		t.Error("SendToDevice should return error for non-existent device")
	}
}

func TestMessageValidator_Integration(t *testing.T) {
	validator := NewMessageValidator()

	// Test complete audio chunk message flow
	audioChunkJSON := `{
		"type": "audio_chunk",
		"device_id": "test-device-1",
		"session_id": "session-123",
		"audio_data": "SGVsbG8gV29ybGQ=",
		"sample_rate": 16000,
		"encoding": "pcm",
		"chunk_sequence": 1,
		"is_final": false,
		"duration_ms": 1000
	}`

	result, err := validator.ValidateMessage([]byte(audioChunkJSON))
	if err != nil {
		t.Errorf("ValidateMessage failed: %v", err)
	}

	audioChunk, ok := result.(*AudioChunkMessage)
	if !ok {
		t.Errorf("Expected *AudioChunkMessage, got %T", result)
	}

	// Verify all fields
	if audioChunk.DeviceID != "test-device-1" {
		t.Errorf("Expected device_id 'test-device-1', got '%s'", audioChunk.DeviceID)
	}

	if audioChunk.SessionID != "session-123" {
		t.Errorf("Expected session_id 'session-123', got '%s'", audioChunk.SessionID)
	}

	if audioChunk.SampleRate != 16000 {
		t.Errorf("Expected sample_rate 16000, got %d", audioChunk.SampleRate)
	}

	if audioChunk.Encoding != "pcm" {
		t.Errorf("Expected encoding 'pcm', got '%s'", audioChunk.Encoding)
	}

	if audioChunk.ChunkSeq != 1 {
		t.Errorf("Expected chunk_sequence 1, got %d", audioChunk.ChunkSeq)
	}

	if audioChunk.IsFinal {
		t.Error("Expected is_final false, got true")
	}

	if audioChunk.Duration != 1000 {
		t.Errorf("Expected duration_ms 1000, got %d", audioChunk.Duration)
	}
}

func TestJWTAuthentication(t *testing.T) {
	// Test JWT token generation and validation
	deviceID := "test-device-123"

	token, err := auth.GenerateDeviceToken(deviceID)
	if err != nil {
		t.Errorf("GenerateDeviceToken failed: %v", err)
	}

	claims, err := auth.ValidateToken(token)
	if err != nil {
		t.Errorf("ValidateToken failed: %v", err)
	}

	if claims.DeviceID != deviceID {
		t.Errorf("Expected device ID %s, got %s", deviceID, claims.DeviceID)
	}

	if claims.Role != "device" {
		t.Errorf("Expected role 'device', got '%s'", claims.Role)
	}

	// Test user token
	userID := "test-user-456"
	userToken, err := auth.GenerateUserToken(userID)
	if err != nil {
		t.Errorf("GenerateUserToken failed: %v", err)
	}

	userClaims, err := auth.ValidateToken(userToken)
	if err != nil {
		t.Errorf("ValidateToken for user failed: %v", err)
	}

	if userClaims.UserID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, userClaims.UserID)
	}

	if userClaims.Role != "user" {
		t.Errorf("Expected role 'user', got '%s'", userClaims.Role)
	}
}

func TestWebSocketUpgrade_WithAuth(t *testing.T) {
	hub, logger := setupTestHub(t)

	// Generate a valid device token
	deviceID := "test-device-123"
	token, err := auth.GenerateDeviceToken(deviceID)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Create test server
	e := echo.New()
	e.GET("/ws", func(c echo.Context) error {
		return HandleWebSocket(hub, c, logger)
	})

	server := httptest.NewServer(e)
	defer server.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?token=" + token

	// Test WebSocket connection with valid token
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Errorf("WebSocket connection failed: %v", err)
		return
	}
	defer ws.Close()

	// Test without token (should fail)
	wsURLNoToken := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	_, _, err = websocket.DefaultDialer.Dial(wsURLNoToken, nil)
	if err == nil {
		t.Error("WebSocket connection should fail without token")
	}
}

func TestClientMessageProcessing(t *testing.T) {
	hub, logger := setupTestHub(t)

	// Create a mock client
	client := &Client{
		hub:       hub,
		deviceID:  "test-device",
		userID:    "test-user",
		sessionID: "test-session",
		send:      make(chan []byte, 256),
		logger:    logger,
	}

	// Test ping message processing
	pingMessage := `{
		"type": "ping",
		"data": "test-ping"
	}`

	client.processMessage([]byte(pingMessage))

	// Check if pong response was sent
	select {
	case response := <-client.send:
		var pongMsg map[string]interface{}
		if err := json.Unmarshal(response, &pongMsg); err != nil {
			t.Errorf("Failed to unmarshal pong response: %v", err)
		}

		if pongMsg["type"] != "pong" {
			t.Errorf("Expected pong type, got %v", pongMsg["type"])
		}
	case <-time.After(time.Second):
		t.Error("Pong response not received within timeout")
	}

	// Test invalid message
	invalidMessage := `{invalid json}`
	client.processMessage([]byte(invalidMessage))

	// Check if error response was sent
	select {
	case response := <-client.send:
		var errorMsg map[string]interface{}
		if err := json.Unmarshal(response, &errorMsg); err != nil {
			t.Errorf("Failed to unmarshal error response: %v", err)
		}

		if errorMsg["type"] != "error" {
			t.Errorf("Expected error type, got %v", errorMsg["type"])
		}
	case <-time.After(time.Second):
		t.Error("Error response not received within timeout")
	}
}

func TestConcurrentClientHandling(t *testing.T) {
	hub, logger := setupTestHub(t)

	// Start hub
	go hub.Run()

	// Create multiple clients concurrently
	numClients := 10
	clients := make([]*Client, numClients)

	for i := 0; i < numClients; i++ {
		client := &Client{
			hub:       hub,
			deviceID:  fmt.Sprintf("device-%d", i),
			userID:    fmt.Sprintf("user-%d", i),
			sessionID: fmt.Sprintf("session-%d", i),
			send:      make(chan []byte, 256),
			logger:    logger,
		}

		clients[i] = client
		hub.register <- client
	}

	// Wait a bit for registration
	time.Sleep(100 * time.Millisecond)

	// Verify all clients are registered
	activeDevices := hub.GetActiveDevices()
	if len(activeDevices) != numClients {
		t.Errorf("Expected %d active devices, got %d", numClients, len(activeDevices))
	}

	// Unregister all clients
	for _, client := range clients {
		hub.unregister <- client
	}

	// Wait a bit for unregistration
	time.Sleep(100 * time.Millisecond)

	// Verify all clients are unregistered
	activeDevices = hub.GetActiveDevices()
	if len(activeDevices) != 0 {
		t.Errorf("Expected 0 active devices, got %d", len(activeDevices))
	}
}

func BenchmarkMessageValidation(b *testing.B) {
	validator := NewMessageValidator()
	
	audioChunkJSON := `{
		"type": "audio_chunk",
		"device_id": "test-device-1",
		"session_id": "session-123",
		"audio_data": "SGVsbG8gV29ybGQ=",
		"sample_rate": 16000,
		"encoding": "pcm",
		"chunk_sequence": 1,
		"is_final": false
	}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validator.ValidateMessage([]byte(audioChunkJSON))
		if err != nil {
			b.Errorf("Validation failed: %v", err)
		}
	}
}

func BenchmarkHubSendToDevice(b *testing.B) {
	hub, logger := setupTestHub(b)

	// Setup a test client
	client := &Client{
		hub:      hub,
		deviceID: "test-device",
		send:     make(chan []byte, 1000), // Large buffer to avoid blocking
		logger:   logger,
	}

	hub.clients["test-device"] = client
	message := []byte(`{"type":"test","data":"benchmark"}`)

	// Start goroutine to consume messages
	go func() {
		for range client.send {
			// Consume messages
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := hub.SendToDevice("test-device", message)
		if err != nil {
			b.Errorf("SendToDevice failed: %v", err)
		}
	}
}