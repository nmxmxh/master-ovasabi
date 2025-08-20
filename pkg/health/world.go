package health

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
)

// ANSI color codes for colorful health output (following hello package pattern).
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorCyan   = "\033[36m"
)

// HealthHandlerFunc is the handler for health check events.
// HandlerFunc is the handler for health check events.
type HandlerFunc func(ctx context.Context, event *nexusv1.EventResponse, log *zap.Logger)

// HealthSubscription represents a health check event subscription.
// Subscription represents a health check event subscription.
type Subscription struct {
	Handler HandlerFunc
	Event   string
}

// HealthCheckResult represents the result of a health check.
// CheckResult represents the result of a health check.
type CheckResult struct {
	ServiceName  string                 `json:"service_name"`
	Status       string                 `json:"status"`        // "healthy", "warning", "down"
	ResponseTime int64                  `json:"response_time"` // milliseconds
	CheckedAt    int64                  `json:"checked_at"`    // unix timestamp
	Metrics      map[string]interface{} `json:"metrics,omitempty"`
	Dependencies map[string]string      `json:"dependencies,omitempty"` // dependency -> status
	ErrorMessage string                 `json:"error_message,omitempty"`
}

// ServiceDependencies holds references to service dependencies for health checking.
type ServiceDependencies struct {
	Database *sql.DB
	Redis    *redis.Cache
	// Add other common dependencies as needed
}

// getServiceColor returns a color for health status.
func getHealthColor(status string) string {
	switch status {
	case "healthy":
		return colorGreen
	case "warning":
		return colorYellow
	case "down":
		return colorRed
	default:
		return colorCyan
	}
}

// performHealthCheck conducts a comprehensive health check for a service.
func performHealthCheck(ctx context.Context, serviceName string, deps *ServiceDependencies) *CheckResult {
	startTime := time.Now()
	result := &CheckResult{
		ServiceName:  serviceName,
		CheckedAt:    time.Now().Unix(),
		Metrics:      make(map[string]interface{}),
		Dependencies: make(map[string]string),
	}

	// Default to healthy, downgrade if issues found
	result.Status = "healthy"

	// Check database connectivity if available
	if deps != nil && deps.Database != nil {
		if err := deps.Database.PingContext(ctx); err != nil {
			result.Dependencies["database"] = "down"
			result.Status = "down"
			if result.ErrorMessage == "" {
				result.ErrorMessage = fmt.Sprintf("Database connectivity failed: %v", err)
			}
		} else {
			result.Dependencies["database"] = "healthy"

			// Get database stats
			stats := deps.Database.Stats()
			result.Metrics["db_open_connections"] = stats.OpenConnections
			result.Metrics["db_max_open_connections"] = stats.MaxOpenConnections
			result.Metrics["db_in_use"] = stats.InUse
			result.Metrics["db_idle"] = stats.Idle
		}
	}

	// Check Redis connectivity if available
	if deps != nil && deps.Redis != nil {
		redisCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		// Test Redis connectivity with a simple operation
		if err := deps.Redis.Set(redisCtx, "health_check", "ping", "pong", 1*time.Second); err != nil {
			result.Dependencies["redis"] = "down"
			if result.Status == "healthy" {
				result.Status = "warning" // Redis might not be critical
			}
			if result.ErrorMessage == "" {
				result.ErrorMessage = fmt.Sprintf("Redis connectivity failed: %v", err)
			}
		} else {
			result.Dependencies["redis"] = "healthy"

			// Add basic Redis metrics
			result.Metrics["redis_connection"] = "active"
			result.Metrics["redis_last_check"] = time.Now().Unix()
		}
	}

	// Add general service metrics
	result.Metrics["service_uptime"] = time.Since(startTime).Milliseconds()
	result.Metrics["go_routines"] = "available" // Could add runtime.NumGoroutine() if needed

	result.ResponseTime = time.Since(startTime).Milliseconds()

	return result
}

