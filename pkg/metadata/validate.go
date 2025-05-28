package metadata

import (
	"fmt"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

// Metadata Standard Reference
// --------------------------
// All service-specific metadata must include the `versioning` field as described in:
//   - docs/services/versioning.md
//   - docs/amadeus/amadeus_context.md
// For all available metadata actions, patterns, and service-specific extensions, see:
//   - docs/services/metadata.md (general metadata documentation)
//   - docs/services/versioning.md (versioning/environment standard)
//
// This file implements metadata validation logic. See above for required fields and integration points.

// Note: Service-specific metadata builders (e.g., BuildContentMetadata) should be implemented in their respective service packages (e.g., internal/service/content/metadata.go).
// This file is for shared validation logic and cross-service helpers only.

const (
	MaxTags             = 20
	MaxFeatures         = 20
	MaxServiceSpecific  = 20
	MaxStringFieldLen   = 256
	MaxMetadataJSONSize = 64 * 1024 // 64KB
)

// ServiceValidator is a function that validates a service-specific metadata structpb.Struct.
type ServiceValidator func(meta *structpb.Struct) error

var serviceValidators = map[string]ServiceValidator{}

// RegisterServiceValidator registers a validator for a service namespace (e.g., "user", "localization").
func RegisterServiceValidator(namespace string, validator ServiceValidator) {
	serviceValidators[namespace] = validator
}

// ValidateMetadata checks that the metadata meets all platform standards.
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

	// Enforce standards for service_specific fields
	if meta.ServiceSpecific != nil {
		for ns, v := range meta.ServiceSpecific.Fields {
			// Each service-specific extension must be a struct
			ss, ok := v.GetKind().(*structpb.Value_StructValue)
			if !ok {
				return fmt.Errorf("service_specific.%s must be an object/struct", ns)
			}
			// Require versioning field
			if _, ok := ss.StructValue.Fields["versioning"]; !ok {
				return fmt.Errorf("service_specific.%s missing required 'versioning' field", ns)
			}
			// Call registered validator if present
			if validator, found := serviceValidators[ns]; found {
				if err := validator(ss.StructValue); err != nil {
					return fmt.Errorf("service_specific.%s validation failed: %w", ns, err)
				}
			}
		}
	}
	return nil
}

// Example: Register a validator for the "localization" namespace
// func init() {
// 	RegisterServiceValidator("localization", func(meta *structpb.Struct) error {
// 		if _, ok := meta.Fields["translation_provenance"]; !ok {
// 			return fmt.Errorf("missing required 'translation_provenance' field")
// 		}
// 		return nil
// 	})
// }

// Example: Register a validator for the "user" namespace
// func init() {
// 	RegisterServiceValidator("user", func(meta *structpb.Struct) error {
// 		// Add user-specific validation here
// 		return nil
// 	})
// }

// Register a validator for the "media" namespace.
func init() {
	RegisterServiceValidator("media", func(meta *structpb.Struct) error {
		if _, ok := meta.Fields["versioning"]; !ok {
			return fmt.Errorf("missing required 'versioning' field in media metadata")
		}
		// Optionally, add more validation for captions, accessibility, etc.
		return nil
	})

	// Register a validator for the 'referral' namespace
	RegisterServiceValidator("referral", func(meta *structpb.Struct) error {
		if _, ok := meta.Fields["versioning"]; !ok {
			return fmt.Errorf("missing required 'versioning' field in referral metadata")
		}
		if _, ok := meta.Fields["fraud_signals"]; !ok {
			return fmt.Errorf("missing required 'fraud_signals' field in referral metadata")
		}
		if _, ok := meta.Fields["audit"]; !ok {
			return fmt.Errorf("missing required 'audit' field in referral metadata")
		}
		return nil
	})
}

// Developers: For each new service-specific metadata extension, register a validator in your service package's init() or main() function.
// Validators should check for required fields, types, and allowed values as described in docs/services/metadata.md.

// BuildReferralMetadata builds a canonical referral metadata struct for storage and analytics.
func BuildReferralMetadata(fraudSignals, rewards, audit, campaign, device map[string]interface{}) (*commonpb.Metadata, error) {
	referralMap := map[string]interface{}{}
	if fraudSignals != nil {
		referralMap["fraud_signals"] = fraudSignals
	}
	if rewards != nil {
		referralMap["rewards"] = rewards
	}
	if audit != nil {
		referralMap["audit"] = audit
	}
	if campaign != nil {
		referralMap["campaign"] = campaign
	}
	if device != nil {
		referralMap["device"] = device
	}
	// Always require versioning for compliance
	if _, ok := referralMap["versioning"]; !ok {
		referralMap["versioning"] = map[string]interface{}{"system_version": "1.0.0"}
	}
	ss := map[string]interface{}{"referral": referralMap}
	ssStruct := NewStructFromMap(ss)
	return &commonpb.Metadata{ServiceSpecific: ssStruct}, nil
}
