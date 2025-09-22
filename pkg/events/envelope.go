package events

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexuspb "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// EventEnvelope is the canonical, extensible wrapper for all event-driven messages in the system.
type EventEnvelope struct {
	ID        string             `json:"id"`
	Type      string             `json:"type"`
	Payload   *commonpb.Payload  `json:"payload"`
	Metadata  *commonpb.Metadata `json:"metadata"`
	Timestamp int64              `json:"timestamp,omitempty"`
}

// CanonicalEventEnvelope wraps the existing protobuf structures with additional validation and consistency
type CanonicalEventEnvelope struct {
	// Core event fields
	Type          string `json:"type"`
	CorrelationID string `json:"correlation_id"`
	Timestamp     string `json:"timestamp"` // RFC3339 string format to match JSON and validation
	Version       string `json:"version"`
	Environment   string `json:"environment"`
	Source        string `json:"source"`

	// Use existing protobuf structures
	Metadata *commonpb.Metadata `json:"metadata"`
	Payload  *commonpb.Payload  `json:"payload,omitempty"`
}

// NewCanonicalEventEnvelope creates a properly structured event
func NewCanonicalEventEnvelope(
	eventType string,
	userID, campaignID, correlationID string,
	payload *commonpb.Payload,
	serviceSpecific map[string]interface{},
) *CanonicalEventEnvelope {
	// Create global context
	globalContext := &commonpb.Metadata_GlobalContext{
		UserId:        userID,
		CampaignId:    campaignID,
		CorrelationId: correlationID,
		SessionId:     generateSessionID(),
		DeviceId:      generateDeviceID(),
		Source:        "backend",
	}

	// Create service-specific metadata
	serviceSpecificStruct, _ := structpb.NewStruct(serviceSpecific)

	// Create audit info
	auditStruct, _ := structpb.NewStruct(map[string]interface{}{
		"created_at": time.Now().Format(time.RFC3339),
		"created_by": userID,
	})

	// Create metadata
	metadata := &commonpb.Metadata{
		GlobalContext:   globalContext,
		EnvelopeVersion: "1.0.0",
		Environment:     getEnvironment(),
		ServiceSpecific: serviceSpecificStruct,
		Features:        []string{},
		Tags:            []string{},
		Audit:           auditStruct,
	}

	return &CanonicalEventEnvelope{
		Type:          eventType,
		CorrelationID: correlationID,
		Timestamp:     time.Now().Format(time.RFC3339),
		Version:       "1.0.0",
		Environment:   getEnvironment(),
		Source:        "backend",
		Metadata:      metadata,
		Payload:       payload,
	}
}

// ToNexusEvent converts to existing Nexus event format
func (e *CanonicalEventEnvelope) ToNexusEvent() *nexuspb.EventRequest {
	campaignIDInt, _ := strconv.ParseInt(e.Metadata.GetGlobalContext().GetCampaignId(), 10, 64)

	return &nexuspb.EventRequest{
		EventId:    e.Metadata.GetGlobalContext().GetCorrelationId(),
		EventType:  e.Type,
		EntityId:   e.Metadata.GetGlobalContext().GetUserId(),
		CampaignId: campaignIDInt,
		Metadata:   e.Metadata,
		Payload:    e.Payload,
	}
}

// Validate ensures the envelope is properly structured
func (e *CanonicalEventEnvelope) Validate() error {
	return ValidateCanonicalEventEnvelope(e)
}

