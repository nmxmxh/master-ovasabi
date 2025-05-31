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

### ComposeMetadataFilter

ComposeMetadataFilter builds a SQL filter string and args for service-specific metadata fields.
Compatible with multi-entity search: use in conjunction with the 'types' field to filter per-entity.
It returns a SQL fragment (e.g., " AND metadata->'service_specific'->>'foo' = ?") and the
corresponding args. Extend as needed for entity-type-specific metadata logic.

### ExtractServiceSpecific

ExtractServiceSpecific returns a map of service-specific metadata fields for a given service
namespace.

### NewService

NewService creates a new SearchService instance with event bus support.

### Register

Register registers the search service with the DI container and event bus support.

### StartEventSubscribers

### ValidateMetadataKeys

ValidateMetadataKeys checks that only allowed keys are present in service-specific metadata.
