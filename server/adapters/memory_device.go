package adapters

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/satriahrh/arunika/server/domain/entities"
)

// MemoryDeviceRepository is a production-ready in-memory implementation of DeviceRepository
// This is suitable for production use as a simple storage backend
type MemoryDeviceRepository struct {
	mu      sync.RWMutex
	devices map[string]*entities.Device           // id -> device mapping
	secrets map[string]string                     // serial_number -> secret_key mapping
	serials map[string]*entities.Device           // serial_number -> device mapping
	owners  map[string][]*entities.Device         // owner_id -> devices mapping
}

// NewMemoryDeviceRepository creates a new in-memory device repository
// This is a clean production implementation without pre-registered test data
func NewMemoryDeviceRepository() *MemoryDeviceRepository {
	return &MemoryDeviceRepository{
		devices: make(map[string]*entities.Device),
		secrets: make(map[string]string),
		serials: make(map[string]*entities.Device),
		owners:  make(map[string][]*entities.Device),
	}
}

// ValidateDevice validates device credentials (serial number + secret)
// This method is used for device authentication
func (m *MemoryDeviceRepository) ValidateDevice(serialNumber, secret string) (*entities.Device, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if secret matches
	storedSecret, exists := m.secrets[serialNumber]
	if !exists {
		return nil, errors.New("device not found")
	}

	if storedSecret != secret {
		return nil, errors.New("invalid credentials")
	}

	// Find and return the device
	device, exists := m.serials[serialNumber]
	if !exists {
		return nil, errors.New("device not found")
	}

	return device, nil
}

// Create implements DeviceRepository interface
func (m *MemoryDeviceRepository) Create(ctx context.Context, device *entities.Device) error {
	if device == nil {
		return errors.New("device cannot be nil")
	}

	if err := device.Validate(); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if device with same serial number already exists
	if _, exists := m.serials[device.SerialNumber]; exists {
		return errors.New("device with this serial number already exists")
	}

	// Generate ID if not provided
	if device.ID == "" {
		device.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	device.CreatedAt = now
	device.UpdatedAt = now

	// Store device
	deviceCopy := *device
	m.devices[device.ID] = &deviceCopy
	m.serials[device.SerialNumber] = &deviceCopy

	// Update owner mapping if owner is set
	if device.OwnerID != nil {
		ownerID := *device.OwnerID
		m.owners[ownerID] = append(m.owners[ownerID], &deviceCopy)
	}

	return nil
}

// GetByID implements DeviceRepository interface
func (m *MemoryDeviceRepository) GetByID(ctx context.Context, id string) (*entities.Device, error) {
	if id == "" {
		return nil, errors.New("device ID cannot be empty")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	device, exists := m.devices[id]
	if !exists {
		return nil, errors.New("device not found")
	}

	// Return a copy to prevent external modifications
	deviceCopy := *device
	return &deviceCopy, nil
}

// GetBySerialNumber implements DeviceRepository interface
func (m *MemoryDeviceRepository) GetBySerialNumber(ctx context.Context, serialNumber string) (*entities.Device, error) {
	if serialNumber == "" {
		return nil, errors.New("serial number cannot be empty")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	device, exists := m.serials[serialNumber]
	if !exists {
		return nil, errors.New("device not found")
	}

	// Return a copy to prevent external modifications
	deviceCopy := *device
	return &deviceCopy, nil
}

// GetByOwnerID implements DeviceRepository interface
func (m *MemoryDeviceRepository) GetByOwnerID(ctx context.Context, ownerID string) ([]*entities.Device, error) {
	if ownerID == "" {
		return nil, errors.New("owner ID cannot be empty")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	devices, exists := m.owners[ownerID]
	if !exists {
		return []*entities.Device{}, nil // Return empty slice instead of nil
	}

	// Return copies to prevent external modifications
	result := make([]*entities.Device, len(devices))
	for i, device := range devices {
		deviceCopy := *device
		result[i] = &deviceCopy
	}

	return result, nil
}

// Update implements DeviceRepository interface
func (m *MemoryDeviceRepository) Update(ctx context.Context, device *entities.Device) error {
	if device == nil {
		return errors.New("device cannot be nil")
	}

	if device.ID == "" {
		return errors.New("device ID cannot be empty")
	}

	if err := device.Validate(); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if device exists
	existingDevice, exists := m.devices[device.ID]
	if !exists {
		return errors.New("device not found")
	}

	// Check if serial number is being changed and conflicts with another device
	if existingDevice.SerialNumber != device.SerialNumber {
		if _, exists := m.serials[device.SerialNumber]; exists {
			return errors.New("device with this serial number already exists")
		}
	}

	// Update timestamps
	device.UpdatedAt = time.Now()
	device.CreatedAt = existingDevice.CreatedAt // Preserve original creation time

	// Remove old mappings
	delete(m.serials, existingDevice.SerialNumber)
	if existingDevice.OwnerID != nil {
		oldOwnerID := *existingDevice.OwnerID
		ownerDevices := m.owners[oldOwnerID]
		for i, d := range ownerDevices {
			if d.ID == device.ID {
				m.owners[oldOwnerID] = append(ownerDevices[:i], ownerDevices[i+1:]...)
				if len(m.owners[oldOwnerID]) == 0 {
					delete(m.owners, oldOwnerID)
				}
				break
			}
		}
	}

	// Store updated device
	deviceCopy := *device
	m.devices[device.ID] = &deviceCopy
	m.serials[device.SerialNumber] = &deviceCopy

	// Update owner mapping if owner is set
	if device.OwnerID != nil {
		ownerID := *device.OwnerID
		m.owners[ownerID] = append(m.owners[ownerID], &deviceCopy)
	}

	return nil
}

// Delete implements DeviceRepository interface
func (m *MemoryDeviceRepository) Delete(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("device ID cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	device, exists := m.devices[id]
	if !exists {
		return errors.New("device not found")
	}

	// Remove from all mappings
	delete(m.devices, id)
	delete(m.serials, device.SerialNumber)
	delete(m.secrets, device.SerialNumber)

	// Remove from owner mapping
	if device.OwnerID != nil {
		ownerID := *device.OwnerID
		ownerDevices := m.owners[ownerID]
		for i, d := range ownerDevices {
			if d.ID == id {
				m.owners[ownerID] = append(ownerDevices[:i], ownerDevices[i+1:]...)
				if len(m.owners[ownerID]) == 0 {
					delete(m.owners, ownerID)
				}
				break
			}
		}
	}

	return nil
}

// RegisterDeviceSecret registers a secret for a device's serial number
// This method is used to set up device authentication credentials
func (m *MemoryDeviceRepository) RegisterDeviceSecret(serialNumber, secret string) error {
	if serialNumber == "" {
		return errors.New("serial number cannot be empty")
	}
	if secret == "" {
		return errors.New("secret cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.secrets[serialNumber] = secret
	return nil
}

// RemoveDeviceSecret removes the secret for a device's serial number
func (m *MemoryDeviceRepository) RemoveDeviceSecret(serialNumber string) error {
	if serialNumber == "" {
		return errors.New("serial number cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.secrets, serialNumber)
	return nil
}