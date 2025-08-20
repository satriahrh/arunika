package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/gorilla/websocket"
)

type DeviceAuthRequest struct {
	SerialNumber string `json:"serial_number"`
	SecretKey    string `json:"secret_key"`
}

type DeviceAuthResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	DeviceID  string    `json:"device_id"`
}

func main() {
	// First, authenticate and get a JWT token
	token, deviceID, err := authenticateDevice()
	if err != nil {
		log.Fatal("Failed to authenticate device:", err)
	}
	log.Printf("Successfully authenticated device: %s", deviceID)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Connect to the WebSocket server with JWT token
	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws"}
	log.Printf("connecting to %s", u.String())

	// Create headers with JWT token
	headers := http.Header{}
	headers.Add("Authorization", "Bearer "+token)

	c, _, err := websocket.DefaultDialer.Dial(u.String(), headers)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})

	// Start a goroutine to read messages from the server
	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s", message)
		}
	}()

	// Test chunked audio functionality
	testChunkedAudio(c)

	// Wait for interrupt signal
	select {
	case <-done:
		return
	case <-interrupt:
		log.Println("interrupt")
		// Cleanly close the connection by sending a close message and then
		// waiting (with timeout) for the server to close the connection.
		err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			log.Println("write close:", err)
			return
		}
		select {
		case <-done:
		case <-time.After(time.Second):
		}
		return
	}
}

func authenticateDevice() (string, string, error) {
	// Use mock device credentials that should work with the mock repository
	authReq := DeviceAuthRequest{
		SerialNumber: "ARUNIKA001",
		SecretKey:    "secret123",
	}

	jsonData, err := json.Marshal(authReq)
	if err != nil {
		return "", "", err
	}

	resp, err := http.Post("http://localhost:8080/api/v1/device/auth", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("authentication failed: %s", string(body))
	}

	var authResp DeviceAuthResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		return "", "", err
	}

	return authResp.Token, authResp.DeviceID, nil
}

func testChunkedAudio(c *websocket.Conn) {
	sessionID := fmt.Sprintf("session_%d", time.Now().Unix())

	// Test 1: Start audio session
	log.Println("Testing audio session start...")
	startMessage := map[string]interface{}{
		"type":       "audio_session_start",
		"session_id": sessionID,
		"timestamp":  time.Now().Unix(),
	}

	if err := sendJSONMessage(c, startMessage); err != nil {
		log.Printf("Error sending session start: %v", err)
		return
	}
	time.Sleep(500 * time.Millisecond)

	// Test 2: Send binary audio chunks from sample_audio.wav
	log.Println("Testing binary audio chunks from sample_audio.wav...")

	// Read the sample audio file
	audioFilePath := filepath.Join(".", "sample_audio.wav")
	audioFileData, err := os.ReadFile(audioFilePath)
	if err != nil {
		log.Printf("Error reading audio file: %v", err)
		return
	}

	log.Printf("Read audio file: %s (%d bytes)", audioFilePath, len(audioFileData))

	// Send audio data in chunks
	chunkSize := 1024 // 1KB chunks
	totalChunks := (len(audioFileData) + chunkSize - 1) / chunkSize

	for i := 0; i < totalChunks/2; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(audioFileData) {
			end = len(audioFileData)
		}

		audioChunk := audioFileData[start:end]

		log.Printf("Sending audio chunk %d/%d (%d bytes)", i+1, totalChunks, len(audioChunk))
		if err := c.WriteMessage(websocket.BinaryMessage, audioChunk); err != nil {
			log.Printf("Error sending audio chunk %d: %v", i, err)
			return
		}
		time.Sleep(100 * time.Millisecond) // Small delay between chunks
	}

	// Test 3: End audio session
	log.Println("Testing audio session end...")
	endMessage := map[string]interface{}{
		"type":       "audio_session_end",
		"session_id": sessionID,
		"timestamp":  time.Now().Unix(),
	}

	if err := sendJSONMessage(c, endMessage); err != nil {
		log.Printf("Error sending session end: %v", err)
		return
	}

	// Test 4: Test ping/pong
	log.Println("Testing ping/pong...")
	pingMessage := map[string]interface{}{
		"type":      "ping",
		"timestamp": time.Now().Unix(),
	}

	if err := sendJSONMessage(c, pingMessage); err != nil {
		log.Printf("Error sending ping: %v", err)
		return
	}

	// Test 5: Test binary chunks with different sizes from real audio (new session)
	log.Println("Testing binary chunks with different sizes from real audio...")
	sessionID2 := fmt.Sprintf("session_binary_%d", time.Now().Unix())

	// Start new session
	startMessage2 := map[string]interface{}{
		"type":       "audio_session_start",
		"session_id": sessionID2,
		"timestamp":  time.Now().Unix(),
	}
	if err := sendJSONMessage(c, startMessage2); err != nil {
		log.Printf("Error sending session start 2: %v", err)
		return
	}
	time.Sleep(200 * time.Millisecond)

	// Read the sample audio file again for the second test
	audioFileData2, err := os.ReadFile(audioFilePath)
	if err != nil {
		log.Printf("Error reading audio file for second test: %v", err)
		return
	}

	// Send binary chunks with different sizes using real audio data
	chunkSizes := []int{512, 1024, 2048, 4096} // Different chunk sizes
	currentOffset := 0

	for i, chunkSize := range chunkSizes {
		if currentOffset >= len(audioFileData2) {
			break // No more data to send
		}

		end := currentOffset + chunkSize
		if end > len(audioFileData2) {
			end = len(audioFileData2)
		}

		audioChunk := audioFileData2[currentOffset:end]
		currentOffset = end

		log.Printf("Sending real audio chunk %d with size %d bytes", i+1, len(audioChunk))
		if err := c.WriteMessage(websocket.BinaryMessage, audioChunk); err != nil {
			log.Printf("Error sending binary chunk %d: %v", i, err)
			return
		}
		time.Sleep(200 * time.Millisecond)
	}

	// End the second session
	log.Println("Testing audio session end for second session...")
	endMessage2 := map[string]interface{}{
		"type":       "audio_session_end",
		"session_id": sessionID2,
		"timestamp":  time.Now().Unix(),
	}

	if err := sendJSONMessage(c, endMessage2); err != nil {
		log.Printf("Error sending session end 2: %v", err)
		return
	}

	log.Println("All tests completed!")
}

func sendJSONMessage(c *websocket.Conn, message map[string]interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return c.WriteMessage(websocket.TextMessage, data)
}
