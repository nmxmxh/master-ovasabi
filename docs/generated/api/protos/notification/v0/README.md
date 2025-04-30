# Package notification

## Constants

### NotificationService_CreateNotification_FullMethodName

## Variables

### File_api_protos_notification_v0_notification_proto

### NotificationService_ServiceDesc

NotificationService_ServiceDesc is the grpc.ServiceDesc for NotificationService service. It's only
intended for direct use with grpc.RegisterService, and not to be introspected or modified (even as a
copy)

## Types

### CreateNotificationRequest

#### Methods

##### Descriptor

Deprecated: Use CreateNotificationRequest.ProtoReflect.Descriptor instead.

##### GetBody

##### GetCampaignId

##### GetChannel

##### GetMasterId

##### GetPayload

##### GetTitle

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateNotificationResponse

#### Methods

##### Descriptor

Deprecated: Use CreateNotificationResponse.ProtoReflect.Descriptor instead.

##### GetNotification

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetNotificationHistoryRequest

GetNotificationHistoryRequest represents the request to get notification history

#### Methods

##### Descriptor

Deprecated: Use GetNotificationHistoryRequest.ProtoReflect.Descriptor instead.

##### GetEndDate

##### GetPage

##### GetPageSize

##### GetStartDate

##### GetType

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetNotificationHistoryResponse

GetNotificationHistoryResponse represents the response containing notification history

#### Methods

##### Descriptor

Deprecated: Use GetNotificationHistoryResponse.ProtoReflect.Descriptor instead.

##### GetNotifications

##### GetPage

##### GetTotalCount

##### GetTotalPages

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetNotificationRequest

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

### ListNotificationsRequest

#### Methods

##### Descriptor

Deprecated: Use ListNotificationsRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetPage

##### GetPageSize

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

#### Methods

##### Descriptor

Deprecated: Use Notification.ProtoReflect.Descriptor instead.

##### GetBody

##### GetCampaignId

##### GetChannel

##### GetCreatedAt

##### GetId

##### GetMasterId

##### GetPayload

##### GetRead

##### GetTitle

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### NotificationHistory

NotificationHistory represents a single notification record

#### Methods

##### Descriptor

Deprecated: Use NotificationHistory.ProtoReflect.Descriptor instead.

##### GetContent

##### GetCreatedAt

##### GetId

##### GetMetadata

##### GetStatus

##### GetType

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### NotificationPreferences

NotificationPreferences represents user notification preferences

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

### NotificationServiceServer

NotificationServiceServer is the server API for NotificationService service. All implementations
must embed UnimplementedNotificationServiceServer for forward compatibility

### SendEmailRequest

SendEmailRequest represents the request to send an email

#### Methods

##### Descriptor

Deprecated: Use SendEmailRequest.ProtoReflect.Descriptor instead.

##### GetBody

##### GetHtml

##### GetMetadata

##### GetSubject

##### GetTo

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SendEmailResponse

SendEmailResponse represents the response from sending an email

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

### SendPushNotificationRequest

SendPushNotificationRequest represents the request to send a push notification

#### Methods

##### Descriptor

Deprecated: Use SendPushNotificationRequest.ProtoReflect.Descriptor instead.

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

SendPushNotificationResponse represents the response from sending a push notification

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

SendSMSRequest represents the request to send an SMS

#### Methods

##### Descriptor

Deprecated: Use SendSMSRequest.ProtoReflect.Descriptor instead.

##### GetMessage

##### GetMetadata

##### GetTo

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SendSMSResponse

SendSMSResponse represents the response from sending an SMS

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

### UnimplementedNotificationServiceServer

UnimplementedNotificationServiceServer must be embedded to have forward compatible implementations.

#### Methods

##### CreateNotification

##### GetNotification

##### GetNotificationHistory

##### ListNotifications

##### SendEmail

##### SendPushNotification

##### SendSMS

##### UpdateNotificationPreferences

### UnsafeNotificationServiceServer

UnsafeNotificationServiceServer may be embedded to opt out of forward compatibility for this
service. Use of this interface is not recommended, as added methods to NotificationServiceServer
will result in compilation errors.

### UpdateNotificationPreferencesRequest

UpdateNotificationPreferencesRequest represents the request to update notification preferences

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

UpdateNotificationPreferencesResponse represents the response from updating notification preferences

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
