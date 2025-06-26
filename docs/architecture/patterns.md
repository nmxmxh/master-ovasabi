# Documentation

version: 2025-05-14

version: 2025-05-14

version: 2025-05-14

## Overview

The pattern system provides a flexible and reusable way to define and execute complex operations in
the application. It supports both system-defined and user-defined patterns, with proper versioning,
categorization, and execution tracking.

## Components

### Pattern Store

The pattern store manages the storage and retrieval of patterns in Redis. It provides:

- Pattern storage with versioning
- Pattern retrieval by ID
- Pattern listing with filters
- Pattern statistics tracking
- Pattern deletion

### Pattern Executor

The pattern executor handles the execution of patterns. Key features include:

- Dependency-based execution ordering
- Concurrent step execution
- Step-level timeout and retry logic
- Transaction support
- Execution statistics tracking

### Pattern Types

#### System Patterns

- Predefined patterns for common operations
- Managed by the system
- Higher security and reliability requirements
- Examples: financial transactions, user onboarding

#### User Patterns

- Custom patterns defined by users
- Validated before storage
- Limited scope and permissions
- Examples: custom notifications, data transformations

## Pattern Structure

### Pattern Definition

```json
{
  "id": "string",
  "name": "string",
  "description": "string",
  "version": "integer",
  "origin": "system|user",
  "category": "finance|notification|user|asset|broadcast|referral",
  "steps": [
    {
      "type": "string",
      "action": "string",
      "parameters": {},
      "depends_on": ["string"],
      "retries": "integer",
      "timeout": "duration"
    }
  ],
  "metadata": {},
  "created_by": "string",
  "created_at": "timestamp",
  "updated_at": "timestamp",
  "is_active": "boolean",
  "usage_count": "integer",
  "success_rate": "float"
}
```

### Step Types

1. Cache Operations

   - get: Retrieve data from cache
   - set: Store data in cache
   - delete: Remove data from cache

2. Pipeline Operations

   - Multiple cache operations in a single round trip
   - Atomic execution
   - Improved performance

3. Transaction Operations
   - Multiple operations in a transaction
   - Automatic rollback on failure
   - Data consistency guarantee

## Pattern Execution

### Execution Flow

1. Pattern retrieval from store
2. Dependency analysis and execution plan creation
3. Concurrent step execution with dependency ordering
4. Result collection and error handling
5. Statistics update

### Error Handling

- Step-level retries with configurable attempts
- Step-level timeouts
- Transaction rollback
- Execution statistics tracking
- Detailed error logging

### Performance Considerations

- Concurrent step execution
- Connection pooling
- Pipeline operations
- Batch processing
- Resource limits

## Security

### Access Control

- Pattern origin validation
- User permission checks
- Resource usage limits
- Input validation

### Data Protection

- Pattern versioning
- Audit logging
- Secure storage
- Execution isolation

## Monitoring

### Metrics

- Pattern usage count
- Success rate
- Execution time
- Error rate
- Resource usage

### Logging

- Pattern execution events
- Error details
- Performance metrics
- Audit trail

## Integration

### Service Integration

```go
// Initialize pattern support
provider.RegisterPatternCache()
if err := provider.InitializePatternSupport(); err != nil {
    log.Fatal("Failed to initialize pattern support", zap.Error(err))
}

// Get pattern store and executor
store := provider.GetPatternStore()
executor := provider.GetPatternExecutor()

// Execute pattern
results, err := executor.ExecutePattern(ctx, patternID, input)
if err != nil {
    log.Error("Pattern execution failed", zap.Error(err))
    return err
}
```

### Example Usage

```go
// Create and store a pattern
pattern := &StoredPattern{
    ID:          "user_onboarding",
    Name:        "User Onboarding Flow",
    Description: "Creates a new user with wallet and referral relationships",
    Version:     1,
    Origin:      PatternOriginSystem,
    Category:    CategoryUser,
    Steps: []OperationStep{
        {
            Type:   "cache",
            Action: "set",
            Parameters: map[string]interface{}{
                "key":   "user:{id}",
                "value": "{user_data}",
                "ttl":   "24h",
            },
            Retries: 3,
            Timeout: 10 * time.Second,
        },
        // Additional steps...
    },
    Metadata: map[string]interface{}{
        "criticality": "high",
        "audit":       true,
    },
}

if err := store.StorePattern(ctx, pattern); err != nil {
    log.Error("Failed to store pattern", zap.Error(err))
    return err
}

// Execute pattern
input := map[string]interface{}{
    "id":        userID,
    "user_data": userData,
}

results, err := executor.ExecutePattern(ctx, pattern.ID, input)
if err != nil {
    log.Error("Pattern execution failed", zap.Error(err))
    return err
}
```

## Best Practices

1. Pattern Design

   - Keep patterns focused and single-purpose
   - Use meaningful names and descriptions
   - Include proper error handling
   - Set appropriate timeouts and retries
   - Document dependencies and requirements

2. Performance

   - Use pipeline operations when possible
   - Set appropriate batch sizes
   - Monitor resource usage
   - Use caching effectively
   - Implement proper cleanup

3. Maintenance

   - Version patterns appropriately
   - Monitor pattern usage and success rates
   - Clean up unused patterns
   - Update documentation
   - Regular testing

4. Security
   - Validate all inputs
   - Implement proper access control
   - Monitor for abuse
   - Regular security audits
   - Keep dependencies updated
