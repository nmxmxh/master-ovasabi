package shared

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// Unified ID generation system for consistency across WASM, frontend, and backend
func GenerateUnifiedID(prefix string, length int, additionalData ...string) string {
	// Create input for hash generation (matching shared/id_generator.go)
	input := fmt.Sprintf("%s_%d_%s", prefix, time.Now().UnixNano(), prefix)
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

// Specific ID generators with consistent prefixes and lengths
func GenerateUserID() string {
	return GenerateUnifiedID("user", 32)
}

func GenerateGuestID() string {
	return GenerateUnifiedID("guest", 32)
}

func GenerateSessionID() string {
	return GenerateUnifiedID("session", 32)
}

func GenerateDeviceID() string {
	return GenerateUnifiedID("device", 32)
}

func GenerateCampaignID() string {
	return GenerateUnifiedID("campaign", 24)
}

func GenerateCorrelationID() string {
	return GenerateUnifiedID("corr", 24)
}
