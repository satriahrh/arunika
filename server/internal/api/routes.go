package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/satriahrh/arunika/server/internal/auth"
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

	// API v1 routes
	v1 := e.Group("/api/v1")

	// Device APIs
	v1.POST("/device/auth", func(c echo.Context) error {
		c.Set("logger", logger)
		return deviceAuth(c)
	})

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

// DeviceAuthRequest represents the request payload for device authentication
type DeviceAuthRequest struct {
	DeviceID string `json:"device_id" validate:"required,min=1,max=100"`
}

// DeviceAuthResponse represents the response payload for device authentication
type DeviceAuthResponse struct {
	Token     string `json:"token"`
	ExpiresIn int64  `json:"expires_in"` // seconds
	DeviceID  string `json:"device_id"`
}

// deviceAuth handles device authentication and returns a JWT token
func deviceAuth(c echo.Context) error {
	logger := c.Get("logger").(*zap.Logger)
	if logger == nil {
		// Fallback logger if not available in context
		logger = zap.NewNop()
	}

	// Parse and validate request payload
	var req DeviceAuthRequest
	if err := c.Bind(&req); err != nil {
		logger.Error("Failed to bind device auth request", zap.Error(err))
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request payload",
		})
	}

	// Validate device_id
	if req.DeviceID == "" {
		logger.Warn("Device auth request with empty device_id")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "device_id is required",
		})
	}

	// Basic device_id validation (no special characters, reasonable length)
	if len(req.DeviceID) > 100 {
		logger.Warn("Device auth request with invalid device_id", zap.String("device_id", req.DeviceID))
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "device_id too long (max 100 characters)",
		})
	}

	// Generate JWT token for the device
	// Note: In a production system, you would validate the device against a database here
	token, err := auth.GenerateDeviceToken(req.DeviceID)
	if err != nil {
		logger.Error("Failed to generate device token", 
			zap.String("device_id", req.DeviceID), 
			zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to generate authentication token",
		})
	}

	logger.Info("Device authenticated successfully", zap.String("device_id", req.DeviceID))

	// Return the JWT token
	response := DeviceAuthResponse{
		Token:     token,
		ExpiresIn: 24 * 60 * 60, // 24 hours in seconds
		DeviceID:  req.DeviceID,
	}

	return c.JSON(http.StatusOK, response)
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
