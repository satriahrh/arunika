package repository

import (
	"context"
	"errors"

	"github.com/satriahrh/arunika/server/domain"
)

// MockDeviceRepository is a mock implementation of DeviceRepository for testing/development
type MockDeviceRepository struct {
	devices map[string]*domain.Device
	secrets map[string]string // serial_number -> secret_key mapping
}

// NewMockDeviceRepository creates a new mock device repository with pre-registered devices
func NewMockDeviceRepository() *MockDeviceRepository {
	repo := &MockDeviceRepository{
		devices: make(map[string]*domain.Device),
		secrets: make(map[string]string),
	}

	// Pre-register some test devices
	repo.addTestDevice("ARUNIKA001", "secret123", "doll-v1", nil)
	repo.addTestDevice("ARUNIKA002", "secret456", "doll-v1", nil)
	repo.addTestDevice("ARUNIKA003", "secret789", "doll-v2", nil)

	return repo
}

// addTestDevice adds a test device to the mock repository
func (m *MockDeviceRepository) addTestDevice(serialNumber, secret, model string, ownerID *string) {
	device := &domain.Device{
		ID:           "device-" + serialNumber,
		SerialNumber: serialNumber,
		Model:        model,
		OwnerID:      ownerID,
	}
	m.devices[device.ID] = device
	m.secrets[serialNumber] = secret
}

// ValidateDevice validates device credentials (serial number + secret)
func (m *MockDeviceRepository) ValidateDevice(serialNumber, secret string) (*domain.Device, error) {
	// Check if secret matches
	storedSecret, exists := m.secrets[serialNumber]
	if !exists {
		return nil, errors.New("device not found")
	}

	if storedSecret != secret {
		return nil, errors.New("invalid credentials")
	}

	// Find and return the device
	for _, device := range m.devices {
		if device.SerialNumber == serialNumber {
			return device, nil
		}
	}

	return nil, errors.New("device not found")
}

// Implementation of DeviceRepository interface
func (m *MockDeviceRepository) Create(ctx context.Context, device *domain.Device) error {
	m.devices[device.ID] = device
	return nil
}

func (m *MockDeviceRepository) GetByID(ctx context.Context, id string) (*domain.Device, error) {
	device, exists := m.devices[id]
	if !exists {
		return nil, errors.New("device not found")
	}
	return device, nil
}

func (m *MockDeviceRepository) GetBySerialNumber(ctx context.Context, serialNumber string) (*domain.Device, error) {
	for _, device := range m.devices {
		if device.SerialNumber == serialNumber {
			return device, nil
		}
	}
	return nil, errors.New("device not found")
}

func (m *MockDeviceRepository) GetByOwnerID(ctx context.Context, ownerID string) ([]*domain.Device, error) {
	var devices []*domain.Device
	for _, device := range m.devices {
		if device.OwnerID != nil && *device.OwnerID == ownerID {
			devices = append(devices, device)
		}
	}
	return devices, nil
}

func (m *MockDeviceRepository) Update(ctx context.Context, device *domain.Device) error {
	if _, exists := m.devices[device.ID]; !exists {
		return errors.New("device not found")
	}
	m.devices[device.ID] = device
	return nil
}

func (m *MockDeviceRepository) Delete(ctx context.Context, id string) error {
	if _, exists := m.devices[id]; !exists {
		return errors.New("device not found")
	}
	delete(m.devices, id)
	return nil
}
