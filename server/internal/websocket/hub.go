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
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"

	"github.com/satriahrh/arunika/server/adapters/stt"
	"github.com/satriahrh/arunika/server/adapters/tts"
	"github.com/satriahrh/arunika/server/domain/entities"
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

	// Session repository for persistent session management
	sessionRepo repositories.SessionRepository

	logger *zap.Logger
}

// NewHub creates a new WebSocket hub
func NewHub(llm repositories.LargeLanguageModel, sessionRepo repositories.SessionRepository, logger *zap.Logger) *Hub {
	return &Hub{
		clients:     make(map[string]*Client),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		llm:         llm,
		sessionRepo: sessionRepo,
		logger:      logger,
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

	// Current session for this device
	currentSession *entities.Session
	sessionMutex   sync.RWMutex

	// Audio streaming session management (legacy - will be replaced)
	audioSessions map[string]*AudioSession
	legacyMutex   sync.RWMutex

	// Current listening state
	isListening bool
	
	// Speech-to-text streaming components for current session
	sttContext context.Context
	sttCancel  context.CancelFunc
	sttRepo    repositories.SpeechToText
	sttStream  repositories.SpeechToTextStreaming
	
	// Text-to-speech repository for current session
	ttsRepo repositories.TextToSpeech
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
	TextToSpeechRepository repositories.TextToSpeech
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
		isListening:   false,
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
	case "listening_start":
		c.handleListeningStart(msg)
	case "listening_end":
		c.handleListeningEnd(msg)
	// Keep legacy support for existing clients
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
	c.logger.Debug("Received binary audio chunk",
		zap.String("deviceID", c.deviceID),
		zap.Int("size", len(data)))

	c.sessionMutex.RLock()
	isListening := c.isListening
	sttStream := c.sttStream
	sessionID := ""
	if c.currentSession != nil {
		sessionID = c.currentSession.ID.Hex()
	}
	c.sessionMutex.RUnlock()

	if !isListening || sttStream == nil {
		c.logger.Warn("Received binary audio chunk but not listening",
			zap.String("deviceID", c.deviceID))
		return
	}

	// Stream audio data to the speech-to-text service
	if err := sttStream.Stream(data); err != nil {
		c.logger.Error("Failed to stream audio data",
			zap.String("sessionID", sessionID),
			zap.Error(err))
		return
	}

	c.logger.Debug("Streamed audio chunk to STT",
		zap.String("sessionID", sessionID),
		zap.Int("size", len(data)))

	// Legacy support - also update old audio sessions if they exist
	c.legacyMutex.Lock()
	defer c.legacyMutex.Unlock()

	// Find any active session for this device (legacy support)
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

		// Stream audio data to the legacy speech-to-text service
		if activeSession.SpeechToTextStream != nil {
			if err := activeSession.SpeechToTextStream.Stream(data); err != nil {
				c.logger.Error("Failed to stream audio data to legacy session",
					zap.String("sessionID", activeSession.SessionID),
					zap.Error(err))
			}
		}

		c.logger.Debug("Updated legacy session with binary chunk",
			zap.String("sessionID", activeSession.SessionID),
			zap.Int("totalChunks", activeSession.ChunkCount))
	}
}

