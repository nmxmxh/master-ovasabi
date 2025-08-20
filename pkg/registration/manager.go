package registration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/nmxmxh/master-ovasabi/config/registry"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"go.uber.org/zap"
)

// DynamicRegistrationManager manages dynamic service registration.
type DynamicRegistrationManager struct {
	logger    *zap.Logger
	generator *DynamicServiceRegistrationGenerator
	inspector *DynamicInspector
	container *di.Container
}

// NewDynamicRegistrationManager creates a new dynamic registration manager.
func NewDynamicRegistrationManager(
	logger *zap.Logger,
	container *di.Container,
	protoPath, srcPath string,
) *DynamicRegistrationManager {
	generator := NewDynamicServiceRegistrationGenerator(logger, protoPath, srcPath)
	inspector := NewDynamicInspector(logger, container, protoPath, srcPath)

	return &DynamicRegistrationManager{
		logger:    logger,
		generator: generator,
		inspector: inspector,
		container: container,
	}
}

// AutoRegisterServices automatically registers services based on proto definitions
// This can replace manual service registration in bootstrap.
func (drm *DynamicRegistrationManager) AutoRegisterServices(ctx context.Context) error {
	drm.logger.Info("Starting automatic service registration")

	// Generate service configurations from proto files
	configs, err := drm.generator.GenerateServiceRegistrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate service configurations: %w", err)
	}

	// Register each service with the registry
	for _, config := range configs {
		serviceReg := registry.ServiceRegistration{
			ServiceName:  config.Name,
			Methods:      drm.convertMethods(config),
			RegisteredAt: time.Now(),
			Description:  fmt.Sprintf("Auto-generated registration for %s service", config.Name),
			Version:      config.Version,
			External:     false,
		}

		registry.RegisterService(serviceReg)
		drm.logger.Info("Auto-registered service",
			zap.String("service", config.Name),
			zap.Int("methods", len(serviceReg.Methods)))
	}

	drm.logger.Info("Completed automatic service registration",
		zap.Int("services", len(configs)))

	return nil
}

// ValidateExistingRegistrations validates existing service registrations.
func (drm *DynamicRegistrationManager) ValidateExistingRegistrations() error {
	drm.logger.Info("Validating existing service registrations")

	// Load existing configurations
	configs, err := drm.loadExistingConfigs()
	if err != nil {
		return fmt.Errorf("failed to load existing configurations: %w", err)
	}

	var invalidServices []string
	for _, config := range configs {
		result, err := drm.inspector.ValidateServiceRegistration(config)
		if err != nil {
			drm.logger.Error("Failed to validate service",
				zap.String("service", config.Name),
				zap.Error(err))
			continue
		}

		if !result.IsValid {
			invalidServices = append(invalidServices, config.Name)
			drm.logger.Warn("Service configuration has issues",
				zap.String("service", config.Name),
				zap.Strings("issues", result.Issues))
		}

		if len(result.Suggestions) > 0 {
			drm.logger.Info("Service optimization suggestions",
				zap.String("service", config.Name),
				zap.Strings("suggestions", result.Suggestions))
		}
	}

	if len(invalidServices) > 0 {
		return fmt.Errorf("invalid service configurations: %v", invalidServices)
	}

	drm.logger.Info("All service registrations are valid")
	return nil
}

// SyncWithProtoFiles synchronizes service registrations with proto file changes.
func (drm *DynamicRegistrationManager) SyncWithProtoFiles(ctx context.Context) error {
	drm.logger.Info("Synchronizing service registrations with proto files")

	// Generate new configurations
	newConfigs, err := drm.generator.GenerateServiceRegistrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate new configurations: %w", err)
	}

	// Load existing configurations
	existingConfigs, err := drm.loadExistingConfigs()
	if err != nil {
		drm.logger.Warn("Failed to load existing configurations, proceeding with new only",
			zap.Error(err))
		existingConfigs = []ServiceRegistrationConfig{}
	}

	// Compare and update
	updated := 0
	for _, newConfig := range newConfigs {
		// Find corresponding existing config
		var existingConfig *ServiceRegistrationConfig
		for _, existing := range existingConfigs {
			if existing.Name == newConfig.Name {
				existingConfig = &existing
				break
			}
		}

		if existingConfig == nil {
			// New service - register it
			drm.registerNewService(newConfig)
			updated++
			drm.logger.Info("Registered new service", zap.String("service", newConfig.Name))
		} else if drm.hasSignificantChanges(*existingConfig, newConfig) {
			drm.updateExistingService(*existingConfig, newConfig)
			updated++
			drm.logger.Info("Updated service registration",
				zap.String("service", newConfig.Name))
		}
	}

	drm.logger.Info("Completed synchronization", zap.Int("updated", updated))
	return nil
}