// createHealthHandler creates a health check event handler for a service.
func createHealthHandler(serviceName string, deps *ServiceDependencies, provider *service.Provider) HandlerFunc {
	return func(ctx context.Context, event *nexusv1.EventResponse, log *zap.Logger) {
		// Use event for diagnostics (lint fix)
		_ = event
		// Perform health check
		healthResult := performHealthCheck(ctx, serviceName, deps)

		// Log health check result with color
		color := getHealthColor(healthResult.Status)
		statusMsg := fmt.Sprintf("Health Check: %s - %s", serviceName, healthResult.Status)
		log.Info(fmt.Sprintf("%s%s%s", color, statusMsg, colorReset),
			zap.String("service", serviceName),
			zap.String("status", healthResult.Status),
			zap.Int64("response_time_ms", healthResult.ResponseTime),
			zap.Any("dependencies", healthResult.Dependencies),
		)

		// Determine response event type based on health status
		var responseEventType string
		switch healthResult.Status {
		case "healthy":
			responseEventType = fmt.Sprintf("%s:health:v1:success", serviceName)
		case "warning":
			responseEventType = fmt.Sprintf("%s:health:v1:success", serviceName) // Still success but with warnings
		case "down":
			responseEventType = fmt.Sprintf("%s:health:v1:failed", serviceName)
		default:
			responseEventType = fmt.Sprintf("%s:health:v1:failed", serviceName)
		}

		// Create response payload with proper type conversion for protobuf
		responsePayload := map[string]interface{}{
			"service_name":  healthResult.ServiceName,
			"status":        healthResult.Status,
			"response_time": healthResult.ResponseTime,
			"checked_at":    healthResult.CheckedAt,
		}

		// Convert dependencies map[string]string to map[string]interface{} for protobuf compatibility
		if len(healthResult.Dependencies) > 0 {
			dependencies := make(map[string]interface{})
			for k, v := range healthResult.Dependencies {
				dependencies[k] = v
			}
			responsePayload["dependencies"] = dependencies
		}

		// Add metrics if available
		if len(healthResult.Metrics) > 0 {
			responsePayload["metrics"] = healthResult.Metrics
		}

		// Add error message if there is one
		if healthResult.ErrorMessage != "" {
			responsePayload["error_message"] = healthResult.ErrorMessage
		}

		// Convert to structpb for the response
		payloadStruct, err := structpb.NewStruct(responsePayload)
		if err != nil {
			log.Error("Failed to create health response payload",
				zap.String("service", healthResult.ServiceName),
				zap.Error(err))
			return
		}

		// Create and emit the health response event
		log.Info("Health check response ready",
			zap.String("response_event_type", responseEventType),
			zap.Any("response_payload", responsePayload),
		)

		// Emit the actual response event via the service's nexus provider
		if provider != nil {
			// Create metadata with the health response payload
			meta := &commonpb.Metadata{
				ServiceSpecific: payloadStruct,
			}

			err := provider.EmitEvent(ctx, responseEventType, "health-check", meta)
			if err != nil {
				log.Error("Failed to emit health response event",
					zap.String("service", serviceName),
					zap.String("event_type", responseEventType),
					zap.Error(err),
				)
			} else {
				log.Info("Health response event emitted successfully",
					zap.String("service", serviceName),
					zap.String("event_type", responseEventType),
				)
			}
		} else {
			log.Warn("No provider available to emit health response event",
				zap.String("service", serviceName),
				zap.String("event_type", responseEventType),
			)
		}
	}
}

// StartHealthSubscriber subscribes to health check events and responds with health status.
func StartHealthSubscriber(ctx context.Context, provider *service.Provider, log *zap.Logger, serviceName string, deps *ServiceDependencies) {
	// Subscribe to health check requests for this service
	healthEventType := fmt.Sprintf("%s:health:v1:requested", serviceName)

	sub := Subscription{
		Event:   healthEventType,
		Handler: createHealthHandler(serviceName, deps, provider),
	}

	go func() {
		err := provider.SubscribeEvents(ctx, []string{sub.Event}, nil, func(ctx context.Context, event *nexusv1.EventResponse) {
			// Use the event parameter for diagnostics
			if event != nil {
				log.Debug("Received health event", zap.String("event_type", event.EventType))
			}
			sub.Handler(ctx, event, log)
		})
		if err != nil {
			log.Error("Failed to subscribe to health check events",
				zap.String("service", serviceName),
				zap.String("event", sub.Event),
				zap.Error(err),
			)
		} else {
			log.Info("Health check subscriber started",
				zap.String("service", serviceName),
				zap.String("listening_for", sub.Event),
			)
		}
	}()
}

// StartHealthHeartbeat emits periodic health status updates (optional).
func StartHealthHeartbeat(ctx context.Context, provider *service.Provider, log *zap.Logger, serviceName string, deps *ServiceDependencies, interval time.Duration) {
	if interval == 0 {
		interval = 60 * time.Second // Default to 1 minute
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Info("Health heartbeat stopped", zap.String("service", serviceName))
				return
			case <-ticker.C:
				// Perform health check
				healthResult := performHealthCheck(ctx, serviceName, deps)

				// Log heartbeat
				color := getHealthColor(healthResult.Status)
				statusMsg := fmt.Sprintf("Health Heartbeat: %s - %s", serviceName, healthResult.Status)
				log.Info(fmt.Sprintf("%s%s%s", color, statusMsg, colorReset),
					zap.String("service", serviceName),
					zap.String("status", healthResult.Status),
					zap.Int64("response_time_ms", healthResult.ResponseTime),
				)

				// Optionally emit a heartbeat event
				if provider != nil {
					heartbeatEventType := fmt.Sprintf("%s:health:v1:heartbeat", serviceName)
					// TODO: Emit heartbeat event via provider
					log.Debug("Health heartbeat event ready", zap.String("event_type", heartbeatEventType))
				}
			}
		}
	}()
}
