package events

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ValidationError represents a validation error with context
type ValidationError struct {
	Field   string
	Message string
	Value   interface{}
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error in field '%s': %s (value: %v)", e.Field, e.Message, e.Value)
}

// EventTypePattern defines the canonical event type format
// Format: {service}:{action}:v{version}:{state}
var EventTypePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*:[a-z][a-z0-9_]*:v[0-9]+:[a-z][a-z0-9_]*$`)

// ValidStates defines the allowed event states
var ValidStates = map[string]bool{
	"request":   true,
	"requested": true,
	"started":   true,
	"success":   true,
	"failed":    true,
	"completed": true,
	"cancelled": true,
	"timeout":   true,
}

// ValidateEventType validates the canonical event type format
func ValidateEventType(eventType string) error {
	if eventType == "" {
		return ValidationError{Field: "type", Message: "event type is required", Value: eventType}
	}

	if !EventTypePattern.MatchString(eventType) {
		return ValidationError{
			Field:   "type",
			Message: "event type must follow format: {service}:{action}:v{version}:{state}",
			Value:   eventType,
		}
	}

	// Extract state from event type
	parts := strings.Split(eventType, ":")
	if len(parts) != 4 {
		return ValidationError{
			Field:   "type",
			Message: "event type must have exactly 4 parts separated by colons",
			Value:   eventType,
		}
	}

	state := parts[3]
	if !ValidStates[state] {
		return ValidationError{
			Field:   "type",
			Message: fmt.Sprintf("invalid state '%s', must be one of: %s", state, getValidStatesString()),
			Value:   eventType,
		}
	}

	return nil
}

// ValidateTimestamp validates ISO 8601 timestamp format
func ValidateTimestamp(timestamp string) error {
	if timestamp == "" {
		return ValidationError{Field: "timestamp", Message: "timestamp is required", Value: timestamp}
	}

	_, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return ValidationError{
			Field:   "timestamp",
			Message: "timestamp must be in ISO 8601 format (RFC3339)",
			Value:   timestamp,
		}
	}

	return nil
}

// ValidateCorrelationID validates correlation ID format
func ValidateCorrelationID(correlationID string) error {
	if correlationID == "" {
		return ValidationError{Field: "correlation_id", Message: "correlation ID is required", Value: correlationID}
	}

	// Correlation ID should be alphanumeric with optional hyphens and underscores
	correlationIDPattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !correlationIDPattern.MatchString(correlationID) {
		return ValidationError{
			Field:   "correlation_id",
			Message: "correlation ID must contain only alphanumeric characters, hyphens, and underscores",
			Value:   correlationID,
		}
	}

	// Minimum length check
	if len(correlationID) < 8 {
		return ValidationError{
			Field:   "correlation_id",
			Message: "correlation ID must be at least 8 characters long",
			Value:   correlationID,
		}
	}

	return nil
}

// ValidateUserID validates user ID format
func ValidateUserID(userID string) error {
	if userID == "" {
		return ValidationError{Field: "user_id", Message: "user ID is required", Value: userID}
	}

	// User ID should be alphanumeric with optional hyphens and underscores
	userIDPattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !userIDPattern.MatchString(userID) {
		return ValidationError{
			Field:   "user_id",
			Message: "user ID must contain only alphanumeric characters, hyphens, and underscores",
			Value:   userID,
		}
	}

	return nil
}

// ValidateCampaignID validates campaign ID format
func ValidateCampaignID(campaignID string) error {
	if campaignID == "" {
		return ValidationError{Field: "campaign_id", Message: "campaign ID is required", Value: campaignID}
	}

	// Campaign ID should be alphanumeric with optional hyphens and underscores
	campaignIDPattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !campaignIDPattern.MatchString(campaignID) {
		return ValidationError{
			Field:   "campaign_id",
			Message: "campaign ID must contain only alphanumeric characters, hyphens, and underscores",
			Value:   campaignID,
		}
	}

	return nil
}

// ValidateSource validates the event source
func ValidateSource(source string) error {
	validSources := map[string]bool{
		"frontend": true,
		"backend":  true,
		"wasm":     true,
	}

	if !validSources[source] {
		return ValidationError{
			Field:   "source",
			Message: fmt.Sprintf("source must be one of: %s", getValidSourcesString()),
			Value:   source,
		}
	}

	return nil
}

// ValidateVersion validates the envelope version
func ValidateVersion(version string) error {
	if version == "" {
		return ValidationError{Field: "version", Message: "version is required", Value: version}
	}

	// Version should follow semantic versioning (simplified)
	versionPattern := regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
	if !versionPattern.MatchString(version) {
		return ValidationError{
			Field:   "version",
			Message: "version must follow semantic versioning format (e.g., 1.0.0)",
			Value:   version,
		}
	}

	return nil
}

// ValidateCanonicalEventEnvelope performs comprehensive validation
func ValidateCanonicalEventEnvelope(envelope *CanonicalEventEnvelope) error {
	if envelope == nil {
		return errors.New("envelope cannot be nil")
	}

	// Validate core fields
	if err := ValidateEventType(envelope.Type); err != nil {
		return err
	}

	if err := ValidateCorrelationID(envelope.CorrelationID); err != nil {
		return err
	}

	if err := ValidateTimestamp(envelope.Timestamp); err != nil {
		return err
	}

	if err := ValidateVersion(envelope.Version); err != nil {
		return err
	}

	if err := ValidateSource(envelope.Source); err != nil {
		return err
	}

	// Validate metadata
	if envelope.Metadata == nil {
		return ValidationError{Field: "metadata", Message: "metadata is required", Value: nil}
	}

	// Validate global context
	if envelope.Metadata.GetGlobalContext() == nil {
		return ValidationError{Field: "metadata.global_context", Message: "global context is required", Value: nil}
	}

	global := envelope.Metadata.GetGlobalContext()
	if err := ValidateUserID(global.GetUserId()); err != nil {
		return err
	}

	if err := ValidateCampaignID(global.GetCampaignId()); err != nil {
		return err
	}

	if err := ValidateCorrelationID(global.GetCorrelationId()); err != nil {
		return err
	}

	if err := ValidateSource(global.GetSource()); err != nil {
		return err
	}

	// Validate envelope version in metadata
	if err := ValidateVersion(envelope.Metadata.GetEnvelopeVersion()); err != nil {
		return err
	}

	// Validate environment
	if envelope.Metadata.GetEnvironment() == "" {
		return ValidationError{Field: "metadata.environment", Message: "environment is required", Value: envelope.Metadata.GetEnvironment()}
	}

	// Validate payload if present
	if envelope.Payload != nil {
		if envelope.Payload.Data == nil {
			return ValidationError{Field: "payload.data", Message: "payload data is required when payload is present", Value: envelope.Payload.Data}
		}
	}

	return nil
}

// Helper functions
func getValidStatesString() string {
	states := make([]string, 0, len(ValidStates))
	for state := range ValidStates {
		states = append(states, state)
	}
	return strings.Join(states, ", ")
}

func getValidSourcesString() string {
	return "frontend, backend, wasm"
}

// SanitizeEventType sanitizes an event type to ensure it follows the canonical format
func SanitizeEventType(eventType string) string {
	// Convert to lowercase
	eventType = strings.ToLower(eventType)

	// Replace spaces and special characters with underscores
	eventType = regexp.MustCompile(`[^a-z0-9:_]`).ReplaceAllString(eventType, "_")

	// Remove multiple consecutive underscores
	eventType = regexp.MustCompile(`_+`).ReplaceAllString(eventType, "_")

	// Remove leading/trailing underscores
	eventType = strings.Trim(eventType, "_")

	return eventType
}

// NormalizeEventType normalizes an event type to the canonical format
func NormalizeEventType(eventType string) (string, error) {
	// First sanitize
	normalized := SanitizeEventType(eventType)

	// Validate the normalized type
	if err := ValidateEventType(normalized); err != nil {
		return "", fmt.Errorf("failed to normalize event type '%s': %w", eventType, err)
	}

	return normalized, nil
}