// UnmarshalJSON handles JSON unmarshaling from WASM module using existing conversion functions
func (e *CanonicalEventEnvelope) UnmarshalJSON(data []byte) error {
	// Parse as plain JSON first
	var temp struct {
		Type          string                 `json:"type"`
		CorrelationID string                 `json:"correlation_id"`
		Timestamp     string                 `json:"timestamp"`
		Version       string                 `json:"version"`
		Environment   string                 `json:"environment"`
		Source        string                 `json:"source"`
		Metadata      map[string]interface{} `json:"metadata"`
		Payload       map[string]interface{} `json:"payload"`
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Set basic fields
	e.Type = temp.Type
	e.CorrelationID = temp.CorrelationID
	e.Version = temp.Version
	e.Environment = temp.Environment
	e.Source = temp.Source

	// Ensure version is set with a default if empty
	if e.Version == "" {
		e.Version = "1.0.0"
	}

	// Set timestamp (already in RFC3339 string format)
	if temp.Timestamp != "" {
		e.Timestamp = temp.Timestamp
	} else {
		e.Timestamp = time.Now().Format(time.RFC3339)
	}

	// Convert metadata to protobuf
	if temp.Metadata != nil {
		// Extract global_context from metadata
		var globalContext *commonpb.Metadata_GlobalContext
		if gcData, ok := temp.Metadata["global_context"].(map[string]interface{}); ok {
			globalContext = &commonpb.Metadata_GlobalContext{
				UserId:        getStringFromMap(gcData, "user_id", ""),
				CampaignId:    getStringFromMap(gcData, "campaign_id", ""),
				CorrelationId: getStringFromMap(gcData, "correlation_id", ""),
				SessionId:     getStringFromMap(gcData, "session_id", ""),
				DeviceId:      getStringFromMap(gcData, "device_id", ""),
				Source:        getStringFromMap(gcData, "source", ""),
			}
		}

		// Create service-specific metadata
		serviceSpecific := make(map[string]interface{})
		for k, v := range temp.Metadata {
			if k != "global_context" && k != "envelope_version" && k != "environment" {
				serviceSpecific[k] = v
			}
		}

		var serviceSpecificStruct *structpb.Struct
		if len(serviceSpecific) > 0 {
			if ss, err := structpb.NewStruct(serviceSpecific); err == nil {
				serviceSpecificStruct = ss
			}
		}

		// Extract envelope_version from metadata
		envelopeVersion := getStringFromMap(temp.Metadata, "envelope_version", "1.0.0")

		e.Metadata = &commonpb.Metadata{
			GlobalContext:   globalContext,
			ServiceSpecific: serviceSpecificStruct,
			Environment:     temp.Environment,
			EnvelopeVersion: envelopeVersion,
		}
	}

	// Convert payload to protobuf
	if temp.Payload != nil {
		if data, ok := temp.Payload["data"]; ok {
			if dataMap, ok := data.(map[string]interface{}); ok && len(dataMap) > 0 {
				if dataStruct, err := structpb.NewStruct(dataMap); err == nil {
					e.Payload = &commonpb.Payload{
						Data: dataStruct,
					}
				}
			}
		}
	}

	return nil
}

// Helper functions
func getStringFromMap(m map[string]interface{}, key, defaultValue string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

func generateSessionID() string {
	return "session_" + strconv.FormatInt(time.Now().UnixNano(), 36)
}

func generateDeviceID() string {
	return "device_" + strconv.FormatInt(time.Now().UnixNano(), 36)
}

func getEnvironment() string {
	env := "development"
	if envVar := os.Getenv("ENVIRONMENT"); envVar != "" {
		env = envVar
	}
	return env
}

// ValidateCanonicalEventEnvelopeComprehensive provides comprehensive validation for canonical event envelopes
func ValidateCanonicalEventEnvelopeComprehensive(envelope *CanonicalEventEnvelope) error {
	if envelope == nil {
		return &ValidationErrorComprehensive{Message: "envelope cannot be nil"}
	}

	// Validate core fields
	if envelope.Type == "" {
		return &ValidationErrorComprehensive{Message: "type is required"}
	}
	if envelope.CorrelationID == "" {
		return &ValidationErrorComprehensive{Message: "correlation_id is required"}
	}
	if envelope.Version == "" {
		return &ValidationErrorComprehensive{Message: "version is required"}
	}
	if envelope.Environment == "" {
		return &ValidationErrorComprehensive{Message: "environment is required"}
	}
	if envelope.Source == "" {
		return &ValidationErrorComprehensive{Message: "source is required"}
	}

	// Validate metadata
	if envelope.Metadata == nil {
		return &ValidationErrorComprehensive{Message: "metadata is required"}
	}
	if envelope.Metadata.GlobalContext == nil {
		return &ValidationErrorComprehensive{Message: "global_context is required"}
	}

	// Validate global context
	gc := envelope.Metadata.GlobalContext
	if gc.UserId == "" {
		return &ValidationErrorComprehensive{Message: "user_id is required"}
	}
	if gc.CampaignId == "" {
		return &ValidationErrorComprehensive{Message: "campaign_id is required"}
	}
	if gc.CorrelationId == "" {
		return &ValidationErrorComprehensive{Message: "correlation_id is required"}
	}
	if gc.SessionId == "" {
		return &ValidationErrorComprehensive{Message: "session_id is required"}
	}
	if gc.DeviceId == "" {
		return &ValidationErrorComprehensive{Message: "device_id is required"}
	}
	if gc.Source == "" {
		return &ValidationErrorComprehensive{Message: "source is required"}
	}

	// Validate canonical event type format
	if !isCanonicalEventType(envelope.Type) {
		return &ValidationErrorComprehensive{Message: "event type must follow canonical format: {service}:{action}:v{version}:{state}"}
	}

	return nil
}

// ValidationErrorComprehensive represents a validation error
type ValidationErrorComprehensive struct {
	Message string
}

func (e *ValidationErrorComprehensive) Error() string {
	return e.Message
}

// isCanonicalEventType validates event type format: {service}:{action}:v{version}:{state}.
func isCanonicalEventType(eventType string) bool {
	// Allow the special echo event type for hello world/testing
	if eventType == "echo" {
		return true
	}
	// Allow all campaign events to pass through
	if strings.HasPrefix(eventType, "campaign:") {
		return true
	}
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
	allowedStates := map[string]struct{}{"requested": {}, "started": {}, "success": {}, "failed": {}, "completed": {}}
	_, ok := allowedStates[state]
	return ok
}
