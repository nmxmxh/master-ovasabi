// Analytics Metadata Builder (GDPR-Compliant, Extensible)
// ------------------------------------------------------
// This file implements the canonical builder for analytics event metadata.
// - Follows the robust, versioned, and namespaced metadata pattern.
// - Supports GDPR compliance: user/sensitive info can be obscured or omitted.
// - Accepts event type, user info (optionally obscured), properties, groups, and context fields.
// - Always includes a versioning field for compliance and audit.
//
// Usage: Use this builder for all analytics event creation, enrichment, and storage.
//
// For more, see docs/services/metadata.md and docs/amadeus/amadeus_context.md.

package analytics

import (
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
)

// BuildAnalyticsMetadata builds robust, GDPR-compliant analytics event metadata.
// If gdprObscure is true, user/sensitive info is omitted or obscured.
func BuildAnalyticsMetadata(
	eventType string,
	userID string,
	userEmail string,
	properties map[string]interface{},
	groups map[string]interface{},
	context map[string]interface{},
	gdprObscure bool,
	serviceSpecific map[string]interface{},
) (*commonpb.Metadata, error) {
	analyticsMap := map[string]interface{}{
		"event_type": eventType,
		"properties": properties,
		"groups":     groups,
		"context":    context,
	}
	if !gdprObscure {
		analyticsMap["user_id"] = userID
		analyticsMap["user_email"] = userEmail
	} else {
		analyticsMap["user_id"] = "obscured"
		analyticsMap["user_email"] = "obscured"
	}
	for k, v := range serviceSpecific {
		analyticsMap[k] = v
	}
	// Always require versioning for compliance
	if _, ok := analyticsMap["versioning"]; !ok {
		analyticsMap["versioning"] = map[string]interface{}{"system_version": "1.0.0"}
	}
	ss := map[string]interface{}{"analytics": analyticsMap}
	ssStruct := metadata.NewStructFromMap(ss, nil)
	return &commonpb.Metadata{
		ServiceSpecific: ssStruct,
	}, nil
}
