# Package metadata

## Constants

### MaxTags

## Types

### AuditMetadata

### AuditRecord

### Hook

### MFAChallengeData

### PasswordResetData

### ServiceMetadata

### ServiceValidator

ServiceValidator is a function that validates a service-specific metadata structpb.Struct.

### VerificationData

### WebAuthnCredential

## Functions

### BuildReferralMetadata

BuildReferralMetadata builds a canonical referral metadata struct for storage and analytics.

### MarshalCanonical

MarshalCanonical marshals a proto.Message using the canonical options for OVASABI metadata.

### MergeStructs

MergeStructs merges two structpb.Structs, with fields from b overwriting a.

### MigrateMetadata

MigrateMetadata ensures metadata is at the latest version and migrates as needed.

### NewStructFromMap

NewStructFromMap creates a structpb.Struct from a map, optionally merging into an existing struct.

### RedactPII

RedactPII removes PII fields from metadata.

### RegisterMetadataHook

RegisterMetadataHook registers a new metadata hook.

### RegisterServiceValidator

RegisterServiceValidator registers a validator for a service namespace (e.g., "user",
"localization").

### RunPostUpdateHooks

RunPostUpdateHooks runs all registered PostUpdate hooks.

### RunPreUpdateHooks

RunPreUpdateHooks runs all registered PreUpdate hooks.

### ServiceMetadataToStruct

ServiceMetadataToStruct converts a \*ServiceMetadata to structpb.Struct.

### ToMap

ToMap safely converts an interface{} to map[string]interface{} if possible, else returns an empty
map.

### UnmarshalCanonical

UnmarshalCanonical unmarshals canonical JSON into a proto.Message.

### UpdateJWTIssueMetadata

UpdateJWTIssueMetadata is a convenience for common JWT issuance fields.

### UpdateJWTMetadata

UpdateJWTMetadata updates the jwt section of a user metadata map.

### ValidateMetadata

ValidateMetadata checks that the metadata meets all platform standards.
