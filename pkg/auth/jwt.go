package auth

import (
	"encoding/json"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ParseAndExtractAuthContext parses a JWT and returns an AuthContext (or error).
func ParseAndExtractAuthContext(tokenStr, secret string) (*Context, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(_ *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	authCtx := &Context{
		UserID:    toString(claims["sub"]),
		Roles:     toStringSlice(claims["roles"]),
		Scopes:    toStringSlice(claims["scopes"]),
		Audience:  toString(claims["aud"]),
		JWTID:     toString(claims["jti"]),
		IssuedAt:  toTime(claims["iat"]),
		ExpiresAt: toTime(claims["exp"]),
		RawClaims: claims,
	}
	if meta, ok := claims["metadata"].(map[string]interface{}); ok {
		authCtx.Metadata = meta
	}
	return authCtx, nil
}

// Helper to convert interface{} to string.
func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// Helper to convert interface{} to []string.
func toStringSlice(v interface{}) []string {
	if v == nil {
		return nil
	}
	if arr, ok := v.([]interface{}); ok {
		res := make([]string, 0, len(arr))
		for _, item := range arr {
			if s, ok := item.(string); ok {
				res = append(res, s)
			}
		}
		return res
	}
	if arr, ok := v.([]string); ok {
		return arr
	}
	return nil
}

// Helper to convert JWT numeric date to time.Time.
func toTime(v interface{}) time.Time {
	if v == nil {
		return time.Time{}
	}
	switch t := v.(type) {
	case float64:
		return time.Unix(int64(t), 0)
	case int64:
		return time.Unix(t, 0)
	case json.Number:
		if i, err := t.Int64(); err == nil {
			return time.Unix(i, 0)
		}
	}
	return time.Time{}
}
