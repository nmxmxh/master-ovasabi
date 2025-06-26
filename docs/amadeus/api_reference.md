# Documentation

version: 2025-05-14

version: 2025-05-14

version: 2025-05-14


This document provides a reference for the Amadeus Knowledge Graph API.

## Knowledge Graph Package

The primary package for interacting with the Amadeus Knowledge Graph is
`github.com/nmxmxh/master-ovasabi/amadeus/pkg/kg`.

### Types

#### KnowledgeGraph

```go
// KnowledgeGraph represents the core structure of the OVASABI knowledge graph
type KnowledgeGraph struct {
    Version     string    `json:"version"`
    LastUpdated time.Time `json:"last_updated"`

    SystemComponents    map[string]interface{} `json:"system_components"`
    RepositoryStructure map[string]interface{} `json:"repository_structure"`
    Services            map[string]interface{} `json:"services"`
    Nexus               map[string]interface{} `json:"nexus"`
    Patterns            map[string]interface{} `json:"patterns"`
    DatabasePractices   map[string]interface{} `json:"database_practices"`
    RedisPractices      map[string]interface{} `json:"redis_practices"`
    AmadeusIntegration  map[string]interface{} `json:"amadeus_integration"`
}
```

### Functions

#### DefaultKnowledgeGraph

```go
// DefaultKnowledgeGraph returns the singleton instance of the knowledge graph
func DefaultKnowledgeGraph() *KnowledgeGraph
```

Returns the default, singleton instance of the knowledge graph. The graph is automatically loaded
from the default path on first access.

Example:

```go
// Get the default knowledge graph
kg := kg.DefaultKnowledgeGraph()
```

### Methods

#### Load

```go
// Load reads the knowledge graph from the specified file
func (kg *KnowledgeGraph) Load(filePath string) error
```

Loads the knowledge graph from the specified file path.

Example:

```go
// Load the knowledge graph from a custom path
kg := &kg.KnowledgeGraph{}
err := kg.Load("path/to/knowledge_graph.json")
```

#### Save

```go
// Save writes the knowledge graph to the specified file
func (kg *KnowledgeGraph) Save(filePath string) error
```

Saves the knowledge graph to the specified file path.

Example:

```go
// Save the knowledge graph to a custom path
err := kg.Save("path/to/knowledge_graph.json")
```

#### GetNode

```go
// GetNode retrieves a value from the knowledge graph using a dot-notation path
func (kg *KnowledgeGraph) GetNode(path string) (interface{}, error)
```

Retrieves a value from the knowledge graph using a dot-notation path.

Example:

```go
// Get information about the user service
userService, err := kg.GetNode("services.core_services.user_service")
```

#### UpdateNode

```go
// UpdateNode updates a node in the knowledge graph using a dot-notation path
func (kg *KnowledgeGraph) UpdateNode(path string, value interface{}) error
```

Updates a node in the knowledge graph using a dot-notation path.

Example:

```go
// Update the version of a service
err := kg.UpdateNode("services.core_services.user_service.version", "1.1.0")
```

#### AddService

```go
// AddService adds a new service to the knowledge graph
func (kg *KnowledgeGraph) AddService(category string, name string, serviceInfo map[string]interface{}) error
```

Adds a new service to the knowledge graph in the specified category.

Example:

```go
// Add a new service
serviceInfo := map[string]interface{}{
    "name":        "my_service",
    "version":     "1.0.0",
    "description": "My new service",
    // ... other service information
}
err := kg.AddService("core_services", "my_service", serviceInfo)
```

#### AddPattern

```go
// AddPattern adds a new pattern to the knowledge graph
func (kg *KnowledgeGraph) AddPattern(category string, name string, patternInfo map[string]interface{}) error
```

Adds a new pattern to the knowledge graph in the specified category.

Example:

```go
// Add a new pattern
patternInfo := map[string]interface{}{
    "name":        "my_pattern",
    "purpose":     "My new pattern",
    "location":    "internal/nexus/patterns/my_pattern",
    // ... other pattern information
}
err := kg.AddPattern("core_patterns", "my_pattern", patternInfo)
```

#### TrackEntityRelationship

```go
// TrackEntityRelationship adds or updates a relationship between two entities
func (kg *KnowledgeGraph) TrackEntityRelationship(sourceType string, sourceID string,
    relationType string, targetType string, targetID string) error
```

