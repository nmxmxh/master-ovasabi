package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	campaignpb "github.com/nmxmxh/master-ovasabi/api/protos/campaign/v1"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	contentpb "github.com/nmxmxh/master-ovasabi/api/protos/content/v1"
	securitypb "github.com/nmxmxh/master-ovasabi/api/protos/security/v1"
	campaignmeta "github.com/nmxmxh/master-ovasabi/internal/service/campaign"
	auth "github.com/nmxmxh/master-ovasabi/pkg/auth"
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
		// Extract authentication context
		authCtx := auth.FromContext(r.Context())
		userID := authCtx.UserID
		isGuest := userID == "" || (len(authCtx.Roles) == 1 && authCtx.Roles[0] == "guest")
		var securitySvc securitypb.SecurityServiceClient
		if err := container.Resolve(&securitySvc); err != nil {
			log.Error("Failed to resolve SecurityService", zap.Error(err))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		buildGuestCommentMeta := func(guestNickname, deviceID string) *commonpb.Metadata {
			fields := map[string]interface{}{
				"guest_comment":  true,
				"guest_nickname": guestNickname,
				"device_id":      deviceID,
			}
			ss := map[string]interface{}{"content": fields}
			ssStruct, err := structpb.NewStruct(ss)
			if err != nil {
				log.Error("Failed to convert guest comment metadata to structpb.Struct", zap.Error(err))
				return nil
			}
			return &commonpb.Metadata{ServiceSpecific: ssStruct}
		}
		switch action {
		case "create_content":
			if isGuest {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			var campaignID int64
			var campaignSlug string
			if v, ok := req["campaign_id"]; ok {
				switch vv := v.(type) {
				case float64:
					campaignID = int64(vv)
				case int64:
					campaignID = vv
				case string:
					campaignSlug = vv
				}
			}
			var authorID string
			if v, ok := req["author_id"].(string); ok {
				authorID = v
			}
			// --- Campaign-based: fetch campaign and check role ---
			switch {
			case campaignID != 0 || campaignSlug != "":
				var campaignSvc campaignpb.CampaignServiceServer
				if err := container.Resolve(&campaignSvc); err != nil {
					log.Error("Failed to resolve CampaignService", zap.Error(err))
					http.Error(w, "internal error", http.StatusInternalServerError)
					return
				}
				var getReq *campaignpb.GetCampaignRequest
				if campaignSlug != "" {
					getReq = &campaignpb.GetCampaignRequest{Slug: campaignSlug}
				} else {
					getReq = &campaignpb.GetCampaignRequest{Slug: ""} // TODO: support lookup by ID if needed
				}
				campResp, err := campaignSvc.GetCampaign(r.Context(), getReq)
				if err != nil || campResp == nil || campResp.Campaign == nil {
					log.Error("Failed to fetch campaign for permission check", zap.Error(err))
					http.Error(w, "failed to fetch campaign", http.StatusInternalServerError)
					return
				}
				role := campaignmeta.GetUserRoleInCampaign(campResp.Campaign.Metadata, userID, campResp.Campaign.OwnerId)
				isSystem := campaignmeta.IsSystemCampaign(campResp.Campaign.Metadata)
				isPlatformAdmin := isAdmin(authCtx.Roles)
				if role != "admin" && role != "user" && (!isSystem || !isPlatformAdmin) {
					http.Error(w, "forbidden: insufficient campaign role", http.StatusForbidden)
					return
				}
				// Optionally, extract subscription info for response
				// typ, price, currency, info := campaignmeta.GetSubscriptionInfo(campResp.Campaign.Metadata)
			case authorID != "":
				if authorID != userID {
					http.Error(w, "forbidden: only author can mutate", http.StatusForbidden)
					return
				}
			default:
				http.Error(w, "missing campaign_id or author_id", http.StatusBadRequest)
				return
			}
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
			if isGuest {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			var campaignID int64
			var campaignSlug string
			if v, ok := req["campaign_id"]; ok {
				switch vv := v.(type) {
				case float64:
					campaignID = int64(vv)
				case int64:
					campaignID = vv
				case string:
					campaignSlug = vv
				}
			}
			var authorID string
			if v, ok := req["author_id"].(string); ok {
				authorID = v
			}
			// --- Campaign-based: fetch campaign and check role ---
			switch {
			case campaignID != 0 || campaignSlug != "":
				var campaignSvc campaignpb.CampaignServiceServer
				if err := container.Resolve(&campaignSvc); err != nil {
					log.Error("Failed to resolve CampaignService", zap.Error(err))
					http.Error(w, "internal error", http.StatusInternalServerError)
					return
				}
				var getReq *campaignpb.GetCampaignRequest
				if campaignSlug != "" {
					getReq = &campaignpb.GetCampaignRequest{Slug: campaignSlug}
				} else {
					getReq = &campaignpb.GetCampaignRequest{Slug: ""} // TODO: support lookup by ID if needed
				}
				campResp, err := campaignSvc.GetCampaign(r.Context(), getReq)
				if err != nil || campResp == nil || campResp.Campaign == nil {
					log.Error("Failed to fetch campaign for permission check", zap.Error(err))
					http.Error(w, "failed to fetch campaign", http.StatusInternalServerError)
					return
				}
				role := campaignmeta.GetUserRoleInCampaign(campResp.Campaign.Metadata, userID, campResp.Campaign.OwnerId)
				isSystem := campaignmeta.IsSystemCampaign(campResp.Campaign.Metadata)
				isPlatformAdmin := isAdmin(authCtx.Roles)
				if role != "admin" && role != "user" && (!isSystem || !isPlatformAdmin) {
					http.Error(w, "forbidden: insufficient campaign role", http.StatusForbidden)
					return
				}
				// Optionally, extract subscription info for response
				// typ, price, currency, info := campaignmeta.GetSubscriptionInfo(campResp.Campaign.Metadata)
			case authorID != "":
				if authorID != userID {
					http.Error(w, "forbidden: only author can mutate", http.StatusForbidden)
					return
				}
			default:
				http.Error(w, "missing campaign_id or author_id", http.StatusBadRequest)
				return
			}
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
			if isGuest {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			var campaignID int64
			var campaignSlug string
			if v, ok := req["campaign_id"]; ok {
				switch vv := v.(type) {
				case float64:
					campaignID = int64(vv)
				case int64:
					campaignID = vv
				case string:
					campaignSlug = vv
				}
			}
			var authorID string
			if v, ok := req["author_id"].(string); ok {
				authorID = v
			}
			// --- Campaign-based: fetch campaign and check role ---
			switch {
			case campaignID != 0 || campaignSlug != "":
				var campaignSvc campaignpb.CampaignServiceServer
				if err := container.Resolve(&campaignSvc); err != nil {
					log.Error("Failed to resolve CampaignService", zap.Error(err))
					http.Error(w, "internal error", http.StatusInternalServerError)
					return
				}
				var getReq *campaignpb.GetCampaignRequest
				if campaignSlug != "" {
					getReq = &campaignpb.GetCampaignRequest{Slug: campaignSlug}
				} else {
					getReq = &campaignpb.GetCampaignRequest{Slug: ""} // TODO: support lookup by ID if needed
				}
				campResp, err := campaignSvc.GetCampaign(r.Context(), getReq)
				if err != nil || campResp == nil || campResp.Campaign == nil {
					log.Error("Failed to fetch campaign for permission check", zap.Error(err))
					http.Error(w, "failed to fetch campaign", http.StatusInternalServerError)
					return
				}
				role := campaignmeta.GetUserRoleInCampaign(campResp.Campaign.Metadata, userID, campResp.Campaign.OwnerId)
				isSystem := campaignmeta.IsSystemCampaign(campResp.Campaign.Metadata)
				isPlatformAdmin := isAdmin(authCtx.Roles)
				if role != "admin" && role != "user" && (!isSystem || !isPlatformAdmin) {
					http.Error(w, "forbidden: insufficient campaign role", http.StatusForbidden)
					return
				}
				// Optionally, extract subscription info for response
				// typ, price, currency, info := campaignmeta.GetSubscriptionInfo(campResp.Campaign.Metadata)
			case authorID != "":
				if authorID != userID {
					http.Error(w, "forbidden: only author can mutate", http.StatusForbidden)
					return
				}
			default:
				http.Error(w, "missing campaign_id or author_id", http.StatusBadRequest)
				return
			}
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
			if isGuest {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			var campaignID int64
			var campaignSlug string
			if v, ok := req["campaign_id"]; ok {
				switch vv := v.(type) {
				case float64:
					campaignID = int64(vv)
				case int64:
					campaignID = vv
				case string:
					campaignSlug = vv
				}
			}
			var authorID string
			if v, ok := req["author_id"].(string); ok {
				authorID = v
			}
			// --- Campaign-based: fetch campaign and check role ---
			switch {
			case campaignID != 0 || campaignSlug != "":
				var campaignSvc campaignpb.CampaignServiceServer
				if err := container.Resolve(&campaignSvc); err != nil {
					log.Error("Failed to resolve CampaignService", zap.Error(err))
					http.Error(w, "internal error", http.StatusInternalServerError)
					return
				}
				var getReq *campaignpb.GetCampaignRequest
				if campaignSlug != "" {
					getReq = &campaignpb.GetCampaignRequest{Slug: campaignSlug}
				} else {
					getReq = &campaignpb.GetCampaignRequest{Slug: ""} // TODO: support lookup by ID if needed
				}
				campResp, err := campaignSvc.GetCampaign(r.Context(), getReq)
				if err != nil || campResp == nil || campResp.Campaign == nil {
					log.Error("Failed to fetch campaign for permission check", zap.Error(err))
					http.Error(w, "failed to fetch campaign", http.StatusInternalServerError)
					return
				}
				role := campaignmeta.GetUserRoleInCampaign(campResp.Campaign.Metadata, userID, campResp.Campaign.OwnerId)
				isSystem := campaignmeta.IsSystemCampaign(campResp.Campaign.Metadata)
				isPlatformAdmin := isAdmin(authCtx.Roles)
				if role != "admin" && role != "user" && (!isSystem || !isPlatformAdmin) {
					http.Error(w, "forbidden: insufficient campaign role", http.StatusForbidden)
					return
				}
				// Optionally, extract subscription info for response
				// typ, price, currency, info := campaignmeta.GetSubscriptionInfo(campResp.Campaign.Metadata)
			case authorID != "":
				if authorID != userID {
					http.Error(w, "forbidden: only author can mutate", http.StatusForbidden)
					return
				}
			default:
				http.Error(w, "missing campaign_id or author_id", http.StatusBadRequest)
				return
			}
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
			if isGuest {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			page := int32(0)
			if p, ok := req["page"].(float64); ok {
				page = int32(p)
			}
			pageSize := int32(20)
			if ps, ok := req["page_size"].(float64); ok {
				pageSize = int32(ps)
			}
			var campaignID int64
			var campaignSlug string
			if v, ok := req["campaign_id"]; ok {
				switch vv := v.(type) {
				case float64:
					campaignID = int64(vv)
				case int64:
					campaignID = vv
				case string:
					campaignSlug = vv
				}
			}
			var authorID string
			if v, ok := req["author_id"].(string); ok {
				authorID = v
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
			// --- Campaign-based: fetch campaign and check role ---
			switch {
			case campaignID != 0 || campaignSlug != "":
				var campaignSvc campaignpb.CampaignServiceServer
				if err := container.Resolve(&campaignSvc); err != nil {
					log.Error("Failed to resolve CampaignService", zap.Error(err))
					http.Error(w, "internal error", http.StatusInternalServerError)
					return
				}
				var getReq *campaignpb.GetCampaignRequest
				if campaignSlug != "" {
					getReq = &campaignpb.GetCampaignRequest{Slug: campaignSlug}
				} else {
					getReq = &campaignpb.GetCampaignRequest{Slug: ""} // TODO: support lookup by ID if needed
				}
				campResp, err := campaignSvc.GetCampaign(r.Context(), getReq)
				if err != nil || campResp == nil || campResp.Campaign == nil {
					log.Error("Failed to fetch campaign for permission check", zap.Error(err))
					http.Error(w, "failed to fetch campaign", http.StatusInternalServerError)
					return
				}
				role := campaignmeta.GetUserRoleInCampaign(campResp.Campaign.Metadata, userID, campResp.Campaign.OwnerId)
				isSystem := campaignmeta.IsSystemCampaign(campResp.Campaign.Metadata)
				isPlatformAdmin := isAdmin(authCtx.Roles)
				if role != "admin" && role != "user" && (!isSystem || !isPlatformAdmin) {
					http.Error(w, "forbidden: insufficient campaign role", http.StatusForbidden)
					return
				}
				// Optionally, extract subscription info for response
				// typ, price, currency, info := campaignmeta.GetSubscriptionInfo(campResp.Campaign.Metadata)
			case authorID != "":
				if authorID != userID {
					http.Error(w, "forbidden: only author can mutate", http.StatusForbidden)
					return
				}
			default:
				http.Error(w, "missing campaign_id or author_id", http.StatusBadRequest)
				return
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
			if isGuest {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
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
			if isGuest {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			contentID, ok := req["content_id"].(string)
			if !ok {
				log.Error("Missing or invalid content_id in add_comment")
				http.Error(w, "missing or invalid content_id", http.StatusBadRequest)
				return
			}
			var authorID string
			var commentMeta *commonpb.Metadata
			if isGuest {
				// Guest comment: require guest_nickname and device_id
				guestNickname, ok1 := req["guest_nickname"].(string)
				deviceID, ok2 := req["device_id"].(string)
				if !ok1 || !ok2 || guestNickname == "" || deviceID == "" {
					log.Error("Missing guest_nickname or device_id for guest comment")
					http.Error(w, "missing guest_nickname or device_id for guest comment", http.StatusBadRequest)
					return
				}
				authorID = "guest:" + deviceID
				commentMeta = buildGuestCommentMeta(guestNickname, deviceID)
			} else {
				// Authenticated user
				authorID = userID
				if m, ok := req["metadata"].(map[string]interface{}); ok {
					if metaStruct, err := structpb.NewStruct(m); err == nil {
						commentMeta = &commonpb.Metadata{ServiceSpecific: metaStruct}
					}
				}
			}
			body, ok := req["body"].(string)
			if !ok {
				log.Error("Missing or invalid body in add_comment")
				http.Error(w, "missing or invalid body", http.StatusBadRequest)
				return
			}
			protoReq := &contentpb.AddCommentRequest{
				ContentId: contentID,
				AuthorId:  authorID,
				Body:      body,
				Metadata:  commentMeta,
			}
			resp, err := contentSvc.AddComment(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to add comment", zap.Error(err))
				http.Error(w, "failed to add comment", http.StatusInternalServerError)
				return
			}
			writeJSON(w, map[string]interface{}{"comment": resp.Comment}, log)
		case "moderate_content":
			if isGuest {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
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
			if isGuest {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			log.Error("Unknown action in content handler", zap.String("action", action))
			http.Error(w, "unknown action", http.StatusBadRequest)
			return
		}
	}
}
