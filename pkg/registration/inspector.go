package registration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/nmxmxh/master-ovasabi/config/registry"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"go.uber.org/zap"
)

// DynamicInspector provides runtime service inspection and registration
type DynamicInspector struct {
	logger    *zap.Logger
	container *di.Container
	generator *DynamicServiceRegistrationGenerator
}

// NewDynamicInspector creates a new dynamic inspector
func NewDynamicInspector(logger *zap.Logger, container *di.Container, protoPath, srcPath string) *DynamicInspector {
	return &DynamicInspector{
		logger:    logger,
		container: container,
		generator: NewDynamicServiceRegistrationGenerator(logger, protoPath, srcPath),
	}
}

// InspectRegisteredServices inspects all services registered in the DI container
func (di *DynamicInspector) InspectRegisteredServices(ctx context.Context) ([]ServiceRegistrationConfig, error) {
	var configs []ServiceRegistrationConfig

	// This would require enhanced DI container functionality to list registered services
	// For now, we'll demonstrate with known service types

	knownServices := []interface{}{
		// These would be actual service interface types
		// (*userpb.UserServiceServer)(nil),
		// (*notificationpb.NotificationServiceServer)(nil),
		// etc.
	}

	for _, serviceType := range knownServices {
		config, err := di.generator.IntrospectService(serviceType)
		if err != nil {
			di.logger.Warn("Failed to introspect service", zap.Error(err))
			continue
		}
		configs = append(configs, *config)
	}

	return configs, nil
}

// AutoRegisterFromRuntime automatically registers services found at runtime
func (di *DynamicInspector) AutoRegisterFromRuntime(ctx context.Context) error {
	// Get all registered services from DI container
	// This is a conceptual implementation - actual implementation would depend on
	// enhanced DI container that can enumerate registered services

	di.logger.Info("Starting automatic service registration from runtime")

	// For demonstration, we'll register with the static registry
	sampleService := registry.ServiceRegistration{
		ServiceName:  "runtime-discovered",
		Methods:      []registry.ServiceMethod{},
		RegisteredAt: time.Now(),
		Description:  "Auto-discovered service at runtime",
		Version:      "v1",
		External:     false,
	}

	registry.RegisterService(sampleService)

	di.logger.Info("Completed automatic service registration")
	return nil
}

// GenerateConfigFromRuntime generates configuration from currently running services
func (di *DynamicInspector) GenerateConfigFromRuntime(ctx context.Context, outputPath string) error {
	configs, err := di.InspectRegisteredServices(ctx)
	if err != nil {
		return err
	}

	// Also generate from proto files
	protoConfigs, err := di.generator.GenerateServiceRegistrations(ctx)
	if err != nil {
		di.logger.Warn("Failed to generate from proto files", zap.Error(err))
	} else {
		configs = append(configs, protoConfigs...)
	}

	// Remove duplicates
	uniqueConfigs := di.removeDuplicateConfigs(configs)

	// Save to file
	jsonData, err := json.MarshalIndent(uniqueConfigs, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
		return err
	}

	di.logger.Info("Generated service registration from runtime",
		zap.String("output", outputPath),
		zap.Int("services", len(uniqueConfigs)))

	return nil
}

// InspectService provides detailed inspection of a specific service
func (di *DynamicInspector) InspectService(serviceName string) (*ServiceInspectionResult, error) {
	result := &ServiceInspectionResult{
		ServiceName: serviceName,
		Timestamp:   time.Now(),
	}

	// Try to resolve from DI container
	// This is conceptual - actual implementation would need DI container enhancement

	// Check if service is in static registry
	serviceRegistry := registry.GetServiceRegistry()
	if svc, exists := serviceRegistry[serviceName]; exists {
		result.IsRegistered = true
		result.RegistrationInfo = &svc
		result.Methods = make([]MethodInfo, len(svc.Methods))

		for i, method := range svc.Methods {
			result.Methods[i] = MethodInfo{
				Name:        method.Name,
				Parameters:  method.Parameters,
				Description: method.Description,
			}
		}
	}

	// Add runtime information
	result.RuntimeInfo = di.gatherRuntimeInfo(serviceName)

	return result, nil
}

