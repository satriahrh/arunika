package mongo

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// Client wraps the MongoDB client and database
type Client struct {
	*mongo.Client
	Database *mongo.Database
	logger   *zap.Logger
}

// NewClient creates a new MongoDB client connection
func NewClient(logger *zap.Logger) (*Client, error) {
	// Get MongoDB URI from environment variable
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017" // Default for development
	}

	// Get database name from environment variable
	dbName := os.Getenv("MONGODB_DATABASE")
	if dbName == "" {
		dbName = "arunika" // Default database name
	}

	// Set client options
	clientOptions := options.Client().
		ApplyURI(uri).
		SetMaxPoolSize(10).
		SetMinPoolSize(1).
		SetMaxConnIdleTime(30 * time.Minute).
		SetServerSelectionTimeout(5 * time.Second).
		SetConnectTimeout(10 * time.Second)

	// Create context for connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping the database to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	logger.Info("Successfully connected to MongoDB",
		zap.String("database", dbName),
		zap.String("uri", uri))

	return &Client{
		Client:   client,
		Database: client.Database(dbName),
		logger:   logger,
	}, nil
}

// Close closes the MongoDB connection
func (c *Client) Close(ctx context.Context) error {
	if err := c.Client.Disconnect(ctx); err != nil {
		c.logger.Error("Failed to disconnect from MongoDB", zap.Error(err))
		return err
	}
	c.logger.Info("Disconnected from MongoDB")
	return nil
}
