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

// ContextWithTimeout creates a context with the default timeout.
func ContextWithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, DefaultTimeout)
}

// ContextWithCustomTimeout creates a context with a custom timeout.
func ContextWithCustomTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}

// ContextWithDeadline creates a context with a deadline.
func ContextWithDeadline(ctx context.Context, deadline time.Time) (context.Context, context.CancelFunc) {
	return context.WithDeadline(ctx, deadline)
}

// MergeContexts creates a new context that is canceled when any of the input contexts are canceled.
func MergeContexts(contexts ...context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		defer cancel()

		done := make(chan struct{})
		for _, c := range contexts {
			go func(c context.Context) {
				ctx, cancel := context.WithCancel(c)
				defer cancel()
				select {
				case <-c.Done():
					close(done)
				case <-ctx.Done():
				}
			}(c)
		}

		<-done
	}()

	return ctx, cancel
}

// WithValue adds a value to the context with type safety.
func WithValue[T any](ctx context.Context, key interface{}, value T) context.Context {
	return context.WithValue(ctx, key, value)
}

// GetValue retrieves a value from the context with type safety.
func GetValue[T any](ctx context.Context, key interface{}) (T, bool) {
	value := ctx.Value(key)
	if value == nil {
		var zero T
		return zero, false
	}

	typed, ok := value.(T)
	if !ok {
		var zero T
		return zero, false
	}

	return typed, true
}

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
