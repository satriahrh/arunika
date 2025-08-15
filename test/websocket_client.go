package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

type DeviceAuthResponse struct {
	Token     string `json:"token"`
	ExpiresIn int64  `json:"expires_in"`
	DeviceID  string `json:"device_id"`
}

func main() {
	serverURL := "http://localhost:8080"
	deviceID := "test-device-websocket"

	// Step 1: Get authentication token
	fmt.Println("Step 1: Getting authentication token...")
	
	authReq := map[string]string{"device_id": deviceID}
	reqBody, _ := json.Marshal(authReq)
	
	resp, err := http.Post(serverURL+"/api/v1/device/auth", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		log.Fatalf("Failed to authenticate device: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Authentication failed with status: %d", resp.StatusCode)
	}

	var authResp DeviceAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		log.Fatalf("Failed to decode auth response: %v", err)
	}

	fmt.Printf("✓ Authentication successful. Token: %s...\n", authResp.Token[:20])

	// Step 2: Connect to WebSocket with token
	fmt.Println("Step 2: Connecting to WebSocket with token...")
	
	wsURL := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws"}
	q := wsURL.Query()
	q.Set("token", authResp.Token)
	wsURL.RawQuery = q.Encode()

	fmt.Printf("Connecting to: %s\n", wsURL.String())

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
	if err != nil {
		if resp != nil {
			log.Fatalf("WebSocket connection failed with status %d: %v", resp.StatusCode, err)
		}
		log.Fatalf("WebSocket connection failed: %v", err)
	}
	defer conn.Close()

	fmt.Println("✓ WebSocket connection successful!")

	// Step 3: Send a ping message
	fmt.Println("Step 3: Sending ping message...")
	
	pingMsg := map[string]interface{}{
		"type": "ping",
		"timestamp": time.Now().Unix(),
	}
	
	if err := conn.WriteJSON(pingMsg); err != nil {
		log.Fatalf("Failed to send ping: %v", err)
	}

	// Step 4: Wait for pong response
	fmt.Println("Step 4: Waiting for pong response...")
	
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, message, err := conn.ReadMessage()
	if err != nil {
		log.Fatalf("Failed to read response: %v", err)
	}

	fmt.Printf("✓ Received response: %s\n", string(message))

	fmt.Println("✓ All WebSocket tests passed!")
}