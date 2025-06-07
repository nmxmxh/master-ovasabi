// Package nexus provides helpers for robust, extensible metadata handling in the Nexus service.
// This file defines the canonical metadata structure, helpers for extraction/validation.
// and query utilities for rich analytics and orchestration.
package nexus

import (
	"encoding/json"
	"fmt"
)

// NexusMetadata is the canonical metadata struct for patterns, orchestrations, and mining.
type Metadata struct {
	Tags            []string               `json:"tags,omitempty"`
	ServiceSpecific map[string]interface{} `json:"service_specific,omitempty"`
	Audit           map[string]interface{} `json:"audit,omitempty"`
	KnowledgeGraph  map[string]interface{} `json:"knowledge_graph,omitempty"`
	CustomRules     map[string]interface{} `json:"custom_rules,omitempty"`
	Features        []string               `json:"features,omitempty"`
	Scheduling      map[string]interface{} `json:"scheduling,omitempty"`
	// Add more fields as needed for extensibility
}

// ParseNexusMetadata parses a JSONB blob into NexusMetadata.
func ParseNexusMetadata(data []byte) (*Metadata, error) {
	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal NexusMetadata: %w", err)
	}
	return &meta, nil
}

// ExtractTagFilter builds a SQL filter for tags.
func ExtractTagFilter(tags []string) (filter string, args []interface{}) {
	if len(tags) == 0 {
		return "", nil
	}
	// Example: WHERE metadata->'tags' ?| array['tag1','tag2']
	args = make([]interface{}, len(tags))
	for i, tag := range tags {
		args[i] = tag
	}
	placeholders := fmt.Sprintf("array[%s]", joinPlaceholders(len(tags)))
	filter = fmt.Sprintf("metadata->'tags' ?| %s", placeholders)
	return filter, args
}

// joinPlaceholders returns a comma-separated list of $1, $2, ... for SQL arrays.
func joinPlaceholders(n int) string {
	out := ""
	for i := 1; i <= n; i++ {
		if i > 1 {
			out += ","
		}
		out += fmt.Sprintf("$%d", i)
	}
	return out
}

// ValidateNexusMetadata checks for required fields and returns an error if missing.
func ValidateNexusMetadata(meta *Metadata) error {
	// Example: require at least one tag and service_specific field
	if len(meta.Tags) == 0 {
		return fmt.Errorf("at least one tag is required")
	}
	if len(meta.ServiceSpecific) == 0 {
		return fmt.Errorf("service_specific metadata is required")
	}
	return nil
}

// ComposeNexusMetadata builds a NexusMetadata struct from components.
func ComposeNexusMetadata(tags []string, serviceSpecific, audit, kg, customRules, scheduling map[string]interface{}, features []string) *Metadata {
	return &Metadata{
		Tags:            tags,
		ServiceSpecific: serviceSpecific,
		Audit:           audit,
		KnowledgeGraph:  kg,
		CustomRules:     customRules,
		Features:        features,
		Scheduling:      scheduling,
	}
}

// Example usage for onboarding and documentation:
// meta := ComposeNexusMetadata(
//   []string{"ai", "orchestration"},
//   map[string]interface{}{ "nexus": map[string]interface{}{ "pattern": "example" } },
//   map[string]interface{}{ "created_by": "admin" },
//   nil, nil, nil, nil,
// )
// if err := ValidateNexusMetadata(meta); err != nil {
//   // handle error
// }
// data, _ := json.Marshal(meta)
// _ = ParseNexusMetadata(data)

// Extend this file with more helpers as new metadata fields and query patterns emerge.

// [CANONICAL] All metadata must be normalized and calculated via metadata.NormalizeAndCalculate before persistence or emission.
// Ensure required fields (versioning, audit, etc.) are present under the correct namespace.

// NormalizeAndCalculate ensures required fields (versioning, audit, etc.) are present under the correct namespace.
func NormalizeAndCalculate(meta *Metadata) *Metadata {
	if meta == nil {
		meta = &Metadata{}
	}
	if meta.ServiceSpecific == nil {
		meta.ServiceSpecific = make(map[string]interface{})
	}
	// Ensure versioning field exists under service_specific.nexus.versioning
	if _, ok := meta.ServiceSpecific["nexus"]; !ok {
		meta.ServiceSpecific["nexus"] = make(map[string]interface{})
	}
	nexusMap, ok := meta.ServiceSpecific["nexus"].(map[string]interface{})
	if !ok {
		// If it's not a map, replace it
		nexusMap = make(map[string]interface{})
		meta.ServiceSpecific["nexus"] = nexusMap
	}
	if _, ok := nexusMap["versioning"]; !ok {
		nexusMap["versioning"] = map[string]interface{}{
			"system_version":   "2025-06-01", // update as needed
			"service_version":  "1.0.0",      // update as needed
			"environment":      "production", // or from env/config
			"last_migrated_at": "2025-06-01T00:00:00Z",
		}
	}
	// Optionally, ensure audit field exists
	if meta.Audit == nil {
		meta.Audit = map[string]interface{}{
			"created_at": "2025-06-01T00:00:00Z", // or time.Now().UTC().Format(time.RFC3339)
		}
	}
	return meta
}
