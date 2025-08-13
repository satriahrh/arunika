package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/satriahrh/arunika/server/domain"
	"github.com/satriahrh/arunika/server/internal/auth"
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
		// TODO: Implement proper origin checking for production
		// For now, allow all origins for development
		origin := r.Header.Get("Origin")
		// In production, you would check against allowed origins
		// allowedOrigins := []string{"https://yourdomain.com", "https://app.yourdomain.com"}
		// return contains(allowedOrigins, origin)
		_ = origin // placeholder to avoid unused variable
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Hub maintains the set of active clients and broadcasts messages to the clients.
type Hub struct {
	// Registered clients by device ID.
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

	// Device session state (in-memory for now, Redis integration can be added later)
	deviceSessions map[string]*DeviceSession

	// Session mutex for thread-safe session access
	sessionMu sync.RWMutex

	logger *zap.Logger
}

// DeviceSession holds session state for a connected device
type DeviceSession struct {
	DeviceID      string    `json:"device_id"`
	UserID        string    `json:"user_id,omitempty"`
	SessionID     string    `json:"session_id"`
	ConnectedAt   time.Time `json:"connected_at"`
	LastActivity  time.Time `json:"last_activity"`
	IsActive      bool      `json:"is_active"`
	ConversationID string   `json:"conversation_id,omitempty"`
}

// NewHub creates a new WebSocket hub
func NewHub(conversationService *usecase.ConversationService, logger *zap.Logger) *Hub {
	return &Hub{
		clients:             make(map[string]*Client),
		register:            make(chan *Client),
		unregister:          make(chan *Client),
		broadcast:           make(chan []byte),
		conversationService: conversationService,
		deviceSessions:      make(map[string]*DeviceSession),
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
			
			// Create or update device session
			h.updateDeviceSession(client.deviceID, client.userID, true)
			
			h.logger.Info("Client registered", 
				zap.String("deviceID", client.deviceID),
				zap.String("userID", client.userID),
				zap.String("sessionID", client.sessionID))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.deviceID]; ok {
				delete(h.clients, client.deviceID)
				close(client.send)
			}
			h.mu.Unlock()
			
			// Update device session to inactive
			h.updateDeviceSession(client.deviceID, client.userID, false)
			
			h.logger.Info("Client unregistered", 
				zap.String("deviceID", client.deviceID),
				zap.String("userID", client.userID))

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

	// User ID (if authenticated)
	userID string

	// Session ID for this connection
	sessionID string

	// JWT claims for authorization
	claims *auth.JWTClaims

	// Logger
	logger *zap.Logger
}

// HandleWebSocket handles websocket requests from the peer.
func HandleWebSocket(hub *Hub, c echo.Context, logger *zap.Logger) error {
	// Extract and validate JWT token from query parameter or header
	var token string
	
	// Try query parameter first
	token = c.QueryParam("token")
	if token == "" {
		// Try Authorization header
		authHeader := c.Request().Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}
	
	if token == "" {
		logger.Warn("WebSocket connection attempted without authentication token")
		return fmt.Errorf("authentication token required")
	}
	
	// Validate JWT token
	claims, err := auth.ValidateToken(token)
	if err != nil {
		logger.Error("Invalid JWT token", zap.Error(err))
		return fmt.Errorf("invalid authentication token")
	}
	
	// Extract device ID and user ID from claims
	deviceID := claims.DeviceID
	userID := claims.UserID
	
	// For device role, deviceID is required
	if claims.Role == "device" && deviceID == "" {
		logger.Error("Device token missing device_id")
		return fmt.Errorf("device token must contain device_id")
	}
	
	// For user role, userID is required
	if claims.Role == "user" && userID == "" {
		logger.Error("User token missing user_id")
		return fmt.Errorf("user token must contain user_id")
	}
	
	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		logger.Error("WebSocket upgrade failed", zap.Error(err))
		return err
	}
	
	// Generate session ID
	sessionID := fmt.Sprintf("ws_%s_%d", deviceID, time.Now().UnixNano())
	
	client := &Client{
		hub:       hub,
		conn:      conn,
		send:      make(chan []byte, 256),
		deviceID:  deviceID,
		userID:    userID,
		sessionID: sessionID,
		claims:    claims,
		logger:    logger,
	}
	
	// Log successful connection
	logger.Info("WebSocket client authenticated",
		zap.String("deviceID", deviceID),
		zap.String("userID", userID),
		zap.String("role", claims.Role),
		zap.String("sessionID", sessionID))
	
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
	// Use message validator for proper validation and parsing
	validator := NewMessageValidator()
	parsedMessage, err := validator.ValidateMessage(message)
	if err != nil {
		c.logger.Error("Message validation failed", 
			zap.Error(err),
			zap.String("deviceID", c.deviceID),
			zap.ByteString("rawMessage", message))
		
		errorMsg := CreateErrorMessage("VALIDATION_ERROR", "Invalid message format", err.Error())
		c.sendMessage(errorMsg)
		return
	}

	// Process based on message type
	switch msg := parsedMessage.(type) {
	case *AudioChunkMessage:
		c.handleAudioChunkV2(msg)
	case *PingMessage:
		c.handlePingV2(msg)
	case *DeviceStatusMessage:
		c.handleDeviceStatus(msg)
	case *AuthMessage:
		c.handleAuthMessage(msg)
	default:
		c.logger.Warn("Unhandled message type", 
			zap.String("type", fmt.Sprintf("%T", msg)),
			zap.String("deviceID", c.deviceID))
		
		errorMsg := CreateErrorMessage("UNSUPPORTED_MESSAGE", "Message type not supported", "")
		c.sendMessage(errorMsg)
	}
}

