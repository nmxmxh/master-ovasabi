package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	mediapb "github.com/nmxmxh/master-ovasabi/api/protos/media/v1"
	"github.com/nmxmxh/master-ovasabi/internal/server/httputil"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// MediaOpsHandler handles media-related actions via the "action" field.
//
// @Summary Media Operations
// @Description Handles media-related actions using the "action" field in the request body. Each action (e.g., upload_media, get_media, etc.) has its own required/optional fields. All requests must include a 'metadata' field following the robust metadata pattern (see docs/services/metadata.md).
// @Tags media
// @Accept json
// @Produce json
// @Param request body object true "Composable request with 'action', required fields for the action, and 'metadata' (see docs/services/metadata.md for schema)"
// @Success 200 {object} object "Response depends on action"
// @Failure 400 {object} ErrorResponse
// @Router /api/media_ops [post]

func MediaOpsHandler(container *di.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { // ctx is created here, but not passed to the generic handler.
		// Inject logger into context
		log := contextx.Logger(r.Context())
		ctx := contextx.WithLogger(r.Context(), log)
		var mediaSvc mediapb.MediaServiceServer
		if err := container.Resolve(&mediaSvc); err != nil {
			log.Error("Failed to resolve MediaService", zap.Error(err))
			httputil.WriteJSONError(w, log, http.StatusInternalServerError, "internal error", err)
			return
		}
		if r.Method != http.MethodPost {
			httputil.WriteJSONError(w, log, http.StatusMethodNotAllowed, "method not allowed", nil)
			return
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid JSON", err)
			return
		}
		authCtx := contextx.Auth(ctx)
		userID, ok := req["user_id"].(string)
		if !ok {
			log.Error("Missing or invalid user_id in media request", zap.Any("value", req["user_id"]))
			errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "missing or invalid user_id", nil)
			errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
			return // This is an early exit, so it's fine to use graceful here.
		}
		action, ok := req["action"].(string)
		if !ok || action == "" {
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid action", nil)
			return
		}
		roles := authCtx.Roles
		isGuest := userID == "" || (len(roles) == 1 && roles[0] == "guest")
		isAdmin := false
		for _, r := range roles {
			if r == "admin" {
				isAdmin = true
			}
		}
		// --- Permission checks and audit metadata for protected actions ---
		protectedActions := map[string]bool{"start_upload": true, "upload_chunk": true, "complete_upload": true, "delete_media": true}
		if protectedActions[action] {
			requestUserID, ok := req["user_id"].(string)
			if !ok || requestUserID == "" {
				log.Error("Missing or invalid user_id in media request", zap.Any("value", req["user_id"]))
				httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid user_id", nil)
				return
			} // This is an early exit, so it's fine to use graceful here.
			if isGuest || (requestUserID != "" && requestUserID != userID && !isAdmin) {
				errResp := graceful.WrapErr(ctx, codes.PermissionDenied, "forbidden: must be authenticated and own the upload (or admin)", nil)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
			// --- Audit/metadata propagation ---
			if m, ok := req["metadata"]; ok && m != nil {
				if metaMap, ok := m.(map[string]interface{}); ok {
					// Convert roles []string to []interface{} for structpb compatibility
					rolesIface := make([]interface{}, len(roles))
					for i, r := range roles {
						rolesIface[i] = r
					} // Convert roles []string to []interface{} for structpb compatibility
					metaMap["roles"] = rolesIface // Convert roles []string to []interface{} for structpb compatibility
					metaMap["audit"] = map[string]interface{}{
						"performed_by": userID,
						"roles":        roles,
						"timestamp":    time.Now().UTC().Format(time.RFC3339),
					}
					req["metadata"] = metaMap
				} // Convert roles []string to []interface{} for structpb compatibility
			} else {
				// Convert roles []string to []interface{} for structpb compatibility
				rolesIface := make([]interface{}, len(roles))
				for i, r := range roles {
					rolesIface[i] = r
				} // Convert roles []string to []interface{} for structpb compatibility
				req["metadata"] = map[string]interface{}{ // Convert roles []string to []interface{} for structpb compatibility
					"roles": rolesIface,
					"audit": map[string]interface{}{
						"performed_by": userID,
						"roles":        roles,
						"timestamp":    time.Now().UTC().Format(time.RFC3339),
					},
				}
			} // Convert roles []string to []interface{} for structpb compatibility
		}

		actionHandlers := map[string]func(){
			"start_upload": func() {
				handleMediaAction(ctx, w, log, req, &mediapb.StartHeavyMediaUploadRequest{}, mediaSvc.StartHeavyMediaUpload)
			},
			"upload_chunk": func() {
				// protojson handles base64 decoding for bytes fields automatically.
				// However, if the client sends a non-base64 string, protojson will return an error.
				// The current implementation manually decodes base64, which is fine, but can be simplified.
				// For consistency with other handlers, we'll let mapToProtoMedia handle it.
				handleMediaAction(ctx, w, log, req, &mediapb.StreamMediaChunkRequest{}, mediaSvc.StreamMediaChunk)
			},
			"complete_upload": func() {
				handleMediaAction(ctx, w, log, req, &mediapb.CompleteMediaUploadRequest{}, mediaSvc.CompleteMediaUpload)
			},
			"get_media": func() {
				handleMediaAction(ctx, w, log, req, &mediapb.GetMediaRequest{}, mediaSvc.GetMedia)
			},
			"list_user_media": func() {
				handleMediaAction(ctx, w, log, req, &mediapb.ListUserMediaRequest{}, mediaSvc.ListUserMedia)
			},
			"delete_media": func() {
				handleMediaAction(ctx, w, log, req, &mediapb.DeleteMediaRequest{}, mediaSvc.DeleteMedia)
			},
		}

		if handler, found := actionHandlers[action]; found {
			handler()
		} else {
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "unknown action", nil)
		}
	}
}

// handleMediaAction is a generic helper to reduce boilerplate in MediaOpsHandler.
func handleMediaAction[T proto.Message, U proto.Message](
	ctx context.Context,
	w http.ResponseWriter,
	log *zap.Logger,
	reqMap map[string]interface{},
	req T,
	svcFunc func(context.Context, T) (U, error),
) {
	if err := mapToProtoMedia(reqMap, req); err != nil {
		httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid request body", err)
		return
	}

	resp, err := svcFunc(ctx, req)
	if err != nil {
		st, _ := status.FromError(err)
		httpStatus := httputil.GRPCStatusToHTTPStatus(st.Code())
		log.Error("media service call failed", zap.Error(err), zap.String("grpc_code", st.Code().String()))
		httputil.WriteJSONError(w, log, httpStatus, st.Message(), nil)
		return
	}

	httputil.WriteJSONResponse(w, log, resp)
}

// mapToProtoMedia converts a map[string]interface{} to a proto.Message using JSON as an intermediate.
func mapToProtoMedia(data map[string]interface{}, v proto.Message) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return protojson.Unmarshal(jsonBytes, v)
}
