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

// ParseUUID parses a UUID string into a UUID object.
func ParseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

// ValidateUUID checks if a string is a valid UUID.
func ValidateUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}
