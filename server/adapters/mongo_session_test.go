package adapters

import (
	"context"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"github.com/satriahrh/arunika/server/domain/entities"
)

// TestMongoSessionRepository_Integration tests the MongoDB session repository
// This test requires a running MongoDB instance (skipped if MONGODB_URI is not set)
func TestMongoSessionRepository_Integration(t *testing.T) {
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		t.Skip("Skipping MongoDB integration test - MONGODB_URI not set")
	}

	// Setup
	ctx := context.Background()
	logger, _ := zap.NewDevelopment()

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	// Use test database
	testDB := client.Database("arunika_test")
	defer func() {
		// Clean up test database
		testDB.Drop(ctx)
	}()

	// Create repository
	repo := NewMongoSessionRepository(testDB, logger)

	t.Run("CreateAndGetSession", func(t *testing.T) {
		deviceID := "test-device-001"
		session := entities.NewSession(deviceID)

		// Create session
		err := repo.Create(ctx, session)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Get session by ID
		retrieved, err := repo.GetByID(ctx, session.ID)
		if err != nil {
			t.Fatalf("Failed to get session: %v", err)
		}

		if retrieved.DeviceID != deviceID {
			t.Errorf("Expected device ID %s, got %s", deviceID, retrieved.DeviceID)
		}

		if retrieved.Status != entities.SessionStatusActive {
			t.Errorf("Expected status %s, got %s", entities.SessionStatusActive, retrieved.Status)
		}
	})

	t.Run("GetActiveByDeviceID", func(t *testing.T) {
		deviceID := "test-device-002"
		session := entities.NewSession(deviceID)

		// Create session
		err := repo.Create(ctx, session)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Get active session
		active, err := repo.GetActiveByDeviceID(ctx, deviceID)
		if err != nil {
			t.Fatalf("Failed to get active session: %v", err)
		}

		if active == nil {
			t.Fatal("Expected active session, got nil")
		}

		if active.DeviceID != deviceID {
			t.Errorf("Expected device ID %s, got %s", deviceID, active.DeviceID)
		}
	})

	t.Run("OneSessionPerDevice", func(t *testing.T) {
		deviceID := "test-device-003"
		session1 := entities.NewSession(deviceID)
		session2 := entities.NewSession(deviceID)

		// Create first session
		err := repo.Create(ctx, session1)
		if err != nil {
			t.Fatalf("Failed to create first session: %v", err)
		}

		// Try to create second session for same device (should fail)
		err = repo.Create(ctx, session2)
		if err == nil {
			t.Error("Expected error when creating second session for same device")
		}
	})

	t.Run("AddMessage", func(t *testing.T) {
		deviceID := "test-device-004"
		session := entities.NewSession(deviceID)

		// Create session
		err := repo.Create(ctx, session)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Add message
		message := entities.SessionMessage{
			Timestamp:  time.Now(),
			Role:       entities.MessageRoleUser,
			Content:    "Hello, world!",
			DurationMs: 1500,
			Metadata:   entities.SessionMessageMetadata{},
		}

		err = repo.AddMessage(ctx, session.ID, message)
		if err != nil {
			t.Fatalf("Failed to add message: %v", err)
		}

		// Retrieve session and check message
		updated, err := repo.GetByID(ctx, session.ID)
		if err != nil {
			t.Fatalf("Failed to get updated session: %v", err)
		}

		if len(updated.Messages) != 1 {
			t.Errorf("Expected 1 message, got %d", len(updated.Messages))
		}

		if updated.Messages[0].Content != "Hello, world!" {
			t.Errorf("Expected content 'Hello, world!', got %s", updated.Messages[0].Content)
		}

		if updated.LastMessageAt == nil {
			t.Error("Expected LastMessageAt to be set")
		}
	})

	t.Run("UpdateSession", func(t *testing.T) {
		deviceID := "test-device-005"
		session := entities.NewSession(deviceID)

		// Create session
		err := repo.Create(ctx, session)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Update session
		session.Metadata.Language = "en-US"
		session.UpdateLastActive()

		err = repo.Update(ctx, session)
		if err != nil {
			t.Fatalf("Failed to update session: %v", err)
		}

		// Retrieve and verify update
		updated, err := repo.GetByID(ctx, session.ID)
		if err != nil {
			t.Fatalf("Failed to get updated session: %v", err)
		}

		if updated.Metadata.Language != "en-US" {
			t.Errorf("Expected language en-US, got %s", updated.Metadata.Language)
		}
	})

	t.Run("ExpireSessions", func(t *testing.T) {
		deviceID := "test-device-006"
		session := entities.NewSession(deviceID)

		// Create session with past expiration
		session.ExpiresAt = time.Now().Add(-1 * time.Hour)

		// Need to bypass the Create validation, so let's manually insert
		collection := testDB.Collection("sessions")
		_, err := collection.InsertOne(ctx, session)
		if err != nil {
			t.Fatalf("Failed to insert expired session: %v", err)
		}

		// Run expiration
		err = repo.ExpireSessions(ctx)
		if err != nil {
			t.Fatalf("Failed to expire sessions: %v", err)
		}

		// Verify session is marked as expired
		expired, err := repo.GetByID(ctx, session.ID)
		if err != nil {
			t.Fatalf("Failed to get expired session: %v", err)
		}

		if expired.Status != entities.SessionStatusExpired {
			t.Errorf("Expected status %s, got %s", entities.SessionStatusExpired, expired.Status)
		}
	})
}

// TestMongoSessionRepository_Unit tests the repository without requiring MongoDB
func TestMongoSessionRepository_Unit(t *testing.T) {
	// Test session validation
	t.Run("SessionValidation", func(t *testing.T) {
		session := entities.NewSession("test-device")
		
		// Valid session should pass
		if err := session.Validate(); err != nil {
			t.Errorf("Valid session should pass validation: %v", err)
		}

		// Invalid device ID should fail
		session.DeviceID = ""
		if err := session.Validate(); err == nil {
			t.Error("Session with empty device ID should fail validation")
		}
	})
}