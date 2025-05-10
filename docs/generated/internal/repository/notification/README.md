# Package repository

## Variables

### ErrNotificationNotFound

## Types

### Notification

Notification represents a notification entry in the service_notification table.

### NotificationRepository

NotificationRepository handles operations on the service_notification table.

#### Methods

##### Create

Create inserts a new notification record.

##### Delete

Delete removes a notification and its master record.

##### GetByID

GetByID retrieves a notification by ID.

##### List

List retrieves a paginated list of notifications.

##### ListByUserID

ListByUserID retrieves all notifications for a specific user.

##### ListPendingScheduled

ListPendingScheduled retrieves all pending notifications that are scheduled to be sent.

##### Update

Update updates a notification record.

### NotificationStatus

NotificationStatus represents the status of a notification.

### NotificationType

NotificationType represents the type of notification.

## Functions

### SetLogger
