package registration

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// HealthChecker provides health checking capabilities for services.
type HealthChecker struct {
	logger *zap.Logger
	client *http.Client
}

// NewHealthChecker creates a new health checker.
func NewHealthChecker(logger *zap.Logger) *HealthChecker {
	return &HealthChecker{
		logger: logger,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// HealthStatus represents the health status of a service.
type HealthStatus struct {
	ServiceName  string                 `json:"service_name"`
	IsHealthy    bool                   `json:"is_healthy"`
	ResponseTime time.Duration          `json:"response_time"`
	StatusCode   int                    `json:"status_code,omitempty"`
	Error        string                 `json:"error,omitempty"`
	LastChecked  time.Time              `json:"last_checked"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// HealthCheckResult contains the results of checking multiple services.
type HealthCheckResult struct {
	TotalServices     int                     `json:"total_services"`
	HealthyServices   int                     `json:"healthy_services"`
	UnhealthyServices int                     `json:"unhealthy_services"`
	Services          map[string]HealthStatus `json:"services"`
	CheckedAt         time.Time               `json:"checked_at"`
}

// CheckServiceHealth checks the health of a single service.
func (hc *HealthChecker) CheckServiceHealth(ctx context.Context, config ServiceRegistrationConfig) HealthStatus {
	status := HealthStatus{
		ServiceName: config.Name,
		LastChecked: time.Now(),
	}

	// Try to determine health check endpoint
	healthEndpoint := hc.getHealthEndpoint(config)
	if healthEndpoint == "" {
		status.Error = "No health endpoint configured"
		return status
	}

	// Perform health check
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthEndpoint, http.NoBody)
	if err != nil {
		status.Error = fmt.Sprintf("Failed to create request: %v", err)
		return status
	}

	resp, err := hc.client.Do(req)
	status.ResponseTime = time.Since(start)

	if err != nil {
		status.Error = fmt.Sprintf("Request failed: %v", err)
		return status
	}
	defer resp.Body.Close()

	status.StatusCode = resp.StatusCode
	status.IsHealthy = resp.StatusCode >= 200 && resp.StatusCode < 300

	if !status.IsHealthy {
		status.Error = fmt.Sprintf("Unhealthy response: %d", resp.StatusCode)
	}

	// Add metadata if available
	status.Metadata = map[string]interface{}{
		"endpoint": healthEndpoint,
		"method":   "GET",
	}

	return status
}

// CheckAllServices checks the health of all configured services.
func (hc *HealthChecker) CheckAllServices(ctx context.Context, configs []ServiceRegistrationConfig) *HealthCheckResult {
	result := &HealthCheckResult{
		TotalServices: len(configs),
		Services:      make(map[string]HealthStatus),
		CheckedAt:     time.Now(),
	}

	// Check each service
	for _, config := range configs {
		status := hc.CheckServiceHealth(ctx, config)
		result.Services[config.Name] = status

		if status.IsHealthy {
			result.HealthyServices++
		} else {
			result.UnhealthyServices++
		}
	}

	return result
}

// getHealthEndpoint tries to determine the health check endpoint for a service.
func (hc *HealthChecker) getHealthEndpoint(config ServiceRegistrationConfig) string {
	// Check if there's a specific health endpoint configured
	if config.HealthCheck != "" {
		return config.HealthCheck
	}

	// Check for standard health endpoints in REST routes
	for _, route := range config.Endpoints {
		if route.Path == "/health" || route.Path == "/healthz" || route.Path == "/ready" {
			// Construct full URL - this is a simplified approach
			// In a real implementation, you'd need the service's base URL
			return fmt.Sprintf("http://localhost:8080%s", route.Path)
		}
	}

	// Try common health endpoints
	baseURL := hc.getServiceBaseURL(config)
	if baseURL != "" {
		return fmt.Sprintf("%s/health", baseURL)
	}

	return ""
}

// getServiceBaseURL tries to determine the base URL for a service.
func (hc *HealthChecker) getServiceBaseURL(config ServiceRegistrationConfig) string {
	// Check for metrics endpoint to infer base URL
	if config.Metrics != "" {
		// Extract base URL from metrics endpoint
		return config.Metrics
	}

	// Default fallback - this would need to be configurable in a real system
	return ""
}

// HealthMonitor provides continuous health monitoring.
type HealthMonitor struct {
	checker  *HealthChecker
	logger   *zap.Logger
	interval time.Duration
	configs  []ServiceRegistrationConfig
}

// NewHealthMonitor creates a new health monitor.
func NewHealthMonitor(logger *zap.Logger, configs []ServiceRegistrationConfig, interval time.Duration) *HealthMonitor {
	return &HealthMonitor{
		checker:  NewHealthChecker(logger),
		logger:   logger,
		interval: interval,
		configs:  configs,
	}
}

// Start begins continuous health monitoring.
func (hm *HealthMonitor) Start(ctx context.Context) <-chan *HealthCheckResult {
	resultChan := make(chan *HealthCheckResult, 1)

	go func() {
		defer close(resultChan)

		ticker := time.NewTicker(hm.interval)
		defer ticker.Stop()

		// Initial health check
		result := hm.checker.CheckAllServices(ctx, hm.configs)
		hm.logHealthResult(result)

		select {
		case resultChan <- result:
		case <-ctx.Done():
			return
		}

		// Continuous monitoring
		for {
			select {
			case <-ticker.C:
				result := hm.checker.CheckAllServices(ctx, hm.configs)
				hm.logHealthResult(result)

				select {
				case resultChan <- result:
				case <-ctx.Done():
					return
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	return resultChan
}

// logHealthResult logs the health check results.
func (hm *HealthMonitor) logHealthResult(result *HealthCheckResult) {
	hm.logger.Info("Health check completed",
		zap.Int("total", result.TotalServices),
		zap.Int("healthy", result.HealthyServices),
		zap.Int("unhealthy", result.UnhealthyServices),
		zap.Time("checked_at", result.CheckedAt),
	)

	// Log unhealthy services
	for name, status := range result.Services {
		if !status.IsHealthy {
			hm.logger.Warn("Service unhealthy",
				zap.String("service", name),
				zap.String("error", status.Error),
				zap.Duration("response_time", status.ResponseTime),
			)
		}
	}
}
