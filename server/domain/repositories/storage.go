package repositories

import (
	"context"

	"github.com/satriahrh/arunika/server/domain/entities"
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
