# Amadeus Knowledge Graph: Consistent Update Guide

This document provides detailed instructions on how to ensure the Amadeus Knowledge Graph remains
consistently up-to-date through various automated mechanisms.

## 1. CI/CD Integration

### 1.1. GitHub Actions Workflow

Add the following to your GitHub Actions workflow to automatically update the knowledge graph
whenever services are changed:

```yaml
name: Update Knowledge Graph

on:
  push:
    branches: [main]
    paths:
      - 'internal/service/**'
      - 'internal/nexus/**'
      - 'amadeus/**'

jobs:
  update-knowledge-graph:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v3
        with:
          go-version: 1.23

      - name: Build kgcli
        run: go build -o bin/kgcli amadeus/cmd/kgcli/main.go

      - name: Scan and Update Service Information
        run: |
          # Scan services for changes
          bin/kgcli scan-services --directory internal/service

          # Validate knowledge graph consistency
          bin/kgcli validate

      - name: Generate Updated Documentation
        run: |
          # Generate visualizations
          bin/kgcli visualize --format mermaid --section services --output docs/diagrams/services.mmd
          bin/kgcli visualize --format mermaid --section patterns --output docs/diagrams/patterns.mmd

      - name: Commit Updates
        uses: stefanzweifel/git-auto-commit-action@v4
        with:
          commit_message: 'chore: Update knowledge graph and documentation'
          file_pattern: amadeus/knowledge_graph.json docs/diagrams/* docs/services/*
```

### 1.2. Integration with Other CI/CD Systems

For other CI/CD systems (Jenkins, CircleCI, etc.), adapt the GitHub Actions workflow above to your
specific system. The key steps are:

1. Building the `kgcli` tool
2. Scanning services for changes
3. Validating the knowledge graph
4. Generating updated documentation
5. Committing changes back to the repository

## 2. Service Lifecycle Hooks

### 2.1. Startup Registration

Integrate service registration at startup to ensure the knowledge graph is updated whenever a
service starts:

```go
func main() {
    // Initialize your service
    service := initializeService()

    // Create knowledge graph hook
    kgHook := NewServiceHook(service)

    // Register with knowledge graph on startup
    err := kgHook.OnServiceStart(context.Background())
    if err != nil {
        log.Printf("Warning: Failed to register with knowledge graph: %v", err)
    }

    // Continue with normal service initialization
    // ...
}
```

### 2.2. Runtime Updates

Update the knowledge graph during runtime when service configuration changes:

```go
// When adding a new endpoint
func (s *Service) AddEndpoint(name string, handler http.Handler, metadata map[string]interface{}) {
    // Register the endpoint with your service
    s.endpoints[name] = handler

    // Update the knowledge graph
    s.kgHook.OnEndpointAdded(context.Background(), name, metadata)
}

// When adding a dependency
func (s *Service) AddDependency(dependencyType, dependencyName string) {
    // Configure the dependency in your service
    s.configureDependency(dependencyType, dependencyName)

    // Update the knowledge graph
    s.kgHook.OnDependencyAdded(context.Background(), dependencyType, dependencyName)
}
```

## 3. Webhook-based Updates

### 3.1. Setting Up the Webhook Server

Create a dedicated webhook server to receive update notifications:

```go
package main

import (
    "encoding/json"
    "io/ioutil"
    "log"
    "net/http"

    "github.com/nmxmxh/master-ovasabi/amadeus/pkg/kg"
)

func main() {
    http.HandleFunc("/webhooks/kg-update", handleKnowledgeGraphUpdate)
    log.Fatal(http.ListenAndServe(":8090", nil))
}

func handleKnowledgeGraphUpdate(w http.ResponseWriter, r *http.Request) {
    // Read request body
    body, err := ioutil.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Failed to read request body", http.StatusBadRequest)
        return
    }

    // Parse update information
    var updateInfo struct {
        Type        string                 `json:"type"`
        Category    string                 `json:"category"`
        Name        string                 `json:"name"`
        Information map[string]interface{} `json:"information"`
    }

    if err := json.Unmarshal(body, &updateInfo); err != nil {
        http.Error(w, "Failed to parse request body", http.StatusBadRequest)
        return
    }

    // Get knowledge graph
    graph := kg.DefaultKnowledgeGraph()

    // Update knowledge graph based on update type
    var updateErr error
    switch updateInfo.Type {
    case "service":
        updateErr = graph.AddService(updateInfo.Category, updateInfo.Name, updateInfo.Information)
    case "pattern":
        updateErr = graph.AddPattern(updateInfo.Category, updateInfo.Name, updateInfo.Information)
    default:
        http.Error(w, "Invalid update type", http.StatusBadRequest)
        return
    }

    if updateErr != nil {
        log.Printf("Failed to update knowledge graph: %v", updateErr)
        http.Error(w, "Failed to update knowledge graph", http.StatusInternalServerError)
        return
    }

    // Save changes
    if err := graph.Save("amadeus/knowledge_graph.json"); err != nil {
        log.Printf("Failed to save knowledge graph: %v", err)
        http.Error(w, "Failed to save knowledge graph", http.StatusInternalServerError)
        return
    }

    // Return success
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{
        "status":  "success",
        "message": "Knowledge graph updated",
    })
}
```

### 3.2. Sending Updates via Webhook

