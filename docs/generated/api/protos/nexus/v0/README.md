# Package nexuspb

## Constants

### NexusService_ExecutePattern_FullMethodName

## Variables

### File_api_protos_nexus_v0_nexus_proto

### NexusService_ServiceDesc

NexusService_ServiceDesc is the grpc.ServiceDesc for NexusService service. It's only intended for
direct use with grpc.RegisterService, and not to be introspected or modified (even as a copy)

## Types

### ExecutePatternRequest

ExecutePatternRequest represents a request to execute a pattern

#### Methods

##### Descriptor

Deprecated: Use ExecutePatternRequest.ProtoReflect.Descriptor instead.

##### GetParameters

##### GetPatternName

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ExecutePatternResponse

ExecutePatternResponse represents the response from pattern execution

#### Methods

##### Descriptor

Deprecated: Use ExecutePatternResponse.ProtoReflect.Descriptor instead.

##### GetResult

##### GetStatus

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetKnowledgeRequest

GetKnowledgeRequest represents a request to get knowledge from the graph

#### Methods

##### Descriptor

Deprecated: Use GetKnowledgeRequest.ProtoReflect.Descriptor instead.

##### GetPath

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetKnowledgeResponse

GetKnowledgeResponse represents the response containing knowledge graph data

#### Methods

##### Descriptor

Deprecated: Use GetKnowledgeResponse.ProtoReflect.Descriptor instead.

##### GetData

##### GetStatus

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### NexusServiceClient

NexusServiceClient is the client API for NexusService service.

For semantics around ctx use and closing/ending streaming RPCs, please refer to
https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.

### NexusServiceServer

NexusServiceServer is the server API for NexusService service. All implementations must embed
UnimplementedNexusServiceServer for forward compatibility

### RegisterPatternRequest

RegisterPatternRequest represents a request to register a new pattern

#### Methods

##### Descriptor

Deprecated: Use RegisterPatternRequest.ProtoReflect.Descriptor instead.

##### GetPatternConfig

##### GetPatternName

##### GetPatternType

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### RegisterPatternResponse

RegisterPatternResponse represents the response from pattern registration

#### Methods

##### Descriptor

Deprecated: Use RegisterPatternResponse.ProtoReflect.Descriptor instead.

##### GetMessage

##### GetStatus

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UnimplementedNexusServiceServer

UnimplementedNexusServiceServer must be embedded to have forward compatible implementations.

#### Methods

##### ExecutePattern

##### GetKnowledge

##### RegisterPattern

### UnsafeNexusServiceServer

UnsafeNexusServiceServer may be embedded to opt out of forward compatibility for this service. Use
of this interface is not recommended, as added methods to NexusServiceServer will result in
compilation errors.

## Functions

### RegisterNexusServiceServer
