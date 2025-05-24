package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	contentpb "github.com/nmxmxh/master-ovasabi/api/protos/content/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
)

// writeJSON is a DRY helper for writing JSON responses (inspired by generics best practices).
func writeJSON(w http.ResponseWriter, v interface{}, log *zap.Logger) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Error("Failed to write JSON response", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

// ContentOpsHandler handles content-related actions via the "action" field.
//
// @Summary Content Operations
// @Description Handles content-related actions using the "action" field in the request body. Each action (e.g., create_content, update_content, etc.) has its own required/optional fields. All requests must include a 'metadata' field following the robust metadata pattern (see docs/services/metadata.md).
// @Tags content
// @Accept json
// @Produce json
// @Param request body object true "Composable request with 'action', required fields for the action, and 'metadata' (see docs/services/metadata.md for schema)"
// @Success 200 {object} object "Response depends on action"
// @Failure 400 {object} ErrorResponse
// @Router /api/content_ops [post].
func ContentOpsHandler(log *zap.Logger, container *di.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var contentSvc contentpb.ContentServiceServer
		if err := container.Resolve(&contentSvc); err != nil {
			log.Error("Failed to resolve ContentService", zap.Error(err))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode content request JSON", zap.Error(err))
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		action, ok := req["action"].(string)
		if !ok || action == "" {
			log.Error("Missing or invalid action in content request", zap.Any("value", req["action"]))
			http.Error(w, "missing or invalid action", http.StatusBadRequest)
			return
		}
		switch action {
		case "create_content":
			title, ok := req["title"].(string)
			if !ok {
				log.Error("Missing or invalid title in create_content")
				http.Error(w, "missing or invalid title", http.StatusBadRequest)
				return
			}
			body, ok := req["body"].(string)
			if !ok {
				log.Error("Missing or invalid body in create_content")
				http.Error(w, "missing or invalid body", http.StatusBadRequest)
				return
			}
			typeStr, ok := req["type"].(string)
			if !ok {
				log.Error("Missing or invalid type in create_content")
				http.Error(w, "missing or invalid type", http.StatusBadRequest)
				return
			}
			tags := []string{}
			if arr, ok := req["tags"].([]interface{}); ok {
				for _, v := range arr {
					if s, ok := v.(string); ok {
						tags = append(tags, s)
					}
				}
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
			if v, ok := req["campaign_id"]; ok {
				switch vv := v.(type) {
				case float64:
					campaignID = int64(vv)
				case int64:
					campaignID = vv
				}
			}
			content := &contentpb.Content{
				Title:      title,
				Body:       body,
				Type:       typeStr,
				Tags:       tags,
				Metadata:   meta,
				CreatedAt:  time.Now().Unix(),
				UpdatedAt:  time.Now().Unix(),
				CampaignId: campaignID,
			}
			protoReq := &contentpb.CreateContentRequest{Content: content, CampaignId: campaignID}
			resp, err := contentSvc.CreateContent(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to create content", zap.Error(err))
				http.Error(w, "failed to create content", http.StatusInternalServerError)
				return
			}
			writeJSON(w, map[string]interface{}{"content": resp.Content}, log)
		case "update_content":
			id, ok := req["id"].(string)
			if !ok {
				log.Error("Missing or invalid id in update_content")
				http.Error(w, "missing or invalid id", http.StatusBadRequest)
				return
			}
			title, ok := req["title"].(string)
			if !ok {
				log.Error("Missing or invalid title in update_content")
				http.Error(w, "missing or invalid title", http.StatusBadRequest)
				return
			}
			body, ok := req["body"].(string)
			if !ok {
				log.Error("Missing or invalid body in update_content")
				http.Error(w, "missing or invalid body", http.StatusBadRequest)
				return
			}
			typeStr, ok := req["type"].(string)
			if !ok {
				log.Error("Missing or invalid type in update_content")
				http.Error(w, "missing or invalid type", http.StatusBadRequest)
				return
			}
			tags := []string{}
			if arr, ok := req["tags"].([]interface{}); ok {
				for _, v := range arr {
					if s, ok := v.(string); ok {
						tags = append(tags, s)
					}
				}
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
			if v, ok := req["campaign_id"]; ok {
				switch vv := v.(type) {
				case float64:
					campaignID = int64(vv)
				case int64:
					campaignID = vv
				}
			}
			content := &contentpb.Content{
				Id:         id,
				Title:      title,
				Body:       body,
				Type:       typeStr,
				Tags:       tags,
				Metadata:   meta,
				UpdatedAt:  time.Now().Unix(),
				CampaignId: campaignID,
			}
			protoReq := &contentpb.UpdateContentRequest{Content: content}
			resp, err := contentSvc.UpdateContent(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to update content", zap.Error(err))
				http.Error(w, "failed to update content", http.StatusInternalServerError)
				return
			}
			writeJSON(w, map[string]interface{}{"content": resp.Content}, log)
		case "delete_content":
			id, ok := req["id"].(string)
			if !ok {
				log.Error("Missing or invalid id in delete_content")
				http.Error(w, "missing or invalid id", http.StatusBadRequest)
				return
			}
			protoReq := &contentpb.DeleteContentRequest{Id: id}
			resp, err := contentSvc.DeleteContent(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to delete content", zap.Error(err))
				http.Error(w, "failed to delete content", http.StatusInternalServerError)
				return
			}
			writeJSON(w, map[string]interface{}{"success": resp.Success}, log)
		case "get_content":
			id, ok := req["id"].(string)
			if !ok {
				log.Error("Missing or invalid id in get_content")
				http.Error(w, "missing or invalid id", http.StatusBadRequest)
				return
			}
			protoReq := &contentpb.GetContentRequest{Id: id}
			resp, err := contentSvc.GetContent(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to get content", zap.Error(err))
				http.Error(w, "failed to get content", http.StatusInternalServerError)
				return
			}
			writeJSON(w, map[string]interface{}{"content": resp.Content}, log)
		case "list_content":
			page := int32(0)
			if p, ok := req["page"].(float64); ok {
				page = int32(p)
			}
			pageSize := int32(20)
			if ps, ok := req["page_size"].(float64); ok {
				pageSize = int32(ps)
			}
			authorID, ok := req["author_id"].(string)
			if !ok {
				log.Error("Missing or invalid author_id in list_content")
				http.Error(w, "missing or invalid author_id", http.StatusBadRequest)
				return
			}
			typeStr, ok := req["type"].(string)
			if !ok {
				log.Error("Missing or invalid type in list_content")
				http.Error(w, "missing or invalid type", http.StatusBadRequest)
				return
			}
			tags := []string{}
			if arr, ok := req["tags"].([]interface{}); ok {
				for _, v := range arr {
					if s, ok := v.(string); ok {
						tags = append(tags, s)
					}
				}
			}
			protoReq := &contentpb.ListContentRequest{
				AuthorId: authorID,
				Type:     typeStr,
				Page:     page,
				PageSize: pageSize,
				Tags:     tags,
			}
			resp, err := contentSvc.ListContent(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to list content", zap.Error(err))
				http.Error(w, "failed to list content", http.StatusInternalServerError)
				return
			}
			writeJSON(w, map[string]interface{}{"contents": resp.Contents, "total": resp.Total}, log)
		case "add_reaction":
			contentID, ok := req["content_id"].(string)
			if !ok {
				log.Error("Missing or invalid content_id in add_reaction")
				http.Error(w, "missing or invalid content_id", http.StatusBadRequest)
				return
			}
			userID, ok := req["user_id"].(string)
			if !ok {
				log.Error("Missing or invalid user_id in add_reaction")
				http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
				return
			}
			reaction, ok := req["reaction"].(string)
			if !ok {
				log.Error("Missing or invalid reaction in add_reaction")
				http.Error(w, "missing or invalid reaction", http.StatusBadRequest)
				return
			}
			protoReq := &contentpb.AddReactionRequest{
				ContentId: contentID,
				UserId:    userID,
				Reaction:  reaction,
			}
			resp, err := contentSvc.AddReaction(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to add reaction", zap.Error(err))
				http.Error(w, "failed to add reaction", http.StatusInternalServerError)
				return
			}
			writeJSON(w, map[string]interface{}{"reaction": resp}, log)
		case "add_comment":
			contentID, ok := req["content_id"].(string)
			if !ok {
				log.Error("Missing or invalid content_id in add_comment")
				http.Error(w, "missing or invalid content_id", http.StatusBadRequest)
				return
			}
			authorID, ok := req["author_id"].(string)
			if !ok {
				log.Error("Missing or invalid author_id in add_comment")
				http.Error(w, "missing or invalid author_id", http.StatusBadRequest)
				return
			}
			body, ok := req["body"].(string)
			if !ok {
				log.Error("Missing or invalid body in add_comment")
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
			protoReq := &contentpb.AddCommentRequest{
				ContentId: contentID,
				AuthorId:  authorID,
				Body:      body,
				Metadata:  meta,
			}
			resp, err := contentSvc.AddComment(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to add comment", zap.Error(err))
				http.Error(w, "failed to add comment", http.StatusInternalServerError)
				return
			}
			writeJSON(w, map[string]interface{}{"comment": resp.Comment}, log)
		case "moderate_content":
			contentID, ok := req["content_id"].(string)
			if !ok {
				log.Error("Missing or invalid content_id in moderate_content")
				http.Error(w, "missing or invalid content_id", http.StatusBadRequest)
				return
			}
			actionStr, ok := req["action_str"].(string) // avoid collision with 'action' field
			if !ok {
				log.Error("Missing or invalid action_str in moderate_content")
				http.Error(w, "missing or invalid action_str", http.StatusBadRequest)
				return
			}
			moderatorID, ok := req["moderator_id"].(string)
			if !ok {
				log.Error("Missing or invalid moderator_id in moderate_content")
				http.Error(w, "missing or invalid moderator_id", http.StatusBadRequest)
				return
			}
			reason, ok := req["reason"].(string)
			if !ok {
				log.Error("Missing or invalid reason in moderate_content")
				http.Error(w, "missing or invalid reason", http.StatusBadRequest)
				return
			}
			protoReq := &contentpb.ModerateContentRequest{
				ContentId:   contentID,
				Action:      actionStr,
				ModeratorId: moderatorID,
				Reason:      reason,
			}
			resp, err := contentSvc.ModerateContent(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to moderate content", zap.Error(err))
				http.Error(w, "failed to moderate content", http.StatusInternalServerError)
				return
			}
			writeJSON(w, map[string]interface{}{"moderation": resp}, log)
		// Extensible: add more actions (add_reaction, add_comment, moderate_content, etc.)
		default:
			log.Error("Unknown action in content handler", zap.String("action", action))
			http.Error(w, "unknown action", http.StatusBadRequest)
			return
		}
	}
}
