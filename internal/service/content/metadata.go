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

// [CANONICAL] All state hydration, analytics, and orchestration must use metadata.ExtractServiceVariables(meta, "content") and metadata.SetServiceSpecificField(meta, "content", key, value) directly.
// Do not add local wrappers for metadata extraction or mutation—use the canonical helpers from pkg/metadata.
// Only business-specific enrichment logic should remain here.

// ServiceMetadata defines the canonical, extensible metadata structure for content entities.
// This struct documents all fields expected under metadata.service_specific["content"] in the common.Metadata proto.
// Reference: docs/services/metadata.md, docs/amadeus/amadeus_context.md
// All extraction and mutation must use canonical helpers from pkg/metadata.
type ServiceMetadata struct {
	Accessibility map[string]interface{}       `json:"accessibility,omitempty"`
	Localization  map[string]interface{}       `json:"localization,omitempty"`
	Moderation    map[string]interface{}       `json:"moderation,omitempty"`
	AIEnrichment  map[string]interface{}       `json:"ai_enrichment,omitempty"`
	Audit         map[string]interface{}       `json:"audit,omitempty"`
	Compliance    map[string]interface{}       `json:"compliance,omitempty"`
	Translations  map[string]map[string]string `json:"translations,omitempty"`
	Versioning    map[string]interface{}       `json:"versioning,omitempty"`
	Custom        map[string]interface{}       `json:"custom,omitempty"`
	// Add other content-specific fields as needed
}
