package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/satriahrh/arunika/server/adapters/stt"
	"github.com/satriahrh/arunika/server/domain/repositories"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512 * 1024 // 512KB for audio chunks
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// TODO: Implement proper origin checking
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Hub maintains the set of active clients and broadcasts messages to the clients.
type Hub struct {
	// Registered clients.
	clients map[string]*Client

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	// Mutex for thread-safe access to clients map
	mu sync.RWMutex

	llm repositories.LargeLanguageModel

	logger *zap.Logger
}

// NewHub creates a new WebSocket hub
func NewHub(llm repositories.LargeLanguageModel, logger *zap.Logger) *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		llm:        llm,
		logger:     logger,
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.deviceID] = client
			h.mu.Unlock()
			h.logger.Info("Client registered", zap.String("deviceID", client.deviceID))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.deviceID]; ok {
				delete(h.clients, client.deviceID)
				close(client.send)
			}
			h.mu.Unlock()
			h.logger.Info("Client unregistered", zap.String("deviceID", client.deviceID))
		}
	}
}

type WriteData struct {
	// MessageType is the type of the websocket message.
	// Expect websocket.TextMessage or websocket.BinaryMessage
	Type    int
	Payload []byte
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan WriteData

	// Device ID for this client
	deviceID string

	// Logger
	logger *zap.Logger

	// Audio streaming session management
	audioSessions map[string]*AudioSession
	sessionMutex  sync.RWMutex
}

// AudioSession manages an ongoing audio streaming session
type AudioSession struct {
	SessionID   string
	StartTime   time.Time
	LastChunk   time.Time
	ChunkCount  int
	TotalChunks int
	ExpectedSeq int
	IsActive    bool

	SpeechToTextContext    context.Context
	SpeechToTextRepository repositories.SpeechToText
	SpeechToTextStream     repositories.SpeechToTextStreaming
	ChatSession            repositories.ChatSession
}

// HandleWebSocket handles websocket requests from the peer.
func HandleWebSocket(hub *Hub, c echo.Context, logger *zap.Logger) error {
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		logger.Error("WebSocket upgrade failed", zap.Error(err))
		return err
	}

	// TODO: Extract device ID from JWT token in query parameter
	deviceID := c.QueryParam("device_id")
	if deviceID == "" {
		deviceID = "unknown" // Temporary fallback
	}

	client := &Client{
		hub:           hub,
		conn:          conn,
		send:          make(chan WriteData, 256),
		deviceID:      deviceID,
		logger:        logger,
		audioSessions: make(map[string]*AudioSession),
	}

	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()

	return nil
}

// HandleWebSocketWithAuth handles websocket requests with pre-authenticated device ID
func HandleWebSocketWithAuth(hub *Hub, c echo.Context, deviceID string, logger *zap.Logger) error {
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		logger.Error("WebSocket upgrade failed", zap.Error(err))
		return err
	}

	client := &Client{
		hub:           hub,
		conn:          conn,
		send:          make(chan WriteData, 256),
		deviceID:      deviceID,
		logger:        logger,
		audioSessions: make(map[string]*AudioSession),
	}

	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()

	return nil
}

// readPump pumps messages from the websocket connection to the hub.
func (c *Client) readPump() {
	defer func() {
		c.cleanupAudioSessions()
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		messageType, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Error("WebSocket error", zap.Error(err))
			}
			break
		}

		// Handle different message types for audio streaming
		switch messageType {
		case websocket.TextMessage:
			// Process JSON messages (control messages, metadata)
			c.processMessage(message)
		case websocket.BinaryMessage:
			// Process binary audio data directly
			c.processBinaryAudioChunk(message)
		default:
			c.logger.Warn("Received unknown message type", zap.Int("type", messageType))
		}
	}
}

