# Package nexus

## Types

### CanonicalEvent

CanonicalEvent wraps the existing Event struct and adds extensibility for multi-event orchestration
and metadata.

### Event

Event represents a cross-service event.

### EventRepository

EventRepository defines the interface for event persistence and orchestration.

### Graph

Graph represents a relationship graph.

### RelationType

RelationType defines the type of relationship between entities.

### Relationship

Relationship represents a connection between two master records.

### Repository

Repository defines the interface for Nexus operations.

### SQLEventRepository

SQLEventRepository is a SQL-backed implementation of EventRepository.

#### Methods

##### GetEvent

##### ListEventsByMaster

##### ListEventsByPattern

##### ListPendingEvents

##### SaveEvent

##### UpdateEventStatus