// GenerateServiceGraph generates and saves service dependency graph.
func (drm *DynamicRegistrationManager) GenerateServiceGraph(outputPath string) error {
	configs, err := drm.loadExistingConfigs()
	if err != nil {
		return fmt.Errorf("failed to load configurations: %w", err)
	}

	if err := drm.inspector.ExportServiceGraph(configs, outputPath); err != nil {
		return fmt.Errorf("failed to export service graph: %w", err)
	}

	drm.logger.Info("Generated service dependency graph", zap.String("output", outputPath))
	return nil
}

// loadExistingConfigs loads existing service configurations.
func (drm *DynamicRegistrationManager) loadExistingConfigs() ([]ServiceRegistrationConfig, error) {
	// Try generated config first, then fallback to manual config
	configs, err := drm.loadConfigFile("config/service_registration_generated.json")
	if err != nil {
		configs, err = drm.loadConfigFile("config/service_registration.json")
		if err != nil {
			return nil, err
		}
	}
	return configs, nil
}

// loadConfigFile loads configuration from a file.
func (drm *DynamicRegistrationManager) loadConfigFile(path string) ([]ServiceRegistrationConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var configs []ServiceRegistrationConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return nil, err
	}

	return configs, nil
}

// convertMethods converts action configs to registry methods.
func (drm *DynamicRegistrationManager) convertMethods(config ServiceRegistrationConfig) []registry.ServiceMethod {
	methods := make([]registry.ServiceMethod, 0, len(config.ActionMap))

	for _, action := range config.ActionMap {
		methods = append(methods, registry.ServiceMethod{
			Name:        action.ProtoMethod,
			Parameters:  action.RestRequiredFields,
			Description: fmt.Sprintf("Auto-generated method for %s", action.ProtoMethod),
		})
	}

	return methods
}

// registerNewService registers a new service with the registry.
func (drm *DynamicRegistrationManager) registerNewService(config ServiceRegistrationConfig) {
	serviceReg := registry.ServiceRegistration{
		ServiceName:  config.Name,
		Methods:      drm.convertMethods(config),
		RegisteredAt: time.Now(),
		Description:  fmt.Sprintf("Auto-registered service: %s", config.Name),
		Version:      config.Version,
		External:     false,
	}

	registry.RegisterService(serviceReg)
}

// updateExistingService updates an existing service registration.
func (drm *DynamicRegistrationManager) updateExistingService(
	_ ServiceRegistrationConfig,
	updatedConfig ServiceRegistrationConfig,
) {
	// For now, just re-register the service
	// In a more sophisticated implementation, you could do incremental updates
	drm.registerNewService(updatedConfig)
}

// hasSignificantChanges checks if there are significant changes between configs.
func (drm *DynamicRegistrationManager) hasSignificantChanges(
	existing ServiceRegistrationConfig,
	updatedConfig ServiceRegistrationConfig,
) bool {
	// Compare key aspects
	if existing.Version != updatedConfig.Version {
		return true
	}

	if len(existing.Schema.Methods) != len(updatedConfig.Schema.Methods) {
		return true
	}

	// Check if methods have changed
	existingMethods := make(map[string]bool)
	for _, method := range existing.Schema.Methods {
		existingMethods[method] = true
	}

	for _, method := range updatedConfig.Schema.Methods {
		if !existingMethods[method] {
			return true // New method found
		}
	}

	return false
}

// EnableAutoSync enables automatic synchronization with proto files
// This could be called during development to automatically update registrations.
func (drm *DynamicRegistrationManager) EnableAutoSync(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			drm.logger.Info("Auto-sync stopped")
			return
		case <-ticker.C:
			if err := drm.SyncWithProtoFiles(ctx); err != nil {
				drm.logger.Error("Auto-sync failed", zap.Error(err))
			}
		}
	}
}
