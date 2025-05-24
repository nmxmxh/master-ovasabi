# Package notificationpb

## Constants

### NotificationService_SendNotification_FullMethodName

## Variables

### NotificationStatus_name

Enum value maps for NotificationStatus.

### File_notification_v1_notification_proto

### NotificationService_ServiceDesc

NotificationService_ServiceDesc is the grpc.ServiceDesc for NotificationService service. It's only
intended for direct use with grpc.RegisterService, and not to be introspected or modified (even as a
copy)

## Types

### AcknowledgeNotificationRequest

#### Methods

##### Descriptor

Deprecated: Use AcknowledgeNotificationRequest.ProtoReflect.Descriptor instead.

##### GetNotificationId

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### AcknowledgeNotificationResponse

#### Methods

##### Descriptor

Deprecated: Use AcknowledgeNotificationResponse.ProtoReflect.Descriptor instead.

##### GetStatus

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### AssetChunk

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

### BroadcastEventRequest

--- Broadcast/Event ---

#### Methods

##### Descriptor

Deprecated: Use BroadcastEventRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetChannel

##### GetMessage

##### GetPayload

##### GetScheduledAt

##### GetSubject

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### BroadcastEventResponse

#### Methods

##### Descriptor

Deprecated: Use BroadcastEventResponse.ProtoReflect.Descriptor instead.

##### GetBroadcastId

##### GetCampaignId

##### GetStatus

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetNotificationRequest

--- Notification Management ---

#### Methods

##### Descriptor

Deprecated: Use GetNotificationRequest.ProtoReflect.Descriptor instead.

##### GetNotificationId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetNotificationResponse

#### Methods

##### Descriptor

Deprecated: Use GetNotificationResponse.ProtoReflect.Descriptor instead.

##### GetNotification

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListNotificationEventsRequest

--- Analytics/Events ---

#### Methods

##### Descriptor

Deprecated: Use ListNotificationEventsRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetNotificationId

##### GetPage

##### GetPageSize

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListNotificationEventsResponse

#### Methods

##### Descriptor

Deprecated: Use ListNotificationEventsResponse.ProtoReflect.Descriptor instead.

##### GetEvents

##### GetTotal

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListNotificationsRequest

#### Methods

##### Descriptor

Deprecated: Use ListNotificationsRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetChannel

##### GetPage

##### GetPageSize

##### GetStatus

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListNotificationsResponse

#### Methods

##### Descriptor

Deprecated: Use ListNotificationsResponse.ProtoReflect.Descriptor instead.

##### GetNotifications

##### GetPage

##### GetTotalCount

##### GetTotalPages

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Notification

--- Notification Core ---

#### Methods

##### Descriptor

Deprecated: Use Notification.ProtoReflect.Descriptor instead.

##### GetBody

##### GetCampaignId

##### GetChannel

##### GetCreatedAt

##### GetId

##### GetMasterId

##### GetMasterUuid

##### GetMetadata

##### GetPayload

##### GetRead

##### GetStatus

##### GetTitle

##### GetUpdatedAt

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### NotificationEvent

#### Methods

##### Descriptor

Deprecated: Use NotificationEvent.ProtoReflect.Descriptor instead.

##### GetCreatedAt

##### GetEventId

##### GetEventType

##### GetNotificationId

##### GetPayload

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### NotificationPreferences

--- Preferences ---

#### Methods

##### Descriptor

Deprecated: Use NotificationPreferences.ProtoReflect.Descriptor instead.

##### GetEmailEnabled

##### GetNotificationTypes

##### GetPushEnabled

##### GetQuietHours

##### GetSmsEnabled

##### GetTimezone

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### NotificationServiceClient

NotificationServiceClient is the client API for NotificationService service.

For semantics around ctx use and closing/ending streaming RPCs, please refer to
https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.

Unified NotificationService: handles notifications, broadcasts, real-time events, and asset
streaming

### NotificationServiceServer

NotificationServiceServer is the server API for NotificationService service. All implementations
must embed UnimplementedNotificationServiceServer for forward compatibility.

Unified NotificationService: handles notifications, broadcasts, real-time events, and asset
streaming

### NotificationService_StreamAssetChunksClient

This type alias is provided for backwards compatibility with existing code that references the prior
non-generic stream type by name.

### NotificationService_StreamAssetChunksServer

This type alias is provided for backwards compatibility with existing code that references the prior
non-generic stream type by name.

