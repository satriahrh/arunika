package api

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/satriahrh/arunika/server/internal/auth"
	"github.com/satriahrh/arunika/server/internal/websocket"
	"github.com/satriahrh/arunika/server/repository"
)

// InitRoutes initializes all API routes
func InitRoutes(e *echo.Echo, hub *websocket.Hub, deviceRepo repository.DeviceRepository, logger *zap.Logger) {
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
		return deviceAuth(c, deviceRepo, logger)
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

	// WebSocket endpoint with JWT validation
	e.GET("/ws", func(c echo.Context) error {
		return websocketWithAuth(hub, c, logger)
	})
}

// Placeholder handlers - to be implemented
func deviceAuth(c echo.Context, deviceRepo repository.DeviceRepository, logger *zap.Logger) error {
	var req DeviceAuthRequest

	// Bind and validate request
	if err := c.Bind(&req); err != nil {
		logger.Error("Failed to bind device auth request", zap.Error(err))
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request format",
		})
	}

	// Validate required fields
	if req.SerialNumber == "" || req.SecretKey == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "missing_fields",
			Message: "Serial number and secret key are required",
		})
	}

	// Validate device credentials using mock repository
	mockRepo, ok := deviceRepo.(*repository.MockDeviceRepository)
	if !ok {
		logger.Error("Expected MockDeviceRepository")
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Repository configuration error",
		})
	}

	device, err := mockRepo.ValidateDevice(req.SerialNumber, req.SecretKey)
	if err != nil {
		logger.Warn("Device authentication failed",
			zap.String("serial_number", req.SerialNumber),
			zap.Error(err))
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "authentication_failed",
			Message: "Invalid device credentials",
		})
	}

	// Generate JWT token for the device
	token, err := auth.GenerateDeviceToken(device.ID)
	if err != nil {
		logger.Error("Failed to generate device token",
			zap.String("device_id", device.ID),
			zap.Error(err))
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "token_generation_failed",
			Message: "Failed to generate authentication token",
		})
	}

	// Calculate expiration time (24 hours from now, matching JWT claims)
	expiresAt := time.Now().Add(24 * time.Hour)

	logger.Info("Device authenticated successfully",
		zap.String("device_id", device.ID),
		zap.String("serial_number", device.SerialNumber))

	return c.JSON(http.StatusOK, DeviceAuthResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		DeviceID:  device.ID,
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

// websocketWithAuth handles WebSocket connections with JWT authentication
func websocketWithAuth(hub *websocket.Hub, c echo.Context, logger *zap.Logger) error {
	// Extract JWT token from Authorization header only
	var token string
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		token = authHeader[7:]
	}

	if token == "" {
		logger.Warn("WebSocket connection rejected: missing token")
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "missing_token",
			Message: "JWT token is required in Authorization header",
		})
	}

	// Validate JWT token
	claims, err := auth.ValidateToken(token)
	if err != nil {
		logger.Warn("WebSocket connection rejected: invalid token", zap.Error(err))
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "invalid_token",
			Message: "Invalid or expired JWT token",
		})
	}

	// Verify this is a device token
	if claims.Role != "device" {
		logger.Warn("WebSocket connection rejected: invalid role",
			zap.String("role", claims.Role))
		return c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "invalid_role",
			Message: "Only device tokens are allowed for WebSocket connections",
		})
	}

	// Extract device ID from JWT claims
	deviceID := claims.DeviceID
	if deviceID == "" {
		logger.Error("WebSocket connection rejected: missing device ID in token")
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_token_claims",
			Message: "Device ID not found in token",
		})
	}

	logger.Info("WebSocket connection authenticated",
		zap.String("device_id", deviceID),
		zap.String("role", claims.Role))

	// Handle WebSocket connection with authenticated device ID
	return websocket.HandleWebSocketWithAuth(hub, c, deviceID, logger)
}
