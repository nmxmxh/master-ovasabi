# Package mediav1

## Constants

### MediaService_UploadLightMedia_FullMethodName

## Variables

### MediaType_name

Enum value maps for MediaType.

### File_media_v1_media_proto

### MediaService_ServiceDesc

MediaService_ServiceDesc is the grpc.ServiceDesc for MediaService service. It's only intended for
direct use with grpc.RegisterService, and not to be introspected or modified (even as a copy)

## Types

### BroadcastSystemMediaRequest

Request to broadcast a system media

#### Methods

##### Descriptor

Deprecated: Use BroadcastSystemMediaRequest.ProtoReflect.Descriptor instead.

##### GetId

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### BroadcastSystemMediaResponse

Response for broadcasting a system media

#### Methods

##### Descriptor

Deprecated: Use BroadcastSystemMediaResponse.ProtoReflect.Descriptor instead.

##### GetError

##### GetStatus

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CompleteMediaUploadRequest

Request to complete an media upload

#### Methods

##### Descriptor

Deprecated: Use CompleteMediaUploadRequest.ProtoReflect.Descriptor instead.

##### GetUploadId

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CompleteMediaUploadResponse

Response for completing an media upload

#### Methods

##### Descriptor

Deprecated: Use CompleteMediaUploadResponse.ProtoReflect.Descriptor instead.

##### GetError

##### GetMedia

##### GetStatus

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### DeleteMediaRequest

Request to delete an media

#### Methods

##### Descriptor

Deprecated: Use DeleteMediaRequest.ProtoReflect.Descriptor instead.

##### GetId

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### DeleteMediaResponse

Response for deleting an media

#### Methods

##### Descriptor

Deprecated: Use DeleteMediaResponse.ProtoReflect.Descriptor instead.

##### GetError

##### GetId

##### GetStatus

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetMediaRequest

Request to get an media

#### Methods

##### Descriptor

Deprecated: Use GetMediaRequest.ProtoReflect.Descriptor instead.

##### GetId

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetMediaResponse

Response for getting an media

#### Methods

##### Descriptor

Deprecated: Use GetMediaResponse.ProtoReflect.Descriptor instead.

##### GetError

##### GetMedia

##### GetStatus

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListSystemMediaRequest

Request to list system media

#### Methods

##### Descriptor

Deprecated: Use ListSystemMediaRequest.ProtoReflect.Descriptor instead.

##### GetFilters

##### GetPageSize

##### GetPageToken

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListSystemMediaResponse

Response for listing system media

#### Methods

##### Descriptor

Deprecated: Use ListSystemMediaResponse.ProtoReflect.Descriptor instead.

##### GetError

##### GetMedia

##### GetNextPageToken

##### GetStatus

##### GetTotalCount

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListUserMediaRequest

Request to list user media

#### Methods

##### Descriptor

Deprecated: Use ListUserMediaRequest.ProtoReflect.Descriptor instead.

##### GetFilters

##### GetPageSize

##### GetPageToken

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListUserMediaResponse

Response for listing user media

#### Methods

##### Descriptor

Deprecated: Use ListUserMediaResponse.ProtoReflect.Descriptor instead.

##### GetError

##### GetMedia

##### GetNextPageToken

##### GetStatus

##### GetTotalCount

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Media

Media represents a media file and its metadata

#### Methods

##### Descriptor

Deprecated: Use Media.ProtoReflect.Descriptor instead.

##### GetCreatedAt

##### GetData

##### GetDeletedAt

##### GetId

##### GetIsSystem

##### GetMasterId

##### GetMetadata

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

### MediaChunk

Chunk of media data for streaming

#### Methods

##### Descriptor

Deprecated: Use MediaChunk.ProtoReflect.Descriptor instead.

##### GetChecksum

##### GetData

##### GetSequence

##### GetUploadId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### MediaServiceClient

MediaServiceClient is the client API for MediaService service.

For semantics around ctx use and closing/ending streaming RPCs, please refer to
https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.

MediaService handles storage and retrieval of media files (images, videos, 3D assets, etc.)

### MediaServiceServer

MediaServiceServer is the server API for MediaService service. All implementations must embed
UnimplementedMediaServiceServer for forward compatibility.

