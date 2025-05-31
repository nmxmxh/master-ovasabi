# Package graceful

## Types

### ContextError

ContextError wraps an error with context, gRPC code, and structured fields.

#### Methods

##### Error

##### GRPCStatus

GRPCStatus returns a gRPC status error for this error context.

##### Orchestrate

Orchestrate runs a list of orchestration hooks on error. Each hook is a func(\*ContextError) error.

##### StandardOrchestrate

StandardOrchestrate runs all standard error orchestration steps based on the config.

### ErrorMapEntry

ErrorMapEntry defines a mapping from an error to a gRPC code and message.

### ErrorOrchestrationConfig

ErrorOrchestrationConfig centralizes all standard orchestration options for an error flow.

### SuccessContext

SuccessContext wraps a successful result with context, process metadata, and orchestration options.

#### Methods

##### Orchestrate

Orchestrate runs a list of orchestration hooks on success. Each hook is a func(\*SuccessContext)
error.

##### OrchestrateWithNexus

OrchestrateWithNexus can be used to trigger pattern/workflow orchestration on success.

##### StandardOrchestrate

StandardOrchestrate runs all standard orchestration steps based on the config. Usage:
success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{...}).

##### String

##### ToStatusSuccess

ToStatusSuccess returns a gRPC status for this success context (for info/logging, not error).

##### ToStatusSuccessErr

ToStatusSuccessErr returns a gRPC status error for this success context (for info/logging, not
error).

### SuccessOrchestrationConfig

SuccessOrchestrationConfig centralizes all standard orchestration options for a successful
operation.

## Functions

### RegisterErrorMap

RegisterErrorMap allows services to register error mappings at runtime.

### ToStatusError

ToStatusError converts an error (ContextError or generic) to a gRPC status error.
