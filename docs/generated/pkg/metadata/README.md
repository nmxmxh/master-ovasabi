# Package metadata

## Constants

### MaxTags

## Types

### AuditMetadata

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

### RegisterServiceValidator

RegisterServiceValidator registers a validator for a service namespace (e.g., "user",
"localization").

### ServiceMetadataToStruct

ServiceMetadataToStruct converts a \*ServiceMetadata to structpb.Struct.

### ValidateMetadata

ValidateMetadata checks that the metadata meets all platform standards.