// writePump pumps messages from the hub to the websocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(message.Type, message.Payload); err != nil {
				c.logger.Error("Failed to write message", zap.Error(err))
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// processMessage processes incoming messages from the device
func (c *Client) processMessage(message []byte) {
	// Parse the message
	var msg map[string]interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		c.logger.Error("Failed to parse message", zap.Error(err))
		return
	}

	msgType, ok := msg["type"].(string)
	if !ok {
		c.logger.Error("Message missing type field")
		return
	}

	switch msgType {
	case "audio_session_start":
		c.handleAudioSessionStart(msg)
	case "audio_session_end":
		c.handleAudioSessionEnd(msg)
	default:
		c.logger.Warn("Unknown message type", zap.String("type", msgType))
	}
}

// Helper functions to extract values from map with type safety
func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// processBinaryAudioChunk handles binary audio data
func (c *Client) processBinaryAudioChunk(data []byte) {
	c.logger.Info("Received binary audio chunk",
		zap.String("deviceID", c.deviceID),
		zap.Int("size", len(data)))

	// For now, we'll assume there's an active session to update counters
	// In a full implementation, you'd extract session ID from binary headers
	// or track the current active session per device

	c.sessionMutex.Lock()
	defer c.sessionMutex.Unlock()

	// Find any active session for this device (simplified approach)
	var activeSession *AudioSession
	for _, session := range c.audioSessions {
		if session.IsActive {
			activeSession = session
			break
		}
	}

	if activeSession != nil {
		// Update session counters
		activeSession.ChunkCount++
		activeSession.LastChunk = time.Now()

		// Stream audio data to the speech-to-text service
		if activeSession.SpeechToTextStream != nil {
			if err := activeSession.SpeechToTextStream.Stream(data); err != nil {
				c.logger.Error("Failed to stream audio data",
					zap.String("sessionID", activeSession.SessionID),
					zap.Error(err))
			}
		}

		c.logger.Debug("Updated session with binary chunk",
			zap.String("sessionID", activeSession.SessionID),
			zap.Int("totalChunks", activeSession.ChunkCount))
	} else {
		c.logger.Warn("Received binary audio chunk but no active session found",
			zap.String("deviceID", c.deviceID))
	}

	// TODO: Process actual audio data with conversation service
}

// handleAudioSessionStart handles the start of an audio streaming session
func (c *Client) handleAudioSessionStart(msg map[string]interface{}) {
	sessionID := getStringFromMap(msg, "session_id")
	var response map[string]interface{}

	c.sessionMutex.Lock()
	defer func() {
		responseBytes, _ := json.Marshal(response)
		select {
		case c.send <- WriteData{
			Type:    websocket.TextMessage,
			Payload: responseBytes,
		}:
		default:
			close(c.send)
		}
		c.sessionMutex.Unlock()
	}()

	session, ok := c.audioSessions[sessionID]
	if !ok {
		session = &AudioSession{
			SessionID:   sessionID,
			StartTime:   time.Now(),
			LastChunk:   time.Now(),
			ChunkCount:  0,
			ExpectedSeq: 0,
			IsActive:    true,

			SpeechToTextContext:    context.Background(),
			SpeechToTextRepository: &stt.GoogleSpeechToText{}, // Replace with actual repository
		}
	}

	// Initialize streaming transcription
	audioConfig := repositories.AudioConfig{
		SampleRate: 48000,      // Example sample rate
		Language:   "id-ID",    // Example language
		Encoding:   "LINEAR16", // Example encoding
	}

	streamInstance, err := session.SpeechToTextRepository.InitTranscribeStreaming(session.SpeechToTextContext, audioConfig)
	if err != nil {
		c.logger.Error("Failed to initialize streaming transcription",
			zap.String("sessionID", sessionID),
			zap.Error(err))
		response = map[string]interface{}{
			"type":       "audio_session_started",
			"session_id": sessionID,
			"timestamp":  time.Now().Unix(),
			"status":     "speech_to_text_not_ready",
		}
		return
	}
	session.SpeechToTextStream = streamInstance

	chatSession, err := c.hub.llm.GenerateChat(context.Background(), []repositories.ChatMessage{})
	if err != nil {
		c.logger.Error("Failed to generate chat session",
			zap.String("sessionID", sessionID),
			zap.Error(err))
		response = map[string]interface{}{
			"type":       "audio_session_started",
			"session_id": sessionID,
			"timestamp":  time.Now().Unix(),
			"status":     "chat_not_ready",
		}
		return
	}
	session.ChatSession = chatSession

	c.audioSessions[sessionID] = session

	c.logger.Info("Audio session started",
		zap.String("deviceID", c.deviceID),
		zap.String("sessionID", sessionID))

	// Send acknowledgment
	response = map[string]interface{}{
		"type":       "audio_session_started",
		"session_id": sessionID,
		"timestamp":  time.Now().Unix(),
		"status":     "ready",
	}
}

