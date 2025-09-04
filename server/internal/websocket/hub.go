package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

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

	llm         repositories.LargeLanguageModel
	ttsRepo     repositories.TextToSpeech
	sttRepo     repositories.SpeechToText
	sessionRepo repositories.SessionRepository

	logger *zap.Logger
}

// NewHub creates a new WebSocket hub
func NewHub(
	llm repositories.LargeLanguageModel,
	ttsRepo repositories.TextToSpeech,
	sttRepo repositories.SpeechToText,
	sessionRepo repositories.SessionRepository,
	logger *zap.Logger,
) *Hub {
	return &Hub{
		clients:     make(map[string]*Client),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		llm:         llm,
		ttsRepo:     ttsRepo,
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

	// Audio streaming session management
	session      *entities.Session
	sttStreaming repositories.SpeechToTextStreaming
	chatSession  repositories.ChatSession

	chunkCount     int
	listeningStart time.Time

	mutex sync.Mutex
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
		hub:      hub,
		conn:     conn,
		send:     make(chan WriteData, 256),
		deviceID: deviceID,
		logger:   logger,
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
		hub:      hub,
		conn:     conn,
		send:     make(chan WriteData, 256),
		deviceID: deviceID,
		logger:   logger,
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
	default:
		c.logger.Warn("Unknown message type", zap.String("type", msgType))
	}
}

// processBinaryAudioChunk handles binary audio data
func (c *Client) processBinaryAudioChunk(data []byte) {
	c.logger.Info("Received binary audio chunk",
		zap.String("deviceID", c.deviceID),
		zap.Int("size", len(data)))

	// For now, we'll assume there's an active session to update counters
	// In a full implementation, you'd extract session ID from binary headers
	// or track the current active session per device

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.session == nil {
		c.logger.Warn("Received binary audio chunk but no active session found",
			zap.String("deviceID", c.deviceID))
		return
	}

	if c.sttStreaming == nil {
		c.logger.Warn("No active STT streaming for current session",
			zap.String("deviceID", c.deviceID),
			zap.String("sessionID", c.session.ID))
		return
	}

	// Update session counters
	c.chunkCount++

	// Stream audio data to the speech-to-text service
	if err := c.sttStreaming.Stream(data); err != nil {
		c.logger.Error("Failed to stream audio data",
			zap.String("sessionID", c.session.ID),
			zap.Error(err))
		// Optionally, you might want to end the session or take other actions here
		return
	}

	c.logger.Debug("Updated session with binary chunk",
		zap.String("sessionID", c.session.ID),
		zap.Int("totalChunks", c.chunkCount))
}

// handleListeningStart handles the start of an audio streaming session
func (c *Client) handleListeningStart(msg map[string]interface{}) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.chunkCount = 0
	c.listeningStart = time.Now()

	var response map[string]interface{} = map[string]interface{}{
		"type":       "listening_start",
		"session_id": c.session.ID,
		"timestamp":  c.listeningStart.Unix(),
	}

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

	var err error
	if c.session != nil {
		c.session, err = c.hub.sessionRepo.GetLastByDeviceID(ctx, c.deviceID)
		if err != nil {
			c.logger.Error("Failed to get last session by device ID",
				zap.String("deviceID", c.deviceID),
				zap.Error(err))
			response["error"] = "failed to get last session"
			return
		}
	}

	if !c.session.CanContinueThisSession() {
		c.session = &entities.Session{
			DeviceID: c.deviceID,
		}
		err := c.hub.sessionRepo.Create(ctx, c.session)
		if err != nil {
			c.logger.Error("Failed to create new session",
				zap.String("deviceID", c.deviceID),
				zap.Error(err))
			response["error"] = "failed to create new session"
			return
		}
	}

	if c.chatSession == nil {
		c.chatSession, err = c.hub.llm.GenerateChat(ctx, c.session.Messages)
		if err != nil {
			c.logger.Error("Failed to create chat session",
				zap.String("deviceID", c.deviceID),
				zap.Error(err))
			response["error"] = "failed to create chat session"
			return
		}
	}

	audioConfig := repositories.AudioConfig{
		SampleRate: 48000,
		Language:   "id-ID",
		Encoding:   "LINEAR16",
	}
	if v, ok := msg["sample_rate"].(float64); ok && v > 0 {
		audioConfig.SampleRate = int(v)
	}
	if v, ok := msg["language"].(string); ok && v != "" {
		audioConfig.Language = v
	}
	if v, ok := msg["encoding"].(string); ok && v != "" {
		audioConfig.Encoding = v
	}

	c.sttStreaming, err = c.hub.sttRepo.InitTranscribeStreaming(ctx, audioConfig)
	if err != nil {
		c.logger.Error("Failed to initialize streaming transcription",
			zap.String("sessionID", c.session.ID),
			zap.Error(err))
		response["error"] = "failed to initialize transcription"
		return
	}

	c.logger.Info("Audio session started",
		zap.String("deviceID", c.deviceID),
		zap.String("sessionID", c.session.ID))

	response["message"] = "listening started"
}

