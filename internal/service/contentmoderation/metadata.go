package contentmoderation

// Metadata defines the canonical, extensible metadata structure for content moderation entities.
// This struct documents all fields expected under metadata.service_specific["contentmoderation"] in the common.Metadata proto.
// Reference: docs/services/metadata.md, docs/amadeus/amadeus_context.md
// All extraction and mutation must use canonical helpers from pkg/metadata.
type Metadata struct {
	Versioning     map[string]interface{} `json:"versioning,omitempty"`
	FlaggedSignals map[string]float64     `json:"flagged_signals,omitempty"`
	Reviewer       *ReviewerMetadata      `json:"reviewer,omitempty"`
	Audit          map[string]interface{} `json:"audit,omitempty"`
	Compliance     map[string]interface{} `json:"compliance,omitempty"`
	Notes          string                 `json:"notes,omitempty"`
	// Add other content moderation-specific fields as needed
}

// ReviewerMetadata documents reviewer information for moderation actions.
type ReviewerMetadata struct {
	ReviewerID   string `json:"reviewer_id,omitempty"`
	ReviewerName string `json:"reviewer_name,omitempty"`
	ReviewedAt   string `json:"reviewed_at,omitempty"`
}

// [CANONICAL] All state hydration, analytics, and orchestration must use metadata.ExtractServiceVariables(meta, "contentmoderation") and metadata.SetServiceSpecificField(meta, "contentmoderation", key, value) directly.
// Do not add local wrappers for metadata extraction or mutation—use the canonical helper from pkg/metadata.
// Only business-specific enrichment logic should remain here.
