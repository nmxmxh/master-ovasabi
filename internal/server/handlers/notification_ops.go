package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	notificationpb "github.com/nmxmxh/master-ovasabi/api/protos/notification/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
)

// NotificationOpsHandler handles notification-related actions via the "action" field.
//
// @Summary Notification Operations
// @Description Handles notification-related actions using the "action" field in the request body. Each action (e.g., send_notification, list_notifications, mark_as_read, etc.) has its own required/optional fields. All requests must include a 'metadata' field following the robust metadata pattern (see docs/services/metadata.md).
// @Tags notification
// @Accept json
// @Produce json
// @Param request body object true "Composable request with 'action', required fields for the action, and 'metadata' (see docs/services/metadata.md for schema)"
// @Success 200 {object} object "Response depends on action"
// @Failure 400 {object} ErrorResponse
// @Router /api/notification_ops [post]

// NotificationHandler handles notification-related actions (send, list, acknowledge, etc.).
func NotificationHandler(container *di.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Inject logger into context
		log := contextx.Logger(r.Context())
		ctx := contextx.WithLogger(r.Context(), log)
		var notifSvc notificationpb.NotificationServiceServer
		if err := container.Resolve(&notifSvc); err != nil {
			log.Error("Failed to resolve NotificationService", zap.Error(err))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode notification request JSON", zap.Error(err))
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		action, ok := req["action"].(string)
		if !ok || action == "" {
			log.Error("Missing or invalid action in notification request", zap.Any("value", req["action"]))
			http.Error(w, "missing or invalid action", http.StatusBadRequest)
			return
		}
		switch action {
		case "send_notification":
			userID := extractAuthContext(r)
			isGuest := userID == ""
			if !isGuest && userID == "" {
				http.Error(w, "unauthenticated: user_id required", http.StatusUnauthorized)
				return
			}
			if isGuest {
				if req["user_id"] != "guest" && req["user_id"] != "" {
					http.Error(w, "guests cannot send direct user notifications", http.StatusForbidden)
					return
				}
			}
			title, ok := req["title"].(string)
			if !ok || title == "" {
				log.Error("Missing or invalid title in send_notification", zap.Any("value", req["title"]))
				http.Error(w, "missing or invalid title", http.StatusBadRequest)
				return
			}
			body, ok := req["body"].(string)
			if !ok || body == "" {
				log.Error("Missing or invalid body in send_notification", zap.Any("value", req["body"]))
				http.Error(w, "missing or invalid body", http.StatusBadRequest)
				return
			}
			channel, ok := req["channel"].(string)
			if !ok || channel == "" {
				log.Error("Missing or invalid channel in send_notification", zap.Any("value", req["channel"]))
				http.Error(w, "missing or invalid channel", http.StatusBadRequest)
				return
			}
			var meta *commonpb.Metadata
			if m, ok := req["metadata"]; ok && m != nil {
				if metaMap, ok := m.(map[string]interface{}); ok {
					metaStruct, err := structpb.NewStruct(metaMap)
					if err != nil {
						log.Error("Failed to convert metadata to structpb.Struct", zap.Error(err))
						http.Error(w, "invalid metadata", http.StatusBadRequest)
						return
					}
					meta = &commonpb.Metadata{ServiceSpecific: metaStruct}
				}
			}
			var campaignID int64
			if v, ok := req["campaign_id"].(float64); ok {
				campaignID = int64(v)
			}
			protoReq := &notificationpb.SendNotificationRequest{
				UserId:     userID,
				Title:      title,
				Body:       body,
				Channel:    channel,
				Metadata:   meta,
				CampaignId: campaignID,
			}
			resp, err := notifSvc.SendNotification(ctx, protoReq)
			if err != nil {
				log.Error("Failed to send notification", zap.Error(err))
				http.Error(w, "failed to send notification", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (send_notification)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "send_email":
			userID := extractAuthContext(r)
			if userID == "" {
				http.Error(w, "unauthenticated: user_id required for email", http.StatusUnauthorized)
				return
			}
			to, ok := req["to"].(string)
			if !ok || to == "" {
				log.Error("Missing or invalid to in send_email", zap.Any("value", req["to"]))
				http.Error(w, "missing or invalid to", http.StatusBadRequest)
				return
			}
			subject, ok := req["subject"].(string)
			if !ok || subject == "" {
				log.Error("Missing or invalid subject in send_email", zap.Any("value", req["subject"]))
				http.Error(w, "missing or invalid subject", http.StatusBadRequest)
				return
			}
			body, ok := req["body"].(string)
			if !ok || body == "" {
				log.Error("Missing or invalid body in send_email", zap.Any("value", req["body"]))
				http.Error(w, "missing or invalid body", http.StatusBadRequest)
				return
			}
			var meta *commonpb.Metadata
			if m, ok := req["metadata"]; ok && m != nil {
				if metaMap, ok := m.(map[string]interface{}); ok {
					metaStruct, err := structpb.NewStruct(metaMap)
					if err != nil {
						log.Error("Failed to convert metadata to structpb.Struct", zap.Error(err))
						http.Error(w, "invalid metadata", http.StatusBadRequest)
						return
					}
					meta = &commonpb.Metadata{ServiceSpecific: metaStruct}
				}
			}
			var campaignID int64
			if v, ok := req["campaign_id"].(float64); ok {
				campaignID = int64(v)
			}
			protoReq := &notificationpb.SendEmailRequest{
				To:         to,
				Subject:    subject,
				Body:       body,
				Metadata:   meta,
				CampaignId: campaignID,
			}
			resp, err := notifSvc.SendEmail(ctx, protoReq)
			if err != nil {
				log.Error("Failed to send email", zap.Error(err))
				http.Error(w, "failed to send email", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (send_email)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "send_sms":
			userID := extractAuthContext(r)
			if userID == "" {
				http.Error(w, "unauthenticated: user_id required for SMS", http.StatusUnauthorized)
				return
			}
			to, ok := req["to"].(string)
			if !ok || to == "" {
				log.Error("Missing or invalid to in send_sms", zap.Any("value", req["to"]))
				http.Error(w, "missing or invalid to", http.StatusBadRequest)
				return
			}
			message, ok := req["message"].(string)
			if !ok || message == "" {
				log.Error("Missing or invalid message in send_sms", zap.Any("value", req["message"]))
				http.Error(w, "missing or invalid message", http.StatusBadRequest)
				return
			}
			var meta *commonpb.Metadata
			if m, ok := req["metadata"]; ok && m != nil {
				if metaMap, ok := m.(map[string]interface{}); ok {
					metaStruct, err := structpb.NewStruct(metaMap)
					if err != nil {
						log.Error("Failed to convert metadata to structpb.Struct", zap.Error(err))
						http.Error(w, "invalid metadata", http.StatusBadRequest)
						return
					}
					meta = &commonpb.Metadata{ServiceSpecific: metaStruct}
				}
			}
			var campaignID int64
			if v, ok := req["campaign_id"].(float64); ok {
				campaignID = int64(v)
			}
			protoReq := &notificationpb.SendSMSRequest{
				To:         to,
				Message:    message,
				Metadata:   meta,
				CampaignId: campaignID,
			}
			resp, err := notifSvc.SendSMS(ctx, protoReq)
			if err != nil {
				log.Error("Failed to send SMS", zap.Error(err))
				http.Error(w, "failed to send SMS", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (send_sms)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "send_push":
			userID := extractAuthContext(r)
			isGuest := userID == ""
			if !isGuest && userID == "" {
				http.Error(w, "unauthenticated: user_id required", http.StatusUnauthorized)
				return
			}
			if isGuest {
				if req["user_id"] != "guest" && req["user_id"] != "" {
					http.Error(w, "guests cannot send direct user notifications", http.StatusForbidden)
					return
				}
			}
			title, ok := req["title"].(string)
			if !ok || title == "" {
				log.Error("Missing or invalid title in send_push", zap.Any("value", req["title"]))
				http.Error(w, "missing or invalid title", http.StatusBadRequest)
				return
			}
			message, ok := req["message"].(string)
			if !ok || message == "" {
				log.Error("Missing or invalid message in send_push", zap.Any("value", req["message"]))
				http.Error(w, "missing or invalid message", http.StatusBadRequest)
				return
			}
			var meta *commonpb.Metadata
			if m, ok := req["metadata"]; ok && m != nil {
				if metaMap, ok := m.(map[string]interface{}); ok {
					metaStruct, err := structpb.NewStruct(metaMap)
					if err != nil {
						log.Error("Failed to convert metadata to structpb.Struct", zap.Error(err))
						http.Error(w, "invalid metadata", http.StatusBadRequest)
						return
					}
					meta = &commonpb.Metadata{ServiceSpecific: metaStruct}
				}
			}
			var campaignID int64
			if v, ok := req["campaign_id"].(float64); ok {
				campaignID = int64(v)
			}
			protoReq := &notificationpb.SendPushNotificationRequest{
				UserId:     userID,
				Title:      title,
				Message:    message,
				Metadata:   meta,
				CampaignId: campaignID,
			}
			resp, err := notifSvc.SendPushNotification(ctx, protoReq)
			if err != nil {
				log.Error("Failed to send push notification", zap.Error(err))
				http.Error(w, "failed to send push notification", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (send_push)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "list_notifications":
			userID := extractAuthContext(r)
			if userID == "" {
				log.Error("Missing or invalid user_id in list_notifications", zap.Any("value", req["user_id"]))
				http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
				return
			}
			page, ok := req["page"].(float64)
			if !ok {
				log.Error("Missing or invalid page in list_notifications", zap.Any("value", req["page"]))
				http.Error(w, "missing or invalid page", http.StatusBadRequest)
				return
			}
			pageSize, ok := req["page_size"].(float64)
			if !ok {
				log.Error("Missing or invalid page_size in list_notifications", zap.Any("value", req["page_size"]))
				http.Error(w, "missing or invalid page_size", http.StatusBadRequest)
				return
			}
			var campaignID int64
			if v, ok := req["campaign_id"].(float64); ok {
				campaignID = int64(v)
			}
			protoReq := &notificationpb.ListNotificationsRequest{
				UserId:     userID,
				Page:       int32(page),
				PageSize:   int32(pageSize),
				CampaignId: campaignID,
			}
			resp, err := notifSvc.ListNotifications(ctx, protoReq)
			if err != nil {
				log.Error("Failed to list notifications", zap.Error(err))
				http.Error(w, "failed to list notifications", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (list_notifications)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "acknowledge_notification":
			notificationID, ok := req["notification_id"].(string)
			if !ok || notificationID == "" {
				log.Error("Missing or invalid notification_id in acknowledge_notification", zap.Any("value", req["notification_id"]))
				http.Error(w, "missing or invalid notification_id", http.StatusBadRequest)
				return
			}
			userID := extractAuthContext(r)
			protoReq := &notificationpb.AcknowledgeNotificationRequest{
				NotificationId: notificationID,
				UserId:         userID,
			}
			resp, err := notifSvc.AcknowledgeNotification(ctx, protoReq)
			if err != nil {
				log.Error("Failed to acknowledge notification", zap.Error(err))
				http.Error(w, "failed to acknowledge notification", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (acknowledge_notification)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "broadcast_event":
			userID := extractAuthContext(r)
			isGuest := userID == ""
			if !isGuest && userID == "" {
				http.Error(w, "unauthenticated: user_id required", http.StatusUnauthorized)
				return
			}
			if isGuest {
				if req["user_id"] != "guest" && req["user_id"] != "" {
					http.Error(w, "guests cannot send direct user notifications", http.StatusForbidden)
					return
				}
			}
			subject, ok := req["subject"].(string)
			if !ok || subject == "" {
				log.Error("Missing or invalid subject in broadcast_event", zap.Any("value", req["subject"]))
				http.Error(w, "missing or invalid subject", http.StatusBadRequest)
				return
			}
			message, ok := req["message"].(string)
			if !ok || message == "" {
				log.Error("Missing or invalid message in broadcast_event", zap.Any("value", req["message"]))
				http.Error(w, "missing or invalid message", http.StatusBadRequest)
				return
			}
			var payload *commonpb.Metadata
			if m, ok := req["payload"]; ok && m != nil {
				if metaMap, ok := m.(map[string]interface{}); ok {
					metaStruct, err := structpb.NewStruct(metaMap)
					if err != nil {
						log.Error("Failed to convert payload to structpb.Struct", zap.Error(err))
						http.Error(w, "invalid payload", http.StatusBadRequest)
						return
					}
					payload = &commonpb.Metadata{ServiceSpecific: metaStruct}
				}
			}
			var campaignID int64
			if v, ok := req["campaign_id"].(float64); ok {
				campaignID = int64(v)
			}
			protoReq := &notificationpb.BroadcastEventRequest{
				Subject:    subject,
				Message:    message,
				Payload:    payload,
				CampaignId: campaignID,
			}
			resp, err := notifSvc.BroadcastEvent(ctx, protoReq)
			if err != nil {
				log.Error("Failed to broadcast event", zap.Error(err))
				http.Error(w, "failed to broadcast event", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (broadcast_event)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "list_notification_events":
			notificationID, ok := req["notification_id"].(string)
			if !ok || notificationID == "" {
				log.Error("Missing or invalid notification_id in list_notification_events", zap.Any("value", req["notification_id"]))
				http.Error(w, "missing or invalid notification_id", http.StatusBadRequest)
				return
			}
			page, ok := req["page"].(float64)
			if !ok {
				log.Error("Missing or invalid page in list_notification_events", zap.Any("value", req["page"]))
				http.Error(w, "missing or invalid page", http.StatusBadRequest)
				return
			}
			pageSize, ok := req["page_size"].(float64)
			if !ok {
				log.Error("Missing or invalid page_size in list_notification_events", zap.Any("value", req["page_size"]))
				http.Error(w, "missing or invalid page_size", http.StatusBadRequest)
				return
			}
			var campaignID int64
			if v, ok := req["campaign_id"].(float64); ok {
				campaignID = int64(v)
			}
			protoReq := &notificationpb.ListNotificationEventsRequest{
				NotificationId: notificationID,
				Page:           int32(page),
				PageSize:       int32(pageSize),
				CampaignId:     campaignID,
			}
			resp, err := notifSvc.ListNotificationEvents(ctx, protoReq)
			if err != nil {
				log.Error("Failed to list notification events", zap.Error(err))
				http.Error(w, "failed to list notification events", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (list_notification_events)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "subscribe_to_events":
			userID := extractAuthContext(r)
			if userID == "" {
				log.Error("Missing or invalid user_id in subscribe_to_events", zap.Any("value", req["user_id"]))
				http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			flusher, ok := w.(http.Flusher)
			if !ok {
				log.Error("Streaming not supported by response writer")
				http.Error(w, "streaming not supported", http.StatusInternalServerError)
				return
			}
			stream := &sseEventStream{w: w, flusher: flusher, log: log}
			protoReq := &notificationpb.SubscribeToEventsRequest{UserId: userID}
			// Simulate streaming: get a single event from the service and send as SSE
			dummyStream := &singleEventStream{onSend: func(e *notificationpb.NotificationEvent) {
				b, err := json.Marshal(e)
				if err != nil {
					log.Error("Failed to marshal event for SSE", zap.Error(err))
					return
				}
				stream.Send(string(b))
			}}
			if err := notifSvc.SubscribeToEvents(protoReq, dummyStream); err != nil {
				log.Error("Failed to subscribe to events", zap.Error(err))
				http.Error(w, "failed to subscribe to events", http.StatusInternalServerError)
				return
			}
			return
		case "stream_asset_chunks":
			assetID, ok := req["asset_id"].(string)
			if !ok || assetID == "" {
				log.Error("Missing or invalid asset_id in stream_asset_chunks", zap.Any("value", req["asset_id"]))
				http.Error(w, "missing or invalid asset_id", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			flusher, ok := w.(http.Flusher)
			if !ok {
				log.Error("Streaming not supported by response writer")
				http.Error(w, "streaming not supported", http.StatusInternalServerError)
				return
			}
			stream := &sseEventStream{w: w, flusher: flusher, log: log}
			protoReq := &notificationpb.StreamAssetChunksRequest{AssetId: assetID}
			// Simulate streaming: get a single chunk from the service and send as SSE
			dummyStream := &singleChunkStream{onSend: func(c *notificationpb.AssetChunk) {
				b, err := json.Marshal(c)
				if err != nil {
					log.Error("Failed to marshal chunk for SSE", zap.Error(err))
					return
				}
				stream.Send(string(b))
			}}
			if err := notifSvc.StreamAssetChunks(protoReq, dummyStream); err != nil {
				log.Error("Failed to stream asset chunks", zap.Error(err))
				http.Error(w, "failed to stream asset chunks", http.StatusInternalServerError)
				return
			}
			return
		default:
			log.Error("Unknown action in notification handler", zap.String("action", action))
			http.Error(w, "unknown action", http.StatusBadRequest)
			return
		}
	}
}

// --- SSE helpers ---.
type sseEventStream struct {
	w       http.ResponseWriter
	flusher http.Flusher
	log     *zap.Logger
}

func (s *sseEventStream) Send(data string) {
	_, err := s.w.Write([]byte("data: " + data + "\n\n"))
	if err != nil {
		s.log.Error("Failed to write SSE data", zap.Error(err))
	}
	s.flusher.Flush()
}

// Simulated gRPC stream for a single event (for SSE demo).
type singleEventStream struct {
	onSend func(*notificationpb.NotificationEvent)
}

func (s *singleEventStream) Send(e *notificationpb.NotificationEvent) error {
	s.onSend(e)
	return nil
}
func (s *singleEventStream) SetHeader(metadata.MD) error  { return nil }
func (s *singleEventStream) SendHeader(metadata.MD) error { return nil }
func (s *singleEventStream) SetTrailer(metadata.MD)       {}
func (s *singleEventStream) Context() context.Context     { return context.Background() }
func (s *singleEventStream) SendMsg(_ interface{}) error  { return nil }
func (s *singleEventStream) RecvMsg(_ interface{}) error  { return nil }

// Simulated gRPC stream for a single chunk (for SSE demo).
type singleChunkStream struct {
	onSend func(*notificationpb.AssetChunk)
}

func (s *singleChunkStream) Send(c *notificationpb.AssetChunk) error {
	s.onSend(c)
	return nil
}
func (s *singleChunkStream) SetHeader(metadata.MD) error  { return nil }
func (s *singleChunkStream) SendHeader(metadata.MD) error { return nil }
func (s *singleChunkStream) SetTrailer(metadata.MD)       {}
func (s *singleChunkStream) Context() context.Context     { return context.Background() }
func (s *singleChunkStream) SendMsg(_ interface{}) error  { return nil }
func (s *singleChunkStream) RecvMsg(_ interface{}) error  { return nil }

// extractAuthContext extracts user_id, device_id from context or metadata.
func extractAuthContext(r *http.Request) (userID string) {
	ctx := r.Context()
	// Try contextx.Auth first
	authCtx := contextx.Auth(ctx)
	if authCtx != nil && authCtx.UserID != "" {
		userID = authCtx.UserID
	}
	// Fallback: try device_id if needed
	if userID == "" {
		if v := ctx.Value("device_id"); v != nil {
			if s, ok := v.(string); ok {
				userID = s
			}
		}
	}
	return userID
}
