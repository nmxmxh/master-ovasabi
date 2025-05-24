package handlers

import (
	"encoding/json"
	"net/http"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	messagingpb "github.com/nmxmxh/master-ovasabi/api/protos/messaging/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"go.uber.org/zap"
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
func MessagingOpsHandler(log *zap.Logger, container *di.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var messagingSvc messagingpb.MessagingServiceServer
		if err := container.Resolve(&messagingSvc); err != nil {
			log.Error("MessagingServiceServer not found in container", zap.Error(err))
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode messaging request JSON", zap.Error(err))
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		action, ok := req["action"].(string)
		if !ok || action == "" {
			log.Error("Missing or invalid action in messaging request", zap.Any("value", req["action"]))
			http.Error(w, "missing or invalid action", http.StatusBadRequest)
			return
		}
		switch action {
		case "send_message":
			threadID, ok := req["thread_id"].(string)
			if !ok && req["thread_id"] != nil {
				log.Error("Invalid thread_id in send_message", zap.Any("value", req["thread_id"]))
				http.Error(w, "invalid thread_id", http.StatusBadRequest)
				return
			}
			conversationID, ok := req["conversation_id"].(string)
			if !ok && req["conversation_id"] != nil {
				log.Error("Invalid conversation_id in send_message", zap.Any("value", req["conversation_id"]))
				http.Error(w, "invalid conversation_id", http.StatusBadRequest)
				return
			}
			chatGroupID, ok := req["chat_group_id"].(string)
			if !ok && req["chat_group_id"] != nil {
				log.Error("Invalid chat_group_id in send_message", zap.Any("value", req["chat_group_id"]))
				http.Error(w, "invalid chat_group_id", http.StatusBadRequest)
				return
			}
			senderID, ok := req["sender_id"].(string)
			if !ok || senderID == "" {
				log.Error("Missing or invalid sender_id in send_message", zap.Any("value", req["sender_id"]))
				http.Error(w, "missing or invalid sender_id", http.StatusBadRequest)
				return
			}
			content, ok := req["content"].(string)
			if !ok || content == "" {
				log.Error("Missing or invalid content in send_message", zap.Any("value", req["content"]))
				http.Error(w, "missing or invalid content", http.StatusBadRequest)
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
			var campaignID int64
			if v, ok := req["campaign_id"].(float64); ok {
				campaignID = int64(v)
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
			resp, err := messagingSvc.SendMessage(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to send message", zap.Error(err))
				http.Error(w, "failed to send message", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (send_message)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "list_messages":
			threadID, ok := req["thread_id"].(string)
			if !ok && req["thread_id"] != nil {
				log.Error("Invalid thread_id in list_messages", zap.Any("value", req["thread_id"]))
				http.Error(w, "invalid thread_id", http.StatusBadRequest)
				return
			}
			conversationID, ok := req["conversation_id"].(string)
			if !ok && req["conversation_id"] != nil {
				log.Error("Invalid conversation_id in list_messages", zap.Any("value", req["conversation_id"]))
				http.Error(w, "invalid conversation_id", http.StatusBadRequest)
				return
			}
			chatGroupID, ok := req["chat_group_id"].(string)
			if !ok && req["chat_group_id"] != nil {
				log.Error("Invalid chat_group_id in list_messages", zap.Any("value", req["chat_group_id"]))
				http.Error(w, "invalid chat_group_id", http.StatusBadRequest)
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
			resp, err := messagingSvc.ListMessages(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to list messages", zap.Error(err))
				http.Error(w, "failed to list messages", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (list_messages)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "update_preferences":
			userID, ok := req["user_id"].(string)
			if !ok || userID == "" {
				log.Error("Missing or invalid user_id in update_preferences", zap.Any("value", req["user_id"]))
				http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
				return
			}
			prefsMap, ok := req["preferences"].(map[string]interface{})
			if !ok {
				log.Error("Missing or invalid preferences in update_preferences", zap.Any("value", req["preferences"]))
				http.Error(w, "missing or invalid preferences", http.StatusBadRequest)
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
			resp, err := messagingSvc.UpdateMessagingPreferences(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to update messaging preferences", zap.Error(err))
				http.Error(w, "failed to update preferences", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (update_preferences)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "list_threads":
			userID, ok := req["user_id"].(string)
			if !ok || userID == "" {
				log.Error("Missing or invalid user_id in list_threads", zap.Any("value", req["user_id"]))
				http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
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
			resp, err := messagingSvc.ListThreads(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to list threads", zap.Error(err))
				http.Error(w, "failed to list threads", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (list_threads)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "add_chat_group_member":
			chatGroupID, ok := req["chat_group_id"].(string)
			if !ok || chatGroupID == "" {
				log.Error("Missing or invalid chat_group_id in add_chat_group_member", zap.Any("value", req["chat_group_id"]))
				http.Error(w, "missing or invalid chat_group_id", http.StatusBadRequest)
				return
			}
			userID, ok := req["user_id"].(string)
			if !ok || userID == "" {
				log.Error("Missing or invalid user_id in add_chat_group_member", zap.Any("value", req["user_id"]))
				http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
				return
			}
			roleVal := req["role"]
			role, okStr := roleVal.(string)
			if !okStr {
				log.Error("Invalid or missing role in request", zap.Any("role", roleVal))
				http.Error(w, "invalid or missing role", http.StatusBadRequest)
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
			resp, err := messagingSvc.AddChatGroupMember(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to add chat group member", zap.Error(err))
				http.Error(w, "failed to add chat group member", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (add_chat_group_member)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "remove_chat_group_member":
			chatGroupID, ok := req["chat_group_id"].(string)
			if !ok || chatGroupID == "" {
				log.Error("Missing or invalid chat_group_id in remove_chat_group_member", zap.Any("value", req["chat_group_id"]))
				http.Error(w, "missing or invalid chat_group_id", http.StatusBadRequest)
				return
			}
			userID, ok := req["user_id"].(string)
			if !ok || userID == "" {
				log.Error("Missing or invalid user_id in remove_chat_group_member", zap.Any("value", req["user_id"]))
				http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
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
			resp, err := messagingSvc.RemoveChatGroupMember(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to remove chat group member", zap.Error(err))
				http.Error(w, "failed to remove chat group member", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (remove_chat_group_member)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "list_chat_group_members":
			chatGroupID, ok := req["chat_group_id"].(string)
			if !ok || chatGroupID == "" {
				log.Error("Missing or invalid chat_group_id in list_chat_group_members", zap.Any("value", req["chat_group_id"]))
				http.Error(w, "missing or invalid chat_group_id", http.StatusBadRequest)
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
			resp, err := messagingSvc.ListChatGroupMembers(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to list chat group members", zap.Error(err))
				http.Error(w, "failed to list chat group members", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (list_chat_group_members)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		default:
			log.Error("Unknown action in messaging_ops", zap.String("action", action))
			http.Error(w, "unknown action", http.StatusBadRequest)
			return
		}
	}
}
