package api

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/satriahrh/arunika/server/internal/websocket"
)

// InitRoutes initializes all API routes
func InitRoutes(e *echo.Echo, hub *websocket.Hub, logger *zap.Logger) {
	// Health check
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status":  "ok",
			"service": "arunika-server",
		})
	})

	// WebSocket stats and management endpoints
	e.GET("/ws/stats", func(c echo.Context) error {
		return getWebSocketStats(c, hub)
	})
	
	e.GET("/ws/devices", func(c echo.Context) error {
		return getActiveDevices(c, hub)
	})

	// API v1 routes
	v1 := e.Group("/api/v1")

	// Device APIs
	v1.POST("/device/auth", deviceAuth)

	// User Management APIs
	v1.POST("/users/register", userRegister)
	v1.POST("/users/login", userLogin)

	// Child Profiles APIs
	v1.GET("/children", getChildren)
	v1.POST("/children", createChild)
	v1.PUT("/children/:id", updateChild)

	// Conversation History APIs
	v1.GET("/conversations", getConversations)

	// WebSocket endpoint
	e.GET("/ws", func(c echo.Context) error {
		return websocket.HandleWebSocket(hub, c, logger)
	})
}

// Placeholder handlers - to be implemented
func deviceAuth(c echo.Context) error {
	// TODO: Implement device authentication
	return c.JSON(http.StatusOK, map[string]string{
		"message": "Device auth endpoint - to be implemented",
	})
}

func userRegister(c echo.Context) error {
	// TODO: Implement user registration
	return c.JSON(http.StatusOK, map[string]string{
		"message": "User register endpoint - to be implemented",
	})
}

func userLogin(c echo.Context) error {
	// TODO: Implement user login
	return c.JSON(http.StatusOK, map[string]string{
		"message": "User login endpoint - to be implemented",
	})
}

func getChildren(c echo.Context) error {
	// TODO: Implement get children profiles
	return c.JSON(http.StatusOK, map[string]string{
		"message": "Get children endpoint - to be implemented",
	})
}

func createChild(c echo.Context) error {
	// TODO: Implement create child profile
	return c.JSON(http.StatusOK, map[string]string{
		"message": "Create child endpoint - to be implemented",
	})
}

func updateChild(c echo.Context) error {
	// TODO: Implement update child profile
	return c.JSON(http.StatusOK, map[string]string{
		"message": "Update child endpoint - to be implemented",
	})
}

func getConversations(c echo.Context) error {
	// TODO: Implement get conversations
	return c.JSON(http.StatusOK, map[string]string{
		"message": "Get conversations endpoint - to be implemented",
	})
}

// WebSocket management endpoints

// getWebSocketStats returns WebSocket connection statistics
func getWebSocketStats(c echo.Context, hub *websocket.Hub) error {
	activeDevices := hub.GetActiveDevices()
	
	stats := map[string]interface{}{
		"active_connections": len(activeDevices),
		"active_devices":     activeDevices,
		"server_time":        time.Now().Format(time.RFC3339),
		"uptime_seconds":     time.Since(startTime).Seconds(),
	}
	
	return c.JSON(http.StatusOK, stats)
}

// getActiveDevices returns list of currently active devices with session info
func getActiveDevices(c echo.Context, hub *websocket.Hub) error {
	activeDeviceIDs := hub.GetActiveDevices()
	devices := make([]map[string]interface{}, 0, len(activeDeviceIDs))
	
	for _, deviceID := range activeDeviceIDs {
		if session, exists := hub.GetDeviceSession(deviceID); exists {
			devices = append(devices, map[string]interface{}{
				"device_id":       session.DeviceID,
				"user_id":         session.UserID,
				"session_id":      session.SessionID,
				"connected_at":    session.ConnectedAt.Format(time.RFC3339),
				"last_activity":   session.LastActivity.Format(time.RFC3339),
				"is_active":       session.IsActive,
				"conversation_id": session.ConversationID,
			})
		} else {
			// Fallback if session not found
			devices = append(devices, map[string]interface{}{
				"device_id": deviceID,
				"status":    "connected_no_session",
			})
		}
	}
	
	return c.JSON(http.StatusOK, map[string]interface{}{
		"devices": devices,
		"count":   len(devices),
	})
}

// Server start time for uptime calculation
var startTime = time.Now()
