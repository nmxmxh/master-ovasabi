# Package nexusservice

## Types

### OperationPattern

OperationPattern represents a predefined set of Nexus operations.

### OperationStep

OperationStep represents a single step in an operation pattern.

### Option

Option is a function that modifies Options.

### Options

Options configures the Nexus service.

### PatternCategory

PatternCategory defines the category of pattern.

### PatternExecutor

PatternExecutor handles the execution of operation patterns.

#### Methods

##### ExecutePattern

ExecutePattern runs a registered pattern with provided input data.

##### RegisterPattern

RegisterPattern adds a new operation pattern.

### PatternOrigin

PatternOrigin defines where a pattern originated from.

### PatternStore

PatternStore manages pattern storage and retrieval.

#### Methods

##### GetPattern

GetPattern retrieves a pattern by ID.

##### ListPatterns

ListPatterns retrieves patterns based on filters.

##### StorePattern

StorePattern stores a new pattern or updates an existing one.

##### UpdatePatternStats

UpdatePatternStats updates pattern usage statistics.

##### ValidatePattern

ValidatePattern validates a pattern.

### PatternValidationResult

PatternValidationResult represents the result of pattern validation.

### StoredPattern

StoredPattern represents a pattern stored in the system.

## Functions

### RegisterServicePattern

RegisterServicePattern modularly registers a service as a pattern in the Nexus orchestrator. This
enables orchestration, introspection, and pattern-based automation for the service in the system.
