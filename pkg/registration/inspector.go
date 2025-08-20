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

// DynamicInspector provides runtime service inspection and registration.
type DynamicInspector struct {
	logger    *zap.Logger
	container *di.Container
	generator *DynamicServiceRegistrationGenerator
}

// convertMethods converts ServiceRegistrationConfig to registry.ServiceMethod slice.
func (inspector *DynamicInspector) convertMethods(config ServiceRegistrationConfig) []registry.ServiceMethod {
	methods := make([]registry.ServiceMethod, 0, len(config.Schema.Methods)+len(config.ActionMap))
	// Handle string slice from Schema.Methods
	for _, methodName := range config.Schema.Methods {
		methods = append(methods, registry.ServiceMethod{
			Name: methodName,
		})
	}
	// Optionally enrich with ActionMap if present
	for name, action := range config.ActionMap {
		methods = append(methods, registry.ServiceMethod{
			Name:        name,
			Description: action.ProtoMethod,
			Parameters:  action.RestRequiredFields,
		})
	}
	return methods
}

// NewDynamicInspector creates a new dynamic inspector.
func NewDynamicInspector(logger *zap.Logger, container *di.Container, protoPath, srcPath string) *DynamicInspector {
	return &DynamicInspector{
		logger:    logger,
		container: container,
		generator: NewDynamicServiceRegistrationGenerator(logger, protoPath, srcPath),
	}
}

// InspectRegisteredServices inspects all services registered in the DI container.
func (inspector *DynamicInspector) InspectRegisteredServices(ctx context.Context) ([]ServiceRegistrationConfig, error) {
	// Use context for diagnostics/cancellation (lint fix)
	if ctx != nil && ctx.Err() != nil {
		return nil, ctx.Err()
	}
	serviceTypes := []interface{}{
		// (*userpb.UserServiceServer)(nil),
		// (*notificationpb.NotificationServiceServer)(nil),
	}
	configs := make([]ServiceRegistrationConfig, 0, len(serviceTypes))
	for _, serviceType := range serviceTypes {
		config, err := inspector.generator.IntrospectService(serviceType)
		if err != nil {
			inspector.logger.Warn("Failed to introspect service", zap.Error(err))
			continue
		}
		configs = append(configs, *config)
	}
	return configs, nil
}

// AutoRegisterFromRuntime automatically registers services found at runtime.
func (inspector *DynamicInspector) AutoRegisterFromRuntime(ctx context.Context) error {
	// Use context for diagnostics/cancellation (lint fix)
	if ctx != nil && ctx.Err() != nil {
		inspector.logger.Warn("AutoRegisterFromRuntime cancelled by context", zap.Error(ctx.Err()))
		return ctx.Err()
	}
	inspector.logger.Info("Starting automatic service registration from runtime")

	serviceTypes := []interface{}{}

	for _, serviceType := range serviceTypes {
		config, err := inspector.generator.IntrospectService(serviceType)
		if err != nil || config == nil {
			inspector.logger.Warn("Failed to introspect service", zap.Error(err))
			continue
		}
		reg := registry.ServiceRegistration{
			ServiceName:  config.Name,
			Methods:      inspector.convertMethods(*config),
			RegisteredAt: time.Now(),
			Description:  fmt.Sprintf("Auto-registered service: %s", config.Name),
			Version:      config.Version,
			External:     false,
		}
		registry.RegisterService(reg)
	}

	inspector.logger.Info("Completed automatic service registration")
	return nil
}

