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

// contextKey is unexported to avoid collisions.
type contextKey struct{}

var authContextKey = &contextKey{}

// NewContext returns a new context with the given AuthContext.
func NewContext(ctx context.Context, auth *Context) context.Context {
	return context.WithValue(ctx, authContextKey, auth)
}

// FromContext extracts the AuthContext from the context, or returns a guest context if not present.
func FromContext(ctx context.Context) *Context {
	val := ctx.Value(authContextKey)
	if val == nil {
		return &Context{Roles: []string{"guest"}}
	}
	authCtx, ok := val.(*Context)
	if !ok {
		return &Context{Roles: []string{"guest"}}
	}
	return authCtx
}

// HasRole checks if the current user has the given role.
func HasRole(ctx context.Context, role string) bool {
	auth := FromContext(ctx)
	for _, r := range auth.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasScope checks if the current user has the given scope.
func HasScope(ctx context.Context, scope string) bool {
	auth := FromContext(ctx)
	for _, s := range auth.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}