Adds or updates a relationship between two entities in the knowledge graph.

Example:

```go
// Track a dependency relationship
err := kg.TrackEntityRelationship(
    "service", "my_service",
    "depends_on",
    "service", "user_service",
)
```

#### GenerateVisualization

```go
// GenerateVisualization generates a visualization of part or all of the knowledge graph
func (kg *KnowledgeGraph) GenerateVisualization(format string, section string) ([]byte, error)
```

Generates a visualization of part or all of the knowledge graph in the specified format.

Example:

```go
// Generate a Mermaid diagram of services
data, err := kg.GenerateVisualization("mermaid", "services")
if err != nil {
    // Handle error
}
// Write to file
err = os.WriteFile("services.mmd", data, 0644)
```

## Service Hook Interface

While not a formal interface in the code, service hooks are expected to implement the following
methods:

### OnServiceStart

```go
// OnServiceStart updates the knowledge graph when the service starts
func (h *ServiceHook) OnServiceStart(ctx context.Context) error
```

Updates the knowledge graph when the service starts, registering the service and its information.

### OnEndpointAdded

```go
// OnEndpointAdded updates the knowledge graph when a new endpoint is added
func (h *ServiceHook) OnEndpointAdded(ctx context.Context, endpointName string, metadata map[string]interface{}) error
```

Updates the knowledge graph when a new endpoint is added to the service.

### OnDependencyAdded

```go
// OnDependencyAdded updates the knowledge graph when a new dependency is added
func (h *ServiceHook) OnDependencyAdded(ctx context.Context, dependencyType string, dependencyName string) error
```

Updates the knowledge graph when a new dependency is added to the service.

## Nexus Pattern Interface

Knowledge graph patterns for Nexus should implement the following interface:

### Execute

```go
// Execute executes the knowledge graph pattern
func (p *KnowledgeGraphPattern) Execute(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error)
```

Executes the knowledge graph pattern with the specified parameters.

Example parameters:

```go
params := map[string]interface{}{
    "action":        "track_service_update",
    "category":      "core_services",
    "service_name":  "my_service",
    "service_info":  serviceInfo,
}
```

## Command Line Interface

The Amadeus CLI provides the following commands:

### get

Gets a value from the knowledge graph.

```bash
bin/kgcli get --path <path> [--output <format>]
```

Options:

- `--path`: Path to the knowledge graph node (e.g., `services.core_services.user_service`)
- `--output`: Output format (json|yaml|text, default: json)

### add-service

Adds a service to the knowledge graph.

```bash
bin/kgcli add-service --category <category> --name <name> --file <file>
```

Options:

- `--category`: Service category (e.g., `core_services`)
- `--name`: Service name
- `--file`: JSON file containing service information

### add-pattern

Adds a pattern to the knowledge graph.

```bash
bin/kgcli add-pattern --category <category> --name <name> --file <file>
```

Options:

- `--category`: Pattern category (e.g., `core_patterns`)
- `--name`: Pattern name
- `--file`: JSON file containing pattern information

### visualize

Generates a visualization of the knowledge graph.

```bash
bin/kgcli visualize --format <format> [--section <section>] [--output <file>]
```

Options:

- `--format`: Visualization format (mermaid|dot|json)
- `--section`: Section of knowledge graph to visualize (optional)
- `--output`: Output file path (optional, stdout if not specified)

## ContentService (contentpb.ContentServiceServer)

The ContentService provides APIs for dynamic content (articles, micro-posts, video), comments, reactions, and full-text search. It integrates with UserService for author info, NotificationService for engagement, SearchService for indexing, and ContentModerationService for compliance.

### RPC Methods
- CreateContent
- GetContent
- UpdateContent
- DeleteContent
- ListContent
- AddReaction
- ListReactions

### Integration Points
- Calls UserService to enrich content with author/user info
- Calls ContentModerationService to submit content for moderation
- Calls NotificationService to notify followers/mentioned users
- Calls SearchService to index content for FTS

### Example Usage

```go
// Create content and trigger cross-service orchestration
resp, err := contentClient.CreateContent(ctx, &contentpb.CreateContentRequest{
    Content: &contentpb.Content{
        AuthorId: "user-uuid",
        Type: "article",
        Title: "My First Post",
        Body: "Hello, world!",
    },
})
if err != nil {
    // handle error
}
fmt.Println("Created content:", resp.Content)
```
