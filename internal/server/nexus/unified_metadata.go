package nexus

import (
	"context"
	"fmt"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
)

// UnifiedMetadataExtractor provides consistent metadata extraction across all Nexus components.
type UnifiedMetadataExtractor struct {
	log *zap.Logger
}

// NewUnifiedMetadataExtractor creates a new unified metadata extractor.
func NewUnifiedMetadataExtractor(log *zap.Logger) *UnifiedMetadataExtractor {
	return &UnifiedMetadataExtractor{log: log}
}

// ExtractedIDs contains all extracted identifiers from metadata.
type ExtractedIDs struct {
	UserID     string
	CampaignID string
	SessionID  string
	DeviceID   string
	TraceID    string
	Source     string
}

// ExtractFromEventRequest extracts all IDs from an EventRequest.
func (e *UnifiedMetadataExtractor) ExtractFromEventRequest(ctx context.Context, event *nexusv1.EventRequest) *ExtractedIDs {
	ids := &ExtractedIDs{}

	if event.Metadata == nil {
		e.log.Debug("[UnifiedMetadata] No metadata available")
		return ids
	}

	// Try new format first (global_context direct field)
	if globalContext := event.Metadata.GetGlobalContext(); globalContext != nil {
		ids.UserID = globalContext.GetUserId()
		ids.CampaignID = globalContext.GetCampaignId()
		ids.SessionID = globalContext.GetSessionId()
		ids.DeviceID = globalContext.GetDeviceId()
		ids.Source = globalContext.GetSource()
		e.log.Debug("[UnifiedMetadata] Extracted from global_context",
			zap.String("user_id", ids.UserID),
			zap.String("campaign_id", ids.CampaignID))
		return ids
	}

	// Fallback to old format (service_specific.global_context)
	if event.Metadata.ServiceSpecific != nil && event.Metadata.ServiceSpecific.Fields != nil {
		if globalContext, ok := event.Metadata.ServiceSpecific.Fields["global_context"]; ok && globalContext != nil {
			if globalContextStruct := globalContext.GetStructValue(); globalContextStruct != nil {
				globalContextMap := globalContextStruct.AsMap()
				ids.UserID = e.getStringFromMap(globalContextMap, "user_id")
				ids.CampaignID = e.getStringFromMap(globalContextMap, "campaign_id")
				ids.SessionID = e.getStringFromMap(globalContextMap, "session_id")
				ids.DeviceID = e.getStringFromMap(globalContextMap, "device_id")
				ids.Source = e.getStringFromMap(globalContextMap, "source")
				e.log.Debug("[UnifiedMetadata] Extracted from service_specific.global_context",
					zap.String("user_id", ids.UserID),
					zap.String("campaign_id", ids.CampaignID))
				return ids
			}
		}
	}

	// Fallback to legacy global format
	if event.Metadata.ServiceSpecific != nil && event.Metadata.ServiceSpecific.Fields != nil {
		if global, ok := event.Metadata.ServiceSpecific.Fields["global"]; ok && global != nil {
			if globalStruct := global.GetStructValue(); globalStruct != nil {
				globalMap := globalStruct.AsMap()
				ids.UserID = e.getStringFromMap(globalMap, "user_id")
				ids.CampaignID = e.getStringFromMap(globalMap, "campaign_id")
				ids.SessionID = e.getStringFromMap(globalMap, "session_id")
				ids.DeviceID = e.getStringFromMap(globalMap, "device_id")
				ids.Source = e.getStringFromMap(globalMap, "source")
				e.log.Debug("[UnifiedMetadata] Extracted from legacy global format",
					zap.String("user_id", ids.UserID),
					zap.String("campaign_id", ids.CampaignID))
				return ids
			}
		}
	}

	// Set defaults
	if ids.CampaignID == "" {
		ids.CampaignID = "0" // Default campaign
	}
	if ids.Source == "" {
		ids.Source = "unknown"
	}

	return ids
}

// ExtractFromEventResponse extracts all IDs from an EventResponse.
func (e *UnifiedMetadataExtractor) ExtractFromEventResponse(event *nexusv1.EventResponse) *ExtractedIDs {
	ids := &ExtractedIDs{}

	if event.Metadata == nil {
		return ids
	}

	// Same extraction logic as EventRequest
	if globalContext := event.Metadata.GetGlobalContext(); globalContext != nil {
		ids.UserID = globalContext.GetUserId()
		ids.CampaignID = globalContext.GetCampaignId()
		ids.SessionID = globalContext.GetSessionId()
		ids.DeviceID = globalContext.GetDeviceId()
		ids.Source = globalContext.GetSource()
		return ids
	}

	// Fallback patterns...
	// (Same as ExtractFromEventRequest but for EventResponse)

	return ids
}

// CreateCanonicalMetadata creates standardized metadata for events.
func (e *UnifiedMetadataExtractor) CreateCanonicalMetadata(ids *ExtractedIDs, additionalData map[string]interface{}) *commonpb.Metadata {
	// Create global context
	globalContext := &commonpb.Metadata_GlobalContext{
		UserId:     ids.UserID,
		CampaignId: ids.CampaignID,
		SessionId:  ids.SessionID,
		DeviceId:   ids.DeviceID,
		Source:     ids.Source,
	}

	// Create service-specific data
	serviceSpecific := make(map[string]*structpb.Value)

	// Add campaign-specific data
	if ids.CampaignID != "" {
		serviceSpecific["campaign"] = &structpb.Value{
			Kind: &structpb.Value_StructValue{
				StructValue: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"campaign_id": {Kind: &structpb.Value_StringValue{StringValue: ids.CampaignID}},
						"timestamp":   {Kind: &structpb.Value_StringValue{StringValue: time.Now().UTC().Format(time.RFC3339)}},
					},
				},
			},
		}
	}

	// Add additional data
	for key, value := range additionalData {
		serviceSpecific[key] = e.interfaceToValue(value)
	}

	return &commonpb.Metadata{
		GlobalContext: globalContext,
		ServiceSpecific: &structpb.Struct{
			Fields: serviceSpecific,
		},
		EnvelopeVersion: "1.0.0",
		Environment:     "production",
	}
}

// Helper method to safely extract string from map.
func (e *UnifiedMetadataExtractor) getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// Helper method to convert interface{} to structpb.Value.
func (e *UnifiedMetadataExtractor) interfaceToValue(v interface{}) *structpb.Value {
	switch val := v.(type) {
	case string:
		return &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: val}}
	case int:
		return &structpb.Value{Kind: &structpb.Value_NumberValue{NumberValue: float64(val)}}
	case float64:
		return &structpb.Value{Kind: &structpb.Value_NumberValue{NumberValue: val}}
	case bool:
		return &structpb.Value{Kind: &structpb.Value_BoolValue{BoolValue: val}}
	default:
		return &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: fmt.Sprintf("%v", val)}}
	}
}
