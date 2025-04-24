package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUUID(t *testing.T) {
	// Generate two UUIDs with a small delay between them
	id1, err := NewUUID()
	require.NoError(t, err)
	time.Sleep(time.Millisecond)
	id2, err := NewUUID()
	require.NoError(t, err)

	// Test that UUIDs are valid
	assert.True(t, ValidateUUID(id1))
	assert.True(t, ValidateUUID(id2))

	// Test that UUIDs are different
	assert.NotEqual(t, id1, id2)

	// Test that UUIDs are time-ordered (UUIDv7 property)
	uuid1, err := ParseUUID(id1)
	require.NoError(t, err)
	uuid2, err := ParseUUID(id2)
	require.NoError(t, err)
	assert.Less(t, uuid1.String(), uuid2.String(), "UUIDs should be time-ordered")
}

func TestMustNewUUID(t *testing.T) {
	// Test normal operation
	assert.NotPanics(t, func() {
		id := MustNewUUID()
		assert.True(t, ValidateUUID(id))
	})
}

func TestParseUUID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid UUID",
			input:   "123e4567-e89b-12d3-a456-426614174000",
			wantErr: false,
		},
		{
			name:    "invalid UUID - wrong format",
			input:   "not-a-uuid",
			wantErr: true,
		},
		{
			name:    "invalid UUID - too short",
			input:   "123e4567",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseUUID(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateUUID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "valid UUID",
			input: "123e4567-e89b-12d3-a456-426614174000",
			want:  true,
		},
		{
			name:  "invalid UUID - wrong format",
			input: "not-a-uuid",
			want:  false,
		},
		{
			name:  "invalid UUID - too short",
			input: "123e4567",
			want:  false,
		},
		{
			name:  "empty string",
			input: "",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ValidateUUID(tt.input))
		})
	}
}
