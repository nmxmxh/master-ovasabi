// Metadata Standard Reference
// --------------------------
// All service-specific metadata must include the `versioning` field as described in:
//   - docs/services/versioning.md
//   - docs/amadeus/amadeus_context.md
// For all available metadata actions, patterns, and service-specific extensions, see:
//   - docs/services/metadata.md (general metadata documentation)
//   - docs/services/versioning.md (versioning/environment standard)
//
// This file implements security service-specific metadata patterns. See above for required fields and integration points.
//
// Service-Specific Metadata Pattern for Security Service
// -----------------------------------------------------
//
// This file defines the canonical Go struct for all security service-specific metadata fields,
// covering all platform standards (risk scoring, audit, compliance, bad actor, escalation, etc.).
//
// Usage:
// - Use ServiceMetadata to read/update all service-specific metadata fields in Go.
// - Use the provided helpers to convert between ServiceMetadata and structpb.Struct.
// - This pattern ensures robust, type-safe, and future-proof handling of security metadata.
//
// Reference: docs/amadeus/amadeus_context.md#cross-service-standards-integration-path
//           (see also: bad actor, compliance, audit, and cross-service calculation patterns)

package security

// ServiceMetadata holds all security service-specific metadata fields.
type ServiceMetadata struct {
	RiskScore       float64             `json:"risk_score,omitempty"`        // Calculated risk score (0-1)
	RiskFactors     []string            `json:"risk_factors,omitempty"`      // List of risk factors (e.g., device, location, behavior)
	LastAudit       string              `json:"last_audit,omitempty"`        // Timestamp of last audit
	AuditHistory    []AuditEntry        `json:"audit_history,omitempty"`     // List of audit entries
	Compliance      *ComplianceMetadata `json:"compliance,omitempty"`        // Compliance info (e.g., SOC2, GDPR)
	BadActor        *BadActorMetadata   `json:"bad_actor,omitempty"`         // Bad actor signals (cross-service)
	LinkedAccounts  []string            `json:"linked_accounts,omitempty"`   // User/content IDs linked by device/location/behavior
	DeviceIDs       []string            `json:"device_ids,omitempty"`        // Devices associated with this entity
	Locations       []LocationMetadata  `json:"locations,omitempty"`         // Locations associated with this entity
	EscalationLevel string              `json:"escalation_level,omitempty"`  // Current escalation level (e.g., info, warn, block, review)
	LastEscalatedAt string              `json:"last_escalated_at,omitempty"` // Timestamp of last escalation
	// Cross-service references for calculation/graphing
	UserID         string `json:"user_id,omitempty"`         // Reference to user
	ContentID      string `json:"content_id,omitempty"`      // Reference to content
	LocalizationID string `json:"localization_id,omitempty"` // Reference to localization/locale
	// Add more as standards and knowledge graph evolve
}

type AuditEntry struct {
	Timestamp string `json:"timestamp,omitempty"`
	Action    string `json:"action,omitempty"`
	Actor     string `json:"actor,omitempty"`
	Result    string `json:"result,omitempty"`
	Details   string `json:"details,omitempty"`
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

type BadActorMetadata struct {
	Score          float64            `json:"score,omitempty"`
	Reason         string             `json:"reason,omitempty"`
	DeviceIDs      []string           `json:"device_ids,omitempty"`
	Locations      []LocationMetadata `json:"locations,omitempty"`
	Frequency      *FrequencyMetadata `json:"frequency,omitempty"`
	AccountsLinked []string           `json:"accounts_linked,omitempty"`
	LastFlaggedAt  string             `json:"last_flagged_at,omitempty"`
	History        []EventMetadata    `json:"history,omitempty"`
}

type FrequencyMetadata struct {
	Window string `json:"window,omitempty"`
	Count  int    `json:"count,omitempty"`
}

type EventMetadata struct {
	Event     string `json:"event,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

type LocationMetadata struct {
	IP      string `json:"ip,omitempty"`
	City    string `json:"city,omitempty"`
	Country string `json:"country,omitempty"`
}
