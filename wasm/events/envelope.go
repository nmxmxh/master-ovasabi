package events

import (
	"time"

	"github.com/nmxmxh/master-ovasabi/wasm/shared"
)

// CanonicalEventEnvelope matches the backend canonical format for WASM usage
type CanonicalEventEnvelope struct {
	// Core event fields - matches backend format exactly
	Type          string `json:"type"`
	CorrelationID string `json:"correlation_id"`
	Timestamp     string `json:"timestamp"` // Use string to match frontend format
	Version       string `json:"version"`
	Environment   string `json:"environment"`
	Source        string `json:"source"`

	// Use protobuf-compatible metadata structure
	Metadata *Metadata `json:"metadata"`
	Payload  *Payload  `json:"payload,omitempty"`
}

// Metadata represents the event metadata
type Metadata struct {
	GlobalContext   *GlobalContext `json:"global_context"`
	EnvelopeVersion string         `json:"envelope_version"`
	Environment     string         `json:"environment"`
}

// GetGlobalContext returns the global context
func (m *Metadata) GetGlobalContext() *GlobalContext {
	return m.GlobalContext
}

// GlobalContext represents the global context
type GlobalContext struct {
	UserId        string `json:"user_id"`
	CampaignId    string `json:"campaign_id"`
	CorrelationId string `json:"correlation_id"`
	SessionId     string `json:"session_id"`
	DeviceId      string `json:"device_id"`
	Source        string `json:"source"`
}

// GetUserId returns the user ID
func (gc *GlobalContext) GetUserId() string {
	return gc.UserId
}

// GetCampaignId returns the campaign ID
func (gc *GlobalContext) GetCampaignId() string {
	return gc.CampaignId
}

// Payload represents the event payload
type Payload struct {
	Data map[string]interface{} `json:"data,omitempty"`
}

// Validate ensures the envelope is properly structured
func (e *CanonicalEventEnvelope) Validate() error {
	if e == nil {
		return &ValidationError{Message: "envelope cannot be nil"}
	}
	if e.Type == "" {
		return &ValidationError{Message: "type is required"}
	}
	if e.CorrelationID == "" {
		return &ValidationError{Message: "correlation_id is required"}
	}
	if e.Timestamp == "" {
		return &ValidationError{Message: "timestamp is required"}
	}
	if e.Version == "" {
		return &ValidationError{Message: "version is required"}
	}
	if e.Environment == "" {
		return &ValidationError{Message: "environment is required"}
	}
	if e.Source == "" {
		return &ValidationError{Message: "source is required"}
	}
	if e.Metadata == nil {
		return &ValidationError{Message: "metadata is required"}
	}
	if e.Metadata.GlobalContext == nil {
		return &ValidationError{Message: "global_context is required"}
	}

	// Validate timestamp format
	if _, err := time.Parse(time.RFC3339, e.Timestamp); err != nil {
		return &ValidationError{Message: "invalid timestamp format"}
	}

	return nil
}

// ValidationError represents a validation error
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// NewCanonicalEventEnvelope creates a properly structured event envelope for WASM
func NewCanonicalEventEnvelope(
	eventType string,
	userID, campaignID, correlationID string,
	payload map[string]interface{},
	serviceSpecific map[string]interface{},
) *CanonicalEventEnvelope {
	now := time.Now()

	// Create global context
	globalContext := &GlobalContext{
		UserId:        userID,
		CampaignId:    campaignID,
		CorrelationId: correlationID,
		SessionId:     shared.GenerateSessionID(),
		DeviceId:      shared.GenerateDeviceID(),
		Source:        "wasm",
	}

	// Create metadata
	metadata := &Metadata{
		GlobalContext:   globalContext,
		EnvelopeVersion: "1.0.0",
		Environment:     getEnvironment(),
	}

	// Create payload if provided
	var eventPayload *Payload
	if payload != nil {
		eventPayload = &Payload{
			Data: payload,
		}
	}

	return &CanonicalEventEnvelope{
		Type:          eventType,
		CorrelationID: correlationID,
		Timestamp:     now.Format(time.RFC3339),
		Version:       "1.0.0",
		Environment:   getEnvironment(),
		Source:        "wasm",
		Metadata:      metadata,
		Payload:       eventPayload,
	}
}

// Helper functions for WASM - now using unified ID generation
// These functions are defined in main.go and imported here

func getEnvironment() string {
	// In WASM, we'll default to development
	// This could be set via JS global or environment variable
	return "development"
}