// ServiceInspectionResult contains detailed service inspection information
type ServiceInspectionResult struct {
	ServiceName      string                        `json:"service_name"`
	IsRegistered     bool                          `json:"is_registered"`
	RegistrationInfo *registry.ServiceRegistration `json:"registration_info,omitempty"`
	Methods          []MethodInfo                  `json:"methods"`
	RuntimeInfo      *RuntimeInfo                  `json:"runtime_info,omitempty"`
	Timestamp        time.Time                     `json:"timestamp"`
}

// MethodInfo contains method inspection information
type MethodInfo struct {
	Name        string   `json:"name"`
	Parameters  []string `json:"parameters"`
	Description string   `json:"description"`
	IsExported  bool     `json:"is_exported"`
}

// RuntimeInfo contains runtime inspection information
type RuntimeInfo struct {
	GoVersion     string            `json:"go_version"`
	MemoryStats   *runtime.MemStats `json:"memory_stats,omitempty"`
	NumGoroutines int               `json:"num_goroutines"`
	ProcessInfo   map[string]string `json:"process_info"`
}

// gatherRuntimeInfo gathers runtime information about the service
func (di *DynamicInspector) gatherRuntimeInfo(serviceName string) *RuntimeInfo {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return &RuntimeInfo{
		GoVersion:     runtime.Version(),
		MemoryStats:   &memStats,
		NumGoroutines: runtime.NumGoroutine(),
		ProcessInfo: map[string]string{
			"goos":   runtime.GOOS,
			"goarch": runtime.GOARCH,
		},
	}
}

// removeDuplicateConfigs removes duplicate service configurations
func (di *DynamicInspector) removeDuplicateConfigs(configs []ServiceRegistrationConfig) []ServiceRegistrationConfig {
	seen := make(map[string]bool)
	var result []ServiceRegistrationConfig

	for _, config := range configs {
		key := fmt.Sprintf("%s-%s", config.Name, config.Version)
		if !seen[key] {
			seen[key] = true
			result = append(result, config)
		}
	}

	return result
}

// ValidateServiceRegistration validates a service registration against runtime
func (di *DynamicInspector) ValidateServiceRegistration(config ServiceRegistrationConfig) (*ValidationResult, error) {
	result := &ValidationResult{
		ServiceName: config.Name,
		IsValid:     true,
		Issues:      []string{},
		Suggestions: []string{},
	}

	// Check if service exists in registry
	serviceRegistry := registry.GetServiceRegistry()
	if _, exists := serviceRegistry[config.Name]; !exists {
		result.Issues = append(result.Issues, "Service not found in registry")
		result.IsValid = false
	}

	// Validate capabilities
	if len(config.Capabilities) == 0 {
		result.Issues = append(result.Issues, "No capabilities defined")
		result.Suggestions = append(result.Suggestions, "Add at least one capability")
	}

	// Validate proto path
	if config.Schema.ProtoPath != "" {
		if _, err := os.Stat(config.Schema.ProtoPath); os.IsNotExist(err) {
			result.Issues = append(result.Issues, fmt.Sprintf("Proto file not found: %s", config.Schema.ProtoPath))
			result.IsValid = false
		}
	}

	// Validate action map consistency
	for actionName, action := range config.ActionMap {
		if action.ProtoMethod == "" {
			result.Issues = append(result.Issues, fmt.Sprintf("Action %s missing proto method", actionName))
			result.IsValid = false
		}
	}

	return result, nil
}

// ValidationResult contains service validation results
type ValidationResult struct {
	ServiceName string   `json:"service_name"`
	IsValid     bool     `json:"is_valid"`
	Issues      []string `json:"issues"`
	Suggestions []string `json:"suggestions"`
}

