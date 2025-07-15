package media

import (
	"context"
	"strings"

	mediapb "github.com/nmxmxh/master-ovasabi/api/protos/media/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
)

// CanonicalEventTypeRegistry provides lookup and validation for canonical event types.
var CanonicalEventTypeRegistry map[string]string

// InitCanonicalEventTypeRegistry initializes the canonical event type registry from service_registration.json.
func InitCanonicalEventTypeRegistry() {
	CanonicalEventTypeRegistry = make(map[string]string)
	evts := loadMediaEvents()
	for _, evt := range evts {
		// Example: evt = "media:upload_light_media:v1:completed"; key = "upload_light_media:completed"
		parts := strings.Split(evt, ":")
		if len(parts) >= 4 {
			key := parts[1] + ":" + parts[3] // action:state
			CanonicalEventTypeRegistry[key] = evt
		}
	}
}

// GetCanonicalEventType returns the canonical event type for a given action and state.
func GetCanonicalEventType(action, state string) string {
	if CanonicalEventTypeRegistry == nil {
		InitCanonicalEventTypeRegistry()
	}
	key := action + ":" + state
	if evt, ok := CanonicalEventTypeRegistry[key]; ok {
		return evt
	}
	return ""
}

// Use generic canonical loader for event types
func loadMediaEvents() []string {
	return events.LoadCanonicalEvents("media")
}

// ActionHandlerFunc defines the signature for business logic handlers for each action.
type ActionHandlerFunc func(ctx context.Context, s *ServiceImpl, event *nexusv1.EventResponse)

// actionHandlers maps action names to their business logic handlers.
var actionHandlers = map[string]ActionHandlerFunc{
	"upload_light_media":       handleUploadLightMedia,
	"start_heavy_media_upload": handleStartHeavyMediaUpload,
	"stream_media_chunk":       handleStreamMediaChunk,
	"complete_media_upload":    handleCompleteMediaUpload,
	"get_media":                handleGetMedia,
	"stream_media_content":     handleStreamMediaContent,
	"delete_media":             handleDeleteMedia,
	"list_user_media":          handleListUserMedia,
	"list_system_media":        handleListSystemMedia,
	"broadcast_system_media":   handleBroadcastSystemMedia,
}

// Handler stubs for each media action
func handleStartHeavyMediaUpload(ctx context.Context, svc *ServiceImpl, event *nexusv1.EventResponse) {
	svc.log.Info("Handling start_heavy_media_upload event", zap.Any("event", event))
	var req mediapb.StartHeavyMediaUploadRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal StartHeavyMediaUploadRequest payload", zap.Error(err))
			if svc.handler != nil {
				svc.handler.Error(ctx, "start_heavy_media_upload", 3, "Failed to unmarshal StartHeavyMediaUploadRequest payload", err, nil, req.Name)
			}
			return
		}
	}
	resp, err := svc.StartHeavyMediaUpload(ctx, &req)
	if err != nil {
		svc.log.Error("StartHeavyMediaUpload failed from event", zap.Error(err))
		if svc.handler != nil {
			svc.handler.Error(ctx, "start_heavy_media_upload", 13, "StartHeavyMediaUpload failed from event", err, nil, req.Name)
		}
	} else {
		svc.log.Info("StartHeavyMediaUpload succeeded from event", zap.Any("response", resp))
		if svc.handler != nil {
			svc.handler.Success(ctx, "start_heavy_media_upload", 0, "StartHeavyMediaUpload succeeded from event", resp, nil, req.Name, nil)
		}
	}
}

