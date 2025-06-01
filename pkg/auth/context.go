package auth

import (
	"context"
	"time"
)

type Context struct {
	UserID    string
	Roles     []string
	Scopes    []string
	Audience  string
	JWTID     string
	IssuedAt  time.Time
	ExpiresAt time.Time
	RawClaims map[string]interface{}
	Metadata  map[string]interface{} // For metadata-driven access
}

// HasRole checks if the current user has the given role.
func HasRole(auth *Context, role string) bool {
	if auth == nil {
		return false
	}
	for _, r := range auth.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasScope checks if the current user has the given scope.
func HasScope(auth *Context, scope string) bool {
	if auth == nil {
		return false
	}
	for _, s := range auth.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// NewContext returns a new context with the given AuthContext.
type contextKey struct{}

func NewContext(ctx context.Context, authCtx *Context) context.Context {
	return context.WithValue(ctx, contextKey{}, authCtx)
}
