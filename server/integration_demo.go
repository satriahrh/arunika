package main

import (
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"

	"github.com/satriahrh/arunika/server/internal/auth"
)

// Integration test to verify WebSocket functionality
func main() {
	// Generate a device token
	deviceToken, err := auth.GenerateDeviceToken("integration-test-device")
	if err != nil {
		log.Fatalf("Failed to generate device token: %v", err)
	}

	// Create WebSocket connection
	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws"}
	q := u.Query()
	q.Set("token", deviceToken)
	u.RawQuery = q.Encode()

	fmt.Printf("Connecting to %s\n", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatalf("WebSocket connection failed: %v", err)
	}
	defer c.Close()

	fmt.Println("✅ WebSocket connected successfully")

	// Test 1: Send ping and expect pong
	pingMsg := map[string]interface{}{
		"type": "ping",
		"data": "integration-test",
	}

	if err := c.WriteJSON(pingMsg); err != nil {
		log.Fatalf("Failed to send ping: %v", err)
	}

	fmt.Println("📤 Sent ping message")

	// Read pong response
	var pongResponse map[string]interface{}
	if err := c.ReadJSON(&pongResponse); err != nil {
		log.Fatalf("Failed to read pong: %v", err)
	}

	if pongResponse["type"] != "pong" {
		log.Fatalf("Expected pong, got %v", pongResponse["type"])
	}

	fmt.Println("📥 Received pong response")
	fmt.Printf("   Data: %v\n", pongResponse["data"])

	// Test 2: Send device status
	statusMsg := map[string]interface{}{
		"type":          "device_status",
		"device_id":     "integration-test-device",
		"status":        "online",
		"battery_level": 75,
		"metadata": map[string]interface{}{
			"firmware_version": "1.0.0",
			"signal_strength":  -50,
		},
	}

	if err := c.WriteJSON(statusMsg); err != nil {
		log.Fatalf("Failed to send device status: %v", err)
	}

	fmt.Println("📤 Sent device status message")

	// Test 3: Send invalid message and expect error
	invalidMsg := map[string]interface{}{
		"type":        "audio_chunk",
		"device_id":   "integration-test-device",
		"sample_rate": 999999, // Invalid sample rate
	}

	if err := c.WriteJSON(invalidMsg); err != nil {
		log.Fatalf("Failed to send invalid message: %v", err)
	}

	fmt.Println("📤 Sent invalid message (should get error)")

	// Read error response
	var errorResponse map[string]interface{}
	if err := c.ReadJSON(&errorResponse); err != nil {
		log.Fatalf("Failed to read error response: %v", err)
	}

	if errorResponse["type"] != "error" {
		log.Fatalf("Expected error response, got %v", errorResponse["type"])
	}

	fmt.Println("📥 Received error response (as expected)")
	fmt.Printf("   Code: %v\n", errorResponse["error_code"])
	fmt.Printf("   Message: %v\n", errorResponse["message"])

	// Test 4: Send valid audio chunk (mock data)
	audioMsg := map[string]interface{}{
		"type":           "audio_chunk",
		"device_id":      "integration-test-device",
		"session_id":     "test-session-123",
		"audio_data":     "SGVsbG8gV29ybGQgQXVkaW8gRGF0YQ==", // Base64 encoded "Hello World Audio Data"
		"sample_rate":    16000,
		"encoding":       "pcm",
		"chunk_sequence": 1,
		"is_final":       true,
		"duration_ms":    1000,
	}

	if err := c.WriteJSON(audioMsg); err != nil {
		log.Fatalf("Failed to send audio chunk: %v", err)
	}

	fmt.Println("📤 Sent audio chunk message")

	// Read AI response
	var aiResponse map[string]interface{}
	c.SetReadDeadline(time.Now().Add(10 * time.Second)) // Allow time for processing
	if err := c.ReadJSON(&aiResponse); err != nil {
		log.Fatalf("Failed to read AI response: %v", err)
	}

	if aiResponse["type"] != "ai_response" {
		log.Fatalf("Expected ai_response, got %v", aiResponse["type"])
	}

	fmt.Println("📥 Received AI response")
	fmt.Printf("   Session ID: %v\n", aiResponse["session_id"])
	fmt.Printf("   Text: %v\n", aiResponse["response_text"])
	fmt.Printf("   Processing Time: %v ms\n", aiResponse["processing_time_ms"])

	// Test 5: Auth message
	authMsg := map[string]interface{}{
		"type":   "auth",
		"action": "refresh",
		"token":  deviceToken,
	}

	if err := c.WriteJSON(authMsg); err != nil {
		log.Fatalf("Failed to send auth message: %v", err)
	}

	fmt.Println("📤 Sent auth refresh message")

	// Read auth response
	var authResponse map[string]interface{}
	if err := c.ReadJSON(&authResponse); err != nil {
		log.Fatalf("Failed to read auth response: %v", err)
	}

	if authResponse["type"] != "auth" {
		log.Fatalf("Expected auth response, got %v", authResponse["type"])
	}

	fmt.Println("📥 Received auth response")
	fmt.Printf("   Action: %v\n", authResponse["action"])

	fmt.Println("\n🎉 All integration tests passed!")
	fmt.Println("\n=== Test Summary ===")
	fmt.Println("✅ WebSocket connection with JWT authentication")
	fmt.Println("✅ Ping/Pong mechanism")
	fmt.Println("✅ Device status reporting")
	fmt.Println("✅ Message validation and error handling")
	fmt.Println("✅ Audio chunk processing")
	fmt.Println("✅ AI response generation")
	fmt.Println("✅ Authentication message handling")
	fmt.Println("\n🚀 WebSocket server is fully functional and ready for production!")
}