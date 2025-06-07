// [CANONICAL] All metadata extraction and mutation in the messaging service must use pkg/metadata bridging helpers:
//   - StructToMap, MapToStruct, SetServiceSpecificField, ExtractServiceVariables
// Do not add local wrappers for metadata extraction or mutation—use the canonical helpers from pkg/metadata.
// Only business-specific enrichment logic (e.g., UpdateDeliveryStatus, ValidateMessagingMetadata) should remain here.
// Reference: docs/services/metadata.md, docs/amadeus/amadeus_context.md
//
// Messaging Metadata Helpers and Types
// -----------------------------------
//
// This file defines the canonical metadata structure and helpers for the Messaging service.
// It follows the robust metadata pattern: all extensible, service-specific, and audit fields
// are stored under common.Metadata, with a service_specific.messaging namespace.
//
// Key fields:
// - delivery: delivery/read/ack status, timestamps, per-user state
// - reactions: emoji, user, timestamp, audit
// - attachments: file info, compliance, audit
// - audit: created_by, last_modified_by, history
// - compliance: accessibility, moderation, legal
// - versioning: system/service/user version, environment, feature flags
// - service_specific.messaging: all messaging-specific extensions
//
// Usage:
//   meta := ExtractMessagingMetadata(msg.Metadata)
//   meta.Delivery.ReadBy = append(meta.Delivery.ReadBy, userID)
//   msg.Metadata = ToStruct(meta)
//
// See docs/amadeus/amadeus_context.md and api/protos/common/v1/metadata.proto for standards.

package messaging

import (
	"fmt"
	"time"
)

// MessagingMetadata is the canonical struct for messaging-specific metadata.
type Metadata struct {
	Delivery    *DeliveryMetadata      `json:"delivery,omitempty"`
	Reactions   []*ReactionMetadata    `json:"reactions,omitempty"`
	Attachments []*AttachmentMetadata  `json:"attachments,omitempty"`
	Audit       *AuditMetadata         `json:"audit,omitempty"`
	Compliance  *ComplianceMetadata    `json:"compliance,omitempty"`
	Versioning  *VersioningMetadata    `json:"versioning,omitempty"`
	Custom      map[string]interface{} `json:"custom,omitempty"`
}

type DeliveryMetadata struct {
	DeliveredBy []string          `json:"delivered_by,omitempty"`
	ReadBy      []string          `json:"read_by,omitempty"`
	AckBy       []string          `json:"ack_by,omitempty"`
	Timestamps  map[string]string `json:"timestamps,omitempty"` // user_id -> RFC3339
}

type ReactionMetadata struct {
	UserID    string         `json:"user_id"`
	Emoji     string         `json:"emoji"`
	ReactedAt string         `json:"reacted_at"`
	Audit     *AuditMetadata `json:"audit,omitempty"`
}

type AttachmentMetadata struct {
	ID         string              `json:"id"`
	Type       string              `json:"type"`
	Filename   string              `json:"filename"`
	Size       int64               `json:"size"`
	URL        string              `json:"url"`
	Compliance *ComplianceMetadata `json:"compliance,omitempty"`
	Audit      *AuditMetadata      `json:"audit,omitempty"`
}

type AuditMetadata struct {
	CreatedBy      string   `json:"created_by,omitempty"`
	LastModifiedBy string   `json:"last_modified_by,omitempty"`
	History        []string `json:"history,omitempty"`
}

type ComplianceMetadata struct {
	Accessibility map[string]interface{} `json:"accessibility,omitempty"`
	Moderation    map[string]interface{} `json:"moderation,omitempty"`
	Legal         map[string]interface{} `json:"legal,omitempty"`
}

type VersioningMetadata struct {
	SystemVersion  string   `json:"system_version,omitempty"`
	ServiceVersion string   `json:"service_version,omitempty"`
	UserVersion    string   `json:"user_version,omitempty"`
	Environment    string   `json:"environment,omitempty"`
	FeatureFlags   []string `json:"feature_flags,omitempty"`
	LastMigratedAt string   `json:"last_migrated_at,omitempty"`
}

// UpdateDeliveryStatus updates the delivery/read/ack status for a user.
func (mm *Metadata) UpdateDeliveryStatus(userID, status string) {
	if mm.Delivery == nil {
		mm.Delivery = &DeliveryMetadata{Timestamps: map[string]string{}}
	}
	ts := time.Now().UTC().Format(time.RFC3339)
	switch status {
	case "delivered":
		mm.Delivery.DeliveredBy = append(mm.Delivery.DeliveredBy, userID)
		mm.Delivery.Timestamps[userID+":delivered"] = ts
	case "read":
		mm.Delivery.ReadBy = append(mm.Delivery.ReadBy, userID)
		mm.Delivery.Timestamps[userID+":read"] = ts
	case "ack":
		mm.Delivery.AckBy = append(mm.Delivery.AckBy, userID)
		mm.Delivery.Timestamps[userID+":ack"] = ts
	}
}

// ValidateMessagingMetadata validates the structure and required fields.
func ValidateMessagingMetadata(mm *Metadata) error {
	// Example: ensure versioning is present
	if mm.Versioning == nil || mm.Versioning.SystemVersion == "" {
		return fmt.Errorf("missing versioning info in messaging metadata")
	}
	return nil
}