MediaService handles storage and retrieval of media files (images, videos, 3D assets, etc.)

### MediaType

Media types

#### Methods

##### Descriptor

##### Enum

##### EnumDescriptor

Deprecated: Use MediaType.Descriptor instead.

##### Number

##### String

##### Type

### StartHeavyMediaUploadRequest

Request to start a heavy media upload

#### Methods

##### Descriptor

Deprecated: Use StartHeavyMediaUploadRequest.ProtoReflect.Descriptor instead.

##### GetMetadata

##### GetMimeType

##### GetName

##### GetSize

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### StartHeavyMediaUploadResponse

Response for starting a heavy media upload

#### Methods

##### Descriptor

Deprecated: Use StartHeavyMediaUploadResponse.ProtoReflect.Descriptor instead.

##### GetChunkSize

##### GetChunksTotal

##### GetError

##### GetStatus

##### GetUploadId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### StreamMediaChunkRequest

Request to stream media chunks

#### Methods

##### Descriptor

Deprecated: Use StreamMediaChunkRequest.ProtoReflect.Descriptor instead.

##### GetChunk

##### GetUploadId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### StreamMediaChunkResponse

Response for streaming media chunks

#### Methods

##### Descriptor

Deprecated: Use StreamMediaChunkResponse.ProtoReflect.Descriptor instead.

##### GetError

##### GetSequence

##### GetStatus

##### GetUploadId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### StreamMediaContentRequest

Request to stream media content

#### Methods

##### Descriptor

Deprecated: Use StreamMediaContentRequest.ProtoReflect.Descriptor instead.

##### GetId

##### GetLength

##### GetOffset

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### StreamMediaContentResponse

Response for streaming media content

#### Methods

##### Descriptor

Deprecated: Use StreamMediaContentResponse.ProtoReflect.Descriptor instead.

##### GetData

##### GetEndOfStream

##### GetError

##### GetLength

##### GetOffset

##### GetStatus

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SubscribeToSystemMediaRequest

Request to subscribe to system media updates

#### Methods

##### Descriptor

Deprecated: Use SubscribeToSystemMediaRequest.ProtoReflect.Descriptor instead.

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SubscribeToSystemMediaResponse

Response for subscribing to system media updates

#### Methods

##### Descriptor

Deprecated: Use SubscribeToSystemMediaResponse.ProtoReflect.Descriptor instead.

##### GetError

##### GetMedia

##### GetStatus

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SubscribeToUserMediaRequest

Request to subscribe to user media updates

#### Methods

##### Descriptor

Deprecated: Use SubscribeToUserMediaRequest.ProtoReflect.Descriptor instead.

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SubscribeToUserMediaResponse

Response for subscribing to user media updates

#### Methods

##### Descriptor

Deprecated: Use SubscribeToUserMediaResponse.ProtoReflect.Descriptor instead.

##### GetError

##### GetMedia

##### GetStatus

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UnimplementedMediaServiceServer

UnimplementedMediaServiceServer must be embedded to have forward compatible implementations.

NOTE: this should be embedded by value instead of pointer to avoid a nil pointer dereference when
methods are called.

#### Methods

##### BroadcastSystemMedia

##### CompleteMediaUpload

##### DeleteMedia

##### GetMedia

##### ListSystemMedia

##### ListUserMedia

##### StartHeavyMediaUpload

##### StreamMediaChunk

##### StreamMediaContent

##### SubscribeToSystemMedia

##### SubscribeToUserMedia

##### UploadLightMedia

### UnsafeMediaServiceServer

UnsafeMediaServiceServer may be embedded to opt out of forward compatibility for this service. Use
of this interface is not recommended, as added methods to MediaServiceServer will result in
compilation errors.

### UploadLightMediaRequest

Request to upload a light media

#### Methods

##### Descriptor

Deprecated: Use UploadLightMediaRequest.ProtoReflect.Descriptor instead.

##### GetData

##### GetMetadata

##### GetMimeType

##### GetName

##### GetSize

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UploadLightMediaResponse

Response for uploading a light media

#### Methods

##### Descriptor

Deprecated: Use UploadLightMediaResponse.ProtoReflect.Descriptor instead.

##### GetError

##### GetMedia

##### GetStatus

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

## Functions

### RegisterMediaServiceServer
