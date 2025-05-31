package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	campaignpb "github.com/nmxmxh/master-ovasabi/api/protos/campaign/v1"
	contentmoderationpb "github.com/nmxmxh/master-ovasabi/api/protos/contentmoderation/v1"
	campaignmeta "github.com/nmxmxh/master-ovasabi/internal/service/campaign"
	auth "github.com/nmxmxh/master-ovasabi/pkg/auth"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"go.uber.org/zap"
)

// ContentModerationOpsHandler handles content moderation actions via the "action" field.
//
// @Summary Content Moderation Operations
// @Description Handles content moderation actions using the "action" field in the request body. Each action (e.g., flag_content, review_content, etc.) has its own required/optional fields. All requests must include a 'metadata' field following the robust metadata pattern (see docs/services/metadata.md).
// @Tags contentmoderation
// @Accept json
// @Produce json
// @Param request body object true "Composable request with 'action', required fields for the action, and 'metadata' (see docs/services/metadata.md for schema)"
// @Success 200 {object} object "Response depends on action"
// @Failure 400 {object} ErrorResponse
// @Router /api/contentmoderation_ops [post]

// ContentModerationOpsHandler: composable handler for content moderation operations.
func ContentModerationOpsHandler(log *zap.Logger, container *di.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var moderationSvc contentmoderationpb.ContentModerationServiceServer
		if err := container.Resolve(&moderationSvc); err != nil {
			log.Error("Failed to resolve ContentModerationService", zap.Error(err))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode content moderation request JSON", zap.Error(err))
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		action, ok := req["action"].(string)
		if !ok || action == "" {
			log.Error("Missing or invalid action in content moderation request", zap.Any("value", req["action"]))
			http.Error(w, "missing or invalid action", http.StatusBadRequest)
			return
		}
		authCtx := auth.FromContext(r.Context())
		userID := authCtx.UserID
		roles := authCtx.Roles
		isGuest := userID == "" || (len(roles) == 1 && roles[0] == "guest")
		isPlatformAdmin := false
		isModerator := false
		for _, r := range roles {
			if r == "admin" {
				isPlatformAdmin = true
			}
			if r == "moderator" {
				isModerator = true
			}
		}
		// Helper: check campaign admin role if campaign_id or campaign_slug is present
		isCampaignAdmin := false
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
		if campaignSlug != "" || campaignID != 0 {
			var campaignSvc campaignpb.CampaignServiceServer
			if err := container.Resolve(&campaignSvc); err == nil {
				var getReq *campaignpb.GetCampaignRequest
				if campaignSlug != "" {
					getReq = &campaignpb.GetCampaignRequest{Slug: campaignSlug}
				} else {
					getReq = &campaignpb.GetCampaignRequest{Slug: ""} // TODO: support lookup by ID if needed
				}
				campResp, err := campaignSvc.GetCampaign(r.Context(), getReq)
				if err == nil && campResp != nil && campResp.Campaign != nil {
					role := campaignmeta.GetUserRoleInCampaign(campResp.Campaign.Metadata, userID, campResp.Campaign.OwnerId)
					if role == "admin" {
						isCampaignAdmin = true
					}
				}
			}
		}
		// --- Permission checks by action ---
		switch action {
		case "approve_content", "reject_content", "list_flagged_content":
			if isGuest || (!isPlatformAdmin && !isModerator && !isCampaignAdmin) {
				http.Error(w, "forbidden: admin or moderator required", http.StatusForbidden)
				return
			}
		case "submit_content_for_moderation", "get_moderation_result":
			// Allow content author or campaign admin
			authorID, ok := req["author_id"].(string)
			if !ok {
				// handle type assertion failure
				return
			}
			if userID != authorID && !isPlatformAdmin && !isCampaignAdmin {
				http.Error(w, "forbidden: only author or admin can perform this action", http.StatusForbidden)
				return
			}
		}
		// --- Audit/metadata propagation ---
		if m, ok := req["metadata"]; ok && m != nil {
			if metaMap, ok := m.(map[string]interface{}); ok {
				metaMap["audit"] = map[string]interface{}{
					"performed_by": userID,
					"roles":        roles,
					"timestamp":    time.Now().UTC().Format(time.RFC3339),
				}
				req["metadata"] = metaMap
			}
		} else {
			req["metadata"] = map[string]interface{}{
				"audit": map[string]interface{}{
					"performed_by": userID,
					"roles":        roles,
					"timestamp":    time.Now().UTC().Format(time.RFC3339),
				},
			}
		}
		ctx := r.Context()
		switch action {
		case "submit_content_for_moderation":
			var protoReq contentmoderationpb.SubmitContentForModerationRequest
			if err := mapToProto(req, &protoReq); err != nil {
				log.Error("Failed to map request to proto (submit)", zap.Error(err))
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
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
			protoReq.CampaignId = campaignID
			resp, err := moderationSvc.SubmitContentForModeration(ctx, &protoReq)
			if err != nil {
				log.Error("Failed to submit content for moderation", zap.Error(err))
				http.Error(w, "failed to submit content for moderation", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (submit)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "get_moderation_result":
			var protoReq contentmoderationpb.GetModerationResultRequest
			if err := mapToProto(req, &protoReq); err != nil {
				log.Error("Failed to map request to proto (get)", zap.Error(err))
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}
			resp, err := moderationSvc.GetModerationResult(ctx, &protoReq)
			if err != nil {
				log.Error("Failed to get moderation result", zap.Error(err))
				http.Error(w, "failed to get moderation result", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (get)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "list_flagged_content":
			var protoReq contentmoderationpb.ListFlaggedContentRequest
			if err := mapToProto(req, &protoReq); err != nil {
				log.Error("Failed to map request to proto (list)", zap.Error(err))
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
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
			protoReq.CampaignId = campaignID
			resp, err := moderationSvc.ListFlaggedContent(ctx, &protoReq)
			if err != nil {
				log.Error("Failed to list flagged content", zap.Error(err))
				http.Error(w, "failed to list flagged content", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (list)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "approve_content":
			var protoReq contentmoderationpb.ApproveContentRequest
			if err := mapToProto(req, &protoReq); err != nil {
				log.Error("Failed to map request to proto (approve)", zap.Error(err))
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
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
			protoReq.CampaignId = campaignID
			resp, err := moderationSvc.ApproveContent(ctx, &protoReq)
			if err != nil {
				log.Error("Failed to approve content", zap.Error(err))
				http.Error(w, "failed to approve content", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (approve)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "reject_content":
			var protoReq contentmoderationpb.RejectContentRequest
			if err := mapToProto(req, &protoReq); err != nil {
				log.Error("Failed to map request to proto (reject)", zap.Error(err))
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
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
			protoReq.CampaignId = campaignID
			resp, err := moderationSvc.RejectContent(ctx, &protoReq)
			if err != nil {
				log.Error("Failed to reject content", zap.Error(err))
				http.Error(w, "failed to reject content", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (reject)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		default:
			log.Error("Unknown action in contentmoderation_ops", zap.Any("action", action))
			http.Error(w, "unknown action", http.StatusBadRequest)
			return
		}
	}
}

// mapToProto is a helper to map a generic map[string]interface{} to a proto message using JSON marshal/unmarshal.
func mapToProto(m map[string]interface{}, pbMsg interface{}) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, pbMsg)
}
