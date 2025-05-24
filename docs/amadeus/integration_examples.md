# Documentation

version: 2025-05-14

version: 2025-05-14

version: 2025-05-14


This document provides practical examples of how to integrate services with the Amadeus Knowledge
Graph.

## 1. Basic Service Integration

### User Service Example

```go
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/nmxmxh/master-ovasabi/amadeus/examples"
	"github.com/nmxmxh/master-ovasabi/internal/service/user"
)

func main() {
	// Initialize user service
	userService := user.NewService()

	// Create knowledge graph hook
	kgHook := examples.NewServiceHookExample("user_service", "core_services")

	// Register with knowledge graph on startup
	if err := kgHook.OnServiceStart(context.Background()); err != nil {
		log.Fatalf("Failed to register with knowledge graph: %v", err)
	}

	// Set up API handlers
	http.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		// Handle user requests
	})

	// Register new endpoint with knowledge graph
	endpointMetadata := map[string]interface{}{
		"method":      "GET",
		"path":        "/users",
		"description": "List all users",
		"auth":        true,
	}
	if err := kgHook.OnEndpointAdded(context.Background(), "listUsers", endpointMetadata); err != nil {
		log.Printf("Warning: Failed to register endpoint with knowledge graph: %v", err)
	}

	// Start server
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## 2. Custom Service Hook Implementation

### Finance Service Example

```go
package finance

import (
	"context"
	"fmt"

	"github.com/nmxmxh/master-ovasabi/amadeus/pkg/kg"
)

// FinanceServiceHook handles knowledge graph integration for finance service
type FinanceServiceHook struct {
	serviceName string
	category    string
	kg          *kg.KnowledgeGraph
	service     *Service
}

// NewFinanceServiceHook creates a new finance service hook
func NewFinanceServiceHook(service *Service) *FinanceServiceHook {
	return &FinanceServiceHook{
		serviceName: "finance_service",
		category:    "core_services",
		kg:          kg.DefaultKnowledgeGraph(),
		service:     service,
	}
}

// OnServiceStart registers the service with the knowledge graph
func (h *FinanceServiceHook) OnServiceStart(ctx context.Context) error {
	serviceInfo := map[string]interface{}{
		"name":        h.serviceName,
		"version":     h.service.Version(),
		"description": "Handles financial transactions and wallet management",
		"status":      "running",
		"endpoints":   h.getEndpointsInfo(),
		"dependencies": []string{
			"user_service",
			"database",
			"redis",
			"exchange_orchestration_service",
		},
		"databases": map[string]interface{}{
			"main": map[string]interface{}{
				"type":   "postgres",
				"tables": []string{"transactions", "wallets", "payment_methods"},
			},
		},
		"redis_usage": map[string]interface{}{
			"caching":      []string{"wallet_balances", "transaction_history"},
			"rate_limits":  []string{"payment_processing"},
			"distributed_locks": []string{"wallet_operations"},
		},
	}

	err := h.kg.AddService(h.category, h.serviceName, serviceInfo)
	if err != nil {
		return fmt.Errorf("failed to update knowledge graph: %w", err)
	}

	return h.kg.Save("amadeus/knowledge_graph.json")
}

// getEndpointsInfo collects actual endpoint information from the service
func (h *FinanceServiceHook) getEndpointsInfo() map[string]interface{} {
	return map[string]interface{}{
		"getWalletBalance": map[string]interface{}{
			"method":      "GET",
			"path":        "/wallets/:id/balance",
			"description": "Get wallet balance",
			"auth":        true,
		},
		"createTransaction": map[string]interface{}{
			"method":      "POST",
			"path":        "/transactions",
			"description": "Create a new transaction",
			"auth":        true,
		},
		"getTransactions": map[string]interface{}{
			"method":      "GET",
			"path":        "/wallets/:id/transactions",
			"description": "Get transaction history for a wallet",
			"auth":        true,
		},
	}
}

// OnWalletCreated updates knowledge graph when a new wallet is created
func (h *FinanceServiceHook) OnWalletCreated(ctx context.Context, walletID string, userID string) error {
	// Track relationship between user and wallet
	err := h.kg.TrackEntityRelationship(
		"user", userID,
		"has_wallet",
		"wallet", walletID,
	)
	if err != nil {
		return fmt.Errorf("failed to track relationship: %w", err)
	}

	return h.kg.Save("amadeus/knowledge_graph.json")
}
```

## 3. Nexus Pattern Usage

### Example: Using Knowledge Graph Pattern in Nexus

```go
package main