// handleAudioSessionEnd handles the end of an audio streaming session
func (c *Client) handleAudioSessionEnd(msg map[string]interface{}) {
	sessionID := getStringFromMap(msg, "session_id")
	var response map[string]interface{}

	c.sessionMutex.Lock()
	defer func() {
		responseBytes, _ := json.Marshal(response)
		select {
		case c.send <- WriteData{
			Type:    websocket.TextMessage,
			Payload: responseBytes,
		}:
		default:
			close(c.send)
		}
		c.sessionMutex.Unlock()
	}()

	session, exists := c.audioSessions[sessionID]
	if exists {
		session.IsActive = false
		// End the streaming transcription and get the result
		var finalTranscription string
		var err error
		if session.SpeechToTextStream != nil {
			finalTranscription, err = session.SpeechToTextStream.End()
			if err != nil {
				c.logger.Error("Failed to end transcription stream",
					zap.String("deviceID", c.deviceID),
					zap.String("sessionID", sessionID),
					zap.Error(err))
			} else {
				c.logger.Info("Transcription completed",
					zap.String("deviceID", c.deviceID),
					zap.String("sessionID", sessionID),
					zap.String("transcription", finalTranscription))
			}
		}

		go c.responseAudio(sessionID, finalTranscription)
	}

	// Send acknowledgment
	response = map[string]interface{}{
		"type":       "audio_session_ended",
		"session_id": sessionID,
		"timestamp":  time.Now().Unix(),
		"status":     "completed",
	}

	if exists {
		response["total_chunks"] = session.ChunkCount
		response["duration"] = time.Since(session.StartTime).Seconds()
	}

	c.logger.Info("Starting audio response goroutine",
		zap.String("deviceID", c.deviceID),
		zap.String("sessionID", sessionID))
}

func (c *Client) responseAudio(sessionID string, finalTranscription string) {
	session := c.audioSessions[sessionID]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	chatResponse, err := session.ChatSession.SendMessage(ctx, repositories.ChatMessage{
		Role:    repositories.UserRole,
		Content: finalTranscription,
	})
	if err != nil {
		c.logger.Error("Failed to send message to chat session",
			zap.String("deviceID", c.deviceID),
			zap.String("sessionID", sessionID),
			zap.Error(err))
		return
	}

	c.logger.Info("Received chat response",
		zap.String("deviceID", c.deviceID),
		zap.String("sessionID", sessionID),
		zap.String("response", chatResponse.Content))
}

