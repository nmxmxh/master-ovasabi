package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	campaignpb "github.com/nmxmxh/master-ovasabi/api/protos/campaign/v1"
	contentmoderationpb "github.com/nmxmxh/master-ovasabi/api/protos/contentmoderation/v1"
	"github.com/nmxmxh/master-ovasabi/internal/server/httputil"
	campaignmeta "github.com/nmxmxh/master-ovasabi/internal/service/campaign"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"go.uber.org/zap"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
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
func ContentModerationOpsHandler(container *di.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Inject logger into context
		log := contextx.Logger(r.Context())
		ctx := contextx.WithLogger(r.Context(), log)
		var moderationSvc contentmoderationpb.ContentModerationServiceServer
		if err := container.Resolve(&moderationSvc); err != nil {
			log.Error("Failed to resolve ContentModerationService", zap.Error(err))
			httputil.WriteJSONError(w, log, http.StatusInternalServerError, "internal error", err)
			return
		}
		if r.Method != http.MethodPost {
			httputil.WriteJSONError(w, log, http.StatusMethodNotAllowed, "method not allowed", nil)
			return
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode content moderation request JSON", zap.Error(err))
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid JSON", err)
			return
		}
		action, ok := req["action"].(string)
		if !ok || action == "" {
			log.Error("Missing or invalid action in content moderation request", zap.Any("value", req["action"]))
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid action", nil, zap.Any("value", req["action"])) // Already correct
			return
		}
		authCtx := contextx.Auth(ctx)
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
				campResp, err := campaignSvc.GetCampaign(ctx, getReq)
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
				// Convert roles []string to []interface{} for structpb compatibility
				rolesIface := make([]interface{}, len(roles))
				for i, r := range roles {
					rolesIface[i] = r
				}
				metaMap["roles"] = rolesIface
				metaMap["audit"] = map[string]interface{}{
					"performed_by": userID,
					"timestamp":    time.Now().UTC().Format(time.RFC3339),
				}
				req["metadata"] = metaMap
			}
		} else {
			// Convert roles []string to []interface{} for structpb compatibility
			rolesIface := make([]interface{}, len(roles))
			for i, r := range roles {
				rolesIface[i] = r
			}
			req["metadata"] = map[string]interface{}{
				"audit": map[string]interface{}{
					"performed_by": userID,
					"roles":        rolesIface,
					"timestamp":    time.Now().UTC().Format(time.RFC3339),
				},
				"roles": rolesIface,
			}
		}

		actionHandlers := map[string]func(){
			"submit_content_for_moderation": func() {
				handleContentModerationAction(ctx, w, log, req, &contentmoderationpb.SubmitContentForModerationRequest{}, moderationSvc.SubmitContentForModeration)
			},
			"get_moderation_result": func() {
				handleContentModerationAction(ctx, w, log, req, &contentmoderationpb.GetModerationResultRequest{}, moderationSvc.GetModerationResult)
			},
			"list_flagged_content": func() {
				handleContentModerationAction(ctx, w, log, req, &contentmoderationpb.ListFlaggedContentRequest{}, moderationSvc.ListFlaggedContent)
			},
			"approve_content": func() {
				handleContentModerationAction(ctx, w, log, req, &contentmoderationpb.ApproveContentRequest{}, moderationSvc.ApproveContent)
			},
			"reject_content": func() {
				handleContentModerationAction(ctx, w, log, req, &contentmoderationpb.RejectContentRequest{}, moderationSvc.RejectContent)
			},
		}

		if handler, found := actionHandlers[action]; found {
			handler()
		} else {
			log.Error("Unknown action in contentmoderation_ops", zap.Any("action", action))
			http.Error(w, "unknown action", http.StatusBadRequest)
		}
	}
}

// handleContentModerationAction is a generic helper to reduce boilerplate in ContentModerationOpsHandler.
func handleContentModerationAction[T proto.Message, U proto.Message](
	ctx context.Context,
	w http.ResponseWriter,
	log *zap.Logger,
	reqMap map[string]interface{},
	req T,
	svcFunc func(context.Context, T) (U, error),
) {
	if err := mapToProtoContentModeration(reqMap, req); err != nil {
		httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid request body", err)
		return
	}

	resp, err := svcFunc(ctx, req)
	if err != nil {
		st, _ := status.FromError(err)
		httpStatus := httputil.GRPCStatusToHTTPStatus(st.Code())
		log.Error("content moderation service call failed", zap.Error(err), zap.String("grpc_code", st.Code().String()))
		httputil.WriteJSONError(w, log, httpStatus, st.Message(), nil)
		return
	}

	httputil.WriteJSONResponse(w, log, resp)
}

// mapToProtoContentModeration converts a map[string]interface{} to a proto.Message using JSON as an intermediate.
func mapToProtoContentModeration(data map[string]interface{}, v proto.Message) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return protojson.Unmarshal(jsonBytes, v)
}
