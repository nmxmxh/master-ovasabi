package utils

import (
	"fmt"

	"github.com/google/uuid"
)

// NewUUID generates a new UUIDv7 (time-based).
func NewUUID() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("failed to generate UUID: %w", err)
	}
	return id.String(), nil
}

// NewUUIDOrDefault generates a new UUIDv7 (time-based) or returns a default if generation fails.
func NewUUIDOrDefault() string {
	id, err := NewUUID()
	if err != nil {
		// Return a nil UUID string as fallback
		return "00000000-0000-0000-0000-000000000000"
	}
	return id
}