func (c *Client) responseWithSampleAlso(sessionID string) {
	c.sessionMutex.RLock()
	session, exists := c.audioSessions[sessionID]
	c.sessionMutex.RUnlock()

	if !exists {
		c.logger.Error("Session not found for audio response",
			zap.String("deviceID", c.deviceID),
			zap.String("sessionID", sessionID))
		return
	}

	// Wait for the session to end (when IsActive becomes false)
	for session.IsActive {
		time.Sleep(100 * time.Millisecond)
	}

	c.logger.Info("Starting delayed audio response",
		zap.String("deviceID", c.deviceID),
		zap.String("sessionID", sessionID),
		zap.String("delay", "5 seconds"))

	time.Sleep(5 * time.Second)

	c.logger.Info("Sending audio response start message",
		zap.String("deviceID", c.deviceID),
		zap.String("sessionID", sessionID))

	startMessage := map[string]interface{}{
		"type":       "audio_response_started",
		"session_id": sessionID,
		"timestamp":  time.Now().Unix(),
	}
	responseBytes, _ := json.Marshal(startMessage)
	c.send <- WriteData{
		Type:    websocket.TextMessage,
		Payload: responseBytes,
	}

	audioFilePath := filepath.Join(".", "sample_audio.wav")
	audioFileData, err := os.ReadFile(audioFilePath)
	if err != nil {
		c.logger.Error("Failed to read sample audio file for response",
			zap.String("deviceID", c.deviceID),
			zap.String("sessionID", sessionID),
			zap.String("filePath", audioFilePath),
			zap.Error(err))
		return
	}

	c.logger.Info("Read sample audio file for response",
		zap.String("deviceID", c.deviceID),
		zap.String("sessionID", sessionID),
		zap.String("filePath", audioFilePath),
		zap.Int("totalBytes", len(audioFileData)))

	// Send audio data in chunks
	chunkSize := 1024 // 1KB chunks
	totalChunks := (len(audioFileData) + chunkSize - 1) / chunkSize
	sendingChunks := totalChunks / 2

	c.logger.Info("Starting to send audio response chunks",
		zap.String("deviceID", c.deviceID),
		zap.String("sessionID", sessionID),
		zap.Int("totalChunks", totalChunks),
		zap.Int("sendingChunks", sendingChunks),
		zap.Int("chunkSize", chunkSize))

	for i := 0; i < sendingChunks; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(audioFileData) {
			end = len(audioFileData)
		}

		audioChunk := audioFileData[start:end]

		c.logger.Debug("Sending audio response chunk",
			zap.String("deviceID", c.deviceID),
			zap.String("sessionID", sessionID),
			zap.Int("chunkNumber", i+1),
			zap.Int("chunkSize", len(audioChunk)))

		c.send <- WriteData{
			Type:    websocket.BinaryMessage,
			Payload: audioChunk,
		}

		time.Sleep(100 * time.Millisecond) // Small delay between chunks
	}

	c.logger.Info("Finished sending audio response chunks",
		zap.String("deviceID", c.deviceID),
		zap.String("sessionID", sessionID),
		zap.Int("totalChunksSent", sendingChunks))

	c.logger.Info("Sending audio response end message",
		zap.String("deviceID", c.deviceID),
		zap.String("sessionID", sessionID))

	endMessage := map[string]interface{}{
		"type":       "audio_response_ended",
		"session_id": sessionID,
		"timestamp":  time.Now().Unix(),
	}
	responseBytes, _ = json.Marshal(endMessage)
	c.send <- WriteData{
		Type:    websocket.TextMessage,
		Payload: responseBytes,
	}

	c.logger.Info("Audio response completed",
		zap.String("deviceID", c.deviceID),
		zap.String("sessionID", sessionID))
}

// cleanupAudioSessions closes all streaming sessions when client disconnects
func (c *Client) cleanupAudioSessions() {
	c.sessionMutex.Lock()
	defer c.sessionMutex.Unlock()

	for sessionID, session := range c.audioSessions {
		if session.SpeechToTextStream != nil {
			// End the streaming session (ignore result since we're cleaning up)
			_, err := session.SpeechToTextStream.End()
			if err != nil {
				c.logger.Warn("Error ending speech-to-text stream during cleanup",
					zap.String("sessionID", sessionID),
					zap.Error(err))
			}
			c.logger.Info("Ended speech-to-text stream for session",
				zap.String("sessionID", sessionID))
		}
		delete(c.audioSessions, sessionID)
	}
}
