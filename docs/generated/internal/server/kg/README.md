# Package server

## Variables

### ErrServiceDegraded

Common errors.

## Types

### KGHooks

KGHooks manages real-time knowledge graph updates.

#### Methods

##### Start

Start begins processing knowledge graph updates.

##### Stop

Stop gracefully shuts down the hooks.

### KGService

KGService manages the knowledge graph service.

#### Methods

##### IsDegraded

IsDegraded returns whether the service is in degraded mode.

##### PublishUpdate

PublishUpdate sends an update to the knowledge graph.

##### RecoverFromDegradedMode

RecoverFromDegradedMode attempts to recover the service from degraded mode.

##### RegisterService

RegisterService registers a new service with the knowledge graph.

##### Start

Start initializes the knowledge graph service.

##### Stop

Stop gracefully shuts down the service.

##### UpdateRelation

UpdateRelation updates a relation in the knowledge graph.

##### UpdateSchema

UpdateSchema updates the schema for a service.

### KGUpdate

KGUpdate represents a knowledge graph update event.

### KGUpdateType

KGUpdateType represents the type of knowledge graph update.