// handleAudioSessionStart handles the start of an audio streaming session
func (c *Client) handleAudioSessionStart(msg map[string]interface{}) {
	sessionID := getStringFromMap(msg, "session_id")
	var response map[string]interface{} = map[string]interface{}{
		"type":       "audio_session_started",
		"session_id": sessionID,
		"timestamp":  time.Now().Unix(),
		"status":     "unknown",
	}

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
		// Initialize TTS repository
		ttsRepoConfig := tts.NewElevenLabsConfigFromEnv()
		ttsRepo, err := tts.NewElevenLabsTTS(ttsRepoConfig, c.logger)
		if err != nil {
			c.logger.Error("Failed to initialize TTS repository",
				zap.String("sessionID", sessionID),
				zap.Error(err))
			response = map[string]interface{}{
				"type":       "audio_session_started",
				"session_id": sessionID,
				"timestamp":  time.Now().Unix(),
				"status":     "tts_not_ready",
			}
			return
		}

		session = &AudioSession{
			SessionID:   sessionID,
			StartTime:   time.Now(),
			LastChunk:   time.Now(),
			ChunkCount:  0,
			ExpectedSeq: 0,
			IsActive:    true,

			SpeechToTextContext:    context.Background(),
			SpeechToTextRepository: &stt.GoogleSpeechToText{}, // Replace with actual repository
			TextToSpeechRepository: ttsRepo,
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
	var response map[string]interface{} = map[string]interface{}{
		"type":       "audio_session_ended",
		"session_id": sessionID,
		"timestamp":  time.Now().Unix(),
		"status":     "unknown",
	}

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

	audioDataChan, err := session.TextToSpeechRepository.ConvertTextToSpeech(ctx, chatResponse.Content)
	if err != nil {
		c.logger.Error("Failed to convert text to speech",
			zap.String("deviceID", c.deviceID),
			zap.String("sessionID", sessionID),
			zap.Error(err))
		return
	}

	responseBytes, _ := json.Marshal(map[string]interface{}{
		"type":       "audio_response_started",
		"session_id": sessionID,
		"timestamp":  time.Now().Unix(),
	})
	c.send <- WriteData{
		Type:    websocket.TextMessage,
		Payload: responseBytes,
	}
	for audioData := range audioDataChan {
		c.send <- WriteData{
			Type:    websocket.BinaryMessage,
			Payload: audioData,
		}
	}

	responseBytes, _ = json.Marshal(map[string]interface{}{
		"type":       "audio_response_ended",
		"session_id": sessionID,
		"timestamp":  time.Now().Unix(),
	})
	c.send <- WriteData{
		Type:    websocket.TextMessage,
		Payload: responseBytes,
	}
}

// responseWithSampleAlso deprecated
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
	c.legacyMutex.Lock()
	defer c.legacyMutex.Unlock()

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
	
	// Clean up current session streaming
	c.sessionMutex.Lock()
	defer c.sessionMutex.Unlock()
	
	if c.sttCancel != nil {
		c.sttCancel()
	}
	
	if c.sttStream != nil {
		c.sttStream.End()
	}
}

