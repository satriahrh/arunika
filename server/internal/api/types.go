package api

import "time"

// DeviceAuthRequest represents the request payload for device authentication
type DeviceAuthRequest struct {
	SerialNumber string `json:"serial_number" validate:"required"`
	SecretKey    string `json:"secret_key" validate:"required"`
}

// DeviceAuthResponse represents the response payload for device authentication
type DeviceAuthResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	DeviceID  string    `json:"device_id"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