```bash
# Example: Update a service via webhook
curl -X POST http://localhost:8090/webhooks/kg-update \
  -H "Content-Type: application/json" \
  -d '{
    "type": "service",
    "category": "core_services",
    "name": "user_service",
    "information": {
      "name": "user_service",
      "version": "1.2.0",
      "description": "Updated user service"
    }
  }'
```

### 3.3. Setting Up as a Systemd Service

For production environments, set up the webhook server as a systemd service:

```
[Unit]
Description=Amadeus Knowledge Graph Webhook Server
After=network.target

[Service]
ExecStart=/usr/local/bin/kgwebhook
WorkingDirectory=/path/to/repository
Restart=always
User=serviceuser
Environment=PATH=/usr/bin:/usr/local/bin

[Install]
WantedBy=multi-user.target
```

## 4. Scheduled Jobs

### 4.1. Cron Jobs

Set up a cron job to periodically validate and update the knowledge graph:

```bash
# Run every 6 hours
0 */6 * * * cd /path/to/repository && bin/kgcli validate --fix && bin/kgcli scan-services --directory internal/service
```

### 4.2. Kubernetes CronJob

For Kubernetes environments, set up a CronJob:

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: amadeus-kg-update
spec:
  schedule: '0 */6 * * *'
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: kgcli
              image: your-registry/kgcli:latest
              command:
                - /bin/sh
                - -c
                - |
                  kgcli validate --fix
                  kgcli scan-services --directory internal/service
                  kgcli visualize --format mermaid --section services --output docs/diagrams/services.mmd
          restartPolicy: OnFailure
```

## 5. Manual Updates

While automated updates are preferred, manual updates may occasionally be necessary:

### 5.1. CLI Tool Updates

```bash
# Update specific service
bin/kgcli add-service --category core_services --name my_service --file service_info.json

# Update pattern
bin/kgcli add-pattern --category core_patterns --name my_pattern --file pattern_info.json

# Scan and update all services
bin/kgcli scan-services --directory internal/service
```

### 5.2. Direct API Updates

Use the Knowledge Graph API directly in custom scripts:

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/nmxmxh/master-ovasabi/amadeus/pkg/kg"
)

func main() {
    // Get the knowledge graph
    graph := kg.DefaultKnowledgeGraph()

    // Update a specific service
    serviceInfo := map[string]interface{}{
        "name":        "my_service",
        "version":     "1.0.0",
        "description": "My service description",
        // ... other service information
    }

    if err := graph.AddService("core_services", "my_service", serviceInfo); err != nil {
        log.Fatalf("Failed to update service: %v", err)
    }

    // Save changes
    if err := graph.Save("amadeus/knowledge_graph.json"); err != nil {
        log.Fatalf("Failed to save knowledge graph: %v", err)
    }
}
```

## 6. Validation and Consistency Checks

To ensure the knowledge graph remains consistent, run regular validation checks:

```bash
# Basic validation
bin/kgcli validate

# Validation with auto-fix
bin/kgcli validate --fix

# Deep validation (checks service existence)
bin/kgcli validate --deep
```

### 6.1. Validation Checks Performed

The validation process checks for:

- Structural consistency of the knowledge graph
- Reference integrity (services referenced in patterns actually exist)
- Version consistency (service versions in different sections match)
- Path validity (service locations exist in the file system)
- Schema compliance (all required fields are present)

## 7. Troubleshooting

### 7.1. Common Issues

| Issue                                 | Resolution                                               |
| ------------------------------------- | -------------------------------------------------------- |
| Knowledge graph file locked           | Check for concurrent processes accessing the file        |
| Webhook server not receiving updates  | Verify network configuration and firewall rules          |
| CI/CD pipeline not updating the graph | Check permissions and SSH keys for the commit step       |
| Inconsistent service information      | Run `bin/kgcli validate --fix` to repair inconsistencies |
| Missing service information           | Run `bin/kgcli scan-services` to rediscover services     |

### 7.2. Logging

Enable detailed logging for troubleshooting:

```go
// In your service
import "log"

// Enable verbose logging
log.SetFlags(log.LstdFlags | log.Lshortfile)
```

For webhook server:

```bash
# Run with debug logging
KG_LOG_LEVEL=debug bin/kgwebhook
```

## 8. Best Practices

### 8.1. Update Frequency

- **Services**: Update on startup and configuration changes
- **Patterns**: Update when pattern implementation changes
- **System components**: Update during deployment
- **Validation**: Run at least daily to catch discrepancies

### 8.2. Handling Conflicts

When multiple sources update the knowledge graph:

1. Use version tracking for each entity
2. Implement merge strategies for conflicting updates
3. Prioritize runtime information over static analysis
4. Log all update conflicts for review

### 8.3. Backup Strategy

Maintain backups of the knowledge graph:

```bash
# In your backup script
timestamp=$(date +%Y%m%d%H%M%S)
cp amadeus/knowledge_graph.json amadeus/backups/knowledge_graph_${timestamp}.json
```

Keep at least the last 7 days of backups for recovery purposes.

## 9. Future Enhancements

Planned improvements to the update process:

- **Real-time event streaming**: Publish knowledge graph updates to an event stream
- **Distributed consistency**: Ensure consistency across multiple instances
- **Conflict resolution AI**: Use machine learning for intelligent merge conflict resolution
- **Automated service discovery**: Detect and register new services automatically
- **Change impact prediction**: Analyze potential impacts before applying updates
