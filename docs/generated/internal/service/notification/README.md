# Package notification

## Variables

### ErrNotificationNotFound

### NotificationEventRegistry

## Types

### AssetChunk

--- Asset Chunk Storage (Optional, stub) ---.

### AzureEmailProvider

#### Methods

##### SendEmail

### AzurePushProvider

#### Methods

##### SendPush

### AzureSMSProvider

#### Methods

##### SendSMS

### EmailProvider

### Event

--- Notification Event Analytics/Audit ---.

### EventEmitter

EventEmitter defines the interface for emitting events in the notification service.

### EventHandlerFunc

### EventRegistry

### EventSubscription

### Metadata

### Notification

Notification represents a notification entry in the service_notification table.

### PushProvider

### Repository

NotificationRepository handles operations on the service_notification table.

#### Methods

##### Create

Create inserts a new notification record.

##### CreateBroadcast

--- Broadcast Support --- Treat broadcasts as notifications with channel/type 'broadcast'.

##### Delete

Delete removes a notification and its master record.

##### GetAssetChunks

##### GetBroadcast

##### GetByID

GetByID retrieves a notification by ID.

##### List

List retrieves a paginated list of notifications.

##### ListBroadcasts

##### ListByUserID

ListByUserID retrieves all notifications for a specific user.

##### ListNotificationEvents

##### ListPendingScheduled

ListPendingScheduled retrieves all pending notifications that are scheduled to be sent.

##### LogNotificationEvent

##### StoreAssetChunk

##### Update

Update updates a notification record.

### SMSProvider

### Service

#### Methods

##### AcknowledgeNotification

##### BroadcastEvent

##### GetNotification

##### ListNotificationEvents

##### ListNotifications

##### SendEmail

##### SendNotification

##### SendPushNotification

##### SendSMS

##### StreamAssetChunks

##### SubscribeToEvents

##### UpdateNotificationPreferences

### Status

NotificationStatus represents the status of a notification.

### Type

NotificationType represents the type of notification.

## Functions

### MetadataToStruct

### NewService

### Register

Register registers the notification service with the DI container and event bus support.

### StartEventSubscribers