// ComparateConfigurations compares two service configurations
func (di *DynamicInspector) CompareConfigurations(config1, config2 ServiceRegistrationConfig) *ComparisonResult {
	result := &ComparisonResult{
		Service1:     config1.Name,
		Service2:     config2.Name,
		Differences:  []string{},
		Similarities: []string{},
	}

	// Compare versions
	if config1.Version != config2.Version {
		result.Differences = append(result.Differences,
			fmt.Sprintf("Version: %s vs %s", config1.Version, config2.Version))
	} else {
		result.Similarities = append(result.Similarities,
			fmt.Sprintf("Same version: %s", config1.Version))
	}

	// Compare capabilities
	cap1Set := make(map[string]bool)
	for _, cap := range config1.Capabilities {
		cap1Set[cap] = true
	}

	cap2Set := make(map[string]bool)
	for _, cap := range config2.Capabilities {
		cap2Set[cap] = true
	}

	// Find unique capabilities
	for cap := range cap1Set {
		if !cap2Set[cap] {
			result.Differences = append(result.Differences,
				fmt.Sprintf("Capability '%s' only in %s", cap, config1.Name))
		}
	}

	for cap := range cap2Set {
		if !cap1Set[cap] {
			result.Differences = append(result.Differences,
				fmt.Sprintf("Capability '%s' only in %s", cap, config2.Name))
		}
	}

	// Find common capabilities
	for cap := range cap1Set {
		if cap2Set[cap] {
			result.Similarities = append(result.Similarities,
				fmt.Sprintf("Common capability: %s", cap))
		}
	}

	return result
}

// ComparisonResult contains service comparison results
type ComparisonResult struct {
	Service1     string   `json:"service1"`
	Service2     string   `json:"service2"`
	Differences  []string `json:"differences"`
	Similarities []string `json:"similarities"`
}

// OptimizeServiceRegistration suggests optimizations for a service configuration
func (di *DynamicInspector) OptimizeServiceRegistration(config ServiceRegistrationConfig) *OptimizationSuggestions {
	suggestions := &OptimizationSuggestions{
		ServiceName: config.Name,
		Suggestions: []string{},
	}

	// Check for redundant capabilities
	if di.hasRedundantCapabilities(config.Capabilities) {
		suggestions.Suggestions = append(suggestions.Suggestions,
			"Remove redundant capabilities to simplify configuration")
	}

	// Check for missing metadata enrichment
	if !config.MetadataEnrichment {
		suggestions.Suggestions = append(suggestions.Suggestions,
			"Enable metadata enrichment for better orchestration")
	}

	// Check for optimal endpoint structure
	if len(config.Endpoints) > 1 {
		suggestions.Suggestions = append(suggestions.Suggestions,
			"Consider consolidating endpoints into a single operations endpoint")
	}

	// Check for proper health and metrics endpoints
	if !strings.Contains(config.HealthCheck, config.Name) {
		suggestions.Suggestions = append(suggestions.Suggestions,
			"Health check endpoint should include service name")
	}

	return suggestions
}

// OptimizationSuggestions contains optimization suggestions
type OptimizationSuggestions struct {
	ServiceName string   `json:"service_name"`
	Suggestions []string `json:"suggestions"`
}

// hasRedundantCapabilities checks for redundant capabilities
func (di *DynamicInspector) hasRedundantCapabilities(capabilities []string) bool {
	// Simple check for obvious redundancies
	capMap := make(map[string]bool)
	for _, cap := range capabilities {
		if capMap[cap] {
			return true // Duplicate found
		}
		capMap[cap] = true
	}
	return false
}

// ExportServiceGraph exports the service dependency graph
func (di *DynamicInspector) ExportServiceGraph(configs []ServiceRegistrationConfig, outputPath string) error {
	graph := &ServiceGraph{
		Services:     make(map[string]ServiceNode),
		Dependencies: []Dependency{},
		Timestamp:    time.Now(),
	}

	// Build nodes
	for _, config := range configs {
		graph.Services[config.Name] = ServiceNode{
			Name:         config.Name,
			Version:      config.Version,
			Capabilities: config.Capabilities,
		}
	}

	// Build dependencies
	for _, config := range configs {
		for _, dep := range config.Dependencies {
			graph.Dependencies = append(graph.Dependencies, Dependency{
				From: config.Name,
				To:   dep,
				Type: "requires",
			})
		}
	}

	// Save to file
	jsonData, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(outputPath, jsonData, 0644)
}

// ServiceGraph represents the service dependency graph
type ServiceGraph struct {
	Services     map[string]ServiceNode `json:"services"`
	Dependencies []Dependency           `json:"dependencies"`
	Timestamp    time.Time              `json:"timestamp"`
}

// ServiceNode represents a service in the graph
type ServiceNode struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Capabilities []string `json:"capabilities"`
}

// Dependency represents a service dependency
type Dependency struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}
