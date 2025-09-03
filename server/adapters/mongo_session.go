package adapters

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"github.com/satriahrh/arunika/server/domain/entities"
	"github.com/satriahrh/arunika/server/domain/repositories"
)

// MongoSessionRepository implements SessionRepository using MongoDB
type MongoSessionRepository struct {
	collection *mongo.Collection
	logger     *zap.Logger
}

// NewMongoSessionRepository creates a new MongoDB session repository
func NewMongoSessionRepository(db *mongo.Database, logger *zap.Logger) repositories.SessionRepository {
	collection := db.Collection("sessions")
	
	// Create indexes for better performance
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		// Index on device_id for faster lookups
		deviceIDIndex := mongo.IndexModel{
			Keys: bson.D{{Key: "device_id", Value: 1}},
		}
		
		// Index on status and expires_at for cleanup operations
		statusExpiresIndex := mongo.IndexModel{
			Keys: bson.D{
				{Key: "status", Value: 1},
				{Key: "expires_at", Value: 1},
			},
		}
		
		// TTL index for automatic cleanup of expired sessions
		ttlIndex := mongo.IndexModel{
			Keys:    bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0),
		}
		
		_, err := collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
			deviceIDIndex,
			statusExpiresIndex,
			ttlIndex,
		})
		
		if err != nil {
			logger.Error("Failed to create session indexes", zap.Error(err))
		} else {
			logger.Info("Session indexes created successfully")
		}
	}()
	
	return &MongoSessionRepository{
		collection: collection,
		logger:     logger,
	}
}

// Create creates a new session
func (r *MongoSessionRepository) Create(ctx context.Context, session *entities.Session) error {
	if err := session.Validate(); err != nil {
		return err
	}
	
	// Ensure device can only have one active session
	existingSession, err := r.GetActiveByDeviceID(ctx, session.DeviceID)
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return err
	}
	
	if existingSession != nil {
		return errors.New("device already has an active session")
	}
	
	_, err = r.collection.InsertOne(ctx, session)
	if err != nil {
		r.logger.Error("Failed to create session", zap.Error(err), zap.String("device_id", session.DeviceID))
		return err
	}
	
	r.logger.Info("Session created", 
		zap.String("session_id", session.ID.Hex()),
		zap.String("device_id", session.DeviceID))
	
	return nil
}

// GetByID retrieves a session by its ID
func (r *MongoSessionRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*entities.Session, error) {
	var session entities.Session
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&session)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("session not found")
		}
		r.logger.Error("Failed to get session by ID", zap.Error(err), zap.String("session_id", id.Hex()))
		return nil, err
	}
	
	return &session, nil
}

// GetActiveByDeviceID retrieves the active session for a device
func (r *MongoSessionRepository) GetActiveByDeviceID(ctx context.Context, deviceID string) (*entities.Session, error) {
	filter := bson.M{
		"device_id": deviceID,
		"status":    entities.SessionStatusActive,
		"expires_at": bson.M{"$gt": time.Now()},
	}
	
	var session entities.Session
	err := r.collection.FindOne(ctx, filter).Decode(&session)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil // No active session found
		}
		r.logger.Error("Failed to get active session", zap.Error(err), zap.String("device_id", deviceID))
		return nil, err
	}
	
	return &session, nil
}

// GetByDeviceID retrieves sessions for a device with limit
func (r *MongoSessionRepository) GetByDeviceID(ctx context.Context, deviceID string, limit int) ([]*entities.Session, error) {
	filter := bson.M{"device_id": deviceID}
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}). // Most recent first
		SetLimit(int64(limit))
	
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		r.logger.Error("Failed to get sessions by device ID", zap.Error(err), zap.String("device_id", deviceID))
		return nil, err
	}
	defer cursor.Close(ctx)
	
	var sessions []*entities.Session
	for cursor.Next(ctx) {
		var session entities.Session
		if err := cursor.Decode(&session); err != nil {
			r.logger.Error("Failed to decode session", zap.Error(err))
			continue
		}
		sessions = append(sessions, &session)
	}
	
	if err := cursor.Err(); err != nil {
		r.logger.Error("Cursor error", zap.Error(err))
		return nil, err
	}
	
	return sessions, nil
}

// Update updates a session
func (r *MongoSessionRepository) Update(ctx context.Context, session *entities.Session) error {
	if err := session.Validate(); err != nil {
		return err
	}
	
	filter := bson.M{"_id": session.ID}
	update := bson.M{"$set": session}
	
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		r.logger.Error("Failed to update session", zap.Error(err), zap.String("session_id", session.ID.Hex()))
		return err
	}
	
	if result.MatchedCount == 0 {
		return errors.New("session not found")
	}
	
	r.logger.Debug("Session updated", zap.String("session_id", session.ID.Hex()))
	return nil
}

// Delete deletes a session
func (r *MongoSessionRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		r.logger.Error("Failed to delete session", zap.Error(err), zap.String("session_id", id.Hex()))
		return err
	}
	
	if result.DeletedCount == 0 {
		return errors.New("session not found")
	}
	
	r.logger.Info("Session deleted", zap.String("session_id", id.Hex()))
	return nil
}

// ExpireSessions marks expired sessions and cleans them up
func (r *MongoSessionRepository) ExpireSessions(ctx context.Context) error {
	// Mark sessions as expired if they're past their expiration time
	filter := bson.M{
		"status":     entities.SessionStatusActive,
		"expires_at": bson.M{"$lt": time.Now()},
	}
	
	update := bson.M{
		"$set": bson.M{
			"status": entities.SessionStatusExpired,
		},
	}
	
	result, err := r.collection.UpdateMany(ctx, filter, update)
	if err != nil {
		r.logger.Error("Failed to expire sessions", zap.Error(err))
		return err
	}
	
	if result.ModifiedCount > 0 {
		r.logger.Info("Expired sessions", zap.Int64("count", result.ModifiedCount))
	}
	
	return nil
}

// AddMessage adds a message to an existing session
func (r *MongoSessionRepository) AddMessage(ctx context.Context, sessionID primitive.ObjectID, message entities.SessionMessage) error {
	filter := bson.M{"_id": sessionID}
	update := bson.M{
		"$push": bson.M{"messages": message},
		"$set": bson.M{
			"last_message_at": message.Timestamp,
			"last_active_at":  time.Now(),
		},
	}
	
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		r.logger.Error("Failed to add message to session", 
			zap.Error(err), 
			zap.String("session_id", sessionID.Hex()))
		return err
	}
	
	if result.MatchedCount == 0 {
		return errors.New("session not found")
	}
	
	r.logger.Debug("Message added to session", 
		zap.String("session_id", sessionID.Hex()),
		zap.String("role", string(message.Role)))
	
	return nil
}