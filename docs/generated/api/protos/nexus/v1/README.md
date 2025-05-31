# Package nexus

## Constants

### NexusService_RegisterPattern_FullMethodName

## Variables

### File_nexus_v1_nexus_proto

### NexusService_ServiceDesc

NexusService_ServiceDesc is the grpc.ServiceDesc for NexusService service. It's only intended for
direct use with grpc.RegisterService, and not to be introspected or modified (even as a copy)

## Types

### EventRequest

#### Methods

##### Descriptor

Deprecated: Use EventRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetEntityId

##### GetEventType

##### GetMetadata

##### GetPayload

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### EventResponse

#### Methods

##### Descriptor

Deprecated: Use EventResponse.ProtoReflect.Descriptor instead.

##### GetMessage

##### GetMetadata

##### GetPayload

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### FeedbackRequest

#### Methods

##### Descriptor

Deprecated: Use FeedbackRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetComments

##### GetMetadata

##### GetPatternId

##### GetScore

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### FeedbackResponse

#### Methods

##### Descriptor

Deprecated: Use FeedbackResponse.ProtoReflect.Descriptor instead.

##### GetError

##### GetMetadata

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### HandleOpsRequest

#### Methods

##### Descriptor

Deprecated: Use HandleOpsRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetMetadata

##### GetOp

##### GetParams

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### HandleOpsResponse

#### Methods

##### Descriptor

Deprecated: Use HandleOpsResponse.ProtoReflect.Descriptor instead.

##### GetData

##### GetMessage

##### GetMetadata

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListPatternsRequest

#### Methods

##### Descriptor

Deprecated: Use ListPatternsRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetMetadata

##### GetPatternType

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListPatternsResponse

#### Methods

##### Descriptor

Deprecated: Use ListPatternsResponse.ProtoReflect.Descriptor instead.

##### GetMetadata

##### GetPatterns

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### MinePatternsRequest

#### Methods

##### Descriptor

Deprecated: Use MinePatternsRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetMetadata

##### GetSource

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### MinePatternsResponse

#### Methods

##### Descriptor

Deprecated: Use MinePatternsResponse.ProtoReflect.Descriptor instead.

##### GetMetadata

##### GetPatterns

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Nexus

#### Methods

##### Descriptor

Deprecated: Use Nexus.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### NexusServiceClient

NexusServiceClient is the client API for NexusService service.

For semantics around ctx use and closing/ending streaming RPCs, please refer to
https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.

NexusService: High-level composer, orchestrator, and pattern identifier

### NexusServiceServer

NexusServiceServer is the server API for NexusService service. All implementations must embed
UnimplementedNexusServiceServer for forward compatibility.

NexusService: High-level composer, orchestrator, and pattern identifier

### NexusService_SubscribeEventsClient

This type alias is provided for backwards compatibility with existing code that references the prior
non-generic stream type by name.

### NexusService_SubscribeEventsServer

This type alias is provided for backwards compatibility with existing code that references the prior
non-generic stream type by name.

### OrchestrateRequest

#### Methods

##### Descriptor

Deprecated: Use OrchestrateRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetInput

##### GetMetadata

##### GetPatternId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### OrchestrateResponse

#### Methods

##### Descriptor

Deprecated: Use OrchestrateResponse.ProtoReflect.Descriptor instead.

##### GetMetadata

##### GetOrchestrationId

##### GetOutput

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Pattern

#### Methods

##### Descriptor

Deprecated: Use Pattern.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetDefinition

##### GetLastUsed

##### GetMetadata

##### GetOrigin

##### GetPatternId

##### GetPatternType

##### GetUsageCount

##### GetVersion

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### RegisterPatternRequest

#### Methods

##### Descriptor

Deprecated: Use RegisterPatternRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetDefinition

##### GetMetadata

##### GetOrigin

##### GetPatternId

##### GetPatternType

##### GetVersion

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### RegisterPatternResponse

#### Methods

##### Descriptor

Deprecated: Use RegisterPatternResponse.ProtoReflect.Descriptor instead.

##### GetError

##### GetMetadata

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SubscribeRequest

#### Methods

##### Descriptor

Deprecated: Use SubscribeRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetEventTypes

##### GetMetadata

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### TracePatternRequest

#### Methods

##### Descriptor

Deprecated: Use TracePatternRequest.ProtoReflect.Descriptor instead.

##### GetMetadata

##### GetOrchestrationId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### TracePatternResponse

#### Methods

##### Descriptor

Deprecated: Use TracePatternResponse.ProtoReflect.Descriptor instead.

##### GetMetadata

##### GetSteps

##### GetTraceId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### TraceStep

#### Methods

##### Descriptor

Deprecated: Use TraceStep.ProtoReflect.Descriptor instead.

##### GetAction

##### GetDetails

##### GetService

##### GetTimestamp

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UnimplementedNexusServiceServer

UnimplementedNexusServiceServer must be embedded to have forward compatible implementations.

NOTE: this should be embedded by value instead of pointer to avoid a nil pointer dereference when
methods are called.

#### Methods

##### EmitEvent

##### Feedback

##### HandleOps

##### ListPatterns

##### MinePatterns

##### Orchestrate

##### RegisterPattern

##### SubscribeEvents

##### TracePattern

### UnsafeNexusServiceServer

UnsafeNexusServiceServer may be embedded to opt out of forward compatibility for this service. Use
of this interface is not recommended, as added methods to NexusServiceServer will result in
compilation errors.

## Functions

### RegisterNexusServiceServer
