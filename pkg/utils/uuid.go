package utils

import (
	"github.com/google/uuid"
)

// NewUUID generates a new UUIDv7 (time-based)
func NewUUID() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

// MustNewUUID generates a new UUIDv7 (time-based) and panics on error
func MustNewUUID() string {
	id, err := NewUUID()
	if err != nil {
		panic(err)
	}
	return id
}

// ParseUUID parses a UUID string into a UUID object
func ParseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

// ValidateUUID checks if a string is a valid UUID
func ValidateUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}
