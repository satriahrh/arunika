package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"

	"github.com/satriahrh/arunika/server/adapters/llm"
	"github.com/satriahrh/arunika/server/adapters/speech"
	"github.com/satriahrh/arunika/server/internal/api"
	"github.com/satriahrh/arunika/server/internal/websocket"
	"github.com/satriahrh/arunika/server/usecase"
)

func main() {
	// Initialize logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Create Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Initialize adapters
	llmService := llm.NewMockGeminiClient()
	speechToText := speech.NewMockSpeechToText(logger)
	textToSpeech := speech.NewMockTextToSpeech(logger)

	// Initialize usecase services
	chatService := usecase.NewChatService(llmService)
	conversationService := usecase.NewConversationService(speechToText, textToSpeech, chatService, logger)

	// Initialize WebSocket hub with conversation service
	hub := websocket.NewHub(conversationService, logger)
	go hub.Run()

	// Initialize API routes
	api.InitRoutes(e, hub, logger)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Graceful shutdown
	go func() {
		if err := e.Start(":" + port); err != nil && err != http.ErrServerClosed {
			logger.Fatal("shutting down the server", zap.Error(err))
		}
	}()

	logger.Info("Server started with clean architecture pattern", zap.String("port", port))

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Info("Server is shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}
