package repositories

import (
	"context"
	"time"

	"github.com/satriahrh/arunika/server/domain/entities"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserRepository defines data access methods for users
type UserRepository interface {
	Create(ctx context.Context, user *entities.User) error
	GetByID(ctx context.Context, id string) (*entities.User, error)
	GetByEmail(ctx context.Context, email string) (*entities.User, error)
	Update(ctx context.Context, user *entities.User) error
	Delete(ctx context.Context, id string) error
}

// DeviceRepository defines data access methods for devices
type DeviceRepository interface {
	Create(ctx context.Context, device *entities.Device) error
	GetByID(ctx context.Context, id string) (*entities.Device, error)
	GetBySerialNumber(ctx context.Context, serialNumber string) (*entities.Device, error)
	GetByOwnerID(ctx context.Context, ownerID string) ([]*entities.Device, error)
	Update(ctx context.Context, device *entities.Device) error
	Delete(ctx context.Context, id string) error
	// ValidateDevice validates device credentials for authentication
	ValidateDevice(serialNumber, secret string) (*entities.Device, error)
}

// ConversationRepository defines data access methods for conversations
type ConversationRepository interface {
	Create(ctx context.Context, conversation *entities.Conversation) error
	GetByID(ctx context.Context, id string) (*entities.Conversation, error)
	GetByDeviceID(ctx context.Context, deviceID string, limit int) ([]*entities.Conversation, error)
	GetByUserID(ctx context.Context, userID string, limit int) ([]*entities.Conversation, error)
	Update(ctx context.Context, conversation *entities.Conversation) error
	Delete(ctx context.Context, id string) error
}

// MessageRepository defines data access methods for messages
type MessageRepository interface {
	Create(ctx context.Context, message *entities.Message) error
	GetByID(ctx context.Context, id string) (*entities.Message, error)
	GetByConversationID(ctx context.Context, conversationID string) ([]*entities.Message, error)
	GetByTimeRange(ctx context.Context, conversationID string, start, end time.Time) ([]*entities.Message, error)
	Update(ctx context.Context, message *entities.Message) error
	Delete(ctx context.Context, id string) error
}

// SessionRepository defines data access methods for conversation sessions
type SessionRepository interface {
	Create(ctx context.Context, session *entities.Session) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*entities.Session, error)
	GetActiveByDeviceID(ctx context.Context, deviceID string) (*entities.Session, error)
	GetByDeviceID(ctx context.Context, deviceID string, limit int) ([]*entities.Session, error)
	Update(ctx context.Context, session *entities.Session) error
	Delete(ctx context.Context, id primitive.ObjectID) error
	ExpireSessions(ctx context.Context) error // Clean up expired sessions
	AddMessage(ctx context.Context, sessionID primitive.ObjectID, message entities.SessionMessage) error
}
