# Package contentmoderationpb

## Constants

### ContentModerationService_SubmitContentForModeration_FullMethodName

## Variables

### ModerationStatus_name

Enum value maps for ModerationStatus.

### ContentModerationService_ServiceDesc

ContentModerationService_ServiceDesc is the grpc.ServiceDesc for ContentModerationService service.
It's only intended for direct use with grpc.RegisterService, and not to be introspected or modified
(even as a copy)

### File_contentmoderation_v1_contentmoderation_proto

## Types

### ApproveContentRequest

#### Methods

##### Descriptor

Deprecated: Use ApproveContentRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetContentId

##### GetMetadata

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ApproveContentResponse

#### Methods

##### Descriptor

Deprecated: Use ApproveContentResponse.ProtoReflect.Descriptor instead.

##### GetResult

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ContentModeration

#### Methods

##### Descriptor

Deprecated: Use ContentModeration.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetContentId

##### GetCreatedAt

##### GetId

##### GetMetadata

##### GetReason

##### GetScores

##### GetStatus

##### GetUpdatedAt

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ContentModerationServiceClient

ContentModerationServiceClient is the client API for ContentModerationService service.

For semantics around ctx use and closing/ending streaming RPCs, please refer to
https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.

### ContentModerationServiceServer

ContentModerationServiceServer is the server API for ContentModerationService service. All
implementations must embed UnimplementedContentModerationServiceServer for forward compatibility.

### GetModerationResultRequest

#### Methods

##### Descriptor

Deprecated: Use GetModerationResultRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetContentId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetModerationResultResponse

#### Methods

##### Descriptor

Deprecated: Use GetModerationResultResponse.ProtoReflect.Descriptor instead.

##### GetResult

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListFlaggedContentRequest

#### Methods

##### Descriptor

Deprecated: Use ListFlaggedContentRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetPage

##### GetPageSize

##### GetStatus

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListFlaggedContentResponse

#### Methods

##### Descriptor

Deprecated: Use ListFlaggedContentResponse.ProtoReflect.Descriptor instead.

##### GetPage

##### GetResults

##### GetTotalCount

##### GetTotalPages

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ModerationResult

#### Methods

##### Descriptor

Deprecated: Use ModerationResult.ProtoReflect.Descriptor instead.

##### GetContentId

##### GetCreatedAt

##### GetId

##### GetMetadata

##### GetReason

##### GetScores

##### GetStatus

##### GetUpdatedAt

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ModerationStatus

#### Methods

##### Descriptor

##### Enum

##### EnumDescriptor

Deprecated: Use ModerationStatus.Descriptor instead.

##### Number

##### String

##### Type

### RejectContentRequest

#### Methods

##### Descriptor

Deprecated: Use RejectContentRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetContentId

##### GetMetadata

##### GetReason

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### RejectContentResponse

#### Methods

##### Descriptor

Deprecated: Use RejectContentResponse.ProtoReflect.Descriptor instead.

##### GetResult

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SubmitContentForModerationRequest

#### Methods

##### Descriptor

Deprecated: Use SubmitContentForModerationRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetContent

##### GetContentId

##### GetContentType

##### GetMetadata

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SubmitContentForModerationResponse

#### Methods

##### Descriptor

Deprecated: Use SubmitContentForModerationResponse.ProtoReflect.Descriptor instead.

##### GetResult

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UnimplementedContentModerationServiceServer

UnimplementedContentModerationServiceServer must be embedded to have forward compatible
implementations.

NOTE: this should be embedded by value instead of pointer to avoid a nil pointer dereference when
methods are called.

#### Methods

##### ApproveContent

##### GetModerationResult

##### ListFlaggedContent

##### RejectContent

##### SubmitContentForModeration

### UnsafeContentModerationServiceServer

UnsafeContentModerationServiceServer may be embedded to opt out of forward compatibility for this
service. Use of this interface is not recommended, as added methods to
ContentModerationServiceServer will result in compilation errors.

## Functions

### RegisterContentModerationServiceServer