func handleStreamMediaChunk(ctx context.Context, svc *ServiceImpl, event *nexusv1.EventResponse) {
	svc.log.Info("Handling stream_media_chunk event", zap.Any("event", event))
	var req mediapb.StreamMediaChunkRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal StreamMediaChunkRequest payload", zap.Error(err))
			if svc.handler != nil {
				svc.handler.Error(ctx, "stream_media_chunk", 3, "Failed to unmarshal StreamMediaChunkRequest payload", err, nil, req.UploadId)
			}
			return
		}
	}
	resp, err := svc.StreamMediaChunk(ctx, &req)
	if err != nil {
		svc.log.Error("StreamMediaChunk failed from event", zap.Error(err))
		if svc.handler != nil {
			svc.handler.Error(ctx, "stream_media_chunk", 13, "StreamMediaChunk failed from event", err, nil, req.UploadId)
		}
	} else {
		svc.log.Info("StreamMediaChunk succeeded from event", zap.Any("response", resp))
		if svc.handler != nil {
			svc.handler.Success(ctx, "stream_media_chunk", 0, "StreamMediaChunk succeeded from event", resp, nil, req.UploadId, nil)
		}
	}
}

func handleCompleteMediaUpload(ctx context.Context, svc *ServiceImpl, event *nexusv1.EventResponse) {
	svc.log.Info("Handling complete_media_upload event", zap.Any("event", event))
	var req mediapb.CompleteMediaUploadRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal CompleteMediaUploadRequest payload", zap.Error(err))
			if svc.handler != nil {
				svc.handler.Error(ctx, "complete_media_upload", 3, "Failed to unmarshal CompleteMediaUploadRequest payload", err, nil, req.UploadId)
			}
			return
		}
	}
	resp, err := svc.CompleteMediaUpload(ctx, &req)
	if err != nil {
		svc.log.Error("CompleteMediaUpload failed from event", zap.Error(err))
		if svc.handler != nil {
			svc.handler.Error(ctx, "complete_media_upload", 13, "CompleteMediaUpload failed from event", err, nil, req.UploadId)
		}
	} else {
		svc.log.Info("CompleteMediaUpload succeeded from event", zap.Any("response", resp))
		if svc.handler != nil {
			svc.handler.Success(ctx, "complete_media_upload", 0, "CompleteMediaUpload succeeded from event", resp, nil, req.UploadId, nil)
		}
	}
}

func handleGetMedia(ctx context.Context, svc *ServiceImpl, event *nexusv1.EventResponse) {
	svc.log.Info("Handling get_media event", zap.Any("event", event))
	var req mediapb.GetMediaRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal GetMediaRequest payload", zap.Error(err))
			if svc.handler != nil {
				svc.handler.Error(ctx, "get_media", 3, "Failed to unmarshal GetMediaRequest payload", err, nil, req.Id)
			}
			return
		}
	}
	resp, err := svc.GetMedia(ctx, &req)
	if err != nil {
		svc.log.Error("GetMedia failed from event", zap.Error(err))
		if svc.handler != nil {
			svc.handler.Error(ctx, "get_media", 13, "GetMedia failed from event", err, nil, req.Id)
		}
	} else {
		svc.log.Info("GetMedia succeeded from event", zap.Any("response", resp))
		if svc.handler != nil {
			svc.handler.Success(ctx, "get_media", 0, "GetMedia succeeded from event", resp, nil, req.Id, nil)
		}
	}
}

func handleStreamMediaContent(ctx context.Context, svc *ServiceImpl, event *nexusv1.EventResponse) {
	svc.log.Info("Handling stream_media_content event", zap.Any("event", event))
	var req mediapb.StreamMediaContentRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal StreamMediaContentRequest payload", zap.Error(err))
			if svc.handler != nil {
				svc.handler.Error(ctx, "stream_media_content", 3, "Failed to unmarshal StreamMediaContentRequest payload", err, nil, req.Id)
			}
			return
		}
	}
	resp, err := svc.StreamMediaContent(ctx, &req)
	if err != nil {
		svc.log.Error("StreamMediaContent failed from event", zap.Error(err))
		if svc.handler != nil {
			svc.handler.Error(ctx, "stream_media_content", 13, "StreamMediaContent failed from event", err, nil, req.Id)
		}
	} else {
		svc.log.Info("StreamMediaContent succeeded from event", zap.Any("response", resp))
		if svc.handler != nil {
			svc.handler.Success(ctx, "stream_media_content", 0, "StreamMediaContent succeeded from event", resp, nil, req.Id, nil)
		}
	}
}

