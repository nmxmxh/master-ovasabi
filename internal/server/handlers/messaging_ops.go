package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	messagingpb "github.com/nmxmxh/master-ovasabi/api/protos/messaging/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"
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
			log.Error("MessagingServiceServer not found in container", zap.Error(err))
			errResp := graceful.WrapErr(ctx, codes.Internal, "internal server error", err)
			errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode messaging request JSON", zap.Error(err))
			errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "invalid JSON", err)
			errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
			return
		}
		action, ok := req["action"].(string)
		if !ok || action == "" {
			log.Error("Missing or invalid action in messaging request", zap.Any("value", req["action"]))
			errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "missing or invalid action", nil)
			errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
			return
		}
		authCtx := contextx.Auth(ctx)
		userID := authCtx.UserID
		roles := authCtx.Roles
		isGuest := userID == "" || (len(roles) == 1 && roles[0] == "guest")
		isAdmin := false
		for _, r := range roles {
			if r == "admin" {
				isAdmin = true
			}
		}
		switch action {
		case "send_message":
			campaignID := int64(0)
			if v, ok := req["campaign_id"].(float64); ok {
				campaignID = int64(v)
			}
			senderID, ok := req["sender_id"].(string)
			if !ok {
				// handle type assertion failure
				return
			}
			guestNickname, ok := req["guest_nickname"].(string)
			if !ok {
				// handle type assertion failure
				errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "guest_nickname and device_id required for guest comment", nil)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
			deviceID, ok := req["device_id"].(string)
			if !ok {
				// handle type assertion failure
				return
			}
			// --- Guest comment logic for campaign-based messaging ---
			if campaignID != 0 && senderID == "" {
				if guestNickname == "" || deviceID == "" {
					errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "guest_nickname and device_id required for guest comment", nil)
					errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
					return
				}
				// Mark as guest comment in metadata
				if m, ok := req["metadata"]; ok && m != nil {
					if metaMap, ok := m.(map[string]interface{}); ok {
						metaMap["guest_comment"] = true
						metaMap["guest_nickname"] = guestNickname
						metaMap["device_id"] = deviceID
						metaMap["audit"] = map[string]interface{}{
							"performed_by":   "guest",
							"guest_nickname": guestNickname,
							"device_id":      deviceID,
							"timestamp":      time.Now().UTC().Format(time.RFC3339),
						}
						// Convert roles []string to []interface{} for structpb compatibility
						rolesIface := make([]interface{}, len(roles))
						for i, r := range roles {
							rolesIface[i] = r
						}
						metaMap["roles"] = rolesIface
						req["metadata"] = metaMap
					}
				} else {
					req["metadata"] = map[string]interface{}{
						"guest_comment":  true,
						"guest_nickname": guestNickname,
						"device_id":      deviceID,
						"audit": map[string]interface{}{
							"performed_by":   "guest",
							"guest_nickname": guestNickname,
							"device_id":      deviceID,
							"timestamp":      time.Now().UTC().Format(time.RFC3339),
						},
						// Convert roles []string to []interface{} for structpb compatibility
						"roles": make([]interface{}, len(roles)),
					}
					// Convert roles []string to []interface{} for structpb compatibility
					var rolesIface []interface{}
					rolesIfaceRaw, ok := req["roles"]
					if !ok {
						// handle missing roles, e.g., log or set to empty
						rolesIface = []interface{}{}
					} else {
						rolesIface, ok = rolesIfaceRaw.([]interface{})
						if !ok {
							// handle wrong type, e.g., log or set to empty
							rolesIface = []interface{}{}
						}
					}
					for i, r := range roles {
						rolesIface[i] = r
					}
				}
				// Allow guest comment to proceed
			} else {
				// --- Authenticated user or admin required for non-guest or non-campaign messages ---
				if isGuest || (senderID != "" && senderID != userID && !isAdmin) {
					errResp := graceful.WrapErr(ctx, codes.PermissionDenied, "forbidden: must be authenticated and own the message (or admin)", nil)
					errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
					return
				}
				// Add audit metadata
				if m, ok := req["metadata"]; ok && m != nil {
					if metaMap, ok := m.(map[string]interface{}); ok {
						// Convert roles []string to []interface{} for structpb compatibility
						rolesIface := make([]interface{}, len(roles))
						for i, r := range roles {
							rolesIface[i] = r
						}
						metaMap["audit"] = map[string]interface{}{
							"performed_by": userID,
							"roles":        rolesIface,
							"timestamp":    time.Now().UTC().Format(time.RFC3339),
						}
						req["metadata"] = metaMap
					}
				} else {
					req["metadata"] = map[string]interface{}{
						"audit": map[string]interface{}{
							"performed_by": userID,
							"roles":        make([]interface{}, len(roles)),
							"timestamp":    time.Now().UTC().Format(time.RFC3339),
						},
					}
					// Convert roles []string to []interface{} for structpb compatibility
					var rolesIface []interface{}
					rolesIfaceRaw, ok := req["roles"]
					if !ok {
						// handle missing roles, e.g., log or set to empty
						rolesIface = []interface{}{}
					} else {
						rolesIface, ok = rolesIfaceRaw.([]interface{})
						if !ok {
							// handle wrong type, e.g., log or set to empty
							rolesIface = []interface{}{}
						}
					}
					for i, r := range roles {
						rolesIface[i] = r
					}
				}
			}
			threadID, ok := req["thread_id"].(string)
			if !ok && req["thread_id"] != nil {
				log.Error("Invalid thread_id in send_message", zap.Any("value", req["thread_id"]))
				errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "invalid thread_id", nil)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
			conversationID, ok := req["conversation_id"].(string)
			if !ok && req["conversation_id"] != nil {
				log.Error("Invalid conversation_id in send_message", zap.Any("value", req["conversation_id"]))
				errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "invalid conversation_id", nil)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
			chatGroupID, ok := req["chat_group_id"].(string)
			if !ok && req["chat_group_id"] != nil {
				log.Error("Invalid chat_group_id in send_message", zap.Any("value", req["chat_group_id"]))
				errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "invalid chat_group_id", nil)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
			content, ok := req["content"].(string)
			if !ok || content == "" {
				log.Error("Missing or invalid content in send_message", zap.Any("value", req["content"]))
				errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "missing or invalid content", nil)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
			recipientIDs := []string{}
			if arr, ok := req["recipient_ids"].([]interface{}); ok {
				for _, v := range arr {
					if s, ok := v.(string); ok {
						recipientIDs = append(recipientIDs, s)
					}
				}
			}
			typeVal, ok := req["type"].(string)
			msgType := messagingpb.MessageType_MESSAGE_TYPE_UNSPECIFIED
			if ok && typeVal != "" {
				if t, ok := messagingpb.MessageType_value[typeVal]; ok {
					msgType = messagingpb.MessageType(t)
				}
			}
			if t, ok := messagingpb.MessageType_value[typeVal]; ok {
				msgType = messagingpb.MessageType(t)
			}
			var meta *commonpb.Metadata
			if m, ok := req["metadata"].(map[string]interface{}); ok {
				if metaStruct, err := structpb.NewStruct(m); err == nil {
					meta = &commonpb.Metadata{ServiceSpecific: metaStruct}
				}
			}
			protoReq := &messagingpb.SendMessageRequest{
				ThreadId:       threadID,
				ConversationId: conversationID,
				ChatGroupId:    chatGroupID,
				SenderId:       senderID,
				RecipientIds:   recipientIDs,
				Content:        content,
				Type:           msgType,
				Metadata:       meta,
				CampaignId:     campaignID,
			}
			resp, err := messagingSvc.SendMessage(ctx, protoReq)
			if err != nil {
				log.Error("Failed to send message", zap.Error(err))
				errResp := graceful.WrapErr(ctx, codes.Internal, "failed to send message", err)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
			success := graceful.WrapSuccess(ctx, codes.OK, "message sent", resp, nil)
			success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: log})
		case "list_messages":
			threadID, ok := req["thread_id"].(string)
			if !ok && req["thread_id"] != nil {
				log.Error("Invalid thread_id in list_messages", zap.Any("value", req["thread_id"]))
				errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "invalid thread_id", nil)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
			conversationID, ok := req["conversation_id"].(string)
			if !ok && req["conversation_id"] != nil {
				log.Error("Invalid conversation_id in list_messages", zap.Any("value", req["conversation_id"]))
				errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "invalid conversation_id", nil)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
			chatGroupID, ok := req["chat_group_id"].(string)
			if !ok && req["chat_group_id"] != nil {
				log.Error("Invalid chat_group_id in list_messages", zap.Any("value", req["chat_group_id"]))
				errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "invalid chat_group_id", nil)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
			page := int32(1)
			if v, ok := req["page"].(float64); ok {
				page = int32(v)
			}
			pageSize := int32(20)
			if v, ok := req["page_size"].(float64); ok {
				pageSize = int32(v)
			}
			var meta *commonpb.Metadata
			if m, ok := req["metadata"].(map[string]interface{}); ok {
				if metaStruct, err := structpb.NewStruct(m); err == nil {
					meta = &commonpb.Metadata{ServiceSpecific: metaStruct}
				}
			}
			var campaignID int64
			if v, ok := req["campaign_id"].(float64); ok {
				campaignID = int64(v)
			}
			protoReq := &messagingpb.ListMessagesRequest{
				ThreadId:       threadID,
				ConversationId: conversationID,
				ChatGroupId:    chatGroupID,
				Page:           page,
				PageSize:       pageSize,
				Metadata:       meta,
				CampaignId:     campaignID,
			}
			resp, err := messagingSvc.ListMessages(ctx, protoReq)
			if err != nil {
				log.Error("Failed to list messages", zap.Error(err))
				errResp := graceful.WrapErr(ctx, codes.Internal, "failed to list messages", err)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
			success := graceful.WrapSuccess(ctx, codes.OK, "messages listed", resp, nil)
			success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: log})
		case "update_preferences":
			userID, ok := req["user_id"].(string)
			if !ok || userID == "" {
				log.Error("Missing or invalid user_id in update_preferences", zap.Any("value", req["user_id"]))
				errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "missing or invalid user_id", nil)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
			prefsMap, ok := req["preferences"].(map[string]interface{})
			if !ok {
				log.Error("Missing or invalid preferences in update_preferences", zap.Any("value", req["preferences"]))
				errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "missing or invalid preferences", nil)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
			prefs := &messagingpb.MessagingPreferences{}
			if v, ok := prefsMap["mute"].(bool); ok {
				prefs.Mute = v
			}
			if v, ok := prefsMap["archive"].(bool); ok {
				prefs.Archive = v
			}
			if m, ok := prefsMap["notification_types"].(map[string]interface{}); ok {
				prefs.NotificationTypes = map[string]bool{}
				for k, v := range m {
					if b, ok := v.(bool); ok {
						prefs.NotificationTypes[k] = b
					}
				}
			}
			if arr, ok := prefsMap["quiet_hours"].([]interface{}); ok {
				for _, v := range arr {
					if s, ok := v.(string); ok {
						prefs.QuietHours = append(prefs.QuietHours, s)
					}
				}
			}
			if v, ok := prefsMap["timezone"].(string); ok {
				prefs.Timezone = v
			}
			if m, ok := prefsMap["metadata"].(map[string]interface{}); ok {
				if metaStruct, err := structpb.NewStruct(m); err == nil {
					prefs.Metadata = &commonpb.Metadata{ServiceSpecific: metaStruct}
				}
			}
			protoReq := &messagingpb.UpdateMessagingPreferencesRequest{
				UserId:      userID,
				Preferences: prefs,
			}
			resp, err := messagingSvc.UpdateMessagingPreferences(ctx, protoReq)
			if err != nil {
				log.Error("Failed to update messaging preferences", zap.Error(err))
				errResp := graceful.WrapErr(ctx, codes.Internal, "failed to update preferences", err)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
			success := graceful.WrapSuccess(ctx, codes.OK, "preferences updated", resp, nil)
			success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: log})
		case "list_threads":
			userID, ok := req["user_id"].(string)
			if !ok || userID == "" {
				log.Error("Missing or invalid user_id in list_threads", zap.Any("value", req["user_id"]))
				errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "missing or invalid user_id", nil)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
			page := int32(1)
			if v, ok := req["page"].(float64); ok {
				page = int32(v)
			}
			pageSize := int32(20)
			if v, ok := req["page_size"].(float64); ok {
				pageSize = int32(v)
			}
			var campaignID int64
			if v, ok := req["campaign_id"].(float64); ok {
				campaignID = int64(v)
			}
			protoReq := &messagingpb.ListThreadsRequest{
				UserId:     userID,
				Page:       page,
				PageSize:   pageSize,
				CampaignId: campaignID,
			}
			resp, err := messagingSvc.ListThreads(ctx, protoReq)
			if err != nil {
				log.Error("Failed to list threads", zap.Error(err))
				errResp := graceful.WrapErr(ctx, codes.Internal, "failed to list threads", err)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
			success := graceful.WrapSuccess(ctx, codes.OK, "threads listed", resp, nil)
			success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: log})
		case "add_chat_group_member":
			chatGroupID, ok := req["chat_group_id"].(string)
			if !ok || chatGroupID == "" {
				log.Error("Missing or invalid chat_group_id in add_chat_group_member", zap.Any("value", req["chat_group_id"]))
				errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "missing or invalid chat_group_id", nil)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
			userID, ok := req["user_id"].(string)
			if !ok || userID == "" {
				log.Error("Missing or invalid user_id in add_chat_group_member", zap.Any("value", req["user_id"]))
				errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "missing or invalid user_id", nil)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
			roleVal := req["role"]
			role, okStr := roleVal.(string)
			if !okStr {
				log.Error("Invalid or missing role in request", zap.Any("role", roleVal))
				errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "invalid or missing role", nil)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
			var campaignID int64
			if v, ok := req["campaign_id"].(float64); ok {
				campaignID = int64(v)
			}
			protoReq := &messagingpb.AddChatGroupMemberRequest{
				ChatGroupId: chatGroupID,
				UserId:      userID,
				Role:        role,
				CampaignId:  campaignID,
			}
			resp, err := messagingSvc.AddChatGroupMember(ctx, protoReq)
			if err != nil {
				log.Error("Failed to add chat group member", zap.Error(err))
				errResp := graceful.WrapErr(ctx, codes.Internal, "failed to add chat group member", err)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
			success := graceful.WrapSuccess(ctx, codes.OK, "chat group member added", resp, nil)
			success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: log})
		case "remove_chat_group_member":
			chatGroupID, ok := req["chat_group_id"].(string)
			if !ok || chatGroupID == "" {
				log.Error("Missing or invalid chat_group_id in remove_chat_group_member", zap.Any("value", req["chat_group_id"]))
				errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "missing or invalid chat_group_id", nil)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
			userID, ok := req["user_id"].(string)
			if !ok || userID == "" {
				log.Error("Missing or invalid user_id in remove_chat_group_member", zap.Any("value", req["user_id"]))
				errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "missing or invalid user_id", nil)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
			var campaignID int64
			if v, ok := req["campaign_id"].(float64); ok {
				campaignID = int64(v)
			}
			protoReq := &messagingpb.RemoveChatGroupMemberRequest{
				ChatGroupId: chatGroupID,
				UserId:      userID,
				CampaignId:  campaignID,
			}
			resp, err := messagingSvc.RemoveChatGroupMember(ctx, protoReq)
			if err != nil {
				log.Error("Failed to remove chat group member", zap.Error(err))
				errResp := graceful.WrapErr(ctx, codes.Internal, "failed to remove chat group member", err)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
			success := graceful.WrapSuccess(ctx, codes.OK, "chat group member removed", resp, nil)
			success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: log})
		case "list_chat_group_members":
			chatGroupID, ok := req["chat_group_id"].(string)
			if !ok || chatGroupID == "" {
				log.Error("Missing or invalid chat_group_id in list_chat_group_members", zap.Any("value", req["chat_group_id"]))
				errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "missing or invalid chat_group_id", nil)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
			page := int32(1)
			if v, ok := req["page"].(float64); ok {
				page = int32(v)
			}
			pageSize := int32(20)
			if v, ok := req["page_size"].(float64); ok {
				pageSize = int32(v)
			}
			var campaignID int64
			if v, ok := req["campaign_id"].(float64); ok {
				campaignID = int64(v)
			}
			protoReq := &messagingpb.ListChatGroupMembersRequest{
				ChatGroupId: chatGroupID,
				Page:        page,
				PageSize:    pageSize,
				CampaignId:  campaignID,
			}
			resp, err := messagingSvc.ListChatGroupMembers(ctx, protoReq)
			if err != nil {
				log.Error("Failed to list chat group members", zap.Error(err))
				errResp := graceful.WrapErr(ctx, codes.Internal, "failed to list chat group members", err)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
			success := graceful.WrapSuccess(ctx, codes.OK, "chat group members listed", resp, nil)
			success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: log})
		default:
			log.Error("Unknown action in messaging_ops", zap.String("action", action))
			errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "unknown action", nil)
			errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
			return
		}
	}
}
