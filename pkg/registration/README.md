# Dynamic Service Registration System

This package provides a dynamic service registration system that can automatically generate service registration configurations by analyzing proto files, Go code, and runtime service information through reflection and introspection.

## Overview

The dynamic service registration system addresses the question: **"Can't we dynamically create the service_registration config?"**

The answer is **YES**, and this package provides multiple approaches:

1. **Proto-based Generation**: Analyze `.proto` files to extract service definitions, methods, and messages
2. **Code Introspection**: Use Go reflection to analyze service interfaces and implementations
3. **Runtime Inspection**: Inspect running services to validate and enhance configurations
4. **Hybrid Approach**: Combine all three methods for comprehensive service registration

## Architecture

```text
pkg/registration/
├── generator.go              # Core dynamic generation logic
├── inspector.go              # Runtime inspection and validation
├── registration.go           # Existing registration patterns (enhanced)
└── cmd/
    ├── generate/             # Standalone generator tool
    └── registry-inspect-dynamic/  # Enhanced registry inspector
```

## Key Components

### 1. DynamicServiceRegistrationGenerator

Generates service registration configurations by:

- Parsing proto files to extract service definitions
- Inferring capabilities from method names and patterns
- Determining dependencies based on method signatures
- Creating REST endpoint mappings
- Analyzing Go source code for additional metadata

### 2. DynamicInspector

Provides runtime inspection capabilities:

- Validates service registrations against actual implementations
- Compares service configurations
- Generates service dependency graphs
- Provides optimization suggestions
- Gathers runtime performance information

## Usage

### Build Tools

```bash
# Build all dynamic tools
./scripts/build-dynamic-tools.sh
```

### Generate Service Registration from Proto Files

```bash
# Generate from default paths
./bin/service-registration-generator

# Specify custom paths
./bin/service-registration-generator \
  -proto-path api/protos \
  -src-path . \
  -output config/service_registration_generated.json
```

### Enhanced Registry Inspection

```bash
# List all services (classic mode)
./bin/registry-inspect-dynamic -mode services

# List all events
./bin/registry-inspect-dynamic -mode events

# Generate dynamic service registration
./bin/registry-inspect-dynamic -mode generate

# Inspect specific service with detailed information
./bin/registry-inspect-dynamic -mode inspect -service user

# Validate service configuration
./bin/registry-inspect-dynamic -mode validate -service user

# Compare two services
./bin/registry-inspect-dynamic -mode compare -service user -compare admin

# Export service dependency graph
./bin/registry-inspect-dynamic -mode graph -output service_graph.json
```

## Generated Configuration Structure

The system generates service registration configurations that are compatible with the existing system but enhanced with additional metadata:

```json
{
  "name": "user",
  "version": "v1",
  "capabilities": [
    "user_mgmt",
    "authentication", 
    "authorization",
    "metadata_enrichment"
  ],
  "dependencies": ["security", "localization"],
  "schema": {
    "proto_path": "api/protos/user/v1/user.proto",
    "methods": ["CreateUser", "GetUser", "UpdateUser", "DeleteUser"]
  },
  "endpoints": [{
    "path": "/api/user_ops",
    "method": "POST", 
    "actions": ["create_user", "get_user", "update_user", "delete_user"],
    "description": "Composable user operations endpoint..."
  }],
  "models": ["User", "CreateUserRequest", "CreateUserResponse"],
  "health_check": "/health/user",
  "metrics": "/metrics/user",
  "metadata_enrichment": true,
  "action_map": {
    "create_user": {
      "proto_method": "CreateUser",
      "request_model": "CreateUserRequest", 
      "response_model": "CreateUserResponse",
      "rest_required_fields": ["name", "email", "metadata"]
    }
  }
}
```

## Integration with Existing System

### 1. Replace Static Configuration

Instead of manually maintaining `config/service_registration.json`, generate it dynamically:

```go
// In your service bootstrap
generator := registration.NewDynamicServiceRegistrationGenerator(
    logger, "api/protos", ".")
    
if err := generator.GenerateAndSaveConfig(ctx, "config/service_registration.json"); err != nil {
    log.Fatal("Failed to generate service registration", zap.Error(err))
}
```

### 2. Runtime Validation

Validate existing configurations against runtime:

```go
inspector := registration.NewDynamicInspector(logger, container, "api/protos", ".")

for _, config := range existingConfigs {
    result, err := inspector.ValidateServiceRegistration(config)
    if err != nil || !result.IsValid {
        log.Warn("Service configuration issues", 
            zap.String("service", config.Name),
            zap.Strings("issues", result.Issues))
    }
}
```

### 3. Enhanced Registry Inspection

Replace the existing `registry-inspect` tool with the dynamic version for more capabilities.

## Capabilities Inference

The system automatically infers service capabilities based on method patterns:

| Method Pattern | Inferred Capability |
|---------------|-------------------|
| CreateUser, GetUser, UpdateUser | user_mgmt |
| CreateSession, RevokeSession | authentication |
| AssignRole, CheckPermission | authorization |
| SendNotification, SendEmail | notification |
| CreateContent, GetContent | content |
| CreateOrder, InitiatePayment | commerce |
| Search, Suggest | search |

## Dependencies Inference

Dependencies are inferred from:

- Method parameter types (e.g., UserID suggests user dependency)
- Message field names
- Import statements in proto files
- Cross-service method calls in Go code

## Benefits

1. **Eliminates Manual Maintenance**: No need to manually update service registration configurations
2. **Reduces Human Error**: Automatically generates accurate configurations
3. **Ensures Consistency**: All services follow the same registration pattern
4. **Provides Validation**: Validates configurations against actual implementations
5. **Enables Analysis**: Provides dependency graphs and service comparisons
6. **Supports Evolution**: Automatically adapts as services change

## Advanced Features

### Service Dependency Graph

Generate visual dependency graphs:

```bash
./bin/registry-inspect-dynamic -mode graph -output service_graph.json
```

Output includes:

- Service nodes with capabilities
- Dependency relationships
- Circular dependency detection
- Impact analysis

### Configuration Optimization

Get suggestions for optimizing service configurations:

```go
suggestions := inspector.OptimizeServiceRegistration(config)
for _, suggestion := range suggestions.Suggestions {
    log.Info("Optimization suggestion", zap.String("suggestion", suggestion))
}
```

### Runtime Performance Integration

The inspector can gather runtime performance metrics and include them in service registrations for better orchestration decisions.

## Future Enhancements

1. **Protocol Buffer Analysis**: Deeper proto analysis for field-level metadata
2. **OpenAPI Integration**: Generate REST documentation alongside registrations  
3. **Metrics Integration**: Include service metrics in registration decisions
4. **AI-Powered Inference**: Use ML to improve capability and dependency inference
5. **Real-time Updates**: Watch for changes and update registrations automatically

## Best Practices

1. **Use as Build Step**: Integrate generation into your build pipeline
2. **Validate Regularly**: Run validation checks in CI/CD
3. **Version Control Generated Files**: Track changes to generated configurations
4. **Combine with Manual Overrides**: Allow manual overrides for special cases
5. **Monitor Dependencies**: Use dependency graphs to understand service evolution
