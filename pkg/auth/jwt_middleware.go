// JWT middleware for net/http. Depends on context.go in the same package.
package auth

import (
	"net/http"
	"strings"
)

// extractBearerToken extracts the token from the Authorization header.
func extractBearerToken(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

// JWTMiddleware is a minimal HTTP middleware for JWT auth.
func JWTMiddleware(secret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr := extractBearerToken(r.Header.Get("Authorization"))
		if tokenStr == "" {
			ctx := NewContext(r.Context(), &Context{Roles: []string{"guest"}})
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		authCtx, err := ParseAndExtractAuthContext(tokenStr, secret)
		if err != nil {
			ctx := NewContext(r.Context(), &Context{Roles: []string{"guest"}})
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		ctx := NewContext(r.Context(), authCtx)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