### NotificationService_SubscribeToEventsClient

This type alias is provided for backwards compatibility with existing code that references the prior
non-generic stream type by name.

### NotificationService_SubscribeToEventsServer

This type alias is provided for backwards compatibility with existing code that references the prior
non-generic stream type by name.

### NotificationStatus

#### Methods

##### Descriptor

##### Enum

##### EnumDescriptor

Deprecated: Use NotificationStatus.Descriptor instead.

##### Number

##### String

##### Type

### PublishAssetChunkRequest

#### Methods

##### Descriptor

Deprecated: Use PublishAssetChunkRequest.ProtoReflect.Descriptor instead.

##### GetAssetId

##### GetChunk

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### PublishAssetChunkResponse

#### Methods

##### Descriptor

Deprecated: Use PublishAssetChunkResponse.ProtoReflect.Descriptor instead.

##### GetStatus

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SendEmailRequest

--- Channel-specific (compatibility) ---

#### Methods

##### Descriptor

Deprecated: Use SendEmailRequest.ProtoReflect.Descriptor instead.

##### GetBody

##### GetCampaignId

##### GetHtml

##### GetMetadata

##### GetSubject

##### GetTo

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SendEmailResponse

#### Methods

##### Descriptor

Deprecated: Use SendEmailResponse.ProtoReflect.Descriptor instead.

##### GetMessageId

##### GetSentAt

##### GetStatus

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SendNotificationRequest

#### Methods

##### Descriptor

Deprecated: Use SendNotificationRequest.ProtoReflect.Descriptor instead.

##### GetBody

##### GetCampaignId

##### GetChannel

##### GetMetadata

##### GetPayload

##### GetTitle

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SendNotificationResponse

#### Methods

##### Descriptor

Deprecated: Use SendNotificationResponse.ProtoReflect.Descriptor instead.

##### GetNotification

##### GetStatus

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SendPushNotificationRequest

#### Methods

##### Descriptor

Deprecated: Use SendPushNotificationRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetDeepLink

##### GetMessage

##### GetMetadata

##### GetTitle

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SendPushNotificationResponse

#### Methods

##### Descriptor

Deprecated: Use SendPushNotificationResponse.ProtoReflect.Descriptor instead.

##### GetNotificationId

##### GetSentAt

##### GetStatus

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SendSMSRequest

#### Methods

##### Descriptor

Deprecated: Use SendSMSRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetMessage

##### GetMetadata

##### GetTo

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SendSMSResponse

#### Methods

##### Descriptor

Deprecated: Use SendSMSResponse.ProtoReflect.Descriptor instead.

##### GetMessageId

##### GetSentAt

##### GetStatus

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### StreamAssetChunksRequest

--- Asset Streaming ---

#### Methods

##### Descriptor

Deprecated: Use StreamAssetChunksRequest.ProtoReflect.Descriptor instead.

##### GetAssetId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SubscribeToEventsRequest

--- Real-time Pub/Sub ---

#### Methods

##### Descriptor

Deprecated: Use SubscribeToEventsRequest.ProtoReflect.Descriptor instead.

##### GetChannels

##### GetFilters

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UnimplementedNotificationServiceServer

UnimplementedNotificationServiceServer must be embedded to have forward compatible implementations.

NOTE: this should be embedded by value instead of pointer to avoid a nil pointer dereference when
methods are called.

#### Methods

##### AcknowledgeNotification

##### BroadcastEvent

##### GetNotification

##### ListNotificationEvents

##### ListNotifications

##### PublishAssetChunk

##### SendEmail

##### SendNotification

##### SendPushNotification

##### SendSMS

##### StreamAssetChunks

##### SubscribeToEvents

##### UpdateNotificationPreferences

### UnsafeNotificationServiceServer

UnsafeNotificationServiceServer may be embedded to opt out of forward compatibility for this
service. Use of this interface is not recommended, as added methods to NotificationServiceServer
will result in compilation errors.

### UpdateNotificationPreferencesRequest

#### Methods

##### Descriptor

Deprecated: Use UpdateNotificationPreferencesRequest.ProtoReflect.Descriptor instead.

##### GetPreferences

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UpdateNotificationPreferencesResponse

#### Methods

##### Descriptor

Deprecated: Use UpdateNotificationPreferencesResponse.ProtoReflect.Descriptor instead.

##### GetPreferences

##### GetUpdatedAt

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

## Functions

### RegisterNotificationServiceServer
