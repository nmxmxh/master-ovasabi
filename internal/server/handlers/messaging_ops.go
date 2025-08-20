package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	messagingpb "github.com/nmxmxh/master-ovasabi/api/protos/messaging/v1"
	"github.com/nmxmxh/master-ovasabi/internal/server/httputil"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"go.uber.org/zap"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// MessagingOpsHandler handles messaging-related actions via the "action" field.
//
// @Summary Messaging Operations
// @Description Handles messaging-related actions using the "action" field in the request body. Each action (e.g., send_message, list_messages, update_preferences, etc.) has its own required/optional fields. All requests must include a 'metadata' field following the robust metadata pattern (see docs/services/metadata.md).
// @Tags messaging
// @Accept json
// @Produce json
// @Param request body object true "Composable request with 'action', required fields for the action, and 'metadata' (see docs/services/metadata.md for schema)"
// @Success 200 {object} object "Response depends on action"
// @Failure 400 {object} ErrorResponse
// @Router /api/messaging_ops [post]

// MessagingOpsHandler: Handles messaging-related actions using the composable API pattern.
func MessagingOpsHandler(container *di.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Inject logger into context
		log := contextx.Logger(r.Context())
		ctx := contextx.WithLogger(r.Context(), log)
		var messagingSvc messagingpb.MessagingServiceServer
		if err := container.Resolve(&messagingSvc); err != nil {
			log.Error("Failed to resolve MessagingService", zap.Error(err))
			httputil.WriteJSONError(w, log, http.StatusInternalServerError, "internal server error", err)
			return
		}
		if r.Method != http.MethodPost {
			httputil.WriteJSONError(w, log, http.StatusMethodNotAllowed, "method not allowed", nil)
			return
		}
		var reqMap map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqMap); err != nil {
			log.Error("Failed to decode messaging request JSON", zap.Error(err)) // Already correct
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid JSON", err)
			return
		}
		action, ok := reqMap["action"].(string)
		if !ok || action == "" {
			log.Error("Missing or invalid action in messaging request", zap.Any("value", reqMap["action"])) // Already correct
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid action", nil)
			return
		}
		authCtx := contextx.Auth(ctx)
		userID := authCtx.UserID
		roles := authCtx.Roles
		isGuest := userID == "" || (len(roles) == 1 && roles[0] == "guest")

		actionHandlers := map[string]func(){
			"send_message": func() {
				senderIDVal, senderIDOk := reqMap["sender_id"]
				var senderID string
				if senderIDOk {
					if s, ok := senderIDVal.(string); ok {
						senderID = s
					} else {
						log.Warn("Type assertion to string failed for senderID", zap.Any("senderIDVal", senderIDVal))
					}
				}
				campaignIDFloatVal, campaignIDFloatOk := reqMap["campaign_id"]
				var campaignIDFloat float64
				if campaignIDFloatOk {
					if f, ok := campaignIDFloatVal.(float64); ok {
						campaignIDFloat = f
					} else {
						log.Warn("Type assertion to float64 failed for campaignIDFloat", zap.Any("campaignIDFloatVal", campaignIDFloatVal))
					}
				}
				campaignIDStr := strconv.FormatInt(int64(campaignIDFloat), 10)

				// --- Guest comment logic for campaign-based messaging ---
				if campaignIDStr != "0" && senderID == "" { // Check if it's a campaign message and sender is not set (implies guest)
					guestNickname, ok1 := reqMap["guest_nickname"].(string)
					deviceID, ok2 := reqMap["device_id"].(string)
					if !ok1 || !ok2 || guestNickname == "" || deviceID == "" {
						httputil.WriteJSONError(w, log, http.StatusBadRequest, "guest_nickname and device_id required for guest comment", nil)
						return
					}
					// Mark as guest comment in metadata
					metaVal, metaOk := reqMap["metadata"]
					var meta map[string]interface{}
					if metaOk {
						if m, ok := metaVal.(map[string]interface{}); ok {
							meta = m
						} else {
							log.Warn("Type assertion to map[string]interface{} failed for meta (guest)", zap.Any("metaVal", metaVal))
							meta = make(map[string]interface{})
						}
					} else {
						meta = make(map[string]interface{})
					}
					serviceSpecificVal, serviceSpecificOk := meta["service_specific"]
					var serviceSpecific map[string]interface{}
					if serviceSpecificOk {
						if ss, ok := serviceSpecificVal.(map[string]interface{}); ok {
							serviceSpecific = ss
						} else {
							log.Warn("Type assertion to map[string]interface{} failed for serviceSpecific (guest)", zap.Any("serviceSpecificVal", serviceSpecificVal))
							serviceSpecific = make(map[string]interface{})
						}
					} else {
						serviceSpecific = make(map[string]interface{})
					}
					messagingMetaVal, messagingMetaOk := serviceSpecific["messaging"]
					var messagingMeta map[string]interface{}
					if messagingMetaOk {
						if mm, ok := messagingMetaVal.(map[string]interface{}); ok {
							messagingMeta = mm
						} else {
							log.Warn("Type assertion to map[string]interface{} failed for messagingMeta (guest)", zap.Any("messagingMetaVal", messagingMetaVal))
							messagingMeta = make(map[string]interface{})
						}
					} else {
						messagingMeta = make(map[string]interface{})
					}

					messagingMeta["guest_comment"] = true
					messagingMeta["guest_nickname"] = guestNickname
					messagingMeta["device_id"] = deviceID
					messagingMeta["audit"] = map[string]interface{}{
						"performed_by":   "guest",
						"guest_nickname": guestNickname,
						"device_id":      deviceID,
						"timestamp":      time.Now().UTC().Format(time.RFC3339),
					}
					messagingMeta["roles"] = roles // Roles from authCtx

					serviceSpecific["messaging"] = messagingMeta
					meta["service_specific"] = serviceSpecific
					reqMap["metadata"] = meta

					// Ensure CampaignId is set in the request map for proto unmarshaling
					reqMap["campaign_id"] = campaignIDStr

				} else {
					// --- Authenticated user or admin required for non-guest or non-campaign messages ---
					if isGuest || (senderID != "" && senderID != userID && !httputil.IsAdmin(roles)) {
						httputil.WriteJSONError(w, log, http.StatusForbidden, "forbidden: must be authenticated and own the message (or admin)", nil)
						return
					}
					// Add audit metadata
					metaVal, metaOk := reqMap["metadata"]
					var meta map[string]interface{}
					if metaOk {
						if m, ok := metaVal.(map[string]interface{}); ok {
							meta = m
						} else {
							log.Warn("Type assertion to map[string]interface{} failed for meta (auth)", zap.Any("metaVal", metaVal))
							meta = make(map[string]interface{})
						}
					} else {
						meta = make(map[string]interface{})
					}
					serviceSpecificVal, serviceSpecificOk := meta["service_specific"]
					var serviceSpecific map[string]interface{}
					if serviceSpecificOk {
						if ss, ok := serviceSpecificVal.(map[string]interface{}); ok {
							serviceSpecific = ss
						} else {
							log.Warn("Type assertion to map[string]interface{} failed for serviceSpecific (auth)", zap.Any("serviceSpecificVal", serviceSpecificVal))
							serviceSpecific = make(map[string]interface{})
						}
					} else {
						serviceSpecific = make(map[string]interface{})
					}
					messagingMetaVal, messagingMetaOk := serviceSpecific["messaging"]
					var messagingMeta map[string]interface{}
					if messagingMetaOk {
						if mm, ok := messagingMetaVal.(map[string]interface{}); ok {
							messagingMeta = mm
						} else {
							log.Warn("Type assertion to map[string]interface{} failed for messagingMeta (auth)", zap.Any("messagingMetaVal", messagingMetaVal))
							messagingMeta = make(map[string]interface{})
						}
					} else {
						messagingMeta = make(map[string]interface{})
					}

					messagingMeta["audit"] = map[string]interface{}{
						"performed_by": userID,
						"roles":        roles,
						"timestamp":    time.Now().UTC().Format(time.RFC3339),
					}
					serviceSpecific["messaging"] = messagingMeta
					meta["service_specific"] = serviceSpecific
					reqMap["metadata"] = meta
				}
				handleMessagingAction(ctx, w, log, reqMap, &messagingpb.SendMessageRequest{}, messagingSvc.SendMessage)
			},
			"list_messages": func() {
				handleMessagingAction(ctx, w, log, reqMap, &messagingpb.ListMessagesRequest{}, messagingSvc.ListMessages)
			},
			"update_preferences": func() {
				requestUserIDVal, requestUserIDOk := reqMap["user_id"]
				var requestUserID string
				if requestUserIDOk {
					if s, ok := requestUserIDVal.(string); ok {
						requestUserID = s
					} else {
						log.Warn("Type assertion to string failed for requestUserID in update_preferences", zap.Any("requestUserIDVal", requestUserIDVal))
					}
				}
				if isGuest || (requestUserID != userID && !httputil.IsAdmin(roles)) {
					httputil.WriteJSONError(w, log, http.StatusForbidden, "forbidden: can only update your own preferences (or admin)", nil)
					return
				}
				handleMessagingAction(ctx, w, log, reqMap, &messagingpb.UpdateMessagingPreferencesRequest{}, messagingSvc.UpdateMessagingPreferences)
			},
			"list_threads": func() {
				requestUserIDVal, requestUserIDOk := reqMap["user_id"]
				var requestUserID string
				if requestUserIDOk {
					if s, ok := requestUserIDVal.(string); ok {
						requestUserID = s
					} else {
						log.Warn("Type assertion to string failed for requestUserID in list_threads", zap.Any("requestUserIDVal", requestUserIDVal))
					}
				}
				if isGuest || (requestUserID != userID && !httputil.IsAdmin(roles)) {
					httputil.WriteJSONError(w, log, http.StatusForbidden, "forbidden: can only list your own threads (or admin)", nil)
					return
				}
				handleMessagingAction(ctx, w, log, reqMap, &messagingpb.ListThreadsRequest{}, messagingSvc.ListThreads)
			},
			"add_chat_group_member": func() {
				if !httputil.IsAdmin(roles) {
					httputil.WriteJSONError(w, log, http.StatusForbidden, "forbidden: admin role required", nil)
					return
				}
				handleMessagingAction(ctx, w, log, reqMap, &messagingpb.AddChatGroupMemberRequest{}, messagingSvc.AddChatGroupMember)
			},
			"remove_chat_group_member": func() {
				if !httputil.IsAdmin(roles) {
					httputil.WriteJSONError(w, log, http.StatusForbidden, "forbidden: admin role required", nil)
					return
				}
				handleMessagingAction(ctx, w, log, reqMap, &messagingpb.RemoveChatGroupMemberRequest{}, messagingSvc.RemoveChatGroupMember)
			},
			"list_chat_group_members": func() {
				handleMessagingAction(ctx, w, log, reqMap, &messagingpb.ListChatGroupMembersRequest{}, messagingSvc.ListChatGroupMembers)
			},
		}

		if handler, found := actionHandlers[action]; found {
			handler()
		} else {
			log.Error("Unknown action in messaging_ops", zap.String("action", action))
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "unknown action", nil)
		}
	}
}

// handleMessagingAction is a generic helper to reduce boilerplate in MessagingOpsHandler.
func handleMessagingAction[T proto.Message, U proto.Message](
	ctx context.Context,
	w http.ResponseWriter,
	log *zap.Logger,
	reqMap map[string]interface{},
	req T,
	svcFunc func(context.Context, T) (U, error),
) {
	if err := mapToProtoMessaging(reqMap, req); err != nil {
		httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid request body", err)
		return
	}

	resp, err := svcFunc(ctx, req)
	if err != nil {
		st, _ := status.FromError(err)
		httpStatus := httputil.GRPCStatusToHTTPStatus(st.Code())
		log.Error("messaging service call failed", zap.Error(err), zap.String("grpc_code", st.Code().String()))
		httputil.WriteJSONError(w, log, httpStatus, st.Message(), nil)
		return
	}

	httputil.WriteJSONResponse(w, log, resp)
}

// mapToProtoMessaging converts a map[string]interface{} to a proto.Message using JSON as an intermediate.
func mapToProtoMessaging(data map[string]interface{}, v proto.Message) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return protojson.Unmarshal(jsonBytes, v)
}
