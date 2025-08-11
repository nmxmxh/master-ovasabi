package health

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
)

// CentralizedHealthChecker manages health checks for all services through the event bus.
type CentralizedHealthChecker struct {
	provider     *service.Provider
	log          *zap.Logger
	services     []string
	healthStatus map[string]*HealthCheckResult
	statusMutex  sync.RWMutex
	lastCheck    time.Time
}

// HealthDashboard represents the overall health status for frontend display.
type HealthDashboard struct {
	OverallStatus string                        `json:"overall_status"` // "healthy", "warning", "down"
	LastUpdated   time.Time                     `json:"last_updated"`
	Services      map[string]*HealthCheckResult `json:"services"`
	Summary       HealthSummary                 `json:"summary"`
}

// HealthSummary provides quick overview statistics.
type HealthSummary struct {
	TotalServices   int `json:"total_services"`
	HealthyServices int `json:"healthy_services"`
	WarningServices int `json:"warning_services"`
	DownServices    int `json:"down_services"`
}

// NewCentralizedHealthChecker creates a new centralized health checker.
func NewCentralizedHealthChecker(provider *service.Provider, log *zap.Logger, services []string) *CentralizedHealthChecker {
	return &CentralizedHealthChecker{
		provider:     provider,
		log:          log,
		services:     services,
		healthStatus: make(map[string]*HealthCheckResult),
		lastCheck:    time.Now(),
	}
}

// StartHealthMonitoring starts the centralized health monitoring system.
func (c *CentralizedHealthChecker) StartHealthMonitoring(ctx context.Context) {
	// Subscribe to health response events
	c.subscribeToHealthResponses(ctx)

	// Start periodic health checks
	go c.periodicHealthCheck(ctx, 2*time.Minute) // Reduced frequency: now every 2 minutes

	// Start health request handler for frontend requests
	c.subscribeToHealthRequests(ctx)
}

// periodicHealthCheck sends health check requests to all services at regular intervals.
func (c *CentralizedHealthChecker) periodicHealthCheck(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Do initial health check
	c.requestHealthCheckFromAllServices(ctx)

	for {
		select {
		case <-ctx.Done():
			c.log.Info("Periodic health check stopped")
			return
		case <-ticker.C:
			c.requestHealthCheckFromAllServices(ctx)
		}
	}
}

// requestHealthCheckFromAllServices sends health check requests to all registered services.
func (c *CentralizedHealthChecker) requestHealthCheckFromAllServices(ctx context.Context) {
	c.log.Info("Requesting health check from all services", zap.Strings("services", c.services))

	for _, serviceName := range c.services {
		// Send canonical health check request event
		eventType := fmt.Sprintf("%s:health:v1:requested", serviceName)

		meta := &commonpb.Metadata{
			ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{
				"health_check": map[string]interface{}{
					"requester":  "centralized-health-checker",
					"timestamp":  time.Now().Format(time.RFC3339),
					"check_type": "periodic",
					"service":    serviceName,
				},
			}, c.log),
		}

		err := c.provider.EmitEvent(ctx, eventType, "health-check-central", meta)
		if err != nil {
			c.log.Error("Failed to emit health check request",
				zap.String("service", serviceName),
				zap.String("event_type", eventType),
				zap.Error(err),
			)

			// Mark service as down if we can't even send the request
			c.statusMutex.Lock()
			c.healthStatus[serviceName] = &HealthCheckResult{
				ServiceName:  serviceName,
				Status:       "down",
				ErrorMessage: fmt.Sprintf("Failed to send health check request: %v", err),
				CheckedAt:    time.Now().Unix(),
				ResponseTime: 0,
			}
			c.statusMutex.Unlock()
		} else {
			c.log.Debug("Sent health check request",
				zap.String("service", serviceName),
				zap.String("event_type", eventType),
			)
		}
	}
}

