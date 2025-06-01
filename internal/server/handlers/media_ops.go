package handlers

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	mediapb "github.com/nmxmxh/master-ovasabi/api/protos/media/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"
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
	return func(w http.ResponseWriter, r *http.Request) {
		// Inject logger into context
		log := contextx.Logger(r.Context())
		ctx := contextx.WithLogger(r.Context(), log)
		var mediaSvc mediapb.MediaServiceServer
		if err := container.Resolve(&mediaSvc); err != nil {
			log.Error("Failed to resolve MediaService", zap.Error(err))
			errResp := graceful.WrapErr(ctx, codes.Internal, "internal error", err)
			errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode media upload request JSON", zap.Error(err))
			errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "invalid JSON", err)
			errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
			return
		}
		action, ok := req["action"].(string)
		if !ok || action == "" {
			log.Error("Missing or invalid action in media upload request", zap.Any("value", req["action"]))
			errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "missing or invalid action", nil)
			errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
			return
		}
		authCtx := contextx.Auth(ctx)
		userID, ok := req["user_id"].(string)
		if !ok {
			log.Error("Missing or invalid user_id in media request", zap.Any("value", req["user_id"]))
			errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "missing or invalid user_id", nil)
			errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
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
		// --- Permission checks and audit metadata for upload actions ---
		uploadActions := map[string]bool{"start_upload": true, "upload_chunk": true, "complete_upload": true}
		if uploadActions[action] {
			requestUserID, ok := req["user_id"].(string)
			if !ok {
				log.Error("Missing or invalid user_id in media request", zap.Any("value", req["user_id"]))
				errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "missing or invalid user_id", nil)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
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
					}
					metaMap["roles"] = rolesIface
					metaMap["audit"] = map[string]interface{}{
						"performed_by": userID,
						"roles":        roles,
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
					"roles": rolesIface,
					"audit": map[string]interface{}{
						"performed_by": userID,
						"roles":        roles,
						"timestamp":    time.Now().UTC().Format(time.RFC3339),
					},
				}
			}
		}
		switch action {
		case "start_upload":
			name, ok := req["name"].(string)
			if !ok || name == "" {
				log.Error("Missing or invalid name in start_upload", zap.Any("value", req["name"]))
				errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "missing or invalid name", nil)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
			mimeType, ok := req["mime_type"].(string)
			if !ok || mimeType == "" {
				log.Error("Missing or invalid mime_type in start_upload", zap.Any("value", req["mime_type"]))
				errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "missing or invalid mime_type", nil)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
			size, ok := req["size"].(float64)
			if !ok {
				log.Error("Missing or invalid size in start_upload", zap.Any("value", req["size"]))
				errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "missing or invalid size", nil)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
			// Metadata is optional and may be a map
			var meta *commonpb.Metadata
			if m, ok := req["metadata"]; ok && m != nil {
				if metaMap, ok := m.(map[string]interface{}); ok {
					var err error
					metaStruct, err := structpb.NewStruct(metaMap)
					if err != nil {
						log.Error("Failed to convert metadata to structpb.Struct", zap.Error(err))
						errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "invalid metadata", err)
						errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
						return
					}
					meta = &commonpb.Metadata{ServiceSpecific: metaStruct}
				}
			}
			// Build proto request
			protoReq := &mediapb.StartHeavyMediaUploadRequest{
				UserId:   userID,
				Name:     name,
				MimeType: mimeType,
				Size:     int64(size),
			}
			if meta != nil {
				protoReq.Metadata = meta
			}
			resp, err := mediaSvc.StartHeavyMediaUpload(ctx, protoReq)
			if err != nil {
				log.Error("Failed to start heavy media upload", zap.Error(err))
				errResp := graceful.WrapErr(ctx, codes.Internal, "failed to start upload", err)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
			success := graceful.WrapSuccess(ctx, codes.OK, "upload started", map[string]interface{}{
				"upload_id":    resp.UploadId,
				"chunk_size":   resp.ChunkSize,
				"chunks_total": resp.ChunksTotal,
				"status":       resp.Status,
				"error":        resp.Error,
			}, nil)
			success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: log})
		case "upload_chunk":
			uploadID, ok := req["upload_id"].(string)
			if !ok || uploadID == "" {
				log.Error("Missing or invalid upload_id in upload_chunk", zap.Any("value", req["upload_id"]))
				errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "missing or invalid upload_id", nil)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
			chunkData, ok := req["chunk"].(string)
			if !ok || chunkData == "" {
				log.Error("Missing or invalid chunk in upload_chunk", zap.Any("value", req["chunk"]))
				errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "missing or invalid chunk", nil)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
			// Optionally: sequence, checksum
			sequence := uint32(0)
			if seq, ok := req["sequence"].(float64); ok {
				sequence = uint32(seq)
			}
			checksumVal, ok := req["checksum"]
			var checksum string
			if ok && checksumVal != nil {
				checksum, ok = checksumVal.(string)
				if !ok {
					log.Error("Invalid checksum type in upload_chunk", zap.Any("value", checksumVal))
					errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "invalid checksum type", nil)
					errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
					return
				}
			}
			// Decode chunkData from base64 if needed
			chunkBytes, err := base64.StdEncoding.DecodeString(chunkData)
			if err != nil {
				log.Error("Failed to decode chunk from base64", zap.Error(err))
				errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "invalid chunk encoding", err)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
			protoReq := &mediapb.StreamMediaChunkRequest{
				UploadId: uploadID,
				Chunk: &mediapb.MediaChunk{
					UploadId: uploadID,
					Data:     chunkBytes,
					Sequence: sequence,
					Checksum: checksum,
				},
			}
			resp, err := mediaSvc.StreamMediaChunk(ctx, protoReq)
			if err != nil {
				log.Error("Failed to stream media chunk", zap.Error(err))
				errResp := graceful.WrapErr(ctx, codes.Internal, "failed to upload chunk", err)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
			success := graceful.WrapSuccess(ctx, codes.OK, "chunk uploaded", map[string]interface{}{
				"upload_id": resp.UploadId,
				"sequence":  resp.Sequence,
				"status":    resp.Status,
				"error":     resp.Error,
			}, nil)
			success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: log})
		case "complete_upload":
			uploadID, ok := req["upload_id"].(string)
			if !ok || uploadID == "" {
				log.Error("Missing or invalid upload_id in complete_upload", zap.Any("value", req["upload_id"]))
				errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "missing or invalid upload_id", nil)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
			protoReq := &mediapb.CompleteMediaUploadRequest{
				UploadId: uploadID,
				UserId:   userID,
			}
			resp, err := mediaSvc.CompleteMediaUpload(ctx, protoReq)
			if err != nil {
				log.Error("Failed to complete media upload", zap.Error(err))
				errResp := graceful.WrapErr(ctx, codes.Internal, "failed to complete upload", err)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return
			}
			success := graceful.WrapSuccess(ctx, codes.OK, "upload complete", map[string]interface{}{
				"media":  resp.Media,
				"status": resp.Status,
				"error":  resp.Error,
			}, nil)
			success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: log})
		default:
			log.Error("Unknown action in media upload handler", zap.String("action", action))
			errResp := graceful.WrapErr(ctx, codes.InvalidArgument, "unknown action", nil)
			errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
			return
		}
	}
}
