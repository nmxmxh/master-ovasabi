# Package searchpb

## Constants

### SearchService_SearchEntities_FullMethodName

## Variables

### File_search_v1_search_proto

### SearchService_ServiceDesc

SearchService_ServiceDesc is the grpc.ServiceDesc for SearchService service. It's only intended for
direct use with grpc.RegisterService, and not to be introspected or modified (even as a copy)

## Types

### EntityResult

#### Methods

##### Descriptor

Deprecated: Use EntityResult.ProtoReflect.Descriptor instead.

##### GetEntityId

##### GetEntityType

##### GetMetadata

##### GetName

##### GetRank

##### GetSimilarity

##### GetSummary

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SearchRequest

#### Methods

##### Descriptor

Deprecated: Use SearchRequest.ProtoReflect.Descriptor instead.

##### GetEntityType

##### GetFields

##### GetFuzzy

##### GetLanguage

##### GetMasterId

##### GetMetadata

##### GetPage

##### GetPageSize

##### GetQuery

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SearchResponse

#### Methods

##### Descriptor

Deprecated: Use SearchResponse.ProtoReflect.Descriptor instead.

##### GetResults

##### GetTotal

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SearchResult

#### Methods

##### Descriptor

Deprecated: Use SearchResult.ProtoReflect.Descriptor instead.

##### GetEntityType

##### GetId

##### GetMasterId

##### GetMetadata

##### GetScore

##### GetSnippet

##### GetTitle

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SearchServiceClient

SearchServiceClient is the client API for SearchService service.

For semantics around ctx use and closing/ending streaming RPCs, please refer to
https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.

### SearchServiceServer

SearchServiceServer is the server API for SearchService service. All implementations must embed
UnimplementedSearchServiceServer for forward compatibility.

### UnimplementedSearchServiceServer

UnimplementedSearchServiceServer must be embedded to have forward compatible implementations.

NOTE: this should be embedded by value instead of pointer to avoid a nil pointer dereference when
methods are called.

#### Methods

##### SearchEntities

### UnsafeSearchServiceServer

UnsafeSearchServiceServer may be embedded to opt out of forward compatibility for this service. Use
of this interface is not recommended, as added methods to SearchServiceServer will result in
compilation errors.

## Functions

### RegisterSearchServiceServer
