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

## Functions

### BuildContentMetadata

BuildContentMetadata builds a canonical content metadata struct for storage, analytics, and
extensibility.

### NewService

### Register

Register registers the content service with the DI container and event bus support.

### StartEventSubscribers
