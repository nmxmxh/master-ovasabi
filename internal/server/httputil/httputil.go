package httputil

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

// WriteJSONError writes a JSON error response and logs the error.
func WriteJSONError(w http.ResponseWriter, log *zap.Logger, status int, msg string, err error, contextFields ...zap.Field) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err != nil {
		log.Error(msg, append(contextFields, zap.Error(err))...)
	} else {
		log.Error(msg, contextFields...)
	}
	errMsg := msg
	details := ""
	if err != nil {
		details = err.Error()
	}
	err = json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   errMsg,
		"details": details,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if _, writeErr := w.Write([]byte("internal server error")); writeErr != nil {
			log.Error("Failed to write error response", zap.Error(writeErr))
		}
	}
}

// WriteJSONResponse writes a JSON response and logs on error.
func WriteJSONResponse(w http.ResponseWriter, log *zap.Logger, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Error("Failed to write JSON response", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

// Usage: import "github.com/nmxmxh/master-ovasabi/internal/server/httputil" and call these helpers in your handlers.