// subscribeToHealthResponses listens for health check response events from all services.
func (c *CentralizedHealthChecker) subscribeToHealthResponses(ctx context.Context) {
	var healthEventTypes []string
	for _, serviceName := range c.services {
		healthEventTypes = append(healthEventTypes,
			fmt.Sprintf("%s:health:v1:success", serviceName),
			fmt.Sprintf("%s:health:v1:failed", serviceName),
		)
	}

	go func() {
		err := c.provider.SubscribeEvents(ctx, healthEventTypes, nil, func(ctx context.Context, event *nexusv1.EventResponse) {
			c.handleHealthResponse(ctx, event)
		})
		if err != nil {
			c.log.Error("Failed to subscribe to health response events", zap.Error(err))
		} else {
			c.log.Info("Subscribed to health response events", zap.Strings("event_types", healthEventTypes))
		}
	}()
}

// subscribeToHealthRequests handles health check requests from frontend/clients.
func (c *CentralizedHealthChecker) subscribeToHealthRequests(ctx context.Context) {
	go func() {
		err := c.provider.SubscribeEvents(ctx, []string{"system:health:v1:requested"}, nil, func(ctx context.Context, event *nexusv1.EventResponse) {
			c.handleHealthDashboardRequest(ctx, event)
		})
		if err != nil {
			c.log.Error("Failed to subscribe to health dashboard requests", zap.Error(err))
		} else {
			c.log.Info("Subscribed to health dashboard requests")
		}
	}()
}

// handleHealthResponse processes health check responses from services.
func (c *CentralizedHealthChecker) handleHealthResponse(ctx context.Context, event *nexusv1.EventResponse) {
	// Example usage: log context deadline if set
	deadline, ok := ctx.Deadline()
	if ok {
		c.log.Debug("Received health response with context deadline",
			zap.String("event_type", event.EventType),
			zap.String("event_id", event.EventId),
			zap.Time("ctx_deadline", deadline),
		)
	} else {
		c.log.Debug("Received health response",
			zap.String("event_type", event.EventType),
			zap.String("event_id", event.EventId),
		)
	}

	// Only process events with allowed suffixes
	if !events.ShouldProcessEvent(event.EventType, []string{":success", ":failed"}) {
		c.log.Debug("Ignoring health event with non-requested suffix", zap.String("event_type", event.EventType))
		return
	}

	// Parse service name from event type
	var serviceName string
	if parts := strings.Split(event.EventType, ":"); len(parts) >= 1 {
		serviceName = parts[0]
	}

	if serviceName == "" {
		c.log.Warn("Could not parse service name from health response", zap.String("event_type", event.EventType))
		return
	}

	// Extract health data from event payload
	var healthResult HealthCheckResult
	if event.Payload != nil && event.Payload.Data != nil {
		payloadMap := event.Payload.Data.AsMap()

		// Convert map to HealthCheckResult
		if jsonBytes, err := json.Marshal(payloadMap); err == nil {
			if err := json.Unmarshal(jsonBytes, &healthResult); err != nil {
				c.log.Error("Failed to unmarshal health result", zap.Error(err))
				return
			}
		}
	}

	// Ensure service name is set
	if healthResult.ServiceName == "" {
		healthResult.ServiceName = serviceName
	}

	// Determine status from event type if not set
	if healthResult.Status == "" {
		switch {
		case strings.HasSuffix(event.EventType, ":success"):
			healthResult.Status = "healthy"
		case strings.HasSuffix(event.EventType, ":failed"):
			healthResult.Status = "down"
		}
	}

	// Update stored health status
	c.statusMutex.Lock()
	c.healthStatus[serviceName] = &healthResult
	c.lastCheck = time.Now()
	c.statusMutex.Unlock()

	c.log.Info("Updated health status",
		zap.String("service", serviceName),
		zap.String("status", healthResult.Status),
		zap.Int64("response_time_ms", healthResult.ResponseTime),
	)
}

