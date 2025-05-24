# Package search

## Constants

### SearchService_Search_FullMethodName

## Variables

### File_search_v1_search_proto

### SearchService_ServiceDesc

SearchService_ServiceDesc is the grpc.ServiceDesc for SearchService service. It's only intended for
direct use with grpc.RegisterService, and not to be introspected or modified (even as a copy)

## Types

### SearchRequest

Request for a search query.

#### Methods

##### Descriptor

Deprecated: Use SearchRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetMetadata

##### GetPageNumber

##### GetPageSize

##### GetQuery

##### GetTypes

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SearchResponse

Response for a search query.

#### Methods

##### Descriptor

Deprecated: Use SearchResponse.ProtoReflect.Descriptor instead.

##### GetMetadata

##### GetPageNumber

##### GetPageSize

##### GetResults

##### GetTotal

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SearchResult

A single search result.

#### Methods

##### Descriptor

Deprecated: Use SearchResult.ProtoReflect.Descriptor instead.

##### GetEntityType

##### GetFields

##### GetId

##### GetMetadata

##### GetScore

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SearchServiceClient

SearchServiceClient is the client API for SearchService service.

For semantics around ctx use and closing/ending streaming RPCs, please refer to
https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.

SearchService provides unified, metadata-driven search across all major entities.

### SearchServiceServer

SearchServiceServer is the server API for SearchService service. All implementations must embed
UnimplementedSearchServiceServer for forward compatibility.

SearchService provides unified, metadata-driven search across all major entities.

### SuggestRequest

Request for suggestions/autocomplete.

#### Methods

##### Descriptor

Deprecated: Use SuggestRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetLimit

##### GetMetadata

##### GetPrefix

##### GetTypes

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SuggestResponse

Response for suggestions/autocomplete.

#### Methods

##### Descriptor

Deprecated: Use SuggestResponse.ProtoReflect.Descriptor instead.

##### GetMetadata

##### GetSuggestions

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UnimplementedSearchServiceServer

UnimplementedSearchServiceServer must be embedded to have forward compatible implementations.

NOTE: this should be embedded by value instead of pointer to avoid a nil pointer dereference when
methods are called.

#### Methods

##### Search

##### Suggest

### UnsafeSearchServiceServer

UnsafeSearchServiceServer may be embedded to opt out of forward compatibility for this service. Use
of this interface is not recommended, as added methods to SearchServiceServer will result in
compilation errors.

## Functions

### RegisterSearchServiceServer
