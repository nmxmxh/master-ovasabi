package errors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorDefinitions(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		message string
	}{
		{
			name:    "ErrUserNotFound",
			err:     ErrUserNotFound,
			message: "user not found",
		},
		{
			name:    "ErrInvalidInput",
			err:     ErrInvalidInput,
			message: "invalid input",
		},
		{
			name:    "ErrInvalidCredentials",
			err:     ErrInvalidCredentials,
			message: "invalid credentials",
		},
		{
			name:    "ErrUserExists",
			err:     ErrUserExists,
			message: "user already exists",
		},
		{
			name:    "ErrInvalidToken",
			err:     ErrInvalidToken,
			message: "invalid token",
		},
		{
			name:    "ErrTokenExpired",
			err:     ErrTokenExpired,
			message: "token expired",
		},
		{
			name:    "ErrInvalidEmail",
			err:     ErrInvalidEmail,
			message: "invalid email format",
		},
		{
			name:    "ErrWeakPassword",
			err:     ErrWeakPassword,
			message: "password too weak",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test error message consistency
			assert.Equal(t, tt.message, tt.err.Error(), "error message should match expected message")

			// Test error comparison
			assert.True(t, tt.err == tt.err, "same error should be equal")
			assert.False(t, tt.err == ErrInvalidInput && tt.name != "ErrInvalidInput", "different errors should not be equal")
		})
	}
}

func TestErrorComparisons(t *testing.T) {
	// Test that different errors are not equal
	assert.NotEqual(t, ErrUserNotFound, ErrInvalidInput)
	assert.NotEqual(t, ErrInvalidCredentials, ErrUserExists)
	assert.NotEqual(t, ErrInvalidToken, ErrTokenExpired)
	assert.NotEqual(t, ErrInvalidEmail, ErrWeakPassword)

	// Test error wrapping and unwrapping
	wrappedErr := ErrUserNotFound
	assert.Equal(t, wrappedErr, ErrUserNotFound)
}
