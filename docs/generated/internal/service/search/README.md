# Package search

## Variables

### SearchEventRegistry

## Types

### EventEmitter

EventEmitter defines the interface for emitting events (canonical platform interface).

### EventHandlerFunc

### EventRegistry

### EventSubscription

### Index

### Repository

#### Methods

##### SearchAllEntities

SearchAllEntities performs FTS and metadata filtering across multiple entity tables (content,
campaign, user, talent). It merges and returns results in a unified format. The 'types' argument
specifies which entity types to search.

##### SearchEntities

SearchEntities performs advanced full-text and fuzzy search on the master table. Supports filtering
by entityType, query, masterID, fields, metadata, fuzzy, and language.

### Result

Result matches the proto definition.

### Service

#### Methods

##### Search

Search implements robust multi-entity, FTS, and metadata filtering search. Supports searching across
multiple entity types as specified in req.Types.

## Functions

### NewService

NewService creates a new SearchService instance with event bus and provider support (canonical
pattern).

### Register

Register registers the search service with the DI container and event bus support (canonical
pattern).

### StartEventSubscribers
