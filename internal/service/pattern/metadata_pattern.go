// Metadata Standard Reference
// --------------------------
// This file now contains only pure metadata helpers (merge, migrate, validate, etc.).
// All orchestration logic (caching, knowledge graph, scheduler, event, nexus) must be handled via hooks in the graceful orchestration config.
//
// For orchestration, see pkg/graceful/success.go.

package pattern

import (
	"fmt"
	"strings"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// --- Pure metadata helpers go here ---
// (Implement or import merge, migrate, validate, etc. as needed)

// NormalizeMetadata ensures required fields, applies defaults, and strips hydration-only fields.
// If partialUpdate is true, only updates provided fields (for PATCH/partial update semantics).
func NormalizeMetadata(meta *commonpb.Metadata, service string, partialUpdate bool) (*commonpb.Metadata, error) {
	if meta == nil {
		return &commonpb.Metadata{}, nil
	}
	ss := meta.ServiceSpecific
	if ss == nil {
		return meta, nil
	}
	fields := ss.AsMap()
	serviceFields, ok := fields[service]
	if !ok {
		return meta, nil
	}
	serviceMap, ok := serviceFields.(map[string]interface{})
	if !ok {
		return meta, nil
	}

	if !partialUpdate {
		// Apply defaults for missing fields (stub)
		if _, ok := serviceMap["versioning"]; !ok {
			serviceMap["versioning"] = map[string]interface{}{"system_version": "1.0.0"}
		}
		// Validate required fields (stub)
		if _, ok := serviceMap["versioning"]; !ok {
			return nil, fmt.Errorf("missing required field: versioning")
		}
	} else {
		// --- PRODUCTION-GRADE PARTIAL UPDATE LOGIC ---
		// 1. Load existing metadata for the service (from DB or previous state)
		//    For this function, assume caller provides the full metadata struct (meta),
		//    and the fields map contains both existing and new fields.
		// 2. Merge: For each field in the incoming (partial) update, update the existing map.
		//    (Assume serviceMap contains only the fields to update.)
		//    We'll merge serviceMap into the existing fields.
		//    If a field is not present in serviceMap, preserve the existing value.
		//    If a field is present, update it.
		//    (If you have a blueprint/schema, you can enforce types here.)
		//
		// For now, we assume fields[service] is the existing map, and serviceMap is the partial update.
		// We'll merge serviceMap into fields[service].
		//
		// Note: In a real implementation, you may want to deep copy or use a helper for deep merge.
		//
		// Example merge logic:
		if existingFields, ok := fields[service].(map[string]interface{}); ok {
			for k, v := range serviceMap {
				existingFields[k] = v
			}
			// Optionally, apply defaults for required fields if missing
			if _, ok := existingFields["versioning"]; !ok {
				existingFields["versioning"] = map[string]interface{}{"system_version": "1.0.0"}
			}
			fields[service] = existingFields
		}
	}
	// Remove hydration-only fields (stub: e.g., computed fields)
	for k := range serviceMap {
		if strings.HasPrefix(k, "_hydrated_") {
			delete(serviceMap, k)
		}
	}
	fields[service] = serviceMap
	newSS, err := structpb.NewStruct(fields)
	if err != nil {
		return nil, err
	}
	meta.ServiceSpecific = newSS
	return meta, nil
}

// DenormalizeMetadata hydrates metadata for API/gRPC/UI responses.
// Optionally expands references, adds computed fields, etc.
func DenormalizeMetadata(meta *commonpb.Metadata, service string) (*commonpb.Metadata, error) {
	if meta == nil {
		return &commonpb.Metadata{}, nil
	}
	ss := meta.ServiceSpecific
	if ss == nil {
		return meta, nil
	}
	fields := ss.AsMap()
	serviceFields, ok := fields[service]
	if !ok {
		return meta, nil
	}
	serviceMap, ok := serviceFields.(map[string]interface{})
	if !ok {
		return meta, nil
	}
	// Example: Add computed/hydrated fields (stub)
	serviceMap["_hydrated_display_name"] = fmt.Sprintf("%s-%s", service, serviceMap["versioning"])
	fields[service] = serviceMap
	newSS, err := structpb.NewStruct(fields)
	if err != nil {
		return nil, err
	}
	meta.ServiceSpecific = newSS
	return meta, nil
}

// MergeMetadataFields merges fields from src into dst for partial updates.
func MergeMetadataFields(dst, src *commonpb.Metadata, service string) (*commonpb.Metadata, error) {
	if dst == nil {
		dst = &commonpb.Metadata{}
	}
	if src == nil {
		return dst, nil
	}
	if dst.ServiceSpecific == nil {
		dst.ServiceSpecific = &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	if src.ServiceSpecific == nil {
		return dst, nil
	}
	dstMap := dst.ServiceSpecific.AsMap()
	srcMap := src.ServiceSpecific.AsMap()
	dstSvcIface, ok := dstMap[service]
	if !ok {
		// handle missing service key
		return dst, nil
	}
	dstSvc, ok := dstSvcIface.(map[string]interface{})
	if !ok {
		// handle type assertion failure
		return dst, nil
	}
	srcSvcIface, ok := srcMap[service]
	if !ok {
		// handle missing service key
		return dst, nil
	}
	srcSvc, ok := srcSvcIface.(map[string]interface{})
	if !ok {
		// handle type assertion failure
		return dst, nil
	}
	if dstSvc == nil {
		dstSvc = map[string]interface{}{}
	}
	for k, v := range srcSvc {
		dstSvc[k] = v
	}
	dstMap[service] = dstSvc
	newSS, err := structpb.NewStruct(dstMap)
	if err != nil {
		return nil, err
	}
	dst.ServiceSpecific = newSS
	return dst, nil
}