import (
	"context"
	"log"

	"github.com/nmxmxh/master-ovasabi/amadeus/nexus/pattern"
	nexuspattern "github.com/nmxmxh/master-ovasabi/internal/nexus/service/pattern"
)

func registerPatterns() {
	// Create knowledge graph pattern
	kgPattern := pattern.NewKnowledgeGraphPattern()

	// Register with Nexus pattern registry
	nexuspattern.Registry().Register("knowledge_graph_pattern", kgPattern)
}

func trackSystemChange(ctx context.Context) error {
	// Example: Track a system-wide change in the knowledge graph
	params := map[string]interface{}{
		"action":        "track_relationship",
		"source_type":   "service",
		"source_id":     "finance_service",
		"relation_type": "depends_on",
		"target_type":   "service",
		"target_id":     "notification_service",
	}

	// Execute the pattern
	result, err := nexuspattern.Registry().Execute(ctx, "knowledge_graph_pattern", params)
	if err != nil {
		return err
	}

	log.Printf("Knowledge graph updated: %v", result["message"])
	return nil
}
```

## 4. Command Line Tool Examples

### Getting Service Information

```bash
bin/kgcli get --path services.core_services.user_service

bin/kgcli get --path services.core_services

bin/kgcli get --path system_components
```

### Adding a New Service

Create a file `notification_service.json`:

```json
{
  "name": "notification_service",
  "version": "1.0.0",
  "description": "Manages user notifications across multiple channels",
  "status": "running",
  "endpoints": {
    "sendNotification": {
      "method": "POST",
      "path": "/notifications",
      "description": "Send a notification to a user",
      "auth": true
    },
    "getNotifications": {
      "method": "GET",
      "path": "/users/:id/notifications",
      "description": "Get notifications for a user",
      "auth": true
    }
  },
  "dependencies": ["user_service", "database", "redis"]
}
```

Add it to the knowledge graph:

```bash
bin/kgcli add-service --category core_services --name notification_service --file notification_service.json
```

### Adding a New Pattern

Create a file `notification_pattern.json`:

```json
{
  "name": "notification_distribution_pattern",
  "purpose": "Distributes notifications across multiple channels based on user preferences",
  "location": "internal/nexus/patterns/notification",
  "services_used": ["notification_service", "user_service"],
  "integration_points": ["notification_sending", "user_preferences", "channel_selection"],
  "composition_potential": "Medium - specialized notification system"
}
```

Add it to the knowledge graph:

```bash
bin/kgcli add-pattern --category service_patterns --name notification_distribution_pattern --file notification_pattern.json
```

### Generating Visualizations

```bash
bin/kgcli visualize --format mermaid --section services --output docs/diagrams/services.mmd

bin/kgcli visualize --format mermaid --section patterns --output docs/diagrams/patterns.mmd

bin/kgcli visualize --format mermaid --section database_practices --output docs/diagrams/database.mmd
```

## 5. CI/CD Integration Example

### GitHub Actions Workflow

```yaml
name: Amadeus Knowledge Graph CI

