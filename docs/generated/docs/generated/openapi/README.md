# Package commonpb

## Variables

### File_metadata_proto

## Types

### Metadata

Central, robust, extensible metadata for all services

- All shared fields are available to every service.
- The service_specific field allows each service to store its own custom, structured data under a
  namespaced key (e.g., "content", "commerce").
- The knowledge_graph field is for graph enrichment and relationships.

#### Methods

##### Descriptor

Deprecated: Use Metadata.ProtoReflect.Descriptor instead.

##### GetAudit

##### GetCustomRules

##### GetFeatures

##### GetKnowledgeGraph

##### GetOwner

##### GetReferral

##### GetScheduling

##### GetServiceSpecific

##### GetTags

##### GetTaxation

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### OwnerMetadata

#### Methods

##### Descriptor

Deprecated: Use OwnerMetadata.ProtoReflect.Descriptor instead.

##### GetId

##### GetWallet

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ReferralMetadata

#### Methods

##### Descriptor

Deprecated: Use ReferralMetadata.ProtoReflect.Descriptor instead.

##### GetId

##### GetWallet

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Taxation

#### Methods

##### Descriptor

Deprecated: Use Taxation.ProtoReflect.Descriptor instead.

##### GetConnectors

##### GetProjectCount

##### GetTotalTax

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### TaxationConnector

#### Methods

##### Descriptor

Deprecated: Use TaxationConnector.ProtoReflect.Descriptor instead.

##### GetAppliedOn

##### GetDefault

##### GetDomain

##### GetEnforced

##### GetJustification

##### GetPercentage

##### GetRecipient

##### GetRecipientWallet

##### GetTiered

##### GetType

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### TieredTax

#### Methods

##### Descriptor

Deprecated: Use TieredTax.ProtoReflect.Descriptor instead.

##### GetMaxProjects

##### GetMinProjects

##### GetPercentage

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String
