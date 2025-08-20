package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	campaignpb "github.com/nmxmxh/master-ovasabi/api/protos/campaign/v1"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	contentpb "github.com/nmxmxh/master-ovasabi/api/protos/content/v1"
	securitypb "github.com/nmxmxh/master-ovasabi/api/protos/security/v1"
	"github.com/nmxmxh/master-ovasabi/internal/server/httputil"
	campaignmeta "github.com/nmxmxh/master-ovasabi/internal/service/campaign"
	"github.com/nmxmxh/master-ovasabi/pkg/auth" // Import the auth package
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/shield"
	"go.uber.org/zap"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// Error messages for permissions.
var (
	errInsufficientCampaignRole = errors.New("forbidden: insufficient campaign role")
	errAuthorMutationOnly       = errors.New("forbidden: only author can mutate")
	errMissingCampaignOrAuthor  = errors.New("missing campaign_id or author_id")
)

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
func ContentOpsHandler(container *di.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Inject logger into context
		log := contextx.Logger(r.Context())
		ctx := contextx.WithLogger(r.Context(), log)
		var contentSvc contentpb.ContentServiceServer
		if err := container.Resolve(&contentSvc); err != nil {
			log.Error("Failed to resolve ContentService", zap.Error(err))
			httputil.WriteJSONError(w, log, http.StatusInternalServerError, "internal error", err)
			return
		}
		if r.Method != http.MethodPost {
			httputil.WriteJSONError(w, log, http.StatusMethodNotAllowed, "method not allowed", nil)
			return
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode content request JSON", zap.Error(err))
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid JSON", err) // Already correct
			return
		}
		action, ok := req["action"].(string)
		if !ok || action == "" {
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid action", nil, zap.Any("value", req["action"]))
			return
		}

		// Extract authentication context
		authCtx := contextx.Auth(ctx)
		userID := authCtx.UserID
		isGuest := userID == "" || (len(authCtx.Roles) == 1 && authCtx.Roles[0] == "guest")
		var securitySvc securitypb.SecurityServiceClient
		if err := container.Resolve(&securitySvc); err != nil {
			log.Error("Failed to resolve SecurityService", zap.Error(err))
			httputil.WriteJSONError(w, log, http.StatusInternalServerError, "internal error", err)
			return
		}
		// Helper to build metadata for guest comments
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

		// --- Action Handlers ---
		actionHandlers := map[string]func(){
			"create_content": func() {
				if err := checkCampaignContentPermission(ctx, container, req, authCtx); err != nil {
					handlePermissionError(w, log, err)
					return
				}
				handleContentAction(ctx, w, log, req, &contentpb.CreateContentRequest{}, contentSvc.CreateContent)
			},
			"update_content": func() {
				if err := checkCampaignContentPermission(ctx, container, req, authCtx); err != nil {
					handlePermissionError(w, log, err)
					return
				}
				handleContentAction(ctx, w, log, req, &contentpb.UpdateContentRequest{}, contentSvc.UpdateContent)
			},
			"delete_content": func() {
				if err := checkCampaignContentPermission(ctx, container, req, authCtx); err != nil {
					handlePermissionError(w, log, err)
					return
				}
				handleContentAction(ctx, w, log, req, &contentpb.DeleteContentRequest{}, contentSvc.DeleteContent)
			},
			"get_content": func() {
				if err := checkCampaignContentPermission(ctx, container, req, authCtx); err != nil {
					handlePermissionError(w, log, err)
					return
				}
				handleContentAction(ctx, w, log, req, &contentpb.GetContentRequest{}, contentSvc.GetContent)
			},
			"list_content": func() {
				if err := checkCampaignContentPermission(ctx, container, req, authCtx); err != nil {
					handlePermissionError(w, log, err)
					return
				}
				handleContentAction(ctx, w, log, req, &contentpb.ListContentRequest{}, contentSvc.ListContent)
			},
			"add_reaction": func() {
				if isGuest {
					httputil.WriteJSONError(w, log, http.StatusUnauthorized, "unauthorized", nil)
					return
				}
				handleContentAction(ctx, w, log, req, &contentpb.AddReactionRequest{}, contentSvc.AddReaction)
			},
			"add_comment": func() {
				var authorID string
				var commentMeta *commonpb.Metadata

				if isGuest {
					guestNickname, ok1 := req["guest_nickname"].(string)
					deviceID, ok2 := req["device_id"].(string)
					if !ok1 || !ok2 || guestNickname == "" || deviceID == "" {
						httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing guest_nickname or device_id for guest comment", nil)
						return
					}
					authorID = "guest:" + deviceID
					commentMeta = buildGuestCommentMeta(guestNickname, deviceID)
				} else {
					authorID = userID
				}

				// Inject author_id and potentially metadata into the request map before unmarshaling
				req["author_id"] = authorID
				if commentMeta != nil {
					metaBytes, err := protojson.Marshal(commentMeta)
					if err != nil {
						httputil.WriteJSONError(w, log, http.StatusInternalServerError, "failed to process guest metadata", err)
						return
					}
					var metaMap map[string]interface{}
					if err := json.Unmarshal(metaBytes, &metaMap); err != nil {
						httputil.WriteJSONError(w, log, http.StatusInternalServerError, "failed to process guest metadata", err)
						return
					}
					req["metadata"] = metaMap
				}

				handleContentAction(ctx, w, log, req, &contentpb.AddCommentRequest{}, contentSvc.AddComment)
			},
			"moderate_content": func() {
				if isGuest {
					httputil.WriteJSONError(w, log, http.StatusUnauthorized, "unauthorized", nil)
					return
				}
				// Additional permission check for moderators can be added here if needed
				handleContentAction(ctx, w, log, req, &contentpb.ModerateContentRequest{}, contentSvc.ModerateContent)
			},
		}

		if handler, found := actionHandlers[action]; found {
			handler()
		} else {
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "unknown action", nil, zap.String("action", action))
		}
	}
}

