package examples

import (
	"context"
	"fmt"
	"log"

	"github.com/nmxmxh/master-ovasabi/amadeus/pkg/kg"
)

// This is an example of how a service would integrate with the
// knowledge graph to keep it updated as the service evolves.

// ServiceHookExample demonstrates how to hook into service lifecycle events
// to update the knowledge graph.
type ServiceHookExample struct {
	serviceName string
	category    string
	kg          *kg.KnowledgeGraph
}

// NewServiceHookExample creates a new ServiceHookExample.
func NewServiceHookExample(serviceName, category string) *ServiceHookExample {
	return &ServiceHookExample{
		serviceName: serviceName,
		category:    category,
		kg:          kg.DefaultKnowledgeGraph(),
	}
}

// OnServiceStart updates the knowledge graph when the service starts.
func (h *ServiceHookExample) OnServiceStart(_ context.Context) error {
	log.Printf("Service %s starting, updating knowledge graph", h.serviceName)

	// Get current service information
	serviceInfo := h.getCurrentServiceInfo()

	// Update the knowledge graph
	err := h.kg.AddService(h.category, h.serviceName, serviceInfo)
	if err != nil {
		return fmt.Errorf("failed to update knowledge graph: %w", err)
	}

	// Save the knowledge graph
	err = h.kg.Save("amadeus/knowledge_graph.json")
	if err != nil {
		return fmt.Errorf("failed to save knowledge graph: %w", err)
	}

	return nil
}

// OnEndpointAdded updates the knowledge graph when a new endpoint is added.
func (h *ServiceHookExample) OnEndpointAdded(_ context.Context, endpointName string, metadata map[string]interface{}) error {
	log.Printf("Service %s added endpoint %s, updating knowledge graph", h.serviceName, endpointName)

	// Get current service information
	serviceInfo := h.getCurrentServiceInfo()

	// Add the new endpoint
	if endpoints, ok := serviceInfo["endpoints"].(map[string]interface{}); ok {
		endpoints[endpointName] = metadata
	} else {
		serviceInfo["endpoints"] = map[string]interface{}{
			endpointName: metadata,
		}
	}

	// Update the knowledge graph
	err := h.kg.AddService(h.category, h.serviceName, serviceInfo)
	if err != nil {
		return fmt.Errorf("failed to update knowledge graph: %w", err)
	}

	// Save the knowledge graph
	err = h.kg.Save("amadeus/knowledge_graph.json")
	if err != nil {
		return fmt.Errorf("failed to save knowledge graph: %w", err)
	}

	return nil
}

// OnDependencyAdded updates the knowledge graph when a new dependency is added.
func (h *ServiceHookExample) OnDependencyAdded(_ context.Context, dependencyType, dependencyName string) error {
	log.Printf("Service %s added dependency %s of type %s, updating knowledge graph",
		h.serviceName, dependencyName, dependencyType)

	// Track the relationship in the knowledge graph
	err := h.kg.TrackEntityRelationship(
		"service", h.serviceName,
		"depends_on",
		dependencyType, dependencyName,
	)
	if err != nil {
		return fmt.Errorf("failed to track relationship: %w", err)
	}

	// Save the knowledge graph
	err = h.kg.Save("amadeus/knowledge_graph.json")
	if err != nil {
		return fmt.Errorf("failed to save knowledge graph: %w", err)
	}

	return nil
}

// Example of how to use the service hook in a real service.
func ExampleUsage() {
	// Create a service hook
	hook := NewServiceHookExample("user_service", "core_services")

	// When service starts
	err := hook.OnServiceStart(context.Background())
	if err != nil {
		log.Fatalf("Failed to update knowledge graph: %v", err)
	}

	// When a new endpoint is added
	err = hook.OnEndpointAdded(context.Background(), "getUserProfile", map[string]interface{}{
		"method":      "GET",
		"path":        "/users/:id/profile",
		"description": "Get user profile information",
		"auth":        true,
	})
	if err != nil {
		log.Fatalf("Failed to update knowledge graph: %v", err)
	}

	// When a dependency is added
	err = hook.OnDependencyAdded(context.Background(), "service", "notification_service")
	if err != nil {
		log.Fatalf("Failed to update knowledge graph: %v", err)
	}
}

// getCurrentServiceInfo gets the current service information.
func (h *ServiceHookExample) getCurrentServiceInfo() map[string]interface{} {
	// In a real implementation, this would dynamically gather information
	// about the service, its endpoints, dependencies, etc.
	return map[string]interface{}{
		"name":        h.serviceName,
		"version":     "1.0.0",
		"description": "Example service for knowledge graph integration",
		"status":      "running",
		"endpoints": map[string]interface{}{
			"getUser": map[string]interface{}{
				"method":      "GET",
				"path":        "/users/:id",
				"description": "Get user by ID",
				"auth":        true,
			},
		},
		"dependencies": []string{
			"database",
			"redis",
		},
	}
}
