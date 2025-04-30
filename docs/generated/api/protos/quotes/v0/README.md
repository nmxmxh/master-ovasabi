# Package quotes

## Constants

### QuotesService_CreateQuote_FullMethodName

## Variables

### File_api_protos_quotes_v0_quotes_proto

### QuotesService_ServiceDesc

QuotesService_ServiceDesc is the grpc.ServiceDesc for QuotesService service. It's only intended for
direct use with grpc.RegisterService, and not to be introspected or modified (even as a copy)

## Types

### BillingQuote

BillingQuote represents a quote for billing purposes

#### Methods

##### Descriptor

Deprecated: Use BillingQuote.ProtoReflect.Descriptor instead.

##### GetAmount

##### GetAuthor

##### GetCampaignId

##### GetCreatedAt

##### GetCurrency

##### GetDescription

##### GetId

##### GetMasterId

##### GetMetadata

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateQuoteRequest

CreateQuoteRequest is the request message for creating a quote

#### Methods

##### Descriptor

Deprecated: Use CreateQuoteRequest.ProtoReflect.Descriptor instead.

##### GetAuthor

##### GetCampaignId

##### GetDescription

##### GetMasterId

##### GetMetadata

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateQuoteResponse

CreateQuoteResponse is the response message for creating a quote

#### Methods

##### Descriptor

Deprecated: Use CreateQuoteResponse.ProtoReflect.Descriptor instead.

##### GetQuote

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetQuoteRequest

GetQuoteRequest is the request message for retrieving a quote

#### Methods

##### Descriptor

Deprecated: Use GetQuoteRequest.ProtoReflect.Descriptor instead.

##### GetQuoteId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetQuoteResponse

GetQuoteResponse is the response message for retrieving a quote

#### Methods

##### Descriptor

Deprecated: Use GetQuoteResponse.ProtoReflect.Descriptor instead.

##### GetQuote

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListQuotesRequest

ListQuotesRequest is the request message for listing quotes

#### Methods

##### Descriptor

Deprecated: Use ListQuotesRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetPage

##### GetPageSize

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListQuotesResponse

ListQuotesResponse is the response message for listing quotes

#### Methods

##### Descriptor

Deprecated: Use ListQuotesResponse.ProtoReflect.Descriptor instead.

##### GetPage

##### GetQuotes

##### GetTotalCount

##### GetTotalPages

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### QuotesServiceClient

QuotesServiceClient is the client API for QuotesService service.

For semantics around ctx use and closing/ending streaming RPCs, please refer to
https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.

### QuotesServiceServer

QuotesServiceServer is the server API for QuotesService service. All implementations must embed
UnimplementedQuotesServiceServer for forward compatibility

### UnimplementedQuotesServiceServer

UnimplementedQuotesServiceServer must be embedded to have forward compatible implementations.

#### Methods

##### CreateQuote

##### GetQuote

##### ListQuotes

### UnsafeQuotesServiceServer

UnsafeQuotesServiceServer may be embedded to opt out of forward compatibility for this service. Use
of this interface is not recommended, as added methods to QuotesServiceServer will result in
compilation errors.

## Functions

### RegisterQuotesServiceServer