// handleAudioChunkV2 processes audio chunks using the new message format
func (c *Client) handleAudioChunkV2(msg *AudioChunkMessage) {
	c.logger.Info("Received audio chunk", 
		zap.String("deviceID", c.deviceID),
		zap.String("sessionID", msg.SessionID),
		zap.Int("chunkSeq", msg.ChunkSeq),
		zap.Bool("isFinal", msg.IsFinal))

	// Convert to domain AudioChunkMessage
	audioMsg := &domain.AudioChunkMessage{
		Type:       "audio_chunk",
		DeviceID:   msg.DeviceID,
		SessionID:  msg.SessionID,
		AudioData:  msg.AudioData,
		SampleRate: msg.SampleRate,
		Encoding:   msg.Encoding,
		Timestamp:  msg.Timestamp,
		ChunkSeq:   msg.ChunkSeq,
		IsFinal:    msg.IsFinal,
	}

	// Process using conversation service
	ctx := context.Background()
	startTime := time.Now()
	
	response, err := c.hub.conversationService.ProcessAudioChunk(ctx, audioMsg)
	if err != nil {
		c.logger.Error("Failed to process audio chunk", 
			zap.Error(err),
			zap.String("deviceID", c.deviceID),
			zap.String("sessionID", msg.SessionID))
		
		errorMsg := CreateErrorMessage("PROCESSING_ERROR", "Failed to process audio", err.Error())
		c.sendMessage(errorMsg)
		return
	}

	// Create enhanced AI response message
	processingTime := time.Since(startTime).Milliseconds()
	aiResponse := &AIResponseMessage{
		BaseMessage: BaseMessage{
			Type:      MessageTypeAIResponse,
			Timestamp: time.Now().Format(time.RFC3339),
		},
		SessionID:      msg.SessionID,
		Text:           response.Text,
		AudioData:      response.AudioData,
		Emotion:        response.Emotion,
		ProcessingTime: processingTime,
		Confidence:     0.95, // Mock confidence score
	}

	c.sendMessage(aiResponse)
	
	c.logger.Info("Audio chunk processed successfully",
		zap.String("deviceID", c.deviceID),
		zap.String("sessionID", msg.SessionID),
		zap.Int64("processingTimeMs", processingTime))
}

// handlePingV2 handles ping messages with new format
func (c *Client) handlePingV2(msg *PingMessage) {
	c.logger.Debug("Received ping", zap.String("deviceID", c.deviceID))
	
	pongMsg := CreatePongMessage(msg.Data)
	c.sendMessage(pongMsg)
}

