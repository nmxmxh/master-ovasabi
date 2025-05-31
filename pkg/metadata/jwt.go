package metadata

import (
	"time"
)

// UpdateJWTMetadata updates the jwt section of a user metadata map.
func UpdateJWTMetadata(serviceMeta, jwtFields map[string]interface{}) map[string]interface{} {
	jwtMetaIface, ok := serviceMeta["jwt"]
	var jwtMeta map[string]interface{}
	if ok {
		jwtMeta, ok = jwtMetaIface.(map[string]interface{})
		if !ok {
			jwtMeta = map[string]interface{}{}
		}
	} else {
		jwtMeta = map[string]interface{}{}
	}
	for k, v := range jwtFields {
		jwtMeta[k] = v
	}
	serviceMeta["jwt"] = jwtMeta
	return serviceMeta
}

// UpdateJWTIssueMetadata is a convenience for common JWT issuance fields.
func UpdateJWTIssueMetadata(serviceMeta map[string]interface{}, jwtID, audience string, scopes []string) map[string]interface{} {
	return UpdateJWTMetadata(serviceMeta, map[string]interface{}{
		"last_jwt_issued_at": time.Now().Format(time.RFC3339),
		"last_jwt_id":        jwtID,
		"jwt_audience":       audience,
		"jwt_scopes":         scopes,
	})
}
