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

ContentModerationMetadata for robust, extensible moderation metadata.

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

### Service

#### Methods

##### ApproveContent

##### GetModerationResult

##### ListFlaggedContent

##### RejectContent

##### SubmitContentForModeration

## Functions

### ExtractAndEnrichContentModerationMetadata

ExtractAndEnrichContentModerationMetadata extracts, validates, and enriches moderation metadata.

### NewContentModerationService

### Register

Register registers the content moderation service with the DI container and event bus support.

### StartEventSubscribers