// handleListeningStart handles the start of user speech listening
func (c *Client) handleListeningStart(msg map[string]interface{}) {
	c.sessionMutex.Lock()
	defer c.sessionMutex.Unlock()

	var response map[string]interface{}
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
	}()

	// Prevent multiple listening sessions
	if c.isListening {
		response = map[string]interface{}{
			"type":      "error",
			"timestamp": time.Now().Unix(),
			"message":   "Already listening",
		}
		return
	}

	// Get or create session for this device
	ctx := context.Background()
	session, err := c.hub.sessionRepo.GetActiveByDeviceID(ctx, c.deviceID)
	if err != nil {
		c.logger.Error("Failed to get active session", zap.Error(err), zap.String("device_id", c.deviceID))
		response = map[string]interface{}{
			"type":      "error",
			"timestamp": time.Now().Unix(),
			"message":   "Failed to retrieve session",
		}
		return
	}

	// Create new session if none exists or if the existing one should be replaced
	if session == nil || session.ShouldCreateNewSession() {
		if session != nil {
			// Terminate the old session
			session.Terminate()
			if err := c.hub.sessionRepo.Update(ctx, session); err != nil {
				c.logger.Error("Failed to terminate old session", zap.Error(err))
			}
		}

		// Create new session
		session = entities.NewSession(c.deviceID)
		if err := c.hub.sessionRepo.Create(ctx, session); err != nil {
			c.logger.Error("Failed to create new session", zap.Error(err), zap.String("device_id", c.deviceID))
			response = map[string]interface{}{
				"type":      "error",
				"timestamp": time.Now().Unix(),
				"message":   "Failed to create session",
			}
			return
		}
	} else {
		// Update existing session activity
		session.UpdateLastActive()
		if err := c.hub.sessionRepo.Update(ctx, session); err != nil {
			c.logger.Error("Failed to update session", zap.Error(err))
		}
	}

	c.currentSession = session

	// Initialize speech-to-text streaming
	c.sttContext, c.sttCancel = context.WithCancel(ctx)
	
	// Initialize STT repository
	c.sttRepo = &stt.GoogleSpeechToText{}

	// Configure audio for STT
	audioConfig := repositories.AudioConfig{
		SampleRate: 48000,
		Language:   session.Metadata.Language,
		Encoding:   "LINEAR16",
	}

	c.sttStream, err = c.sttRepo.InitTranscribeStreaming(c.sttContext, audioConfig)
	if err != nil {
		c.logger.Error("Failed to initialize STT streaming", zap.Error(err))
		response = map[string]interface{}{
			"type":      "error",
			"timestamp": time.Now().Unix(),
			"message":   "Speech recognition not ready",
		}
		return
	}

	// Initialize TTS repository
	ttsRepoConfig := tts.NewElevenLabsConfigFromEnv()
	c.ttsRepo, err = tts.NewElevenLabsTTS(ttsRepoConfig, c.logger)
	if err != nil {
		c.logger.Error("Failed to initialize TTS repository", zap.Error(err))
		// TTS failure is not critical for listening start
	}

	c.isListening = true

	c.logger.Info("Listening started",
		zap.String("device_id", c.deviceID),
		zap.String("session_id", session.ID.Hex()))

	response = map[string]interface{}{
		"type":       "listening_started",
		"session_id": session.ID.Hex(),
		"timestamp":  time.Now().Unix(),
		"status":     "ready",
	}
}

// handleListeningEnd handles the end of user speech listening
func (c *Client) handleListeningEnd(msg map[string]interface{}) {
	c.sessionMutex.Lock()
	defer c.sessionMutex.Unlock()

	var response map[string]interface{}
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
	}()

	if !c.isListening || c.currentSession == nil {
		response = map[string]interface{}{
			"type":      "error",
			"timestamp": time.Now().Unix(),
			"message":   "Not currently listening",
		}
		return
	}

	c.isListening = false

	// End the STT streaming and get final transcription
	var finalTranscription string
	var err error
	if c.sttStream != nil {
		finalTranscription, err = c.sttStream.End()
		if err != nil {
			c.logger.Error("Failed to end STT streaming", zap.Error(err))
		}
	}

	// Cancel STT context
	if c.sttCancel != nil {
		c.sttCancel()
	}

	c.logger.Info("Listening ended",
		zap.String("device_id", c.deviceID),
		zap.String("session_id", c.currentSession.ID.Hex()),
		zap.String("transcription", finalTranscription))

	response = map[string]interface{}{
		"type":       "listening_ended",
		"session_id": c.currentSession.ID.Hex(),
		"timestamp":  time.Now().Unix(),
		"status":     "completed",
	}

	// Process the transcription asynchronously
	if finalTranscription != "" {
		go c.processUserMessage(c.currentSession.ID, finalTranscription)
	}
}