func handleDeleteMedia(ctx context.Context, svc *ServiceImpl, event *nexusv1.EventResponse) {
	svc.log.Info("Handling delete_media event", zap.Any("event", event))
	var req mediapb.DeleteMediaRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal DeleteMediaRequest payload", zap.Error(err))
			if svc.handler != nil {
				svc.handler.Error(ctx, "delete_media", 3, "Failed to unmarshal DeleteMediaRequest payload", err, nil, req.Id)
			}
			return
		}
	}
	resp, err := svc.DeleteMedia(ctx, &req)
	if err != nil {
		svc.log.Error("DeleteMedia failed from event", zap.Error(err))
		if svc.handler != nil {
			svc.handler.Error(ctx, "delete_media", 13, "DeleteMedia failed from event", err, nil, req.Id)
		}
	} else {
		svc.log.Info("DeleteMedia succeeded from event", zap.Any("response", resp))
		if svc.handler != nil {
			svc.handler.Success(ctx, "delete_media", 0, "DeleteMedia succeeded from event", resp, nil, req.Id, nil)
		}
	}
}

func handleListUserMedia(ctx context.Context, svc *ServiceImpl, event *nexusv1.EventResponse) {
	svc.log.Info("Handling list_user_media event", zap.Any("event", event))
	var req mediapb.ListUserMediaRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal ListUserMediaRequest payload", zap.Error(err))
			if svc.handler != nil {
				svc.handler.Error(ctx, "list_user_media", 3, "Failed to unmarshal ListUserMediaRequest payload", err, nil, req.UserId)
			}
			return
		}
	}
	resp, err := svc.ListUserMedia(ctx, &req)
	if err != nil {
		svc.log.Error("ListUserMedia failed from event", zap.Error(err))
		if svc.handler != nil {
			svc.handler.Error(ctx, "list_user_media", 13, "ListUserMedia failed from event", err, nil, req.UserId)
		}
	} else {
		svc.log.Info("ListUserMedia succeeded from event", zap.Any("response", resp))
		if svc.handler != nil {
			svc.handler.Success(ctx, "list_user_media", 0, "ListUserMedia succeeded from event", resp, nil, req.UserId, nil)
		}
	}
}

func handleListSystemMedia(ctx context.Context, svc *ServiceImpl, event *nexusv1.EventResponse) {
	svc.log.Info("Handling list_system_media event", zap.Any("event", event))
	var req mediapb.ListSystemMediaRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal ListSystemMediaRequest payload", zap.Error(err))
			if svc.handler != nil {
				svc.handler.Error(ctx, "list_system_media", 3, "Failed to unmarshal ListSystemMediaRequest payload", err, nil, "system")
			}
			return
		}
	}
	resp, err := svc.ListSystemMedia(ctx, &req)
	if err != nil {
		svc.log.Error("ListSystemMedia failed from event", zap.Error(err))
		if svc.handler != nil {
			svc.handler.Error(ctx, "list_system_media", 13, "ListSystemMedia failed from event", err, nil, "system")
		}
	} else {
		svc.log.Info("ListSystemMedia succeeded from event", zap.Any("response", resp))
		if svc.handler != nil {
			svc.handler.Success(ctx, "list_system_media", 0, "ListSystemMedia succeeded from event", resp, nil, "system", nil)
		}
	}
}

func handleBroadcastSystemMedia(ctx context.Context, svc *ServiceImpl, event *nexusv1.EventResponse) {
	svc.log.Info("Handling broadcast_system_media event", zap.Any("event", event))
	var req mediapb.BroadcastSystemMediaRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal BroadcastSystemMediaRequest payload", zap.Error(err))
			if svc.handler != nil {
				svc.handler.Error(ctx, "broadcast_system_media", 3, "Failed to unmarshal BroadcastSystemMediaRequest payload", err, nil, req.UserId)
			}
			return
		}
	}
	resp, err := svc.BroadcastSystemMedia(ctx, &req)
	if err != nil {
		svc.log.Error("BroadcastSystemMedia failed from event", zap.Error(err))
		if svc.handler != nil {
			svc.handler.Error(ctx, "broadcast_system_media", 13, "BroadcastSystemMedia failed from event", err, nil, req.UserId)
		}
	} else {
		svc.log.Info("BroadcastSystemMedia succeeded from event", zap.Any("response", resp))
		if svc.handler != nil {
			svc.handler.Success(ctx, "broadcast_system_media", 0, "BroadcastSystemMedia succeeded from event", resp, nil, req.UserId, nil)
		}
	}
}

