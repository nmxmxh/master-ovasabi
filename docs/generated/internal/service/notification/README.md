# Package notification

## Types

### NotificationType

NotificationType represents the type of notification

### QueuedNotification

QueuedNotification represents a notification in the queue

### Service

Service implements the NotificationService gRPC interface.

#### Methods

##### GetNotificationHistory

GetNotificationHistory implements the GetNotificationHistory RPC method.

##### SendEmail

SendEmail implements the SendEmail RPC method.

##### SendPushNotification

SendPushNotification implements the SendPushNotification RPC method.

##### SendSMS

SendSMS implements the SendSMS RPC method.

##### UpdateNotificationPreferences

UpdateNotificationPreferences implements the UpdateNotificationPreferences RPC method.

## Functions

### NewNotificationService

NewNotificationService creates a new instance of NotificationService.
