# Package repository

## Variables

### ErrNotificationNotFound

## Types

### AssetChunk

--- Asset Chunk Storage (Optional, stub) ---.

### Notification

Notification represents a notification entry in the service_notification table.

### NotificationEvent

--- Notification Event Analytics/Audit ---.

### NotificationRepository

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

### NotificationStatus

NotificationStatus represents the status of a notification.

### NotificationType

NotificationType represents the type of notification.
