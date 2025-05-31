# Package metaversion

Package metaversion provides canonical versioning and feature flag management for the OVASABI
platform.

## Constants

### InitialVersion

InitialVersion is the default version for all fields at package init.

## Types

### Evaluator

Evaluator defines the interface for feature flag and AB test evaluation.

### OpenFeatureEvaluator

OpenFeatureEvaluator implements Evaluator using OpenFeature.

#### Methods

##### AssignABTest

AssignABTest deterministically assigns a user to an A/B group.

##### EvaluateFlags

EvaluateFlags returns the enabled feature flags for a user.

### Versioning

Versioning holds all versioning and feature flag metadata for a user/session/entity.

#### Methods

##### ToMap

ToMap converts Versioning to a map for embedding in metadata or JWT claims.

## Functions

### InjectContext

InjectContext returns a new context with the given Versioning.

### MergeMetadata

MergeMetadata merges Versioning into a metadata map under service_specific.user.versioning.

### Middleware

Middleware returns an HTTP middleware that injects and validates Versioning in the request context.

### NowUTC

NowUTC returns the current time in UTC. Used for testability.

### ValidateVersioning

ValidateVersioning checks that all version fields are semantic and required fields are set.