// RegisterActionHandler allows registration of business logic handlers for actions.
func RegisterActionHandler(action string, handler ActionHandlerFunc) {
	actionHandlers[action] = handler
}

// parseActionAndState extracts the action and state from a canonical event type.
func parseActionAndState(eventType string) (action, state string) {
	parts := strings.Split(eventType, ":")
	if len(parts) >= 4 {
		return parts[1], parts[3]
	}
	return "", ""
}

// HandleMediaServiceEvent is the generic event handler for all media service actions.
func HandleMediaServiceEvent(ctx context.Context, s *ServiceImpl, event *nexusv1.EventResponse) {
	eventType := event.GetEventType()
	action, _ := parseActionAndState(eventType)
	handler, ok := actionHandlers[action]
	if !ok {
		if s.log != nil {
			s.log.Warn("No handler for action", zap.String("action", action), zap.String("event_type", eventType))
		}
		return
	}
	expectedPrefix := "media:" + action + ":"
	if !strings.HasPrefix(eventType, expectedPrefix) {
		if s.log != nil {
			s.log.Warn("Event type does not match handler action, ignoring", zap.String("event_type", eventType), zap.String("expected_prefix", expectedPrefix))
		}
		return
	}
	handler(ctx, s, event)
}

// Example handler for upload_light_media
func handleUploadLightMedia(ctx context.Context, svc *ServiceImpl, event *nexusv1.EventResponse) {
	svc.log.Info("Handling upload_light_media event", zap.Any("event", event))
	var req mediapb.UploadLightMediaRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal UploadLightMediaRequest payload", zap.Error(err))
			if svc.handler != nil {
				svc.handler.Error(ctx, "upload_light_media", 3, "Failed to unmarshal UploadLightMediaRequest payload", err, nil, req.Name)
			}
			return
		}
	}
	resp, err := svc.UploadLightMedia(ctx, &req)
	if err != nil {
		svc.log.Error("UploadLightMedia failed from event", zap.Error(err))
		if svc.handler != nil {
			svc.handler.Error(ctx, "upload_light_media", 13, "UploadLightMedia failed from event", err, nil, req.Name)
		}
	} else {
		svc.log.Info("UploadLightMedia succeeded from event", zap.Any("response", resp))
		if svc.handler != nil {
			svc.handler.Success(ctx, "upload_light_media", 0, "UploadLightMedia succeeded from event", resp, nil, req.Name, nil)
		}
	}
}

// Register all canonical event types to the generic handler
var eventTypeToHandler = func() map[string]ActionHandlerFunc {
	InitCanonicalEventTypeRegistry()
	m := make(map[string]ActionHandlerFunc)
	for _, evt := range loadMediaEvents() {
		m[evt] = HandleMediaServiceEvent
	}
	return m
}()

// MediaEventRegistry defines all event subscriptions for the media service, using canonical event types.
var MediaEventRegistry = func() []EventSubscription {
	InitCanonicalEventTypeRegistry()
	evts := loadMediaEvents()
	var subs []EventSubscription
	for _, evt := range evts {
		if handler, ok := eventTypeToHandler[evt]; ok {
			subs = append(subs, EventSubscription{
				EventTypes: []string{evt},
				Handler:    handler,
			})
		}
	}
	return subs
}()

// EventSubscription defines a subscription to canonical event types and their handler.
type EventSubscription struct {
	EventTypes []string
	Handler    ActionHandlerFunc
}
