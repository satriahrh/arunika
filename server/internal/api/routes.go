package api

import (
	"net/http"

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