// handleHealthDashboardRequest responds to requests for the health dashboard.
func (c *CentralizedHealthChecker) handleHealthDashboardRequest(ctx context.Context, event *nexusv1.EventResponse) {
	c.log.Info("Received health dashboard request", zap.String("event_id", event.EventId))

	dashboard := c.GetHealthDashboard()

	// Convert dashboard to protobuf struct
	dashboardBytes, err := json.Marshal(dashboard)
	if err != nil {
		c.log.Error("Failed to marshal health dashboard", zap.Error(err))
		return
	}

	var dashboardMap map[string]interface{}
	if err := json.Unmarshal(dashboardBytes, &dashboardMap); err != nil {
		c.log.Error("Failed to unmarshal health dashboard for protobuf conversion", zap.Error(err))
		return
	}

	dashboardStruct, err := structpb.NewStruct(dashboardMap)
	if err != nil {
		c.log.Error("Failed to create protobuf struct from health dashboard", zap.Error(err))
		return
	}

	// Emit health dashboard response
	meta := &commonpb.Metadata{
		ServiceSpecific: dashboardStruct,
	}

	responseEventType := "system:health:v1:success"
	err = c.provider.EmitEvent(ctx, responseEventType, "health-dashboard", meta)
	if err != nil {
		c.log.Error("Failed to emit health dashboard response",
			zap.String("event_type", responseEventType),
			zap.Error(err),
		)
	} else {
		c.log.Info("Sent health dashboard response",
			zap.String("event_type", responseEventType),
			zap.String("overall_status", dashboard.OverallStatus),
			zap.Int("total_services", dashboard.Summary.TotalServices),
		)
	}
}

// GetHealthDashboard returns the current health dashboard.
func (c *CentralizedHealthChecker) GetHealthDashboard() HealthDashboard {
	c.statusMutex.RLock()
	defer c.statusMutex.RUnlock()

	dashboard := HealthDashboard{
		LastUpdated: c.lastCheck,
		Services:    make(map[string]*HealthCheckResult),
		Summary:     HealthSummary{},
	}

	// Copy current health status
	var healthyCount, warningCount, downCount int
	overallHealthy := true

	for serviceName, result := range c.healthStatus {
		dashboard.Services[serviceName] = result
		dashboard.Summary.TotalServices++

		switch result.Status {
		case "healthy":
			healthyCount++
		case "warning":
			warningCount++
			overallHealthy = false
		case "down":
			downCount++
			overallHealthy = false
		}
	}

	// Add services that haven't responded yet
	for _, serviceName := range c.services {
		if _, exists := dashboard.Services[serviceName]; !exists {
			dashboard.Services[serviceName] = &HealthCheckResult{
				ServiceName:  serviceName,
				Status:       "unknown",
				ErrorMessage: "No health check response received",
				CheckedAt:    time.Now().Unix(),
				ResponseTime: 0,
			}
			dashboard.Summary.TotalServices++
			downCount++
			overallHealthy = false
		}
	}

	dashboard.Summary.HealthyServices = healthyCount
	dashboard.Summary.WarningServices = warningCount
	dashboard.Summary.DownServices = downCount

	// Determine overall status
	if overallHealthy && healthyCount > 0 {
		dashboard.OverallStatus = "healthy"
	} else if downCount > 0 {
		dashboard.OverallStatus = "down"
	} else {
		dashboard.OverallStatus = "warning"
	}

	return dashboard
}

// GetServiceNames returns the list of services being monitored.
func (c *CentralizedHealthChecker) GetServiceNames() []string {
	return c.services
}

// AddService adds a new service to the monitoring list.
func (c *CentralizedHealthChecker) AddService(serviceName string) {
	for _, existing := range c.services {
		if existing == serviceName {
			return // Already exists
		}
	}
	c.services = append(c.services, serviceName)
	c.log.Info("Added service to health monitoring", zap.String("service", serviceName))
}

// RemoveService removes a service from the monitoring list.
func (c *CentralizedHealthChecker) RemoveService(serviceName string) {
	for i, existing := range c.services {
		if existing == serviceName {
			c.services = append(c.services[:i], c.services[i+1:]...)
			c.statusMutex.Lock()
			delete(c.healthStatus, serviceName)
			c.statusMutex.Unlock()
			c.log.Info("Removed service from health monitoring", zap.String("service", serviceName))
			return
		}
	}
}
