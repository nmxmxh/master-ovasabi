package search

import (
	"fmt"
	"strings"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
)

// ComposeMetadataFilter builds a SQL filter string and args for service-specific metadata fields.
// Compatible with multi-entity search: use in conjunction with the 'types' field to filter per-entity.
// It returns a SQL fragment (e.g., " AND metadata->'service_specific'->>'foo' = ?") and the corresponding args.
// Extend as needed for entity-type-specific metadata logic.
func ComposeMetadataFilter(metadata *commonpb.Metadata) (filter string, args []interface{}) {
	if metadata == nil || metadata.ServiceSpecific == nil || len(metadata.ServiceSpecific.Fields) == 0 {
		return "", nil
	}
	filters := []string{}
	args = []interface{}{}
	for k, v := range metadata.ServiceSpecific.Fields {
		filters = append(filters, fmt.Sprintf("metadata->'service_specific'->>'%s' = ?", k))
		args = append(args, v.GetStringValue())
	}
	if len(filters) == 0 {
		return "", nil
	}
	return " AND " + strings.Join(filters, " AND "), args
}

// ExtractServiceSpecific returns a map of service-specific metadata fields for a given service namespace.
func ExtractServiceSpecific(metadata *commonpb.Metadata, service string) map[string]string {
	if metadata == nil || metadata.ServiceSpecific == nil {
		return nil
	}
	fields := map[string]string{}
	if v, ok := metadata.ServiceSpecific.Fields[service]; ok && v != nil {
		if structVal, ok := v.GetStructValue().Fields[service]; ok && structVal != nil {
			for k, val := range structVal.GetStructValue().Fields {
				fields[k] = val.GetStringValue()
			}
		}
	}
	return fields
}

// ValidateMetadataKeys checks that only allowed keys are present in service-specific metadata.
func ValidateMetadataKeys(metadata *commonpb.Metadata, allowedKeys []string) error {
	if metadata == nil || metadata.ServiceSpecific == nil {
		return nil
	}
	allowed := map[string]struct{}{}
	for _, k := range allowedKeys {
		allowed[k] = struct{}{}
	}
	for k := range metadata.ServiceSpecific.Fields {
		if _, ok := allowed[k]; !ok {
			return fmt.Errorf("unexpected metadata key: %s", k)
		}
	}
	return nil
}

// Document: This file provides robust helpers for extracting, validating, and composing metadata filters
// for the search service, following the platform's extensible metadata pattern. Extend as needed for
// analytics, audit, and advanced orchestration.
