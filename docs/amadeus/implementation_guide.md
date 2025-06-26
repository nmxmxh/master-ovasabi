# Documentation

version: 2025-05-14

version: 2025-05-14

version: 2025-05-14


This guide provides detailed information on implementing and integrating with the Amadeus Knowledge
Graph system.

## Core Files

- Knowledge Graph Store: `amadeus/knowledge_graph.json`
- Knowledge Graph API: `amadeus/pkg/kg/knowledge_graph.go`
- CLI Tool: `amadeus/cmd/kgcli/main.go`
- Nexus Pattern: `amadeus/nexus/pattern/knowledge_graph_pattern.go`
- Service Hook Example: `amadeus/examples/service_hook_example.go`

## 1. Overview

Amadeus Knowledge Graph is a comprehensive system that maintains a living map of the entire OVASABI
architecture. It enables:

- Self-documenting system architecture
- Impact analysis for changes
- Architectural compliance enforcement
- Automatic visualization generation
- Decision intelligence for system evolution

## 2. System Components

The Amadeus system consists of:

| Component            | Description                                       | Location                       |
| -------------------- | ------------------------------------------------- | ------------------------------ |
| Knowledge Graph JSON | Core data store for system knowledge              | `amadeus/knowledge_graph.json` |
| Knowledge Graph API  | Go package for programmatic access                | `amadeus/pkg/kg`               |
| CLI Tool             | Command-line interface for knowledge graph access | `amadeus/cmd/kgcli`            |
| Nexus Pattern        | Pattern for integrating with Nexus orchestration  | `amadeus/nexus/pattern`        |
| Service Hooks        | Integration points for services                   | `amadeus/examples`             |

## 3. Installation

### 3.1. Prerequisites

- Go 1.23 or later
- Access to OVASABI codebase
- Permissions to modify service code

### 3.2. Building the CLI Tool

```bash
go build -o bin/kgcli amadeus/cmd/kgcli/main.go
```

### 3.3. Verifying Installation

```bash
bin/kgcli get --path system_components
```

## 4. Service Integration

### 4.1. Basic Integration

To integrate a service with Amadeus, add the following code to your service initialization:

```go
import (
    "context"
    "log"

    "github.com/nmxmxh/master-ovasabi/amadeus/examples"
)

func initializeService() {
    // Create a service hook
    serviceHook := examples.NewServiceHookExample("your_service_name", "core_services")

    // Register service with knowledge graph on startup
    err := serviceHook.OnServiceStart(context.Background())
    if err != nil {
        log.Fatalf("Failed to register service with knowledge graph: %v", err)
    }

    // Continue with normal service initialization
    // ...
}
```

### 4.2. Advanced Integration

For more comprehensive integration, implement hooks at key lifecycle points:

```go
// When adding a new endpoint
func AddEndpoint(name string, handler http.Handler, metadata map[string]interface{}) {
    // Register the endpoint with your service
    service.AddEndpoint(name, handler)

    // Update the knowledge graph
    serviceHook.OnEndpointAdded(context.Background(), name, metadata)
}

// When adding a dependency
func AddDependency(dependencyType, dependencyName string) {
    // Configure the dependency in your service
    service.ConfigureDependency(dependencyType, dependencyName)

    // Update the knowledge graph
    serviceHook.OnDependencyAdded(context.Background(), dependencyType, dependencyName)
}
```

### 4.3. Custom Service Hook Implementation

For production use, implement a custom service hook that gathers accurate information about your
service:

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
    // Get real service information from your actual service
    serviceInfo := map[string]interface{}{
        "name":        h.service.Name(),
        "version":     h.service.Version(),
        "description": h.service.Description(),
        "endpoints":   h.getEndpointsInfo(),
        "dependencies": h.getDependenciesInfo(),
        // Add any other relevant service information
    }

    // Update the knowledge graph
    err := h.kg.AddService(h.category, h.serviceName, serviceInfo)
    if err != nil {
        return err
    }

    // Save changes
    return h.kg.Save("amadeus/knowledge_graph.json")
}

// Implement other hook methods as needed
```

## 5. Nexus Integration

### 5.1. Registering the Knowledge Graph Pattern

To integrate with Nexus, register the Knowledge Graph pattern:

```go
import (
    "github.com/nmxmxh/master-ovasabi/amadeus/nexus/pattern"
    "github.com/nmxmxh/master-ovasabi/internal/nexus/service/pattern" as nexuspattern
)

