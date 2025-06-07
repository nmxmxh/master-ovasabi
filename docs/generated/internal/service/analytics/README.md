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

##### CountEventsByType

CountEventsByType returns the number of analytics events with the given event_type.

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

BuildAnalyticsMetadata builds robust, GDPR-compliant analytics event metadata. User information
(userID, userEmail) is always obscured unless gdprObscure is explicitly false. If user info is
provided but gdprObscure is true, a warning is logged using the provided logger. [CANONICAL] All
metadata must be normalized and calculated via metadata.NormalizeAndCalculate before persistence or
emission. Ensure required fields (versioning, audit, etc.) are present under the correct namespace.

### NewService

### Register

Register registers the analytics service with the DI container and event bus support.

### StartEventSubscribers
