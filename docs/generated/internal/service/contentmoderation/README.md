# Package contentmoderation

## Variables

### ContentModerationEventRegistry

## Types

### EventEmitter

EventEmitter defines the interface for emitting events in the content moderation service.

### EventHandlerFunc

### EventRegistry

### EventSubscription

### Metadata

Metadata defines the canonical, extensible metadata structure for content moderation entities. This
struct documents all fields expected under metadata.service_specific["contentmoderation"] in the
common.Metadata proto. Reference: docs/services/metadata.md, docs/amadeus/amadeus_context.md All
extraction and mutation must use canonical helpers from pkg/metadata.

### Moderation

### ModerationResult

### PostgresRepository

#### Methods

##### ApproveContent

##### GetModerationResult

##### ListFlaggedContent

##### RejectContent

##### SubmitContentForModeration

### Repository

### ReviewerMetadata

ReviewerMetadata documents reviewer information for moderation actions.

### Service

#### Methods

##### ApproveContent

##### GetModerationResult

##### ListFlaggedContent

##### RejectContent

##### SubmitContentForModeration

## Functions

### NewContentModerationService

### Register

Register registers the content moderation service with the DI container and event bus support.

### StartEventSubscribers
