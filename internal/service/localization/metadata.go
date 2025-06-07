// Metadata Standard Reference
// --------------------------
// All service-specific metadata must include the `versioning` field as described in:
//   - docs/services/versioning.md
//   - docs/amadeus/amadeus_context.md
// For all available metadata actions, patterns, and service-specific extensions, see:
//   - docs/services/metadata.md (general metadata documentation)
//   - docs/services/versioning.md (versioning/environment standard)
//
// This file implements localization service-specific metadata patterns. See above for required fields and integration points.
//
// Service-Specific Metadata Pattern for Localization Service
// --------------------------------------------------------
//
// This file defines the canonical Go struct for all localization service-specific metadata fields,
// covering all platform standards (locale, language, region, timezone, compliance, accessibility, etc.).
//
// Usage:
// - Use LocalizationServiceMetadata to read/update all service-specific metadata fields in Go.
// - Use the provided helpers to convert between LocalizationServiceMetadata and structpb.Struct.
// - This pattern ensures robust, type-safe, and future-proof handling of localization metadata.
//
// Reference: docs/amadeus/amadeus_context.md#cross-service-standards-integration-path

package localization

// ServiceMetadata holds all localization service-specific metadata fields.
// This struct documents all fields expected under metadata.service_specific["localization"] in the common.Metadata proto.
// Reference: docs/services/metadata.md, docs/amadeus/amadeus_context.md
// All extraction and mutation must use canonical helpers from pkg/metadata.
type ServiceMetadata struct {
	Locale                string                         `json:"locale,omitempty"`            // Target locale (e.g., en-US)
	Language              string                         `json:"language,omitempty"`          // Target language (e.g., en)
	Region                string                         `json:"region,omitempty"`            // Target region (e.g., US, FR)
	Timezone              string                         `json:"timezone,omitempty"`          // Target timezone (e.g., Europe/Paris)
	LastLocalizedAt       string                         `json:"last_localized_at,omitempty"` // Timestamp of last localization
	Compliance            *ComplianceMetadata            `json:"compliance,omitempty"`        // Accessibility/compliance info
	Accessibility         *AccessibilityMetadata         `json:"accessibility,omitempty"`     // Accessibility features/results
	TranslationProvenance *TranslationProvenanceMetadata `json:"translation_provenance,omitempty"`
	Versioning            *VersioningMetadata            `json:"versioning,omitempty"`
	Audit                 *AuditMetadata                 `json:"audit,omitempty"`
	// Add more as standards evolve
}

type ComplianceMetadata struct {
	Standards []ComplianceStandard `json:"standards,omitempty"`
	CheckedBy string               `json:"checked_by,omitempty"`
	CheckedAt string               `json:"checked_at,omitempty"`
	Method    string               `json:"method,omitempty"`
	Issues    []ComplianceIssue    `json:"issues_found,omitempty"`
}

type ComplianceStandard struct {
	Name      string `json:"name,omitempty"`
	Level     string `json:"level,omitempty"`
	Version   string `json:"version,omitempty"`
	Compliant bool   `json:"compliant,omitempty"`
}

type ComplianceIssue struct {
	Type     string `json:"type,omitempty"`
	Location string `json:"location,omitempty"`
	Resolved bool   `json:"resolved,omitempty"`
}

type AccessibilityMetadata struct {
	Features map[string]bool `json:"features,omitempty"`
	// Add more fields as needed (e.g., alt_text, captions, etc.)
}

// TranslationProvenanceMetadata describes how a translation was produced.
type TranslationProvenanceMetadata struct {
	Type           string  `json:"type,omitempty"`   // "machine" or "human"
	Engine         string  `json:"engine,omitempty"` // e.g., "google_translate_v3"
	TranslatorID   string  `json:"translator_id,omitempty"`
	TranslatorName string  `json:"translator_name,omitempty"`
	ReviewedBy     string  `json:"reviewed_by,omitempty"`
	QualityScore   float64 `json:"quality_score,omitempty"`
	Timestamp      string  `json:"timestamp,omitempty"`
}

type VersioningMetadata struct {
	SystemVersion  string `json:"system_version,omitempty"`
	ServiceVersion string `json:"service_version,omitempty"`
	Environment    string `json:"environment,omitempty"`
	LastMigratedAt string `json:"last_migrated_at,omitempty"`
}

type AuditMetadata struct {
	CreatedBy      string   `json:"created_by,omitempty"`
	LastModifiedBy string   `json:"last_modified_by,omitempty"`
	History        []string `json:"history,omitempty"`
}

// [CANONICAL] All metadata must be normalized and calculated via metadata.NormalizeAndCalculate before persistence or emission.
// Ensure required fields (versioning, audit, etc.) are present under the correct namespace.
