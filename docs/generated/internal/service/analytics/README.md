# Package analytics

## Variables

### AnalyticsEventRegistry

### ErrEventNotFound

Define a package-level error for event not found.

## Types

### Event

### EventEmitter

EventEmitter defines the interface for emitting events.

### EventHandlerFunc

### EventRegistry

### EventSubscription

### Repository

PostgresRepository provides analytics event storage.

#### Methods

##### BatchTrackEvents

##### GetProductEvents

##### GetReport

##### GetUserEvents

##### ListReports

##### TrackEvent

### RepositoryItf

### Service

#### Methods

##### BatchTrackEvents

##### CaptureEvent

CaptureEvent ingests a new analytics event with robust, GDPR-compliant metadata.

##### EnrichEventMetadata

EnrichEventMetadata allows for post-hoc enrichment of event metadata.

##### GetProductEvents

##### GetReport

##### GetUserEvents

##### ListEvents

ListEvents returns all captured analytics events (paginated in production).

##### ListReports

##### TrackEvent

## Functions

### BuildAnalyticsMetadata

BuildAnalyticsMetadata builds robust, GDPR-compliant analytics event metadata. If gdprObscure is
true, user/sensitive info is omitted or obscured.

### NewService

### Register

Register registers the analytics service with the DI container and event bus support.

### StartEventSubscribers
