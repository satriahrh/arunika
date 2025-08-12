package repository

import (
	"context"
	"time"

	"github.com/satriahrh/arunika/server/domain"
)

// UserRepository defines data access methods for users
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id string) error
}

// DeviceRepository defines data access methods for devices
type DeviceRepository interface {
	Create(ctx context.Context, device *domain.Device) error
	GetByID(ctx context.Context, id string) (*domain.Device, error)
	GetBySerialNumber(ctx context.Context, serialNumber string) (*domain.Device, error)
	GetByOwnerID(ctx context.Context, ownerID string) ([]*domain.Device, error)
	Update(ctx context.Context, device *domain.Device) error
	Delete(ctx context.Context, id string) error
}

// ConversationRepository defines data access methods for conversations
type ConversationRepository interface {
	Create(ctx context.Context, conversation *domain.Conversation) error
	GetByID(ctx context.Context, id string) (*domain.Conversation, error)
	GetByDeviceID(ctx context.Context, deviceID string, limit int) ([]*domain.Conversation, error)
	GetByUserID(ctx context.Context, userID string, limit int) ([]*domain.Conversation, error)
	Update(ctx context.Context, conversation *domain.Conversation) error
	Delete(ctx context.Context, id string) error
}

// MessageRepository defines data access methods for messages
type MessageRepository interface {
	Create(ctx context.Context, message *domain.Message) error
	GetByID(ctx context.Context, id string) (*domain.Message, error)
	GetByConversationID(ctx context.Context, conversationID string) ([]*domain.Message, error)
	GetByTimeRange(ctx context.Context, conversationID string, start, end time.Time) ([]*domain.Message, error)
	Update(ctx context.Context, message *domain.Message) error
	Delete(ctx context.Context, id string) error
}
