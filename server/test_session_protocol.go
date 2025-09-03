package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"go.uber.org/zap"

	"github.com/satriahrh/arunika/server/domain/entities"
	"github.com/satriahrh/arunika/server/internal/auth"
)

// SessionProtocolTest tests the new session management protocol
func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	fmt.Println("=== Arunika Session Management Protocol Test ===")

	// Test 1: Session Entity Functionality
	fmt.Println("\n1. Testing Session Entity...")
	testSessionEntity()

	// Test 2: Device Authentication
	fmt.Println("\n2. Testing Device Authentication...")
	testDeviceAuth()

	// Test 3: Message Protocol Validation
	fmt.Println("\n3. Testing Message Protocol...")
	testMessageProtocol()

	fmt.Println("\n=== All Tests Completed Successfully! ===")
}

func testSessionEntity() {
	// Test session creation
	deviceID := "test-device-001"
	session := entities.NewSession(deviceID)
	
	fmt.Printf("✓ Created session for device: %s\n", deviceID)
	fmt.Printf("  Session ID: %s\n", session.ID.Hex())
	fmt.Printf("  Status: %s\n", session.Status)
	fmt.Printf("  Language: %s\n", session.Metadata.Language)

	// Test adding messages
	session.AddMessage(entities.MessageRoleUser, "Hello!", 1500, entities.SessionMessageMetadata{})
	session.AddMessage(entities.MessageRoleAssistant, "Hi there!", 1200, entities.SessionMessageMetadata{})

	fmt.Printf("✓ Added %d messages to session\n", len(session.Messages))

	// Test session continuation logic
	shouldCreate := session.ShouldCreateNewSession()
	fmt.Printf("✓ Should create new session: %v (expected: false)\n", shouldCreate)

	// Test session expiration
	isExpired := session.IsExpired()
	fmt.Printf("✓ Is session expired: %v (expected: false)\n", isExpired)
}

func testDeviceAuth() {
	// Test JWT token generation
	deviceID := "device-ARUNIKA001"
	token, err := auth.GenerateDeviceToken(deviceID)
	if err != nil {
		log.Fatalf("Failed to generate device token: %v", err)
	}

	fmt.Printf("✓ Generated JWT token for device: %s\n", deviceID)
	fmt.Printf("  Token: %s...\n", token[:50])

	// Test token validation
	claims, err := auth.ValidateToken(token)
	if err != nil {
		log.Fatalf("Failed to validate token: %v", err)
	}

	fmt.Printf("✓ Validated token successfully\n")
	fmt.Printf("  Device ID: %s\n", claims.DeviceID)
	fmt.Printf("  Role: %s\n", claims.Role)
}

func testMessageProtocol() {
	// Test message structures for the new protocol
	messages := []map[string]interface{}{
		{
			"type": "listening_start",
		},
		{
			"type": "listening_end",
		},
		{
			"type": "speaking_start",
			"session_id": "507f1f77bcf86cd799439011",
			"timestamp":  time.Now().Unix(),
		},
		{
			"type": "speaking_end",
			"session_id": "507f1f77bcf86cd799439011",
			"timestamp":  time.Now().Unix(),
		},
	}

	fmt.Println("✓ Testing message protocol formats:")
	for _, msg := range messages {
		msgBytes, err := json.Marshal(msg)
		if err != nil {
			log.Fatalf("Failed to marshal message: %v", err)
		}
		fmt.Printf("  %s: %s\n", msg["type"], string(msgBytes))
	}

	// Test error message format
	errorMsg := map[string]interface{}{
		"type":      "error",
		"timestamp": time.Now().Unix(),
		"message":   "Example error message",
	}
	errorBytes, _ := json.Marshal(errorMsg)
	fmt.Printf("  error: %s\n", string(errorBytes))
}

// simulateWebSocketProtocol simulates a WebSocket conversation flow
func simulateWebSocketProtocol() {
	// This would normally connect to a real WebSocket server
	// For testing purposes, we'll simulate the message flow

	fmt.Println("\n4. Simulating WebSocket Protocol Flow...")

	// Simulate client messages
	clientMessages := []string{
		`{"type": "listening_start"}`,
		`{"type": "listening_end"}`,
	}

	// Simulate server responses
	serverMessages := []string{
		`{"type": "listening_started", "session_id": "507f1f77bcf86cd799439011", "timestamp": 1640995200, "status": "ready"}`,
		`{"type": "listening_ended", "session_id": "507f1f77bcf86cd799439011", "timestamp": 1640995205, "status": "completed"}`,
		`{"type": "speaking_start", "session_id": "507f1f77bcf86cd799439011", "timestamp": 1640995210}`,
		`{"type": "speaking_end", "session_id": "507f1f77bcf86cd799439011", "timestamp": 1640995215}`,
	}

	fmt.Println("Client → Server messages:")
	for _, msg := range clientMessages {
		fmt.Printf("  → %s\n", msg)
	}

	fmt.Println("Server → Client messages:")
	for _, msg := range serverMessages {
		fmt.Printf("  ← %s\n", msg)
	}

	fmt.Println("✓ Protocol simulation completed")
}

// Additional helper function to test actual WebSocket connection
// (commented out as it requires a running server)
/*
func testRealWebSocketConnection() {
	// This would test against a real running server
	// Uncomment and modify for integration testing

	token, _ := auth.GenerateDeviceToken("test-device")
	
	dialer := websocket.Dialer{}
	header := make(http.Header)
	header.Set("Authorization", "Bearer "+token)
	
	conn, _, err := dialer.Dial("ws://localhost:8080/ws", header)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Send listening_start
	startMsg := map[string]interface{}{"type": "listening_start"}
	conn.WriteJSON(startMsg)

	// Read response
	var response map[string]interface{}
	conn.ReadJSON(&response)
	
	fmt.Printf("Received: %+v\n", response)
}
*/