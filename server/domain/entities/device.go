package entities

import "time"

// Device represents a doll device
type Device struct {
	ID           string    `json:"id" bson:"_id" db:"id"`
	SerialNumber string    `json:"serial_number" bson:"serial_number" db:"serial_number"`
	SecretKey    string    `json:"secret_key" bson:"secret_key" db:"secret_key"`
	Model        string    `json:"model" bson:"model" db:"model"`
	OwnerID      *string   `json:"owner_id" bson:"owner_id" db:"owner_id"`
	CreatedAt    time.Time `json:"created_at" bson:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" bson:"updated_at" db:"updated_at"`
}