// GenerateConfigFromRuntime generates configuration from currently running services.
func (inspector *DynamicInspector) GenerateConfigFromRuntime(ctx context.Context, outputPath string) error {
	configs, err := inspector.InspectRegisteredServices(ctx)
	if err != nil {
		return err
	}

	// Also generate from proto files
	protoConfigs, err := inspector.generator.GenerateServiceRegistrations(ctx)
	if err != nil {
		inspector.logger.Warn("Failed to generate from proto files", zap.Error(err))
	} else {
		configs = append(configs, protoConfigs...)
	}

	// Remove duplicates
	uniqueConfigs := inspector.removeDuplicateConfigs(configs)

	// Save to file
	jsonData, err := json.MarshalIndent(uniqueConfigs, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(outputPath, jsonData, 0o600); err != nil {
		return err
	}

	inspector.logger.Info("Generated service registration from runtime",
		zap.String("output", outputPath),
		zap.Int("services", len(uniqueConfigs)))

	return nil
}

// InspectService provides detailed inspection of a specific service.
func (inspector *DynamicInspector) InspectService(serviceName string) (*ServiceInspectionResult, error) {
	result := &ServiceInspectionResult{
		ServiceName: serviceName,
		Timestamp:   time.Now(),
	}

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
	result.RuntimeInfo = inspector.gatherRuntimeInfo(serviceName)

	return result, nil
}

// ServiceInspectionResult contains detailed service inspection information.
type ServiceInspectionResult struct {
	ServiceName      string                        `json:"service_name"`
	IsRegistered     bool                          `json:"is_registered"`
	RegistrationInfo *registry.ServiceRegistration `json:"registration_info,omitempty"`
	Methods          []MethodInfo                  `json:"methods"`
	RuntimeInfo      *RuntimeInfo                  `json:"runtime_info,omitempty"`
	Timestamp        time.Time                     `json:"timestamp"`
}

// MethodInfo contains method inspection information.
type MethodInfo struct {
	Name        string   `json:"name"`
	Parameters  []string `json:"parameters"`
	Description string   `json:"description"`
	IsExported  bool     `json:"is_exported"`
}

// RuntimeInfo contains runtime inspection information.
type RuntimeInfo struct {
	GoVersion     string            `json:"go_version"`
	MemoryStats   *runtime.MemStats `json:"memory_stats,omitempty"`
	NumGoroutines int               `json:"num_goroutines"`
	ProcessInfo   map[string]string `json:"process_info"`
}

// gatherRuntimeInfo gathers runtime information about the service.
func (inspector *DynamicInspector) gatherRuntimeInfo(_ string) *RuntimeInfo {
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

// removeDuplicateConfigs removes duplicate service configurations.
func (inspector *DynamicInspector) removeDuplicateConfigs(configs []ServiceRegistrationConfig) []ServiceRegistrationConfig {
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

// ValidateServiceRegistration validates a service registration against runtime.
func (inspector *DynamicInspector) ValidateServiceRegistration(config ServiceRegistrationConfig) (*ValidationResult, error) {
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

// ValidationResult contains service validation results.
type ValidationResult struct {
	ServiceName string   `json:"service_name"`
	IsValid     bool     `json:"is_valid"`
	Issues      []string `json:"issues"`
	Suggestions []string `json:"suggestions"`
}

// ComparateConfigurations compares two service configurations.
func (inspector *DynamicInspector) CompareConfigurations(config1, config2 ServiceRegistrationConfig) *ComparisonResult {
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

// ComparisonResult contains service comparison results.
type ComparisonResult struct {
	Service1     string   `json:"service1"`
	Service2     string   `json:"service2"`
	Differences  []string `json:"differences"`
	Similarities []string `json:"similarities"`
}

// OptimizeServiceRegistration suggests optimizations for a service configuration.
func (inspector *DynamicInspector) OptimizeServiceRegistration(config ServiceRegistrationConfig) *OptimizationSuggestions {
	suggestions := &OptimizationSuggestions{
		ServiceName: config.Name,
		Suggestions: []string{},
	}

	// Check for redundant capabilities
	if inspector.hasRedundantCapabilities(config.Capabilities) {
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

// OptimizationSuggestions contains optimization suggestions.
type OptimizationSuggestions struct {
	ServiceName string   `json:"service_name"`
	Suggestions []string `json:"suggestions"`
}

// hasRedundantCapabilities checks for redundant capabilities.
func (inspector *DynamicInspector) hasRedundantCapabilities(capabilities []string) bool {
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

// ExportServiceGraph exports the service dependency graph.
func (inspector *DynamicInspector) ExportServiceGraph(configs []ServiceRegistrationConfig, outputPath string) error {
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

	return os.WriteFile(outputPath, jsonData, 0o600)
}

// ServiceGraph represents the service dependency graph.
type ServiceGraph struct {
	Services     map[string]ServiceNode `json:"services"`
	Dependencies []Dependency           `json:"dependencies"`
	Timestamp    time.Time              `json:"timestamp"`
}

// ServiceNode represents a service in the graph.
type ServiceNode struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Capabilities []string `json:"capabilities"`
}

// Dependency represents a service dependency.
type Dependency struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}
