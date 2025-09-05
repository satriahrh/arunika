package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/satriahrh/arunika/server/domain/entities"
	"github.com/satriahrh/arunika/server/domain/repositories"
)

type SessionRepository struct {
	collection *mongo.Collection
}

// NewSessionRepository creates a new MongoDB session repository
func NewSessionRepository(db *mongo.Database) repositories.SessionRepository {
	return &SessionRepository{
		collection: db.Collection("sessions"),
	}
}

// Create implements repositories.SessionRepository
func (r *SessionRepository) Create(ctx context.Context, session *entities.Session) error {
	if session == nil {
		return errors.New("session cannot be nil")
	}

	// Set creation timestamps if not already set
	now := time.Now()
	if session.CreatedAt.IsZero() {
		session.CreatedAt = now
	}
	if session.LastMessageAt.IsZero() {
		session.LastMessageAt = now
	}

	// Convert to MongoDB document
	doc := bson.M{
		"device_id":       session.DeviceID,
		"created_at":      session.CreatedAt,
		"last_message_at": session.LastMessageAt,
		"messages":        session.Messages,
		"metadata":        session.Metadata,
	}

	// Insert the document
	result, err := r.collection.InsertOne(ctx, doc)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	// Set the generated ID back to the session
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		session.ID = oid.Hex()
	}

	return nil
}

// GetLastByDeviceID implements repositories.SessionRepository
func (r *SessionRepository) GetLastByDeviceID(ctx context.Context, deviceID string) (*entities.Session, error) {
	if deviceID == "" {
		return nil, errors.New("device ID cannot be empty")
	}

	// Find the most recent session for the device
	filter := bson.M{"device_id": deviceID}
	opts := options.FindOne().SetSort(bson.M{"last_message_at": -1})

	var session entities.Session
	err := r.collection.FindOne(ctx, filter, opts).Decode(&session)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil // No session found, return nil without error
		}
		return nil, fmt.Errorf("failed to get last session for device %s: %w", deviceID, err)
	}

	return &session, nil
}

// Update implements repositories.SessionRepository
func (r *SessionRepository) Update(ctx context.Context, session *entities.Session) error {
	if session == nil {
		return errors.New("session cannot be nil")
	}
	if session.ID == "" {
		return errors.New("session ID cannot be empty")
	}

	// Convert string ID to ObjectID
	objectID, err := primitive.ObjectIDFromHex(session.ID)
	if err != nil {
		return fmt.Errorf("invalid session ID format: %w", err)
	}

	// Prepare update document
	update := bson.M{
		"$set": bson.M{
			"device_id":       session.DeviceID,
			"last_message_at": session.LastMessageAt,
			"messages":        session.Messages,
			"metadata":        session.Metadata,
		},
	}

	// Update the document
	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		update,
	)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	// Check if the document was found and updated
	if result.MatchedCount == 0 {
		return fmt.Errorf("session with ID %s not found", session.ID)
	}

	return nil
}
