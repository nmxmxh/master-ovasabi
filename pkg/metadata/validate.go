package metadata

import (
	"fmt"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	MaxTags             = 20
	MaxFeatures         = 20
	MaxServiceSpecific  = 20
	MaxStringFieldLen   = 256
	MaxMetadataJSONSize = 16 * 1024 // 16KB
)

// ValidateMetadata checks that the metadata does not exceed size or field limits.
func ValidateMetadata(meta *commonpb.Metadata) error {
	if meta == nil {
		return nil
	}
	if len(meta.Tags) > MaxTags {
		return fmt.Errorf("too many tags (max %d)", MaxTags)
	}
	if len(meta.Features) > MaxFeatures {
		return fmt.Errorf("too many features (max %d)", MaxFeatures)
	}
	if meta.ServiceSpecific != nil && len(meta.ServiceSpecific.Fields) > MaxServiceSpecific {
		return fmt.Errorf("too many service-specific fields (max %d)", MaxServiceSpecific)
	}
	for _, tag := range meta.Tags {
		if len(tag) > MaxStringFieldLen {
			return fmt.Errorf("tag too long (max %d chars)", MaxStringFieldLen)
		}
	}
	b, err := protojson.Marshal(meta)
	if err != nil {
		// TODO: log.Warn("protojson.Marshal failed in ValidateMetadata", zap.Error(err))
		return fmt.Errorf("failed to marshal metadata in ValidateMetadata: %w", err)
	}
	if len(b) > MaxMetadataJSONSize {
		return fmt.Errorf("metadata too large (max %d bytes)", MaxMetadataJSONSize)
	}
	return nil
}
