// Content Metadata Builder (Service-Specific Standard)
// ---------------------------------------------------
// This file implements the canonical builder for content metadata.
// All service-specific metadata builders (e.g., BuildContentMetadata) must:
//   - Be implemented in their respective service packages (not in pkg/metadata)
//   - Follow the extensible, versioned, and namespaced pattern
//   - Be referenced in docs/services/metadata.md and onboarding docs
//
// This is the standard for all service-specific metadata in the OVASABI platform.

package content

import (
	"fmt"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// BuildContentMetadata builds a canonical content metadata struct for storage, analytics, and extensibility.
func BuildContentMetadata(
	accessibility, localization, moderation, aiEnrichment, audit, compliance map[string]interface{},
	tags []string,
	serviceSpecific map[string]interface{},
) (*commonpb.Metadata, error) {
	contentMap := map[string]interface{}{}
	if accessibility != nil {
		contentMap["accessibility"] = accessibility
	}
	if localization != nil {
		contentMap["localization"] = localization
	}
	if moderation != nil {
		contentMap["moderation"] = moderation
	}
	if aiEnrichment != nil {
		contentMap["ai_enrichment"] = aiEnrichment
	}
	if audit != nil {
		contentMap["audit"] = audit
	}
	if compliance != nil {
		contentMap["compliance"] = compliance
	}
	for k, v := range serviceSpecific {
		contentMap[k] = v
	}
	// Always require versioning for compliance
	if _, ok := contentMap["versioning"]; !ok {
		contentMap["versioning"] = map[string]interface{}{"system_version": "1.0.0"}
	}
	ss := map[string]interface{}{"content": contentMap}
	ssStruct, err := structpb.NewStruct(ss)
	if err != nil {
		return nil, fmt.Errorf("failed to build service_specific struct: %w", err)
	}
	return &commonpb.Metadata{
		ServiceSpecific: ssStruct,
		Tags:            tags,
	}, nil
}
