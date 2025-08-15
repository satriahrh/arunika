package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/satriahrh/arunika/server/domain"
	"github.com/satriahrh/arunika/server/usecase"
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

	// Inbound messages from the clients.
	broadcast chan []byte

	// Mutex for thread-safe access to clients map
	mu sync.RWMutex

	// Conversation service for processing audio
	conversationService *usecase.ConversationService

	logger *zap.Logger
}

// NewHub creates a new WebSocket hub
func NewHub(conversationService *usecase.ConversationService, logger *zap.Logger) *Hub {
	return &Hub{
		clients:             make(map[string]*Client),
		register:            make(chan *Client),
		unregister:          make(chan *Client),
		broadcast:           make(chan []byte),
		conversationService: conversationService,
		logger:              logger,
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

		case message := <-h.broadcast:
			h.mu.RLock()
			for deviceID, client := range h.clients {
				select {
				case client.send <- message:
				default:
					delete(h.clients, deviceID)
					close(client.send)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	// Device ID for this client
	deviceID string

	// Logger
	logger *zap.Logger
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
		send:     make(chan []byte, 256),
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
		send:     make(chan []byte, 256),
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
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Error("WebSocket error", zap.Error(err))
			}
			break
		}

		// Process the message (audio chunk, etc.)
		c.processMessage(message)
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

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
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
	case "audio_chunk":
		c.handleAudioChunk(msg)
	case "ping":
		c.handlePing()
	default:
		c.logger.Warn("Unknown message type", zap.String("type", msgType))
	}
}

// handleAudioChunk processes audio chunks from the device using conversation service
func (c *Client) handleAudioChunk(msg map[string]interface{}) {
	c.logger.Info("Received audio chunk", zap.String("deviceID", c.deviceID))

	// Convert map to AudioChunkMessage
	audioMsg := &domain.AudioChunkMessage{
		Type:       "audio_chunk",
		DeviceID:   c.deviceID,
		SessionID:  getStringFromMap(msg, "session_id"),
		AudioData:  getStringFromMap(msg, "audio_data"),
		SampleRate: getIntFromMap(msg, "sample_rate"),
		Encoding:   getStringFromMap(msg, "encoding"),
		Timestamp:  getStringFromMap(msg, "timestamp"),
		ChunkSeq:   getIntFromMap(msg, "chunk_sequence"),
		IsFinal:    getBoolFromMap(msg, "is_final"),
	}

	// Process using conversation service
	ctx := context.Background()
	response, err := c.hub.conversationService.ProcessAudioChunk(ctx, audioMsg)
	if err != nil {
		c.logger.Error("Failed to process audio chunk", zap.Error(err))
		c.sendErrorResponse("Failed to process audio")
		return
	}

	// Send response back to client
	responseBytes, err := json.Marshal(response)
	if err != nil {
		c.logger.Error("Failed to marshal response", zap.Error(err))
		c.sendErrorResponse("Failed to generate response")
		return
	}

	select {
	case c.send <- responseBytes:
	default:
		close(c.send)
	}
}

// sendErrorResponse sends an error response to the client
func (c *Client) sendErrorResponse(message string) {
	errorResponse := map[string]interface{}{
		"type":      "error",
		"message":   message,
		"timestamp": time.Now().Unix(),
	}

	responseBytes, err := json.Marshal(errorResponse)
	if err != nil {
		c.logger.Error("Failed to marshal error response", zap.Error(err))
		return
	}

	select {
	case c.send <- responseBytes:
	default:
		close(c.send)
	}
}

// Helper functions to extract values from map with type safety
func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getIntFromMap(m map[string]interface{}, key string) int {
	if val, ok := m[key].(float64); ok {
		return int(val)
	}
	if val, ok := m[key].(int); ok {
		return val
	}
	return 0
}

func getBoolFromMap(m map[string]interface{}, key string) bool {
	if val, ok := m[key].(bool); ok {
		return val
	}
	return false
}

// handlePing responds to ping messages
func (c *Client) handlePing() {
	response := map[string]interface{}{
		"type":      "pong",
		"timestamp": time.Now().Unix(),
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		c.logger.Error("Failed to marshal pong response", zap.Error(err))
		return
	}

	select {
	case c.send <- responseBytes:
	default:
		close(c.send)
	}
}
