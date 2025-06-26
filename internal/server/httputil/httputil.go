package httputil

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/nmxmxh/master-ovasabi/pkg/shield"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
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

// GRPCStatusToHTTPStatus converts a gRPC status code to an appropriate HTTP status code.
func GRPCStatusToHTTPStatus(code codes.Code) int {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.Canceled:
		return 499 // Client Closed Request
	case codes.Unknown:
		return http.StatusInternalServerError
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.FailedPrecondition:
		return http.StatusBadRequest
	case codes.Aborted:
		return http.StatusConflict
	case codes.OutOfRange:
		return http.StatusBadRequest
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Internal:
		return http.StatusInternalServerError
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DataLoss:
		return http.StatusInternalServerError
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

// Usage: import "github.com/nmxmxh/master-ovasabi/internal/server/httputil" and call these helpers in your handlers.

// HandleShieldError maps a shield error to an HTTP response.
func HandleShieldError(w http.ResponseWriter, log *zap.Logger, err error) {
	switch {
	case errors.Is(err, shield.ErrUnauthenticated):
		WriteJSONError(w, log, http.StatusUnauthorized, "unauthorized", err)
	case errors.Is(err, shield.ErrPermissionDenied):
		WriteJSONError(w, log, http.StatusForbidden, "forbidden", err)
	default:
		WriteJSONError(w, log, http.StatusInternalServerError, "internal server error", err)
	}
}

// HasRole checks if a user has any of the specified roles.
func HasRole(userRoles []string, requiredRoles ...string) bool {
	userRoleSet := make(map[string]struct{}, len(userRoles))
	for _, r := range userRoles {
		userRoleSet[r] = struct{}{}
	}
	for _, r := range requiredRoles {
		if _, ok := userRoleSet[r]; ok {
			return true
		}
	}
	return false
}

// IsAdmin checks if a user has the 'admin' role.
func IsAdmin(roles []string) bool {
	return HasRole(roles, "admin")
}
