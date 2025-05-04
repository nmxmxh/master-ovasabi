# Package broadcast

## Constants

### BroadcastService_BroadcastAction_FullMethodName

## Variables

### BroadcastService_ServiceDesc

BroadcastService_ServiceDesc is the grpc.ServiceDesc for BroadcastService service. It's only
intended for direct use with grpc.RegisterService, and not to be introspected or modified (even as a
copy)

### File_api_protos_broadcast_v0_broadcast_proto

## Types

### ActionSummary

ActionSummary contains a summary of user actions

#### Methods

##### Descriptor

Deprecated: Use ActionSummary.ProtoReflect.Descriptor instead.

##### GetActionType

##### GetApplicationId

##### GetMetadata

##### GetTimestamp

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### AssetChunk

Add for live asset streaming

#### Methods

##### Descriptor

Deprecated: Use AssetChunk.ProtoReflect.Descriptor instead.

##### GetData

##### GetSequence

##### GetUploadId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Broadcast

Broadcast message for campaigns or services

#### Methods

##### Descriptor

Deprecated: Use Broadcast.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetChannel

##### GetCreatedAt

##### GetId

##### GetMasterId

##### GetMessage

##### GetPayload

##### GetScheduledAt

##### GetSubject

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### BroadcastActionRequest

BroadcastActionRequest contains the action to broadcast

#### Methods

##### Descriptor

Deprecated: Use BroadcastActionRequest.ProtoReflect.Descriptor instead.

##### GetActionType

##### GetApplicationId

##### GetMetadata

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### BroadcastActionResponse

BroadcastActionResponse contains the broadcast result

#### Methods

##### Descriptor

Deprecated: Use BroadcastActionResponse.ProtoReflect.Descriptor instead.

##### GetMessage

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### BroadcastServiceClient

BroadcastServiceClient is the client API for BroadcastService service.

For semantics around ctx use and closing/ending streaming RPCs, please refer to
https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.

### BroadcastServiceServer

BroadcastServiceServer is the server API for BroadcastService service. All implementations must
embed UnimplementedBroadcastServiceServer for forward compatibility

### BroadcastService_SubscribeToActionsClient

### BroadcastService_SubscribeToActionsServer

### BroadcastService_SubscribeToLiveAssetChunksClient

### BroadcastService_SubscribeToLiveAssetChunksServer

### CreateBroadcastRequest

#### Methods

##### Descriptor

Deprecated: Use CreateBroadcastRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetChannel

##### GetMasterId

##### GetMessage

##### GetPayload

##### GetScheduledAt

##### GetSubject

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateBroadcastResponse

#### Methods

##### Descriptor

Deprecated: Use CreateBroadcastResponse.ProtoReflect.Descriptor instead.

##### GetBroadcast

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetBroadcastRequest

#### Methods

##### Descriptor

Deprecated: Use GetBroadcastRequest.ProtoReflect.Descriptor instead.

##### GetBroadcastId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetBroadcastResponse

#### Methods

##### Descriptor

Deprecated: Use GetBroadcastResponse.ProtoReflect.Descriptor instead.

##### GetBroadcast

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListBroadcastsRequest

#### Methods

##### Descriptor

Deprecated: Use ListBroadcastsRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetPage

##### GetPageSize

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListBroadcastsResponse

#### Methods

##### Descriptor

Deprecated: Use ListBroadcastsResponse.ProtoReflect.Descriptor instead.

##### GetBroadcasts

##### GetPage

##### GetTotalCount

##### GetTotalPages

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SubscribeRequest

SubscribeRequest contains subscription parameters

#### Methods

##### Descriptor

Deprecated: Use SubscribeRequest.ProtoReflect.Descriptor instead.

##### GetActionTypes

##### GetApplicationId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SubscribeToLiveAssetChunksRequest

#### Methods

##### Descriptor

Deprecated: Use SubscribeToLiveAssetChunksRequest.ProtoReflect.Descriptor instead.

##### GetAssetId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UnimplementedBroadcastServiceServer

UnimplementedBroadcastServiceServer must be embedded to have forward compatible implementations.

#### Methods

##### BroadcastAction

##### CreateBroadcast

##### GetBroadcast

##### ListBroadcasts

##### PublishLiveAssetChunk

##### SubscribeToActions

##### SubscribeToLiveAssetChunks

### UnsafeBroadcastServiceServer

UnsafeBroadcastServiceServer may be embedded to opt out of forward compatibility for this service.
Use of this interface is not recommended, as added methods to BroadcastServiceServer will result in
compilation errors.

## Functions

### RegisterBroadcastServiceServer