// processUserMessage processes the user's message and generates a response
func (c *Client) processUserMessage(sessionID primitive.ObjectID, userMessage string) {
	ctx := context.Background()

	// Add user message to session
	userMsg := entities.SessionMessage{
		Timestamp:  time.Now(),
		Role:       entities.MessageRoleUser,
		Content:    userMessage,
		DurationMs: 0, // Could be calculated from audio duration
		Metadata: entities.SessionMessageMetadata{
			TranscriptionConfidence: nil, // Could be provided by STT
		},
	}

	if err := c.hub.sessionRepo.AddMessage(ctx, sessionID, userMsg); err != nil {
		c.logger.Error("Failed to add user message to session", zap.Error(err))
		return
	}

	// Get session with conversation history
	session, err := c.hub.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		c.logger.Error("Failed to get session for LLM processing", zap.Error(err))
		return
	}

	// Convert session messages to LLM format
	var chatMessages []repositories.ChatMessage
	for _, msg := range session.Messages {
		role := repositories.UserRole
		if msg.Role == entities.MessageRoleAssistant {
			role = repositories.DollRole
		}
		chatMessages = append(chatMessages, repositories.ChatMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	// Generate response using LLM
	chatSession, err := c.hub.llm.GenerateChat(ctx, chatMessages)
	if err != nil {
		c.logger.Error("Failed to generate LLM response", zap.Error(err))
		return
	}

	// Get the LLM response
	responseMsg, err := chatSession.SendMessage(ctx, repositories.ChatMessage{
		Role:    repositories.UserRole,
		Content: userMessage,
	})
	if err != nil {
		c.logger.Error("Failed to get LLM response", zap.Error(err))
		return
	}

	// Add assistant message to session
	assistantMsg := entities.SessionMessage{
		Timestamp:  time.Now(),
		Role:       entities.MessageRoleAssistant,
		Content:    responseMsg.Content,
		DurationMs: 0, // Will be calculated during TTS
		Metadata:   entities.SessionMessageMetadata{},
	}

	if err := c.hub.sessionRepo.AddMessage(ctx, sessionID, assistantMsg); err != nil {
		c.logger.Error("Failed to add assistant message to session", zap.Error(err))
		return
	}

	// Generate and stream TTS response
	c.generateSpeechResponse(sessionID, responseMsg.Content)
}

// generateSpeechResponse generates and streams speech from text
func (c *Client) generateSpeechResponse(sessionID primitive.ObjectID, text string) {
	if c.ttsRepo == nil {
		c.logger.Error("TTS repository not available")
		return
	}

	// Send speaking_start message
	speakingStartMsg := map[string]interface{}{
		"type":       "speaking_start",
		"session_id": sessionID.Hex(),
		"timestamp":  time.Now().Unix(),
	}

	speakingStartBytes, _ := json.Marshal(speakingStartMsg)
	select {
	case c.send <- WriteData{
		Type:    websocket.TextMessage,
		Payload: speakingStartBytes,
	}:
	default:
		c.logger.Error("Failed to send speaking_start message")
		return
	}

	c.logger.Info("Started speech generation",
		zap.String("device_id", c.deviceID),
		zap.String("session_id", sessionID.Hex()))

	// Stream TTS audio
	ctx := context.Background()
	audioStream, err := c.ttsRepo.ConvertTextToSpeech(ctx, text)
	if err != nil {
		c.logger.Error("Failed to generate speech", zap.Error(err))
		return
	}

	// Stream audio chunks to client
	for audioChunk := range audioStream {
		select {
		case c.send <- WriteData{
			Type:    websocket.BinaryMessage,
			Payload: audioChunk,
		}:
		default:
			c.logger.Error("Failed to send audio chunk")
			return
		}
	}

	// Send speaking_end message
	speakingEndMsg := map[string]interface{}{
		"type":       "speaking_end",
		"session_id": sessionID.Hex(),
		"timestamp":  time.Now().Unix(),
	}

	speakingEndBytes, _ := json.Marshal(speakingEndMsg)
	select {
	case c.send <- WriteData{
		Type:    websocket.TextMessage,
		Payload: speakingEndBytes,
	}:
	default:
		c.logger.Error("Failed to send speaking_end message")
	}

	c.logger.Info("Completed speech generation",
		zap.String("device_id", c.deviceID),
		zap.String("session_id", sessionID.Hex()))
}
