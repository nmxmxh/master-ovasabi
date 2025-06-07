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
	"fmt"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"go.uber.org/zap"
)

// BuildAnalyticsMetadata builds robust, GDPR-compliant analytics event metadata.
// User information (userID, userEmail) is always obscured unless gdprObscure is explicitly false.
// If user info is provided but gdprObscure is true, a warning is logged using the provided logger.
// [CANONICAL] All metadata must be normalized and calculated via metadata.NormalizeAndCalculate before persistence or emission.
// Ensure required fields (versioning, audit, etc.) are present under the correct namespace.
func BuildAnalyticsMetadata(
	eventType string,
	userID string,
	userEmail string,
	properties map[string]interface{},
	groups map[string]interface{},
	context map[string]interface{},
	gdprObscure bool,
	serviceSpecific map[string]interface{},
	log *zap.Logger,
) (*commonpb.Metadata, error) {
	analyticsMap := map[string]interface{}{
		"event_type": eventType,
		"properties": properties,
		"groups":     groups,
		"context":    context,
	}
	// Privacy-first: always obscure user info unless explicitly requested
	if !gdprObscure {
		analyticsMap["user_id"] = userID
		analyticsMap["user_email"] = userEmail
	} else {
		analyticsMap["user_id"] = "obscured"
		analyticsMap["user_email"] = "obscured"
		// Log a warning if real user info is provided but is being obscured
		if (userID != "" || userEmail != "") && log != nil {
			log.Warn("[Analytics] User info provided but obscured",
				zap.String("user_id", userID),
				zap.String("user_email", userEmail),
			)
		}
	}
	for k, v := range serviceSpecific {
		analyticsMap[k] = v
	}
	// Always require versioning for compliance
	if _, ok := analyticsMap["versioning"]; !ok {
		analyticsMap["versioning"] = map[string]interface{}{"system_version": "1.0.0"}
	}
	ss := map[string]interface{}{"analytics": analyticsMap}
	ssStruct := metadata.NewStructFromMap(ss, log)
	if ssStruct == nil {
		return nil, fmt.Errorf("failed to create service specific struct")
	}
	meta := &commonpb.Metadata{
		ServiceSpecific: ssStruct,
	}
	metaMap := metadata.ProtoToMap(meta)
	if metaMap == nil {
		return nil, fmt.Errorf("failed to convert metadata to map")
	}
	normMap := metadata.Handler{}.NormalizeAndCalculate(metaMap, "", "", []string{}, "success", "enrich analytics metadata")
	if normMap == nil {
		return nil, fmt.Errorf("failed to normalize and calculate metadata")
	}
	proto := metadata.MapToProto(normMap)
	if proto == nil {
		return nil, fmt.Errorf("failed to convert normalized map to proto")
	}
	return proto, nil
}
