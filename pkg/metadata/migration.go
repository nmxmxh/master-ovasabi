package metadata

import (
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// MigrateMetadata ensures metadata is at the latest version and migrates as needed.
func MigrateMetadata(meta *commonpb.Metadata) {
	if meta == nil || meta.ServiceSpecific == nil {
		return
	}
	ss := meta.ServiceSpecific.AsMap()
	for ns, v := range ss {
		m, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		ver, vok := m["versioning"].(map[string]interface{})
		if !vok {
			// Add versioning field if missing
			m["versioning"] = map[string]interface{}{
				"system_version":   "1.0.0",
				"service_version":  "1.0.0",
				"user_version":     "1.0.0",
				"environment":      "dev",
				"feature_flags":    []string{},
				"last_migrated_at": time.Now().Format(time.RFC3339),
			}
		} else if ver["system_version"] == "0.9.0" {
			ver["system_version"] = "1.0.0"
			ver["last_migrated_at"] = time.Now().Format(time.RFC3339)
			m["versioning"] = ver
		}
		ss[ns] = m
	}
	newStruct, err := structpb.NewStruct(ss)
	if err != nil {
		// Optionally log or handle the error, here we just return
		return
	}
	meta.ServiceSpecific = newStruct
}
