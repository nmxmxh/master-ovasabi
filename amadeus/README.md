# Amadeus Knowledge Graph System

Amadeus is the knowledge persistence system for the OVASABI platform, providing a comprehensive and
programmatically accessible knowledge graph of all system components and their relationships.

## Overview

The knowledge graph system enables:

1. **System Documentation** - Comprehensive documentation of system architecture, components and
   relationships
2. **Automated Discovery** - Services and patterns register themselves with the knowledge graph
3. **Visualization** - Generate visual representations of system relationships
4. **Impact Analysis** - Analyze how changes in one part of the system affect others
5. **Evolution Tracking** - Track system changes and evolution over time

## Directory Structure

```
amadeus/
  ├── cmd/                  # Command-line tools
  │   └── kgcli/            # Knowledge graph CLI tool
  ├── examples/             # Example usage of the knowledge graph
  ├── nexus/                # Nexus integration for the knowledge graph
  │   └── pattern/          # Knowledge graph patterns
  ├── pkg/                  # Go packages for knowledge graph management
  │   └── kg/               # Core knowledge graph package
  ├── knowledge_graph.json  # Structured knowledge graph data
  ├── README.md             # This file
  └── system_knowledge_graph.md  # Detailed documentation of the system
```

## Usage

### Knowledge Graph CLI

The `kgcli` tool provides command-line access to the knowledge graph:

```bash
# Get a node from the knowledge graph
./kgcli get --path services.core_services.user_service

# Add a service to the knowledge graph
./kgcli add-service --category core_services --name new_service --file service_info.json

# Add a pattern to the knowledge graph
./kgcli add-pattern --category core_patterns --name new_pattern --file pattern_info.json

# Generate a visualization of the knowledge graph
./kgcli visualize --format mermaid --section services --output services.mmd
```

### Programmatic Usage

The knowledge graph can be accessed programmatically using the Go package:

```go
import "github.com/nmxmxh/master-ovasabi/amadeus/pkg/kg"

// Get the knowledge graph
knowledgeGraph := kg.DefaultKnowledgeGraph()

// Query the knowledge graph
serviceInfo, err := knowledgeGraph.GetNode("services.core_services.user_service")

// Update the knowledge graph
serviceData := map[string]interface{}{
    "name": "user_service",
    "version": "1.0.0",
    // ...
}
err = knowledgeGraph.AddService("core_services", "user_service", serviceData)

// Save changes
err = knowledgeGraph.Save("amadeus/knowledge_graph.json")
```

### Nexus Integration

The knowledge graph is integrated with the Nexus pattern system, allowing services to automatically
update the knowledge graph:

```go
// Example Nexus pattern execution
params := map[string]interface{}{
    "action": "track_service_update",
    "category": "core_services",
    "service_name": "user_service",
    "service_info": serviceInfo,
}
result, err := nexus.ExecutePattern("knowledge_graph_pattern", params)
```

## Service Integration

Services can integrate with the knowledge graph using the provided hooks:

```go
// Create a service hook
hook := NewServiceHookExample("user_service", "core_services")

// Update the knowledge graph when service starts
err := hook.OnServiceStart(context.Background())

// Update the knowledge graph when endpoints are added
err = hook.OnEndpointAdded(context.Background(), "getUserProfile", metadata)

// Update the knowledge graph when dependencies are added
err = hook.OnDependencyAdded(context.Background(), "service", "notification_service")
```

## Knowledge Graph Structure

The knowledge graph is structured as a JSON document with the following top-level sections:

1. **system_components** - High-level system components
2. **repository_structure** - Repository structure and organization
3. **services** - Service descriptions and relationships
4. **nexus** - Nexus orchestration system components
5. **patterns** - Pattern descriptions and relationships
6. **database_practices** - Database usage and practices
7. **redis_practices** - Redis usage and practices
8. **amadeus_integration** - Integration points for the knowledge graph system

See `system_knowledge_graph.md` for a detailed description of each section.

## Future Enhancements

1. **Real-time Updates** - Event-driven updates to the knowledge graph
2. **Versioning** - Track changes to the knowledge graph over time
3. **Advanced Visualization** - Generate interactive visualizations of the system
4. **Query Language** - Develop a dedicated query language for the knowledge graph
5. **AI Integration** - Integrate with AI systems to analyze system patterns and recommend
   improvements
