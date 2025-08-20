package utils

import (
	"context"
	"time"
)

// DefaultTimeout is the default timeout for operations.
const DefaultTimeout = 30 * time.Second

// ContextUserIDKey is the key for the authenticated user ID in the context.
const ContextUserIDKey = "user_id"

// ContextRolesKey is the key for the authenticated user roles in the context.
const ContextRolesKey = "roles"

// GetAuthenticatedUserID retrieves the authenticated user ID from the context.
func GetAuthenticatedUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(ContextUserIDKey).(string)
	return userID, ok
}

// GetAuthenticatedUserRoles retrieves the authenticated user roles from the context.
func GetAuthenticatedUserRoles(ctx context.Context) ([]string, bool) {
	roles, ok := ctx.Value(ContextRolesKey).([]string)
	return roles, ok
}

// IsAdmin checks if the given roles include the "admin" role.
func IsAdmin(roles []string) bool {
	for _, r := range roles {
		if r == "admin" {
			return true
		}
	}
	return false
}

// IsServiceAdmin checks if the given roles include the global admin or a service-specific admin role.
func IsServiceAdmin(roles []string, service string) bool {
	adminRole := service + "_admin"
	for _, r := range roles {
		if r == "admin" || r == adminRole {
			return true
		}
	}
	return false
}

// GetContextFields extracts common fields from context for logging and error context.
func GetContextFields(ctx context.Context) map[string]interface{} {
	fields := make(map[string]interface{})
	if ctx == nil {
		return fields
	}
	if userID, ok := ctx.Value(ContextUserIDKey).(string); ok && userID != "" {
		fields["user_id"] = userID
	}
	if roles, ok := ctx.Value(ContextRolesKey).([]string); ok && len(roles) > 0 {
		fields["roles"] = roles
	}
	if reqID, ok := ctx.Value("request_id").(string); ok && reqID != "" {
		fields["request_id"] = reqID
	}
	if traceID, ok := ctx.Value("trace_id").(string); ok && traceID != "" {
		fields["trace_id"] = traceID
	}
	return fields
}

// GetStringFromContext retrieves a string value from the context by key, or returns an empty string if not found.
func GetStringFromContext(ctx context.Context, key string) string {
	if v, ok := ctx.Value(key).(string); ok {
		return v
	}
	return ""
}
