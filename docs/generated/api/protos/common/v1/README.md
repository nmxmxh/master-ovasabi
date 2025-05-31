# Package commonpb

## Variables

### File_common_v1_metadata_proto

### File_common_v1_payload_proto

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

##### GetScheduling

##### GetServiceSpecific

##### GetTags

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Payload

Canonical Payload message for all event-driven and cross-service communication.

#### Methods

##### Descriptor

Deprecated: Use Payload.ProtoReflect.Descriptor instead.

##### GetData

##### GetVersioning

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String
