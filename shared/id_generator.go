package shared

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// IDGenerator provides unified ID generation across all systems.
type IDGenerator struct {
	// Common prefixes for different ID types
	prefixes map[string]string
	// Lengths for different ID types
	lengths map[string]int
}

// NewIDGenerator creates a new ID generator with standardized configuration.
func NewIDGenerator() *IDGenerator {
	return &IDGenerator{
		prefixes: map[string]string{
			"user":        "user",
			"session":     "session",
			"device":      "device",
			"campaign":    "campaign",
			"correlation": "corr",
			"guest":       "guest",
		},
		lengths: map[string]int{
			"user":        32,
			"session":     32,
			"device":      32,
			"campaign":    24,
			"correlation": 24,
			"guest":       32,
		},
	}
}

// GenerateID creates a standardized ID with the given type.
func (g *IDGenerator) GenerateID(idType string, additionalData ...string) string {
	prefix, exists := g.prefixes[idType]
	if !exists {
		prefix = "id"
	}

	length, exists := g.lengths[idType]
	if !exists {
		length = 32
	}

	// Create input for hash generation
	input := fmt.Sprintf("%s_%d_%s", prefix, time.Now().UnixNano(), idType)
	for _, data := range additionalData {
		input += "_" + data
	}

	// Generate SHA256 hash
	hash := sha256.Sum256([]byte(input))
	hashStr := hex.EncodeToString(hash[:])

	// Ensure consistent length
	if len(hashStr) > length {
		hashStr = hashStr[:length]
	} else if len(hashStr) < length {
		// Pad with additional entropy if needed
		additional := sha256.Sum256([]byte(hashStr + time.Now().String()))
		additionalStr := hex.EncodeToString(additional[:])
		hashStr = hashStr + additionalStr[:length-len(hashStr)]
	}

	return prefix + "_" + hashStr
}

// GenerateUserID creates a user ID.
func (g *IDGenerator) GenerateUserID() string {
	return g.GenerateID("user")
}

// GenerateGuestID creates a guest user ID.
func (g *IDGenerator) GenerateGuestID() string {
	return g.GenerateID("guest")
}

// GenerateSessionID creates a session ID.
func (g *IDGenerator) GenerateSessionID() string {
	return g.GenerateID("session")
}

// GenerateDeviceID creates a device ID.
func (g *IDGenerator) GenerateDeviceID() string {
	return g.GenerateID("device")
}

// GenerateCampaignID creates a campaign ID.
func (g *IDGenerator) GenerateCampaignID() string {
	return g.GenerateID("campaign")
}

// GenerateCorrelationID creates a correlation ID.
func (g *IDGenerator) GenerateCorrelationID() string {
	return g.GenerateID("correlation")
}

// ValidateID checks if an ID follows the expected format.
func (g *IDGenerator) ValidateID(id, expectedType string) bool {
	expectedPrefix, exists := g.prefixes[expectedType]
	if !exists {
		return false
	}

	expectedLength, exists := g.lengths[expectedType]
	if !exists {
		return false
	}

	// Check prefix
	if len(id) <= len(expectedPrefix)+1 {
		return false
	}

	if id[:len(expectedPrefix)+1] != expectedPrefix+"_" {
		return false
	}

	// Check length (prefix + underscore + hash)
	expectedTotalLength := len(expectedPrefix) + 1 + expectedLength
	return len(id) == expectedTotalLength
}
