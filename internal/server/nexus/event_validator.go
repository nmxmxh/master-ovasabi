package nexus

import "strings"

// EventTypeValidator provides unified event type validation.
type EventTypeValidator struct{}

// NewEventTypeValidator creates a new event type validator.
func NewEventTypeValidator() *EventTypeValidator {
	return &EventTypeValidator{}
}

// IsValidEventType validates event type format and content.
func (v *EventTypeValidator) IsValidEventType(eventType string) bool {
	// Allow special echo event type for testing
	if eventType == "echo" {
		return true
	}

	// Allow all campaign events to pass through
	if strings.HasPrefix(eventType, "campaign:") {
		return true
	}

	// Check if it's a health event
	if v.IsHealthEventType(eventType) {
		return true
	}

	// Check if it's a canonical event type
	return v.IsCanonicalEventType(eventType)
}

// IsCanonicalEventType validates canonical event type format: {service}:{action}:v{version}:{state}.
func (v *EventTypeValidator) IsCanonicalEventType(eventType string) bool {
	parts := strings.Split(eventType, ":")
	if len(parts) != 4 {
		return false
	}

	// service: non-empty, action: non-empty, version: v[0-9]+, state: controlled vocab
	service, action, version, state := parts[0], parts[1], parts[2], parts[3]
	if service == "" || action == "" {
		return false
	}
	if !strings.HasPrefix(version, "v") || len(version) < 2 {
		return false
	}

	allowedStates := map[string]struct{}{
		"requested": {}, "started": {}, "success": {}, "failed": {}, "completed": {},
	}
	_, ok := allowedStates[state]
	return ok
}

// IsHealthEventType validates health event type format: {service}:health:v{version}:{state}.
func (v *EventTypeValidator) IsHealthEventType(eventType string) bool {
	parts := strings.Split(eventType, ":")
	if len(parts) != 4 {
		return false
	}

	// Format: {service}:health:v{version}:{state}
	service, action, version, state := parts[0], parts[1], parts[2], parts[3]
	if service == "" || action != "health" {
		return false
	}
	if !strings.HasPrefix(version, "v") || len(version) < 2 {
		return false
	}

	// Health events allow additional states beyond the standard canonical states
	healthStates := map[string]struct{}{
		"requested": {}, "success": {}, "failed": {},
		"heartbeat": {}, // Health-specific state for periodic heartbeats
	}
	_, ok := healthStates[state]
	return ok
}

// GetEventTypeCategory returns the category of an event type.
func (v *EventTypeValidator) GetEventTypeCategory(eventType string) string {
	if eventType == "echo" {
		return "test"
	}
	if strings.HasPrefix(eventType, "campaign:") {
		return "campaign"
	}
	if v.IsHealthEventType(eventType) {
		return "health"
	}
	if v.IsCanonicalEventType(eventType) {
		return "canonical"
	}
	return "unknown"
}
