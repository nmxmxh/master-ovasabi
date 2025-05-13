package server

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	mediapb "github.com/nmxmxh/master-ovasabi/api/protos/media/v1"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"go.uber.org/zap"
)

// RegisterMediaUploadHandlers registers all media upload endpoints to the mux.
func RegisterMediaUploadHandlers(mux *http.ServeMux, log *zap.Logger, provider *service.Provider, wsClients *sync.Map) {
	redisClient := provider.RedisClient().Client

	// Helper for consistent error responses
	writeJSONError := func(w http.ResponseWriter, log *zap.Logger, msg string, err error, status int) {
		w.WriteHeader(status)
		if err2 := json.NewEncoder(w).Encode(map[string]string{"error": msg}); err2 != nil {
			log.Error("Failed to write JSON error response", zap.Error(err2))
		}
		if err != nil {
			log.Error(msg, zap.Error(err))
		}
	}

	mux.HandleFunc("/api/media/start_upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			UserID   string                 `json:"user_id"`
			Name     string                 `json:"name"`
			MimeType string                 `json:"mime_type"`
			Size     int64                  `json:"size"`
			Metadata map[string]interface{} `json:"metadata"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, log, "invalid JSON", err, http.StatusBadRequest)
			return
		}
		if req.UserID == "" {
			req.UserID = generateGuestID()
		}
		var mediaSvc mediapb.MediaServiceClient
		if err := provider.Container().Resolve(&mediaSvc); err != nil {
			writeJSONError(w, log, "media service unavailable", err, http.StatusInternalServerError)
			return
		}
		grpcReq := &mediapb.StartHeavyMediaUploadRequest{
			UserId:   req.UserID,
			Name:     req.Name,
			MimeType: req.MimeType,
			Size:     req.Size,
			// Metadata: ...
		}
		resp, err := mediaSvc.StartHeavyMediaUpload(r.Context(), grpcReq)
		if err != nil {
			writeJSONError(w, log, "failed to start upload", err, http.StatusInternalServerError)
			return
		}
		progressKey := "media:upload:progress:" + resp.UploadId
		if err := redisClient.HSet(r.Context(), progressKey, map[string]interface{}{
			"user_id":  req.UserID,
			"received": 0,
			"total":    resp.ChunksTotal,
		}).Err(); err != nil {
			writeJSONError(w, log, "failed to set progress in Redis", err, http.StatusInternalServerError)
			return
		}
		log.Info("Started media upload", zap.String("upload_id", resp.UploadId), zap.String("user_id", req.UserID), zap.Int32("chunks_total", resp.ChunksTotal))
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"upload_id":    resp.UploadId,
			"chunk_size":   resp.ChunkSize,
			"chunks_total": resp.ChunksTotal,
			"status":       resp.Status,
		}); err != nil {
			log.Error("Failed to write JSON response", zap.Error(err))
		}
	})

	mux.HandleFunc("/api/media/upload_chunk", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			UploadID string `json:"upload_id"`
			UserID   string `json:"user_id"`
			Chunk    []byte `json:"chunk"`
			Sequence int    `json:"sequence"`
			Checksum string `json:"checksum"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, log, "invalid JSON", err, http.StatusBadRequest)
			return
		}
		var mediaSvc mediapb.MediaServiceClient
		if err := provider.Container().Resolve(&mediaSvc); err != nil {
			writeJSONError(w, log, "media service unavailable", err, http.StatusInternalServerError)
			return
		}
		seq := req.Sequence
		if seq < 0 || seq > int(^uint32(0)) {
			writeJSONError(w, log, "Sequence out of uint32 range", nil, http.StatusBadRequest)
			return
		}
		grpcReq := &mediapb.StreamMediaChunkRequest{
			UploadId: req.UploadID,
			Chunk: &mediapb.MediaChunk{
				UploadId: req.UploadID,
				Data:     req.Chunk,
				Sequence: uint32(seq),
				Checksum: req.Checksum,
			},
		}
		_, err := mediaSvc.StreamMediaChunk(r.Context(), grpcReq)
		if err != nil {
			writeJSONError(w, log, "failed to stream media chunk", err, http.StatusInternalServerError)
			return
		}
		progressKey := "media:upload:progress:" + req.UploadID
		userID, err := redisClient.HGet(r.Context(), progressKey, "user_id").Result()
		if err != nil {
			writeJSONError(w, log, "failed to get user_id from Redis", err, http.StatusInternalServerError)
			return
		}
		total, err := redisClient.HGet(r.Context(), progressKey, "total").Int()
		if err != nil {
			writeJSONError(w, log, "failed to get total from Redis", err, http.StatusInternalServerError)
			return
		}
		if err := redisClient.HIncrBy(r.Context(), progressKey, "received", 1).Err(); err != nil {
			writeJSONError(w, log, "failed to increment received in Redis", err, http.StatusInternalServerError)
			return
		}
		received, err := redisClient.HGet(r.Context(), progressKey, "received").Int()
		if err != nil {
			writeJSONError(w, log, "failed to get received from Redis", err, http.StatusInternalServerError)
			return
		}
		percent := float64(received) / float64(total) * 100
		log.Info("Chunk uploaded", zap.String("upload_id", req.UploadID), zap.String("user_id", userID), zap.Int("received", received), zap.Int("total", total), zap.Float64("percent", percent))
		if conn, ok := wsClients.Load(userID); ok {
			if err := conn.(*websocket.Conn).WriteJSON(map[string]interface{}{
				"type":            "media_event",
				"event":           "upload_progress",
				"upload_id":       req.UploadID,
				"user_id":         userID,
				"received_chunks": received,
				"total_chunks":    total,
				"percent":         percent,
			}); err != nil {
				log.Error("failed to write JSON to websocket (upload_progress)",
					zap.Error(err),
					zap.String("user_id", userID),
					zap.String("upload_id", req.UploadID),
					zap.String("event", "upload_progress"),
				)
			}
		}
		if err := json.NewEncoder(w).Encode(map[string]interface{}{"status": "chunk uploaded", "percent": percent}); err != nil {
			log.Error("Failed to write JSON response", zap.Error(err))
		}
	})

	mux.HandleFunc("/api/media/complete_upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			UploadID string `json:"upload_id"`
			UserID   string `json:"user_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, log, "invalid JSON", err, http.StatusBadRequest)
			return
		}
		var mediaSvc mediapb.MediaServiceClient
		if err := provider.Container().Resolve(&mediaSvc); err != nil {
			writeJSONError(w, log, "media service unavailable", err, http.StatusInternalServerError)
			return
		}
		grpcReq := &mediapb.CompleteMediaUploadRequest{
			UploadId: req.UploadID,
			UserId:   req.UserID,
		}
		_, err := mediaSvc.CompleteMediaUpload(r.Context(), grpcReq)
		if err != nil {
			writeJSONError(w, log, "failed to complete media upload", err, http.StatusInternalServerError)
			return
		}
		progressKey := "media:upload:progress:" + req.UploadID
		userID, err := redisClient.HGet(r.Context(), progressKey, "user_id").Result()
		if err != nil {
			writeJSONError(w, log, "failed to get user_id from Redis", err, http.StatusInternalServerError)
			return
		}
		log.Info("Upload complete", zap.String("upload_id", req.UploadID), zap.String("user_id", userID))
		if conn, ok := wsClients.Load(userID); ok {
			if err := conn.(*websocket.Conn).WriteJSON(map[string]interface{}{
				"type":      "media_event",
				"event":     "upload_complete",
				"upload_id": req.UploadID,
				"user_id":   userID,
				"status":    "ready",
			}); err != nil {
				log.Error("failed to write JSON to websocket (upload_complete)",
					zap.Error(err),
					zap.String("user_id", userID),
					zap.String("upload_id", req.UploadID),
					zap.String("event", "upload_complete"),
				)
			}
		}
		if err2 := json.NewEncoder(w).Encode(map[string]interface{}{"status": "upload complete"}); err2 != nil {
			log.Error("Failed to write JSON response", zap.Error(err2))
		}
	})

	mux.Handle("/media/hls/", http.StripPrefix("/media/hls/", http.FileServer(http.Dir("./media/hls/"))))
}