// handleContentAction is a generic helper to reduce boilerplate in ContentOpsHandler.
func handleContentAction[T proto.Message, U proto.Message](
	ctx context.Context,
	w http.ResponseWriter,
	log *zap.Logger,
	reqMap map[string]interface{},
	req T,
	svcFunc func(context.Context, T) (U, error),
) {
	if err := mapToProtoContent(reqMap, req); err != nil {
		httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid request body", err)
		return
	}

	resp, err := svcFunc(ctx, req)
	if err != nil {
		st, _ := status.FromError(err)
		httpStatus := httputil.GRPCStatusToHTTPStatus(st.Code())
		log.Error("content service call failed", zap.Error(err), zap.String("grpc_code", st.Code().String()))
		httputil.WriteJSONError(w, log, httpStatus, st.Message(), nil)
		return
	}

	httputil.WriteJSONResponse(w, log, resp)
}

// mapToProtoContent converts a map[string]interface{} to a proto.Message using JSON as an intermediate.
func mapToProtoContent(data map[string]interface{}, v proto.Message) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return protojson.Unmarshal(jsonBytes, v)
}

// checkCampaignContentPermission centralizes the complex permission logic for content actions.
func checkCampaignContentPermission(ctx context.Context, container *di.Container, req map[string]interface{}, authCtx *auth.Context) error {
	log := contextx.Logger(ctx)
	userID := authCtx.UserID

	if userID == "" || (len(authCtx.Roles) == 1 && authCtx.Roles[0] == "guest") {
		return shield.ErrUnauthenticated
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

	switch {
	case campaignID != 0 || campaignSlug != "":
		var campaignSvc campaignpb.CampaignServiceServer
		if err := container.Resolve(&campaignSvc); err != nil {
			log.Error("Failed to resolve CampaignService", zap.Error(err))
			return errors.New("internal error")
		}
		var getReq *campaignpb.GetCampaignRequest
		if campaignSlug != "" {
			getReq = &campaignpb.GetCampaignRequest{Slug: campaignSlug}
		} else {
			// The current campaignpb.GetCampaignRequest does not support lookup by ID.
			// To enable this, the campaign.proto file needs to be updated and then the Go protobufs regenerated.
			// For now, we return an error if only campaignID is provided.
			return errors.New("campaign lookup by ID is not supported by the current API definition; use slug instead")
		}
		campResp, err := campaignSvc.GetCampaign(ctx, getReq)
		if err != nil || campResp == nil || campResp.Campaign == nil {
			log.Error("Failed to fetch campaign for permission check", zap.Error(err))
			return errors.New("failed to fetch campaign")
		}
		role := campaignmeta.GetUserRoleInCampaign(campResp.Campaign.Metadata, userID, campResp.Campaign.OwnerId)
		isSystem := campaignmeta.IsSystemCampaign(campResp.Campaign.Metadata)
		isPlatformAdmin := httputil.IsAdmin(authCtx.Roles)
		if role != "admin" && role != "user" && (!isSystem || !isPlatformAdmin) {
			return errInsufficientCampaignRole
		}
	case authorID != "":
		if authorID != userID {
			return errAuthorMutationOnly
		}
	default:
		return errMissingCampaignOrAuthor
	}

	return nil
}

// handlePermissionError maps a permission error to an HTTP status code.
func handlePermissionError(w http.ResponseWriter, log *zap.Logger, err error) {
	switch {
	case errors.Is(err, shield.ErrUnauthenticated):
		httputil.WriteJSONError(w, log, http.StatusUnauthorized, "unauthorized", err)
	case errors.Is(err, errInsufficientCampaignRole), errors.Is(err, errAuthorMutationOnly):
		httputil.WriteJSONError(w, log, http.StatusForbidden, err.Error(), err)
	case errors.Is(err, errMissingCampaignOrAuthor):
		httputil.WriteJSONError(w, log, http.StatusBadRequest, err.Error(), err)
	default:
		httputil.WriteJSONError(w, log, http.StatusInternalServerError, "internal error during permission check", err)
	}
}
