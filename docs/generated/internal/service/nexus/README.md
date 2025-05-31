# Package nexus

Package nexus provides helpers for robust, extensible metadata handling in the Nexus service. This
file defines the canonical metadata structure, helpers for extraction/validation. and query
utilities for rich analytics and orchestration.

Package nexus provides the repository layer for the Nexus Service. See docs/services/nexus.md and
api/protos/nexus/v1/nexus_service.proto for full context.

## Constants

### EventUserCreated

--- User events ---.

### EventAdminUserCreated

--- Admin events ---.

### EventCampaignCreated

--- Campaign events ---.

### EventContentCreated

--- Content events ---.

### EventNotificationSent

--- Notification events ---.

### EventReferralCreated

--- Referral events ---.

### EventSecurityAuthenticated

--- Security events ---.

### EventCommerceQuoteCreated

--- Commerce events ---.

### EventAnalyticsEventTracked

--- Analytics events ---.

### EventMessagingSent

--- Messaging events ---.

### EventMediaUploaded

--- Media events ---.

### EventSchedulerJobCreated

--- Scheduler events ---.

### EventLocalizationTranslated

--- Localization events ---.

### EventSearchPerformed

--- Search events ---.

### EventTalentProfileCreated

--- Talent events ---.

### EventContentModerationSubmitted

--- Content Moderation events ---.

### EventProductCreated

--- Product events ---.

### EventNexusCircuitBreakerTripped

--- Resilience & Orchestration events ---.

### EventNexusPatternRegistered

--- Nexus events ---.

## Types

### Metadata

NexusMetadata is the canonical metadata struct for patterns, orchestrations, and mining.

### Repository

Repository handles pattern registration, orchestration, mining, and feedback.

#### Methods

##### AddTraceStep

AddTraceStep inserts a trace step for an orchestration.

##### FacetPatternTags

FacetPatternTags returns a count of patterns per tag for faceted search.

##### Feedback

Feedback records feedback for a pattern.

##### GetPatternRequirements

GetPatternRequirements extracts requirements from a pattern's definition.

##### InsertEvent

InsertEvent persists an event to the service_nexus_event table.

##### ListMinedPatterns

ListMinedPatterns returns all mined patterns.

##### ListPatterns

ListPatterns returns all patterns, optionally filtered by type.

##### MineAndStorePatterns

MineAndStorePatterns analyzes trace data to discover frequent (service, action) patterns and stores
them.

##### MinePatterns

MinePatterns returns mined patterns if source == "mined", otherwise queries by origin.

##### Orchestrate

Orchestrate executes a pattern and records the orchestration, with rollback on failure.

##### RegisterPattern

RegisterPattern inserts or updates a pattern in the database, with provenance.

##### SearchOrchestrationsByMetadata

SearchOrchestrationsByMetadata finds orchestrations matching a metadata key path and value.

##### SearchPatternsByJSONPath

SearchPatternsByJSONPath supports arbitrary-depth metadata search using JSONPath (Postgres 12+).

##### SearchPatternsByMetadata

SearchPatternsByMetadata finds patterns matching a metadata key path and value. Supports top-level
and service_specific metadata (e.g., "tags", "service_specific.content.editor_mode").

##### SearchPatternsExplainable

SearchPatternsExplainable returns patterns and the matching metadata fields for explainability.

##### TracePattern

TracePattern returns the trace for a given orchestration.

##### ValidatePattern

ValidatePattern checks if input satisfies pattern requirements.

### Service

Service implements the NexusServiceServer gRPC interface and business logic, fully
repository-backed.

#### Methods

##### EmitEvent

EmitEvent handles event emission to the Nexus event bus with structured logging and persistence.

##### Feedback

##### HandleOps

##### ListPatterns

##### MinePatterns

##### Orchestrate

##### RegisterPattern

##### SubscribeEvents

SubscribeEvents handles event subscriptions with structured logging.

##### TracePattern

## Functions

### BuildEventMetadata

Helper to build standard event metadata.

### BuildEventType

Helper to build event type strings dynamically.

### ExtractTagFilter

ExtractTagFilter builds a SQL filter for tags.

### NewNexusClient

NewNexusClient creates a new gRPC client connection and returns a NexusServiceClient and a cleanup
function.

### NewNexusService

NewNexusService constructs a new NexusServiceServer instance.

### NewNexusServiceProvider

Canonical provider function for DI/bootstrap.

### Register

Register registers the NexusServiceServer with the DI container.

### ValidateNexusMetadata

ValidateNexusMetadata checks for required fields and returns an error if missing.
