package entities

import (
	"errors"
	"time"
)

// Device represents a doll device
type Device struct {
	ID           string    `json:"id" db:"id"`
	SerialNumber string    `json:"serial_number" db:"serial_number"`
	Model        string    `json:"model" db:"model"`
	OwnerID      *string   `json:"owner_id" db:"owner_id"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// User represents a parent/user account
type User struct {
	ID        string    `json:"id" db:"id"`
	Email     string    `json:"email" db:"email"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Domain validation methods
func (u *User) Validate() error {
	if u.Email == "" {
		return errors.New("email is required")
	}
	if u.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

func (d *Device) Validate() error {
	if d.SerialNumber == "" {
		return errors.New("serial number is required")
	}
	if d.Model == "" {
		return errors.New("model is required")
	}
	return nil
}