on:
  push:
    branches: [main]
    paths:
      - 'internal/service/**'
      - 'internal/nexus/**'
      - 'amadeus/**'
  pull_request:
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

      - name: Update Service Information
        run: |
          # Scan service directories and generate service info JSON files
          go run scripts/scan_services.go

          # Update knowledge graph with new service information
          for file in service_info/*.json; do
            service_name=$(basename "$file" .json)
            bin/kgcli add-service --category core_services --name "$service_name" --file "$file"
          done

      - name: Generate Documentation
        run: |
          # Generate visualizations
          bin/kgcli visualize --format mermaid --section services --output docs/diagrams/services.mmd
          bin/kgcli visualize --format mermaid --section patterns --output docs/diagrams/patterns.mmd

          # Generate service documentation
          go run scripts/generate_service_docs.go

      - name: Commit updates
        uses: stefanzweifel/git-auto-commit-action@v4
        with:
          commit_message: 'chore: Update knowledge graph and documentation'
          file_pattern: amadeus/knowledge_graph.json docs/diagrams/* docs/services/*
```

## 6. Advanced Usage Examples

### Knowledge Graph Query Tool

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/nmxmxh/master-ovasabi/amadeus/pkg/kg"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: kgquery <query>")
		os.Exit(1)
	}

	query := os.Args[1]
	graph := kg.DefaultKnowledgeGraph()

	// Parse query
	parts := strings.Split(query, ".")
	if len(parts) == 0 {
		fmt.Println("Invalid query")
		os.Exit(1)
	}

	// Handle special queries
	switch parts[0] {
	case "services-using-pattern":
		if len(parts) < 2 {
			fmt.Println("Usage: services-using-pattern.<pattern_name>")
			os.Exit(1)
		}
		findServicesUsingPattern(graph, parts[1])
	case "service-dependencies":
		if len(parts) < 2 {
			fmt.Println("Usage: service-dependencies.<service_name>")
			os.Exit(1)
		}
		findServiceDependencies(graph, parts[1])
	case "pattern-compositions":
		if len(parts) < 2 {
			fmt.Println("Usage: pattern-compositions.<pattern_name>")
			os.Exit(1)
		}
		findPatternCompositions(graph, parts[1])
	default:
		// Standard query
		result, err := graph.GetNode(query)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		jsonBytes, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(string(jsonBytes))
	}
}

func findServicesUsingPattern(graph *kg.KnowledgeGraph, patternName string) {
	// Implementation details
}

func findServiceDependencies(graph *kg.KnowledgeGraph, serviceName string) {
	// Implementation details
}

func findPatternCompositions(graph *kg.KnowledgeGraph, patternName string) {
	// Implementation details
}
```

### Impact Analysis Tool

```go
package main

import (
	"fmt"
	"os"

	"github.com/nmxmxh/master-ovasabi/amadeus/pkg/kg"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: impact-analysis <entity_type> <entity_name>")
		fmt.Println("Example: impact-analysis service user_service")
		os.Exit(1)
	}

	entityType := os.Args[1]
	entityName := os.Args[2]

	graph := kg.DefaultKnowledgeGraph()

	// Perform impact analysis
	switch entityType {
	case "service":
		analyzeServiceImpact(graph, entityName)
	case "pattern":
		analyzePatternImpact(graph, entityName)
	case "database":
		analyzeDatabaseImpact(graph, entityName)
	default:
		fmt.Printf("Unknown entity type: %s\n", entityType)
		os.Exit(1)
	}
}

func analyzeServiceImpact(graph *kg.KnowledgeGraph, serviceName string) {
	fmt.Printf("Impact Analysis for Service: %s\n\n", serviceName)

	// Find all services that depend on this service
	fmt.Println("Services directly dependent on this service:")
	// Implementation details

	// Find all patterns that use this service
	fmt.Println("\nPatterns using this service:")
	// Implementation details

	// Find all components that would be affected by a change
	fmt.Println("\nPotential impact areas:")
	// Implementation details
}

func analyzePatternImpact(graph *kg.KnowledgeGraph, patternName string) {
	// Implementation details
}

func analyzeDatabaseImpact(graph *kg.KnowledgeGraph, databaseName string) {
	// Implementation details
}
```

## 7. Webhook Integration Example

### Auto-update Knowledge Graph on Service Deployment

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
	http.HandleFunc("/webhook/service-deployed", handleServiceDeployed)
	log.Fatal(http.ListenAndServe(":8090", nil))
}

func handleServiceDeployed(w http.ResponseWriter, r *http.Request) {
	// Read request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Parse deployment information
	var deployInfo struct {
		ServiceName    string                 `json:"service_name"`
		ServiceCategory string                `json:"service_category"`
		Version        string                 `json:"version"`
		ServiceInfo    map[string]interface{} `json:"service_info"`
	}

	if err := json.Unmarshal(body, &deployInfo); err != nil {
		http.Error(w, "Failed to parse request body", http.StatusBadRequest)
		return
	}

	// Update knowledge graph
	graph := kg.DefaultKnowledgeGraph()

	err = graph.AddService(deployInfo.ServiceCategory, deployInfo.ServiceName, deployInfo.ServiceInfo)
	if err != nil {
		log.Printf("Failed to update knowledge graph: %v", err)
		http.Error(w, "Failed to update knowledge graph", http.StatusInternalServerError)
		return
	}

	// Save changes
	err = graph.Save("amadeus/knowledge_graph.json")
	if err != nil {
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

Example webhook payload:

```json
{
  "service_name": "user_service",
  "service_category": "core_services",
  "version": "1.2.0",
  "service_info": {
    "name": "user_service",
    "version": "1.2.0",
    "description": "Manages user identity, profiles, and authentication",
    "status": "running",
    "endpoints": {
      "getUser": {
        "method": "GET",
        "path": "/users/:id",
        "description": "Get user by ID",
        "auth": true
      },
      "updateUser": {
        "method": "PUT",
        "path": "/users/:id",
        "description": "Update user profile",
        "auth": true
      }
    },
    "dependencies": ["database", "redis", "notification_service"]
  }
}
```
