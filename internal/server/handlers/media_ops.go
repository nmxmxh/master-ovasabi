package handlers

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	mediapb "github.com/nmxmxh/master-ovasabi/api/protos/media/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/auth"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"go.uber.org/zap"
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

func MediaOpsHandler(log *zap.Logger, container *di.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var mediaSvc mediapb.MediaServiceServer
		if err := container.Resolve(&mediaSvc); err != nil {
			log.Error("Failed to resolve MediaService", zap.Error(err))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode media upload request JSON", zap.Error(err))
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		action, ok := req["action"].(string)
		if !ok || action == "" {
			log.Error("Missing or invalid action in media upload request", zap.Any("value", req["action"]))
			http.Error(w, "missing or invalid action", http.StatusBadRequest)
			return
		}
		authCtx := auth.FromContext(r.Context())
		userID, ok := req["user_id"].(string)
		if !ok {
			log.Error("Missing or invalid user_id in media request", zap.Any("value", req["user_id"]))
			http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
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
				http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
				return
			}
			if isGuest || (requestUserID != "" && requestUserID != userID && !isAdmin) {
				http.Error(w, "forbidden: must be authenticated and own the upload (or admin)", http.StatusForbidden)
				return
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
		}
		switch action {
		case "start_upload":
			name, ok := req["name"].(string)
			if !ok || name == "" {
				log.Error("Missing or invalid name in start_upload", zap.Any("value", req["name"]))
				http.Error(w, "missing or invalid name", http.StatusBadRequest)
				return
			}
			mimeType, ok := req["mime_type"].(string)
			if !ok || mimeType == "" {
				log.Error("Missing or invalid mime_type in start_upload", zap.Any("value", req["mime_type"]))
				http.Error(w, "missing or invalid mime_type", http.StatusBadRequest)
				return
			}
			size, ok := req["size"].(float64)
			if !ok {
				log.Error("Missing or invalid size in start_upload", zap.Any("value", req["size"]))
				http.Error(w, "missing or invalid size", http.StatusBadRequest)
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
						http.Error(w, "invalid metadata", http.StatusBadRequest)
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
			resp, err := mediaSvc.StartHeavyMediaUpload(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to start heavy media upload", zap.Error(err))
				http.Error(w, "failed to start upload", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				"upload_id":    resp.UploadId,
				"chunk_size":   resp.ChunkSize,
				"chunks_total": resp.ChunksTotal,
				"status":       resp.Status,
				"error":        resp.Error,
			}); err != nil {
				log.Error("Failed to write JSON response (start_upload)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "upload_chunk":
			uploadID, ok := req["upload_id"].(string)
			if !ok || uploadID == "" {
				log.Error("Missing or invalid upload_id in upload_chunk", zap.Any("value", req["upload_id"]))
				http.Error(w, "missing or invalid upload_id", http.StatusBadRequest)
				return
			}
			chunkData, ok := req["chunk"].(string)
			if !ok || chunkData == "" {
				log.Error("Missing or invalid chunk in upload_chunk", zap.Any("value", req["chunk"]))
				http.Error(w, "missing or invalid chunk", http.StatusBadRequest)
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
					http.Error(w, "invalid checksum type", http.StatusBadRequest)
					return
				}
			}
			// Decode chunkData from base64 if needed
			chunkBytes, err := base64.StdEncoding.DecodeString(chunkData)
			if err != nil {
				log.Error("Failed to decode chunk from base64", zap.Error(err))
				http.Error(w, "invalid chunk encoding", http.StatusBadRequest)
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
			resp, err := mediaSvc.StreamMediaChunk(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to stream media chunk", zap.Error(err))
				http.Error(w, "failed to upload chunk", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				"upload_id": resp.UploadId,
				"sequence":  resp.Sequence,
				"status":    resp.Status,
				"error":     resp.Error,
			}); err != nil {
				log.Error("Failed to write JSON response (upload_chunk)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "complete_upload":
			uploadID, ok := req["upload_id"].(string)
			if !ok || uploadID == "" {
				log.Error("Missing or invalid upload_id in complete_upload", zap.Any("value", req["upload_id"]))
				http.Error(w, "missing or invalid upload_id", http.StatusBadRequest)
				return
			}
			protoReq := &mediapb.CompleteMediaUploadRequest{
				UploadId: uploadID,
				UserId:   userID,
			}
			resp, err := mediaSvc.CompleteMediaUpload(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to complete media upload", zap.Error(err))
				http.Error(w, "failed to complete upload", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				"media":  resp.Media,
				"status": resp.Status,
				"error":  resp.Error,
			}); err != nil {
				log.Error("Failed to write JSON response (complete_upload)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		default:
			log.Error("Unknown action in media upload handler", zap.String("action", action))
			http.Error(w, "unknown action", http.StatusBadRequest)
			return
		}
	}
}
