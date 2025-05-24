# Package nexus

Package nexus provides the repository layer for the Nexus Service. See docs/services/nexus.md and
api/protos/nexus/v1/nexus_service.proto for full context.

## Types

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
