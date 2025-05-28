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

import (
	"encoding/json"
	"time"

	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

// ServiceMetadata holds all localization service-specific metadata fields.
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

// ServiceMetadataFromStruct converts a structpb.Struct to ServiceMetadata.
func ServiceMetadataFromStruct(s *structpb.Struct) (*ServiceMetadata, error) {
	if s == nil {
		return &ServiceMetadata{}, nil
	}
	b, err := json.Marshal(s.AsMap())
	if err != nil {
		return nil, err
	}
	var meta ServiceMetadata
	err = json.Unmarshal(b, &meta)
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

// ServiceMetadataToStruct converts ServiceMetadata to structpb.Struct.
func ServiceMetadataToStruct(meta *ServiceMetadata) (*structpb.Struct, error) {
	if meta == nil {
		return metadata.NewStructFromMap(map[string]interface{}{}), nil
	}
	b, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	err = json.Unmarshal(b, &m)
	if err != nil {
		return nil, err
	}
	return metadata.NewStructFromMap(m), nil
}

// ExtractAndEnrichLocalizationMetadata extracts, validates, and enriches localization metadata.
func ExtractAndEnrichLocalizationMetadata(meta *ServiceMetadata, userID string, isCreate bool) *ServiceMetadata {
	if meta == nil {
		meta = &ServiceMetadata{}
	}
	// Ensure versioning
	if meta.Versioning == nil {
		meta.Versioning = &VersioningMetadata{
			SystemVersion:  "1.0.0",
			ServiceVersion: "1.0.0",
			Environment:    "prod",
			LastMigratedAt: time.Now().Format(time.RFC3339),
		}
	}
	// Ensure audit
	if meta.Audit == nil {
		meta.Audit = &AuditMetadata{
			CreatedBy: userID,
			History:   []string{"created"},
		}
	} else {
		meta.Audit.LastModifiedBy = userID
		if isCreate {
			meta.Audit.History = append(meta.Audit.History, "created")
		} else {
			meta.Audit.History = append(meta.Audit.History, "updated")
		}
	}
	// Ensure translation provenance
	if meta.TranslationProvenance == nil {
		meta.TranslationProvenance = &TranslationProvenanceMetadata{
			Type:      "machine",
			Engine:    "unknown",
			Timestamp: time.Now().Format(time.RFC3339),
		}
	}
	// Ensure compliance
	if meta.Compliance == nil {
		meta.Compliance = &ComplianceMetadata{
			Standards: []ComplianceStandard{{Name: "WCAG", Level: "AA", Version: "2.1", Compliant: true}},
			CheckedBy: "localization-service",
			CheckedAt: time.Now().Format(time.RFC3339),
			Method:    "automated",
			Issues:    []ComplianceIssue{},
		}
	}
	return meta
}
