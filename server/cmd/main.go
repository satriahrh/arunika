package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"

	"github.com/satriahrh/arunika/server/adapters"
	"github.com/satriahrh/arunika/server/adapters/llm"
	"github.com/satriahrh/arunika/server/adapters/mongo"
	"github.com/satriahrh/arunika/server/adapters/stt"
	"github.com/satriahrh/arunika/server/adapters/tts"
	"github.com/satriahrh/arunika/server/domain/entities"
	"github.com/satriahrh/arunika/server/internal/api"
	"github.com/satriahrh/arunika/server/internal/websocket"
)

func main() {
	godotenv.Load()

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
	// Initialize MongoDB client
	mongoClient, err := mongo.NewClient(logger)
	if err != nil {
		logger.Fatal("Failed to create MongoDB client", zap.Error(err))
	}
	defer func() {
		ctx := context.Background()
		mongoClient.Close(ctx)
	}()

	// Initialize repositories
	sessionRepo := mongo.NewSessionRepository(mongoClient.Database)
	deviceRepo := adapters.NewMemoryDeviceRepository()
	sttRepo := &stt.GoogleSpeechToText{}
	ttsRepoConfig := tts.NewElevenLabsConfigFromEnv()
	ttsRepo, err := tts.NewElevenLabsTTS(ttsRepoConfig, logger)
	if err != nil {
		logger.Fatal("Failed to create TTS repository", zap.Error(err))
	}
	geminiLLMRepo, err := llm.NewGeminiLLM(logger)
	if err != nil {
		logger.Fatal("Failed to create Gemini LLM", zap.Error(err))
	}

	// Bootstrap with demo devices for development (in production, devices would be provisioned through separate APIs)
	if err := bootstrapDemoDevices(deviceRepo, logger); err != nil {
		logger.Warn("Failed to bootstrap demo devices", zap.Error(err))
	}

	// Initialize WebSocket hub with conversation service
	hub := websocket.NewHub(geminiLLMRepo, ttsRepo, sttRepo, sessionRepo, logger)
	go hub.Run()

	// Initialize API routes
	api.InitRoutes(e, hub, deviceRepo, logger)

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

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

// bootstrapDemoDevices sets up demo devices for development and testing
// In production, devices would be provisioned through device management APIs
func bootstrapDemoDevices(deviceRepo *adapters.MemoryDeviceRepository, logger *zap.Logger) error {
	ctx := context.Background()

	// Demo devices with production-style IDs and credentials
	demoDevices := []struct {
		serialNumber string
		secret       string
		model        string
	}{
		{"ARUNIKA001", "secret123", "doll-v1"},
		{"ARUNIKA002", "secret456", "doll-v1"},
		{"ARUNIKA003", "secret789", "doll-v2"},
	}

	for _, demo := range demoDevices {
		// Create device entity
		device := &entities.Device{
			SerialNumber: demo.serialNumber,
			Model:        demo.model,
			OwnerID:      nil, // No owner initially
		}

		// Create device in repository
		if err := deviceRepo.Create(ctx, device); err != nil {
			return err
		}

		// Register device secret for authentication
		if err := deviceRepo.RegisterDeviceSecret(demo.serialNumber, demo.secret); err != nil {
			return err
		}

		logger.Info("Bootstrapped demo device",
			zap.String("serial_number", demo.serialNumber),
			zap.String("device_id", device.ID),
			zap.String("model", demo.model))
	}

	return nil
}
