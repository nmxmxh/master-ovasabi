# Package assetv0

## Constants

### AssetService_UploadLightAsset_FullMethodName

## Variables

### AssetType_name

Enum value maps for AssetType.

### AssetService_ServiceDesc

AssetService_ServiceDesc is the grpc.ServiceDesc for AssetService service. It's only intended for
direct use with grpc.RegisterService, and not to be introspected or modified (even as a copy)

### File_api_protos_asset_v1_asset_proto

## Types

### Asset

Asset represents a 3D asset and its metadata

#### Methods

##### Descriptor

Deprecated: Use Asset.ProtoReflect.Descriptor instead.

##### GetCreatedAt

##### GetData

##### GetDeletedAt

##### GetId

##### GetIsSystem

##### GetMimeType

##### GetName

##### GetSize

##### GetType

##### GetUpdatedAt

##### GetUrl

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### AssetChunk

Chunk of asset data for streaming

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

### AssetServiceClient

AssetServiceClient is the client API for AssetService service.

For semantics around ctx use and closing/ending streaming RPCs, please refer to
https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.

Asset service handles storage and retrieval of 3D assets

### AssetServiceServer

AssetServiceServer is the server API for AssetService service. All implementations must embed
UnimplementedAssetServiceServer for forward compatibility.

Asset service handles storage and retrieval of 3D assets

### AssetType

Asset types

#### Methods

##### Descriptor

##### Enum

##### EnumDescriptor

Deprecated: Use AssetType.Descriptor instead.

##### Number

##### String

##### Type

### BroadcastSystemAssetRequest

Request to broadcast a system asset

#### Methods

##### Descriptor

Deprecated: Use BroadcastSystemAssetRequest.ProtoReflect.Descriptor instead.

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### BroadcastSystemAssetResponse

Response for broadcasting a system asset

#### Methods

##### Descriptor

Deprecated: Use BroadcastSystemAssetResponse.ProtoReflect.Descriptor instead.

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CompleteAssetUploadRequest

Request to complete an asset upload

#### Methods

##### Descriptor

Deprecated: Use CompleteAssetUploadRequest.ProtoReflect.Descriptor instead.

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CompleteAssetUploadResponse

Response for completing an asset upload

#### Methods

##### Descriptor

Deprecated: Use CompleteAssetUploadResponse.ProtoReflect.Descriptor instead.

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### DeleteAssetRequest

Request to delete an asset

#### Methods

##### Descriptor

Deprecated: Use DeleteAssetRequest.ProtoReflect.Descriptor instead.

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### DeleteAssetResponse

Response for deleting an asset

#### Methods

##### Descriptor

Deprecated: Use DeleteAssetResponse.ProtoReflect.Descriptor instead.

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetAssetRequest

Request to get an asset

#### Methods

##### Descriptor

Deprecated: Use GetAssetRequest.ProtoReflect.Descriptor instead.

##### GetId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetAssetResponse

Response for getting an asset

#### Methods

##### Descriptor

Deprecated: Use GetAssetResponse.ProtoReflect.Descriptor instead.

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListSystemAssetsRequest

Request to list system assets

#### Methods

##### Descriptor

Deprecated: Use ListSystemAssetsRequest.ProtoReflect.Descriptor instead.

##### GetPageSize

##### GetPageToken

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListSystemAssetsResponse

Response for listing system assets

#### Methods

##### Descriptor

Deprecated: Use ListSystemAssetsResponse.ProtoReflect.Descriptor instead.

##### GetAssets

##### GetNextPageToken

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListUserAssetsRequest

Request to list user assets

#### Methods

##### Descriptor

Deprecated: Use ListUserAssetsRequest.ProtoReflect.Descriptor instead.

##### GetPageSize

##### GetPageToken

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListUserAssetsResponse

Response for listing user assets

#### Methods

##### Descriptor

Deprecated: Use ListUserAssetsResponse.ProtoReflect.Descriptor instead.

##### GetAssets

##### GetNextPageToken

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### StartHeavyAssetUploadRequest

Request to start a heavy asset upload

#### Methods

##### Descriptor

Deprecated: Use StartHeavyAssetUploadRequest.ProtoReflect.Descriptor instead.

##### GetMimeType

##### GetName

##### GetSize

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### StartHeavyAssetUploadResponse

Response for starting a heavy asset upload

#### Methods

##### Descriptor

Deprecated: Use StartHeavyAssetUploadResponse.ProtoReflect.Descriptor instead.

##### GetChunkSize

##### GetChunksTotal

##### GetUploadId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### StreamAssetChunkRequest

Request to stream asset chunks

#### Methods

##### Descriptor

Deprecated: Use StreamAssetChunkRequest.ProtoReflect.Descriptor instead.

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### StreamAssetChunkResponse

Response for streaming asset chunks

#### Methods

##### Descriptor

Deprecated: Use StreamAssetChunkResponse.ProtoReflect.Descriptor instead.

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### StreamAssetContentRequest

Request to stream asset content

#### Methods

##### Descriptor

Deprecated: Use StreamAssetContentRequest.ProtoReflect.Descriptor instead.

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### StreamAssetContentResponse

Response for streaming asset content

#### Methods

##### Descriptor

Deprecated: Use StreamAssetContentResponse.ProtoReflect.Descriptor instead.

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SubscribeToSystemAssetsRequest

Request to subscribe to system asset updates

#### Methods

##### Descriptor

Deprecated: Use SubscribeToSystemAssetsRequest.ProtoReflect.Descriptor instead.

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SubscribeToSystemAssetsResponse

Response for subscribing to system asset updates

#### Methods

##### Descriptor

Deprecated: Use SubscribeToSystemAssetsResponse.ProtoReflect.Descriptor instead.

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SubscribeToUserAssetsRequest

Request to subscribe to user asset updates

#### Methods

##### Descriptor

Deprecated: Use SubscribeToUserAssetsRequest.ProtoReflect.Descriptor instead.

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SubscribeToUserAssetsResponse

Response for subscribing to user asset updates

#### Methods

##### Descriptor

Deprecated: Use SubscribeToUserAssetsResponse.ProtoReflect.Descriptor instead.

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UnimplementedAssetServiceServer

UnimplementedAssetServiceServer must be embedded to have forward compatible implementations.

NOTE: this should be embedded by value instead of pointer to avoid a nil pointer dereference when
methods are called.

#### Methods

##### BroadcastSystemAsset

##### CompleteAssetUpload

##### DeleteAsset

##### GetAsset

##### ListSystemAssets

##### ListUserAssets

##### StartHeavyAssetUpload

##### StreamAssetChunk

##### StreamAssetContent

##### SubscribeToSystemAssets

##### SubscribeToUserAssets

##### UploadLightAsset

### UnsafeAssetServiceServer

UnsafeAssetServiceServer may be embedded to opt out of forward compatibility for this service. Use
of this interface is not recommended, as added methods to AssetServiceServer will result in
compilation errors.

### UploadLightAssetRequest

Request to upload a light asset

#### Methods

##### Descriptor

Deprecated: Use UploadLightAssetRequest.ProtoReflect.Descriptor instead.

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UploadLightAssetResponse

Response for uploading a light asset

#### Methods

##### Descriptor

Deprecated: Use UploadLightAssetResponse.ProtoReflect.Descriptor instead.

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

## Functions

### RegisterAssetServiceServer
