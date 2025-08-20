package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	notificationpb "github.com/nmxmxh/master-ovasabi/api/protos/notification/v1"
	"github.com/nmxmxh/master-ovasabi/internal/server/httputil"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
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
			httputil.WriteJSONError(w, log, http.StatusInternalServerError, "internal error", err) // Already correct
			return
		}
		if r.Method != http.MethodPost {
			httputil.WriteJSONError(w, log, http.StatusMethodNotAllowed, "method not allowed", nil) // Already correct
			return
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode notification request JSON", zap.Error(err))
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid JSON", err) // Already correct
			return
		}
		action, ok := req["action"].(string) // req is now reqMap
		if !ok || action == "" {
			log.Error("Missing or invalid action in notification request", zap.Any("value", req["action"]))
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid action", nil) // Already correct
			return
		}
		authCtx := contextx.Auth(ctx)
		userID := authCtx.UserID
		isGuest := userID == ""

		actionHandlers := map[string]func(){
			"send_notification": func() {
				if isGuest {
					if req["user_id"] != "guest" && req["user_id"] != "" {
						httputil.WriteJSONError(w, log, http.StatusForbidden, "guests cannot send direct user notifications", nil)
						return
					}
				}
				handleNotificationAction(ctx, w, log, req, &notificationpb.SendNotificationRequest{}, notifSvc.SendNotification)
			},
			"send_email": func() {
				if isGuest {
					httputil.WriteJSONError(w, log, http.StatusUnauthorized, "unauthenticated: user_id required for email", nil)
					return
				}
				handleNotificationAction(ctx, w, log, req, &notificationpb.SendEmailRequest{}, notifSvc.SendEmail)
			},
			"send_sms": func() {
				if isGuest {
					httputil.WriteJSONError(w, log, http.StatusUnauthorized, "unauthenticated: user_id required for SMS", nil)
					return
				}
				handleNotificationAction(ctx, w, log, req, &notificationpb.SendSMSRequest{}, notifSvc.SendSMS)
			},
			"send_push": func() {
				if isGuest {
					if req["user_id"] != "guest" && req["user_id"] != "" {
						httputil.WriteJSONError(w, log, http.StatusForbidden, "guests cannot send direct user notifications", nil)
						return
					}
				}
				handleNotificationAction(ctx, w, log, req, &notificationpb.SendPushNotificationRequest{}, notifSvc.SendPushNotification)
			},
			"list_notifications": func() {
				if isGuest {
					httputil.WriteJSONError(w, log, http.StatusUnauthorized, "unauthenticated: user_id required", nil)
					return
				}
				handleNotificationAction(ctx, w, log, req, &notificationpb.ListNotificationsRequest{}, notifSvc.ListNotifications)
			},
			"acknowledge_notification": func() {
				if isGuest {
					httputil.WriteJSONError(w, log, http.StatusUnauthorized, "unauthenticated: user_id required", nil)
					return
				}
				handleNotificationAction(ctx, w, log, req, &notificationpb.AcknowledgeNotificationRequest{}, notifSvc.AcknowledgeNotification)
			},
			"broadcast_event": func() {
				// Broadcasts are typically admin/system actions, but we'll allow authenticated users for now.
				if isGuest {
					httputil.WriteJSONError(w, log, http.StatusUnauthorized, "unauthenticated: user_id required", nil)
					return
				}
				handleNotificationAction(ctx, w, log, req, &notificationpb.BroadcastEventRequest{}, notifSvc.BroadcastEvent)
			},
			"list_notification_events": func() {
				if isGuest {
					httputil.WriteJSONError(w, log, http.StatusUnauthorized, "unauthenticated: user_id required", nil)
					return
				}
				handleNotificationAction(ctx, w, log, req, &notificationpb.ListNotificationEventsRequest{}, notifSvc.ListNotificationEvents)
			},
			"subscribe_to_events": func() {
				if isGuest {
					httputil.WriteJSONError(w, log, http.StatusUnauthorized, "unauthenticated: user_id required", nil)
					return
				}
				w.Header().Set("Content-Type", "text/event-stream")
				w.Header().Set("Cache-Control", "no-cache")
				w.Header().Set("Connection", "keep-alive")
				flusher, ok := w.(http.Flusher)
				if !ok {
					httputil.WriteJSONError(w, log, http.StatusInternalServerError, "streaming not supported", nil)
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
					// Don't write another header if one was already written
				}
			},
			"stream_asset_chunks": func() {
				w.Header().Set("Content-Type", "text/event-stream")
				w.Header().Set("Cache-Control", "no-cache")
				w.Header().Set("Connection", "keep-alive")
				flusher, ok := w.(http.Flusher)
				if !ok {
					httputil.WriteJSONError(w, log, http.StatusInternalServerError, "streaming not supported", nil)
					return
				}
				stream := &sseEventStream{w: w, flusher: flusher, log: log}
				var protoReq notificationpb.StreamAssetChunksRequest
				if err := mapToProtoNotification(req, &protoReq); err != nil {
					httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid request body", err)
					return
				}
				// Simulate streaming: get a single chunk from the service and send as SSE
				dummyStream := &singleChunkStream{onSend: func(c *notificationpb.AssetChunk) {
					b, err := json.Marshal(c)
					if err != nil {
						log.Error("Failed to marshal chunk for SSE", zap.Error(err))
						return
					}
					stream.Send(string(b))
				}}
				if err := notifSvc.StreamAssetChunks(&protoReq, dummyStream); err != nil {
					log.Error("Failed to stream asset chunks", zap.Error(err))
					// Don't write another header if one was already written
				}
			},
		}

		if handler, found := actionHandlers[action]; found {
			handler()
		} else {
			log.Error("Unknown action in notification handler", zap.String("action", action))
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "unknown action", nil)
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

// handleNotificationAction is a generic helper to reduce boilerplate in NotificationHandler.
func handleNotificationAction[T proto.Message, U proto.Message](
	ctx context.Context,
	w http.ResponseWriter,
	log *zap.Logger,
	reqMap map[string]interface{},
	req T,
	svcFunc func(context.Context, T) (U, error),
) {
	if err := mapToProtoNotification(reqMap, req); err != nil {
		httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid request body", err)
		return
	}

	resp, err := svcFunc(ctx, req)
	if err != nil {
		st, _ := status.FromError(err)
		httpStatus := httputil.GRPCStatusToHTTPStatus(st.Code())
		if st.Code() == codes.Unauthenticated || st.Code() == codes.PermissionDenied {
			httpStatus = http.StatusForbidden
		}
		log.Error("notification service call failed", zap.Error(err), zap.String("grpc_code", st.Code().String()))
		httputil.WriteJSONError(w, log, httpStatus, st.Message(), nil)
		return
	}

	httputil.WriteJSONResponse(w, log, resp)
}

// mapToProtoNotification converts a map[string]interface{} to a proto.Message using JSON as an intermediate.
func mapToProtoNotification(data map[string]interface{}, v proto.Message) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return protojson.Unmarshal(jsonBytes, v)
}
