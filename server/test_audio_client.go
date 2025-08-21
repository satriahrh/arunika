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
	go handleIncomingMessage(c, done)

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
	log.Printf("🚀 Testing audio session start for session: %s at %s", sessionID, time.Now().Format("15:04:05.000"))
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
	log.Printf("📤 Testing binary audio chunks from sample_audio.wav for session: %s", sessionID)

	// Read the sample audio file
	audioFilePath := filepath.Join(".", "sample_audio.wav")
	audioFileData, err := os.ReadFile(audioFilePath)
	if err != nil {
		log.Printf("Error reading audio file: %v", err)
		return
	}

	log.Printf("📁 Read audio file: %s (%d bytes)", audioFilePath, len(audioFileData))

	// Send audio data in chunks
	chunkSize := 1024 // 1KB chunks
	totalChunks := (len(audioFileData) + chunkSize - 1) / chunkSize
	sendingChunks := totalChunks / 2

	log.Printf("📤 Sending %d/%d audio chunks (chunk size: %d bytes)", sendingChunks, totalChunks, chunkSize)
	audioStartTime := time.Now()

	for i := 0; i < sendingChunks; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(audioFileData) {
			end = len(audioFileData)
		}

		audioChunk := audioFileData[start:end]

		log.Printf("📤 Sending audio chunk %d/%d (%d bytes)", i+1, sendingChunks, len(audioChunk))
		if err := c.WriteMessage(websocket.BinaryMessage, audioChunk); err != nil {
			log.Printf("Error sending audio chunk %d: %v", i, err)
			return
		}
		time.Sleep(100 * time.Millisecond) // Small delay between chunks
	}

	audioDuration := time.Since(audioStartTime)
	log.Printf("📤 Finished sending audio chunks in %v", audioDuration)

	// Test 3: End audio session
	log.Printf("🛑 Testing audio session end for session: %s at %s", sessionID, time.Now().Format("15:04:05.000"))
	endMessage := map[string]interface{}{
		"type":       "audio_session_end",
		"session_id": sessionID,
		"timestamp":  time.Now().Unix(),
	}

	if err := sendJSONMessage(c, endMessage); err != nil {
		log.Printf("Error sending session end: %v", err)
		return
	}

	log.Printf("✅ All tests completed for session: %s! Waiting for server response...", sessionID)
}

func sendJSONMessage(c *websocket.Conn, message map[string]interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return c.WriteMessage(websocket.TextMessage, data)
}

func handleIncomingMessage(c *websocket.Conn, done chan struct{}) {
	defer close(done)
	var audioFile *os.File
	var audioResponseStartTime time.Time
	var audioChunkCount int

	for {
		messageType, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return
		}

		if messageType == websocket.TextMessage {
			log.Printf("Received text message: %s", string(message))

			var msg map[string]interface{}
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Println("unmarshal error:", err)
				continue
			}

			// Handle different message types if needed
			if msgType, ok := msg["type"].(string); ok {
				sessionID := ""
				if sid, exists := msg["session_id"]; exists {
					sessionID = fmt.Sprintf("%v", sid)
				}

				switch msgType {
				case "audio_response_started":
					audioResponseStartTime = time.Now()
					audioChunkCount = 0
					log.Printf("🎵 Audio response started for session: %s at %s", sessionID, audioResponseStartTime.Format("15:04:05.000"))
					audioDir := "audio_responses"
					if err := os.MkdirAll(audioDir, 0755); err != nil {
						log.Printf("Error creating audio response directory: %v", err)
						return
					}
					filename := fmt.Sprintf("%d.wav", time.Now().Unix())
					filepath := filepath.Join(audioDir, filename)
					audioFile, err = os.Create(filepath)
					if err != nil {
						log.Printf("Error creating audio response file: %v", err)
						return
					}
					log.Printf("📁 Created audio response file: %s", filepath)
				case "audio_response_ended":
					duration := time.Since(audioResponseStartTime)
					log.Printf("🎵 Audio response ended for session: %s", sessionID)
					log.Printf("📊 Audio response stats - Duration: %v, Chunks received: %d", duration, audioChunkCount)
					if audioFile != nil {
						audioFile.Close()
						log.Println("📁 Audio response file closed")
					}
				case "audio_session_started":
					log.Printf("✅ Audio session start acknowledged for session: %s", sessionID)
				case "audio_session_ended":
					log.Printf("✅ Audio session end acknowledged for session: %s", sessionID)
					if totalChunks, exists := msg["total_chunks"]; exists {
						log.Printf("📊 Session stats - Total chunks processed: %v", totalChunks)
					}
					if duration, exists := msg["duration"]; exists {
						log.Printf("📊 Session duration: %v seconds", duration)
					}
				default:
					log.Printf("Received unknown message type: %s", msgType)
				}
			}
		} else if messageType == websocket.BinaryMessage {
			audioChunkCount++
			log.Printf("🎵 Received audio response chunk #%d (%d bytes)", audioChunkCount, len(message))
			if audioFile != nil {
				if _, err := audioFile.Write(message); err != nil {
					log.Printf("Error writing audio chunk to file: %v", err)
				}
			}
		}
	}
}