// handleDeviceStatus handles device status updates
func (c *Client) handleDeviceStatus(msg *DeviceStatusMessage) {
	c.logger.Info("Device status update", 
		zap.String("deviceID", msg.DeviceID),
		zap.String("status", msg.Status),
		zap.Int("batteryLevel", msg.BatteryLevel))
	
	// Update device session with status information
	if session, exists := c.hub.GetDeviceSession(msg.DeviceID); exists {
		session.LastActivity = time.Now()
		// Could store additional status info in session metadata
	}
	
	// Could broadcast status to interested parties (parents, admin dashboard, etc.)
	// For now, just log it
}

// handleAuthMessage handles authentication-related messages
func (c *Client) handleAuthMessage(msg *AuthMessage) {
	c.logger.Info("Auth message received", 
		zap.String("action", msg.Action),
		zap.String("deviceID", c.deviceID))
	
	switch msg.Action {
	case "refresh":
		// Handle token refresh request
		// For now, just acknowledge
		response := &AuthMessage{
			BaseMessage: BaseMessage{
				Type:      MessageTypeAuth,
				Timestamp: time.Now().Format(time.RFC3339),
			},
			Action: "refreshed",
		}
		c.sendMessage(response)
		
	case "logout":
		// Handle logout request
		c.logger.Info("Client logout requested", zap.String("deviceID", c.deviceID))
		// Close connection gracefully
		c.conn.Close()
		
	default:
		c.logger.Warn("Unknown auth action", zap.String("action", msg.Action))
	}
}

// sendMessage sends a typed message to the client
func (c *Client) sendMessage(message interface{}) {
	messageBytes, err := json.Marshal(message)
	if err != nil {
		c.logger.Error("Failed to marshal message", zap.Error(err))
		return
	}

	select {
	case c.send <- messageBytes:
	default:
		close(c.send)
		c.logger.Warn("Client send channel full, closing connection", 
			zap.String("deviceID", c.deviceID))
	}
}

// Legacy methods for backward compatibility - these can be removed once all clients use new format

// sendErrorResponse sends an error response to the client (legacy method)
func (c *Client) sendErrorResponse(message string) {
	errorMsg := CreateErrorMessage("LEGACY_ERROR", message, "")
	c.sendMessage(errorMsg)
}

// handlePing responds to ping messages (legacy method)
func (c *Client) handlePing() {
	pongMsg := CreatePongMessage("")
	c.sendMessage(pongMsg)
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



// updateDeviceSession creates or updates a device session
func (h *Hub) updateDeviceSession(deviceID, userID string, isActive bool) {
	h.sessionMu.Lock()
	defer h.sessionMu.Unlock()
	
	session, exists := h.deviceSessions[deviceID]
	if !exists {
		session = &DeviceSession{
			DeviceID:    deviceID,
			UserID:      userID,
			SessionID:   fmt.Sprintf("session_%s_%d", deviceID, time.Now().UnixNano()),
			ConnectedAt: time.Now(),
		}
		h.deviceSessions[deviceID] = session
	}
	
	session.IsActive = isActive
	session.LastActivity = time.Now()
	
	if !isActive {
		h.logger.Info("Device session ended",
			zap.String("deviceID", deviceID),
			zap.String("sessionID", session.SessionID),
			zap.Duration("duration", time.Since(session.ConnectedAt)))
	}
}

// GetDeviceSession returns the session for a device
func (h *Hub) GetDeviceSession(deviceID string) (*DeviceSession, bool) {
	h.sessionMu.RLock()
	defer h.sessionMu.RUnlock()
	session, exists := h.deviceSessions[deviceID]
	return session, exists
}

// GetActiveDevices returns a list of currently active device IDs
func (h *Hub) GetActiveDevices() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	devices := make([]string, 0, len(h.clients))
	for deviceID := range h.clients {
		devices = append(devices, deviceID)
	}
	return devices
}

// SendToDevice sends a message to a specific device
func (h *Hub) SendToDevice(deviceID string, message []byte) error {
	h.mu.RLock()
	client, exists := h.clients[deviceID]
	h.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("device not connected: %s", deviceID)
	}
	
	select {
	case client.send <- message:
		return nil
	default:
		return fmt.Errorf("failed to send message to device: %s", deviceID)
	}
}
