# Package metadata

## Constants

### MaxTags

## Types

### AuditMetadata

### AuditRecord

### Handler

MetadataHandler is the canonical handler for all metadata operations (creation, chaining,
idempotency, calculation, search).

#### Methods

##### AddScore

AddScore adds or updates a score in the calculation field.

##### AddTax

AddTax adds or updates a tax value in the calculation field.

##### AppendAudit

AppendAudit appends an entry to the audit or lineage field in metadata.

##### DefaultMetadata

DefaultMetadata returns a canonical metadata map with all required fields initialized.

##### GenerateIdempotentKey

GenerateIdempotentKey generates a unique, idempotent key for a metadata instance based on its
normalized content and context.

##### GetChainLinks

GetChainLinks retrieves prev, next, and related state ids from metadata.

##### GrepMetadata

GrepMetadata searches for a field or value in metadata and returns matching keys/values.

##### NormalizeAndCalculate

NormalizeAndCalculate normalizes metadata and performs default calculations for success/error
states. calculationType should be "success" or "error".

##### NormalizeMetadata

NormalizeMetadata ensures the metadata is canonical: sets chain links, sorts keys, and returns a
normalized map.

##### SetAvailableBalance

SetAvailableBalance sets the available balance in the calculation field.

##### SetChainLinks

SetChainLinks sets prev, next, and related state ids in metadata.

##### SetPending

SetPending sets the pending value in the calculation field.

##### TransferOwnership

TransferOwnership updates the owner, audit, prev_state_id, and updated_at fields, and returns the
new idempotent key.

##### UpdateCalculation

UpdateCalculation updates the calculation field in metadata (e.g., score, tax, etc.).

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

### CanonicalEnrichMetadata

CanonicalEnrichMetadata enriches a metadata map with system context, event/step, and extra fields.

### ExtractServiceVariables

ExtractServiceVariables extracts key variables (score, badges, gamification, compliance, etc.) from
any service_specific namespace in a commonpb.Metadata. This is the canonical, extensible function
for state hydration, leaderboard, trending, and gamification. Usage: vars :=
metadata.ExtractServiceVariables(meta, "user"), metadata.ExtractServiceVariables(meta, "campaign"),
etc.

### JSONToMap

JSONToMap unmarshals JSON bytes to map[string]interface{}.

### MapToJSON

MapToJSON marshals a map[string]interface{} to JSON bytes.

### MapToProto

MapToProto converts a map[string]interface{} (from Handler) to a \*commonpb.Metadata proto.

### MapToStruct

MapToStruct converts a map[string]interface{} to \*structpb.Struct.

### MarshalCanonical

MarshalCanonical marshals a proto.Message using the canonical options for OVASABI metadata.

### MergeStructs

MergeStructs merges two structpb.Structs, with fields from b overwriting a.

### MigrateMetadata

MigrateMetadata ensures metadata is at the latest version and migrates as needed.

### NewStructFromMap

NewStructFromMap creates a structpb.Struct from a map, optionally merging into an existing struct.

### ProtoToMap

ProtoToMap converts a \*commonpb.Metadata proto to a map[string]interface{} for use with Handler.

### ProtoToStruct

ProtoToStruct converts a *commonpb.Metadata proto to *structpb.Struct (for storage as jsonb or for
gRPC).

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

### SetServiceSpecificField

Usage: metadata.SetServiceSpecificField(meta, "admin", "versioning", versioningMap).

### StructToMap

StructToMap converts a \*structpb.Struct to map[string]interface{}.

### StructToProto

StructToProto converts a *structpb.Struct to *commonpb.Metadata proto.

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
