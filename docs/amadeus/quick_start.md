# Amadeus Knowledge Graph Quick Start Guide

This guide will help you get started quickly with Amadeus Knowledge Graph.

## 1. Build the CLI Tool

First, build the Knowledge Graph CLI tool:

```bash
# From the repository root
go build -o bin/kgcli amadeus/cmd/kgcli/main.go
```

## 2. Explore the Knowledge Graph

Use the CLI tool to explore the existing knowledge graph:

```bash
# View system components
bin/kgcli get --path system_components

# View core services
bin/kgcli get --path services.core_services

# View patterns
bin/kgcli get --path patterns
```

## 3. Add a Service to the Knowledge Graph

Create a JSON file with your service information:

```json
{
  "name": "my_service",
  "version": "1.0.0",
  "description": "My new service",
  "location": "internal/service/my_service",
  "repositories": ["my_repository"],
  "patterns_used": ["identity_unification_pattern"],
  "integration_points": ["user_service"],
  "endpoints": {
    "getItem": {
      "method": "GET",
      "path": "/items/:id",
      "description": "Get item by ID",
      "auth": true
    }
  }
}
```

Add the service to the knowledge graph:

```bash
bin/kgcli add-service --category core_services --name my_service --file my_service.json
```

## 4. Add a Pattern to the Knowledge Graph

Create a JSON file with your pattern information:

```json
{
  "name": "my_pattern",
  "purpose": "Purpose of my pattern",
  "location": "internal/nexus/patterns/my_pattern",
  "services_used": ["my_service", "user_service"],
  "integration_points": ["endpoint1", "endpoint2"],
  "composition_potential": "Medium"
}
```

Add the pattern to the knowledge graph:

```bash
bin/kgcli add-pattern --category core_patterns --name my_pattern --file my_pattern.json
```

## 5. Integrate with Your Service

Add these imports to your service:

```go
import (
    "context"
    "log"

    "github.com/nmxmxh/master-ovasabi/amadeus/examples"
)
```

Initialize the knowledge graph hook in your service:

```go
func main() {
    // Initialize your service
    service := initializeService()

    // Create knowledge graph hook
    kgHook := examples.NewServiceHookExample("my_service", "core_services")

    // Register service with knowledge graph
    err := kgHook.OnServiceStart(context.Background())
    if err != nil {
        log.Printf("Warning: Failed to register with knowledge graph: %v", err)
    }

    // Continue with normal service initialization
    // ...
}
```

Update the knowledge graph when adding endpoints:

```go
// When adding a new endpoint
endpointMetadata := map[string]interface{}{
    "method":      "GET",
    "path":        "/new-endpoint",
    "description": "Description of the endpoint",
    "auth":        true,
}
kgHook.OnEndpointAdded(context.Background(), "newEndpoint", endpointMetadata)
```

Update the knowledge graph when adding dependencies:

```go
// When adding a dependency
kgHook.OnDependencyAdded(context.Background(), "service", "other_service_name")
```

## 6. Implement a Custom Service Hook

For production use, implement a custom service hook:

```go
// YourServiceHook implements service-specific knowledge graph updates
type YourServiceHook struct {
    serviceName string
    category    string
    kg          *kg.KnowledgeGraph
    service     *YourService // Your actual service implementation
}

// NewYourServiceHook creates a new service hook
func NewYourServiceHook(service *YourService) *YourServiceHook {
    return &YourServiceHook{
        serviceName: service.Name(),
        category:    "core_services",
        kg:          kg.DefaultKnowledgeGraph(),
        service:     service,
    }
}

// OnServiceStart registers service with knowledge graph
func (h *YourServiceHook) OnServiceStart(ctx context.Context) error {
    // Get service information
    serviceInfo := map[string]interface{}{
        "name":        h.service.Name(),
        "version":     h.service.Version(),
        "description": h.service.Description(),
        "endpoints":   h.getEndpointsInfo(),
        "dependencies": h.getDependenciesInfo(),
    }

    // Update knowledge graph
    err := h.kg.AddService(h.category, h.serviceName, serviceInfo)
    if err != nil {
        return err
    }

    // Save changes
    return h.kg.Save("amadeus/knowledge_graph.json")
}

// Implement other hook methods
```

## 7. Integrate with Nexus

Register the knowledge graph pattern with Nexus:

```go
import (
    "github.com/nmxmxh/master-ovasabi/amadeus/nexus/pattern"
    nexuspattern "github.com/nmxmxh/master-ovasabi/internal/nexus/service/pattern"
)

func registerPatterns() {
    // Create knowledge graph pattern
    kgPattern := pattern.NewKnowledgeGraphPattern()

    // Register with Nexus pattern registry
    nexuspattern.Registry().Register("knowledge_graph_pattern", kgPattern)
}
```

Use the pattern in Nexus:

```go
func updateRelationship(ctx context.Context) error {
    params := map[string]interface{}{
        "action":        "track_relationship",
        "source_type":   "service",
        "source_id":     "my_service",
        "relation_type": "depends_on",
        "target_type":   "service",
        "target_id":     "user_service",
    }

    _, err := nexuspattern.Registry().Execute(ctx, "knowledge_graph_pattern", params)
    return err
}
```

## 8. Generate Visualizations

Generate visual representations of the knowledge graph:

```bash
# Generate service architecture diagram
bin/kgcli visualize --format mermaid --section services --output docs/diagrams/services.mmd

# Generate pattern diagram
bin/kgcli visualize --format mermaid --section patterns --output docs/diagrams/patterns.mmd
```

## 9. Next Steps

- Explore the [Amadeus Knowledge Graph Implementation Guide](implementation_guide.md) for detailed
  implementation instructions
- See [Amadeus Knowledge Graph Integration Examples](integration_examples.md) for more integration
  examples
- Review the [Amadeus Knowledge Graph Architecture](architecture.md) for architectural details

## 10. Common Issues

- **Knowledge graph not updating**: Check file permissions and error handling in your hooks
- **CLI tool failing**: Verify the path to the knowledge graph JSON file
- **Import errors**: Make sure the import paths match your module name in go.mod