// handleListeningEnd handles the end of an audio streaming session
func (c *Client) handleListeningEnd(msg map[string]interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var response map[string]interface{} = map[string]interface{}{
		"type":       "listening_end",
		"session_id": c.session.ID,
	}

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

	var finalTranscription string
	var err error
	finalTranscription, err = c.sttStreaming.End()
	if err != nil {
		c.logger.Error("Failed to end transcription stream",
			zap.String("deviceID", c.deviceID),
			zap.String("sessionID", c.session.ID),
			zap.Error(err))
		response["error"] = "failed to end transcription"
		return
	}

	c.logger.Info("Transcription completed",
		zap.String("deviceID", c.deviceID),
		zap.String("sessionID", c.session.ID),
		zap.String("transcription", finalTranscription))

	chatMessage := entities.Message{
		Timestamp:  c.listeningStart,
		Role:       entities.UserRole,
		Content:    finalTranscription,
		DurationMs: time.Since(c.listeningStart).Milliseconds(),
	}
	response["chat"] = chatMessage

	go c.responseAudio(chatMessage)

	c.logger.Info("Starting audio response goroutine",
		zap.String("deviceID", c.deviceID),
		zap.String("sessionID", c.session.ID))

	fmt.Println("Adding user message to session:", msg)
}

func (c *Client) responseAudio(message entities.Message) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	chatResponse, err := c.chatSession.SendMessage(ctx, message)
	if err != nil {
		c.logger.Error("Failed to send message to chat session",
			zap.String("deviceID", c.deviceID),
			zap.String("sessionID", c.session.ID),
			zap.Error(err))
		return
	}

	c.logger.Info("Received chat response",
		zap.String("deviceID", c.deviceID),
		zap.String("sessionID", c.session.ID),
		zap.String("response", chatResponse.Content))

	audioDataChan, err := c.hub.ttsRepo.ConvertTextToSpeech(ctx, chatResponse.Content)
	if err != nil {
		c.logger.Error("Failed to convert text to speech",
			zap.String("deviceID", c.deviceID),
			zap.String("sessionID", c.session.ID),
			zap.Error(err))
		return
	}

	c.send <- WriteData{
		Type: websocket.TextMessage,
		Payload: func() []byte {
			responseBytes, _ := json.Marshal(map[string]interface{}{
				"type":       "speaking_start",
				"session_id": c.session.ID,
				"chat":       chatResponse,
			})
			return responseBytes
		}(),
	}
	for audioData := range audioDataChan {
		c.send <- WriteData{
			Type:    websocket.BinaryMessage,
			Payload: audioData,
		}
	}

	c.send <- WriteData{
		Type: websocket.TextMessage,
		Payload: func() []byte {
			responseBytes, _ := json.Marshal(map[string]interface{}{
				"type":       "speaking_end",
				"session_id": c.session.ID,
				"timestamp":  time.Now().Unix(),
			})
			return responseBytes
		}(),
	}

	c.session.AddMessage(func(s *entities.Session) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := c.hub.sessionRepo.Update(ctx, s)
		if err != nil {
			c.logger.Error("Failed to update session with new messages",
				zap.String("deviceID", c.deviceID),
				zap.String("sessionID", c.session.ID),
				zap.Error(err))
			return err
		}
		return nil
	}, message, chatResponse)
}

// responseWithSampleAlso deprecated
func (c *Client) responseWithSampleAlso() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	time.Sleep(100 * time.Millisecond)

	c.logger.Info("Starting delayed audio response",
		zap.String("deviceID", c.deviceID),
		zap.String("sessionID", c.session.ID),
		zap.String("delay", "5 seconds"))

	time.Sleep(5 * time.Second)

	c.logger.Info("Sending audio response start message",
		zap.String("deviceID", c.deviceID),
		zap.String("sessionID", c.session.ID))

	startMessage := map[string]interface{}{
		"type":       "speaking_start",
		"session_id": c.session.ID,
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
			zap.String("sessionID", c.session.ID),
			zap.String("filePath", audioFilePath),
			zap.Error(err))
		return
	}

	c.logger.Info("Read sample audio file for response",
		zap.String("deviceID", c.deviceID),
		zap.String("sessionID", c.session.ID),
		zap.String("filePath", audioFilePath),
		zap.Int("totalBytes", len(audioFileData)))

	// Send audio data in chunks
	chunkSize := 1024 // 1KB chunks
	totalChunks := (len(audioFileData) + chunkSize - 1) / chunkSize
	sendingChunks := totalChunks / 2

	c.logger.Info("Starting to send audio response chunks",
		zap.String("deviceID", c.deviceID),
		zap.String("sessionID", c.session.ID),
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
			zap.String("sessionID", c.session.ID),
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
		zap.String("sessionID", c.session.ID),
		zap.Int("totalChunksSent", sendingChunks))

	c.logger.Info("Sending audio response end message",
		zap.String("deviceID", c.deviceID),
		zap.String("sessionID", c.session.ID))

	endMessage := map[string]interface{}{
		"type":       "audio_response_ended",
		"session_id": c.session.ID,
		"timestamp":  time.Now().Unix(),
	}
	responseBytes, _ = json.Marshal(endMessage)
	c.send <- WriteData{
		Type:    websocket.TextMessage,
		Payload: responseBytes,
	}

	c.logger.Info("Audio response completed",
		zap.String("deviceID", c.deviceID),
		zap.String("sessionID", c.session.ID))
}
