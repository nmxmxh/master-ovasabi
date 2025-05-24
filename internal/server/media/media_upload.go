package media

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/nmxmxh/master-ovasabi/internal/service"
	"go.uber.org/zap"
)

// UploadOpsHandler handles media upload actions via the "action" field.
func UploadOpsHandler(log *zap.Logger, provider *service.Provider, _ *sync.Map) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		switch action {
		case "start_upload":
			log.Info("Start upload", zap.Any("provider", provider))
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{"status":"start_upload stub"}`)); err != nil {
				log.Error("Failed to write start_upload response", zap.Error(err))
			}
		case "upload_chunk":
			log.Info("Upload chunk", zap.Any("provider", provider))
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{"status":"upload_chunk stub"}`)); err != nil {
				log.Error("Failed to write upload_chunk response", zap.Error(err))
			}
		case "complete_upload":
			log.Info("Complete upload", zap.Any("provider", provider))
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{"status":"complete_upload stub"}`)); err != nil {
				log.Error("Failed to write complete_upload response", zap.Error(err))
			}
		default:
			log.Error("Unknown action in media upload ops", zap.String("action", action))
			http.Error(w, "unknown action", http.StatusBadRequest)
		}
	}
}
