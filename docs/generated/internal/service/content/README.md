# Package content

## Variables

### ContentEventRegistry

## Types

### EventEmitter

EventEmitter defines the interface for emitting events in the content service.

### EventHandlerFunc

### EventRegistry

### EventSubscription

### Repository

#### Methods

##### AddComment

AddComment adds a comment to content.

##### AddReaction

Reactions.

##### CreateContent

Content CRUD.

##### DeleteComment

DeleteComment deletes a comment by ID.

##### DeleteContent

##### GetComment

GetComment fetches a single comment by ID.

##### GetContent

##### ListComments

ListComments lists comments for a content item.

##### ListContent

##### ListContentFlexible

Flexible ListContent with filters.

##### ListReactions

##### LogContentEvent

LogContentEvent logs a content event for analytics/audit.

##### ModerateContent

Moderation stubs.

##### SearchContent

Full-text/context search with master_id support.

##### SearchContentFlexible

SearchContent with flexible filters.

##### UpdateContent

### Service

#### Methods

##### AddComment

##### AddReaction

##### CreateContent

##### DeleteComment

##### DeleteContent

##### GetContent

##### ListComments

##### ListContent

##### ListReactions

##### LogContentEvent

##### ModerateContent

##### SearchContent

##### UpdateContent

### ServiceMetadata

ServiceMetadata defines the canonical, extensible metadata structure for content entities. This
struct documents all fields expected under metadata.service_specific["content"] in the
common.Metadata proto. Reference: docs/services/metadata.md, docs/amadeus/amadeus_context.md All
extraction and mutation must use canonical helpers from pkg/metadata.

## Functions

### NewService

### Register

Register registers the content service with the DI container and event bus support.

### StartEventSubscribers
