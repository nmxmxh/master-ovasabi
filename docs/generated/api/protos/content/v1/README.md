# Package contentpb

## Constants

### ContentService_CreateContent_FullMethodName

## Variables

### ContentService_ServiceDesc

ContentService_ServiceDesc is the grpc.ServiceDesc for ContentService service. It's only intended
for direct use with grpc.RegisterService, and not to be introspected or modified (even as a copy)

### File_content_v1_content_proto

## Types

### AddCommentRequest

#### Methods

##### Descriptor

Deprecated: Use AddCommentRequest.ProtoReflect.Descriptor instead.

##### GetAuthorId

##### GetBody

##### GetContentId

##### GetMetadata

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### AddReactionRequest

#### Methods

##### Descriptor

Deprecated: Use AddReactionRequest.ProtoReflect.Descriptor instead.

##### GetContentId

##### GetReaction

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Comment

#### Methods

##### Descriptor

Deprecated: Use Comment.ProtoReflect.Descriptor instead.

##### GetAuthorId

##### GetBody

##### GetContentId

##### GetCreatedAt

##### GetId

##### GetMasterId

##### GetMetadata

##### GetUpdatedAt

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CommentResponse

#### Methods

##### Descriptor

Deprecated: Use CommentResponse.ProtoReflect.Descriptor instead.

##### GetComment

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Content

#### Methods

##### Descriptor

Deprecated: Use Content.ProtoReflect.Descriptor instead.

##### GetAuthorId

##### GetBody

##### GetCommentCount

##### GetCreatedAt

##### GetId

##### GetMasterId

##### GetMediaUrls

##### GetMetadata

##### GetParentId

##### GetReactionCounts

##### GetTags

##### GetTitle

##### GetType

##### GetUpdatedAt

##### GetVisibility

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ContentEvent

#### Methods

##### Descriptor

Deprecated: Use ContentEvent.ProtoReflect.Descriptor instead.

##### GetContentId

##### GetEventType

##### GetId

##### GetMasterId

##### GetOccurredAt

##### GetPayload

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ContentResponse

#### Methods

##### Descriptor

Deprecated: Use ContentResponse.ProtoReflect.Descriptor instead.

##### GetContent

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ContentServiceClient

ContentServiceClient is the client API for ContentService service.

For semantics around ctx use and closing/ending streaming RPCs, please refer to
https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.

### ContentServiceServer

ContentServiceServer is the server API for ContentService service. All implementations must embed
UnimplementedContentServiceServer for forward compatibility.

### CreateContentRequest

#### Methods

##### Descriptor

Deprecated: Use CreateContentRequest.ProtoReflect.Descriptor instead.

##### GetContent

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### DeleteCommentRequest

#### Methods

##### Descriptor

Deprecated: Use DeleteCommentRequest.ProtoReflect.Descriptor instead.

##### GetCommentId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### DeleteCommentResponse

#### Methods

##### Descriptor

Deprecated: Use DeleteCommentResponse.ProtoReflect.Descriptor instead.

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### DeleteContentRequest

#### Methods

##### Descriptor

Deprecated: Use DeleteContentRequest.ProtoReflect.Descriptor instead.

##### GetId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### DeleteContentResponse

#### Methods

##### Descriptor

Deprecated: Use DeleteContentResponse.ProtoReflect.Descriptor instead.

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetContentRequest

#### Methods

##### Descriptor

Deprecated: Use GetContentRequest.ProtoReflect.Descriptor instead.

##### GetId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListCommentsRequest

#### Methods

##### Descriptor

Deprecated: Use ListCommentsRequest.ProtoReflect.Descriptor instead.

##### GetContentId

##### GetPage

##### GetPageSize

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListCommentsResponse

#### Methods

##### Descriptor

Deprecated: Use ListCommentsResponse.ProtoReflect.Descriptor instead.

##### GetComments

##### GetTotal

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListContentRequest

#### Methods

##### Descriptor

Deprecated: Use ListContentRequest.ProtoReflect.Descriptor instead.

##### GetAuthorId

##### GetMetadata

##### GetPage

##### GetPageSize

##### GetParentId

##### GetSearchQuery

##### GetTags

##### GetType

##### GetVisibility

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListContentResponse

#### Methods

##### Descriptor

Deprecated: Use ListContentResponse.ProtoReflect.Descriptor instead.

##### GetContents

##### GetTotal

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListReactionsRequest

#### Methods

##### Descriptor

Deprecated: Use ListReactionsRequest.ProtoReflect.Descriptor instead.

##### GetContentId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListReactionsResponse

#### Methods

##### Descriptor

Deprecated: Use ListReactionsResponse.ProtoReflect.Descriptor instead.

##### GetReactions

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### LogContentEventRequest

#### Methods

##### Descriptor

Deprecated: Use LogContentEventRequest.ProtoReflect.Descriptor instead.

##### GetEvent

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### LogContentEventResponse

#### Methods

##### Descriptor

Deprecated: Use LogContentEventResponse.ProtoReflect.Descriptor instead.

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ModerateContentRequest

Moderation hooks (stub)

#### Methods

##### Descriptor

Deprecated: Use ModerateContentRequest.ProtoReflect.Descriptor instead.

##### GetAction

##### GetContentId

##### GetModeratorId

##### GetReason

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ModerateContentResponse

#### Methods

##### Descriptor

Deprecated: Use ModerateContentResponse.ProtoReflect.Descriptor instead.

##### GetStatus

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ReactionResponse

#### Methods

##### Descriptor

Deprecated: Use ReactionResponse.ProtoReflect.Descriptor instead.

##### GetContentId

##### GetCount

##### GetReaction

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SearchContentRequest

#### Methods

##### Descriptor

Deprecated: Use SearchContentRequest.ProtoReflect.Descriptor instead.

##### GetMetadata

##### GetPage

##### GetPageSize

##### GetQuery

##### GetTags

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UnimplementedContentServiceServer

UnimplementedContentServiceServer must be embedded to have forward compatible implementations.

NOTE: this should be embedded by value instead of pointer to avoid a nil pointer dereference when
methods are called.

#### Methods

##### AddComment

##### AddReaction

##### CreateContent

##### DeleteComment

##### DeleteContent

##### GetContent

##### ListComments

##### ListContent

##### ListReactions

##### LogContentEvent

##### ModerateContent

##### SearchContent

##### UpdateContent

### UnsafeContentServiceServer

UnsafeContentServiceServer may be embedded to opt out of forward compatibility for this service. Use
of this interface is not recommended, as added methods to ContentServiceServer will result in
compilation errors.

### UpdateContentRequest

#### Methods

##### Descriptor

Deprecated: Use UpdateContentRequest.ProtoReflect.Descriptor instead.

##### GetContent

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

## Functions

### RegisterContentServiceServer