func initializeNexus() {
    // Create knowledge graph pattern
    kgPattern := pattern.NewKnowledgeGraphPattern()

    // Register with Nexus pattern registry
    nexuspattern.Registry().Register("knowledge_graph_pattern", kgPattern)
}
```

### 5.2. Using the Pattern in Nexus

To use the pattern from Nexus:

```go
func updateServiceInKnowledgeGraph(serviceInfo map[string]interface{}) error {
    params := map[string]interface{}{
        "action":      "track_service_update",
        "category":    "core_services",
        "service_name": serviceInfo["name"].(string),
        "service_info": serviceInfo,
    }

    result, err := nexuspattern.Registry().Execute("knowledge_graph_pattern", params)
    if err != nil {
        return err
    }

    // Process result if needed
    return nil
}
```

## 6. Programmatic Usage

### 6.1. Querying the Knowledge Graph

```go
import "github.com/nmxmxh/master-ovasabi/amadeus/pkg/kg"

func getServiceInformation(serviceName string) (interface{}, error) {
    // Get the knowledge graph
    graph := kg.DefaultKnowledgeGraph()

    // Query the graph
    path := fmt.Sprintf("services.core_services.%s", serviceName)
    return graph.GetNode(path)
}
```

### 6.2. Updating the Knowledge Graph

```go
import "github.com/nmxmxh/master-ovasabi/amadeus/pkg/kg"

func updateServiceVersion(serviceName, version string) error {
    // Get the knowledge graph
    graph := kg.DefaultKnowledgeGraph()

    // Get existing service info
    path := fmt.Sprintf("services.core_services.%s", serviceName)
    serviceInfo, err := graph.GetNode(path)
    if err != nil {
        return err
    }

    // Update version
    serviceInfoMap := serviceInfo.(map[string]interface{})
    serviceInfoMap["version"] = version

    // Update the knowledge graph
    err = graph.AddService("core_services", serviceName, serviceInfoMap)
    if err != nil {
        return err
    }

    // Save changes
    return graph.Save("amadeus/knowledge_graph.json")
}
```

## 7. CI/CD Integration

### 7.1. Automated Visualization Generation

Add the following to your CI/CD pipeline to generate visualizations on each build:

```yaml
steps:
  # Other CI steps...

  - name: Generate System Visualizations
    run: |
      bin/kgcli visualize --format mermaid --section services --output docs/diagrams/services.mmd
      bin/kgcli visualize --format mermaid --section patterns --output docs/diagrams/patterns.mmd
```

### 7.2. Knowledge Graph Validation

Add validation to ensure system consistency:

```yaml
steps:
  # Other CI steps...

  - name: Validate Knowledge Graph
    run: |
      # Custom script to validate knowledge graph integrity
      scripts/validate_knowledge_graph.sh
```

## 8. Best Practices

### 8.1. Service Registration

- Register all services with the knowledge graph on startup
- Keep service information up-to-date with actual implementation
- Include detailed metadata about endpoints, dependencies, and capabilities

### 8.2. Pattern Registration

- Document all patterns in the knowledge graph
- Use patterns consistently across services
- Update pattern documentation when pattern implementation changes

### 8.3. Knowledge Graph Maintenance

- Treat the knowledge graph as a critical system resource
- Version control the knowledge graph JSON
- Validate knowledge graph integrity regularly

## 9. Troubleshooting

### 9.1. Common Issues

| Issue                        | Resolution                                           |
| ---------------------------- | ---------------------------------------------------- |
| Knowledge graph not updating | Check service hook implementation and error handling |
| CLI tool failing             | Verify build and paths to knowledge graph JSON       |
| Inconsistent data            | Run validation tools and fix inconsistencies         |

### 9.2. Logging

All Amadeus components include detailed logging. Check logs for issues:

```go
// Enable debug logging
import "log"

log.SetFlags(log.LstdFlags | log.Lshortfile)
```

## 10. Future Development

### 10.1. Planned Enhancements

- Real-time knowledge graph updates via event streams
- Advanced visualization capabilities
- AI-assisted system analysis
- Knowledge graph query language
- System evolution tracking

### 10.2. Contributing

To contribute to Amadeus development:

1. Fork the repository
2. Make your changes
3. Add tests for new functionality
4. Submit a pull request

## 11. References

- Knowledge Graph Data Structure: `amadeus/knowledge_graph.json`
- Knowledge Graph Package Documentation: `amadeus/pkg/kg/knowledge_graph.go`
- CLI Tool Documentation: `amadeus/cmd/kgcli/main.go`
- Nexus Pattern Documentation: `amadeus/nexus/pattern/knowledge_graph_pattern.go`
- Service Hook Example: `amadeus/examples/service_hook_example.go`
