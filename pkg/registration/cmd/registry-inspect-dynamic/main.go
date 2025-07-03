package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/nmxmxh/master-ovasabi/config/registry"
	"github.com/nmxmxh/master-ovasabi/pkg/registration"
	"go.uber.org/zap"
)

func main() {
	mode := flag.String("mode", "services", "Mode: 'services', 'events', 'health', 'generate', 'inspect', 'validate', 'compare', 'graph', 'watch', 'help'")
	service := flag.String("service", "", "Service name for specific operations")
	output := flag.String("output", "", "Output file path")
	format := flag.String("format", "json", "Output format: 'json', 'yaml', 'table'")
	protoPath := flag.String("proto-path", "api/protos", "Path to proto files")
	srcPath := flag.String("src-path", ".", "Path to source files")
	compare := flag.String("compare", "", "Second service name for comparison")
	watch := flag.Bool("watch", false, "Enable watch mode for continuous regeneration")
	debounce := flag.Int("debounce", 1000, "Debounce time in milliseconds for watch mode")
	monitor := flag.Bool("monitor", false, "Enable continuous health monitoring")
	interval := flag.Int("interval", 30, "Health check interval in seconds")
	flag.Parse()

	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}

	// Always load from disk for local inspection
	_ = registry.LoadRegistriesFromDisk("config/registry")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Create dynamic inspector
	absProtoPath, _ := filepath.Abs(*protoPath)
	absSrcPath, _ := filepath.Abs(*srcPath)

	inspector := registration.NewDynamicInspector(
		logger,
		nil, // DI container would be passed here in real usage
		absProtoPath,
		absSrcPath,
	)

	switch *mode {
	case "help":
		printHelp()

	case "services":
		printServices()

	case "events":
		printEvents()

	case "health":
		checkHealth(*monitor, *interval)

	case "generate":
		if *output == "" {
			*output = "config/service_registration_dynamic.json"
		}

		generator := registration.NewDynamicServiceRegistrationGenerator(
			logger,
			absProtoPath,
			absSrcPath,
		)

		if *watch {
			// Watch mode - continuously regenerate on changes
			watchConfig := registration.WatcherConfig{
				ProtoPath:  absProtoPath,
				OutputPath: *output,
				DebounceMs: *debounce,
				AutoReload: true,
			}

			watcher, err := registration.NewConfigWatcherWithConfig(logger, generator, watchConfig)
			if err != nil {
				logger.Fatal("Failed to create config watcher", zap.Error(err))
			}
			defer watcher.Stop()

			// Generate initial config
			if err := generator.GenerateAndSaveConfig(ctx, *output); err != nil {
				logger.Fatal("Failed to generate initial service registration", zap.Error(err))
			}
			fmt.Printf("Generated initial service registration configuration: %s\n", *output)

			// Start watching
			if err := watcher.Start(ctx); err != nil {
				logger.Fatal("Failed to start config watcher", zap.Error(err))
			}

			fmt.Println("Watching for proto file changes... Press Ctrl+C to stop")

			// Wait for interrupt signal
			<-ctx.Done()
			fmt.Println("\nShutting down...")
		} else {
			// One-time generation
			if err := generator.GenerateAndSaveConfig(ctx, *output); err != nil {
				logger.Fatal("Failed to generate service registration", zap.Error(err))
			}
			fmt.Printf("Generated service registration configuration: %s\n", *output)
		}

	case "watch":
		// Standalone watch mode (alias for generate --watch)
		if *output == "" {
			*output = "config/service_registration_dynamic.json"
		}

		generator := registration.NewDynamicServiceRegistrationGenerator(
			logger,
			absProtoPath,
			absSrcPath,
		)

		watchConfig := registration.WatcherConfig{
			ProtoPath:  absProtoPath,
			OutputPath: *output,
			DebounceMs: *debounce,
			AutoReload: true,
		}

		watcher, err := registration.NewConfigWatcherWithConfig(logger, generator, watchConfig)
		if err != nil {
			logger.Fatal("Failed to create config watcher", zap.Error(err))
		}
		defer watcher.Stop()

		// Generate initial config
		if err := generator.GenerateAndSaveConfig(ctx, *output); err != nil {
			logger.Fatal("Failed to generate initial service registration", zap.Error(err))
		}
		fmt.Printf("Generated initial service registration configuration: %s\n", *output)

		// Start watching
		if err := watcher.Start(ctx); err != nil {
			logger.Fatal("Failed to start config watcher", zap.Error(err))
		}

		fmt.Println("Watching for proto file changes... Press Ctrl+C to stop")

		// Wait for interrupt signal
		<-ctx.Done()
		fmt.Println("\nShutting down...")

	case "inspect":
		if *service == "" {
			fmt.Fprintf(os.Stderr, "Service name is required for inspection\n")
			os.Exit(1)
		}

		result, err := inspector.InspectService(*service)
		if err != nil {
			logger.Fatal("Failed to inspect service", zap.Error(err))
		}

		printInspectionResult(result, *format)

	case "validate":
		if *service == "" {
			fmt.Fprintf(os.Stderr, "Service name is required for validation\n")
			os.Exit(1)
		}

		// Load service configuration and validate
		configs, err := loadServiceConfigs()
		if err != nil {
			logger.Fatal("Failed to load service configurations", zap.Error(err))
		}

		for _, config := range configs {
			if config.Name == *service {
				result, err := inspector.ValidateServiceRegistration(config)
				if err != nil {
					logger.Fatal("Failed to validate service", zap.Error(err))
				}
				printValidationResult(result)
				return
			}
		}

		fmt.Printf("Service '%s' not found in configurations\n", *service)

	case "compare":
		if *service == "" || *compare == "" {
			fmt.Fprintf(os.Stderr, "Both service names are required for comparison\n")
			os.Exit(1)
		}

		configs, err := loadServiceConfigs()
		if err != nil {
			logger.Fatal("Failed to load service configurations", zap.Error(err))
		}

		var config1, config2 *registration.ServiceRegistrationConfig
		for _, config := range configs {
			if config.Name == *service {
				config1 = &config
			}
			if config.Name == *compare {
				config2 = &config
			}
		}

		if config1 == nil || config2 == nil {
			fmt.Fprintf(os.Stderr, "One or both services not found\n")
			os.Exit(1)
		}

		result := inspector.CompareConfigurations(*config1, *config2)
		printComparisonResult(result)

	case "graph":
		if *output == "" {
			*output = "amadeus/service_graph.json"
		}

		configs, err := loadServiceConfigs()
		if err != nil {
			logger.Fatal("Failed to load service configurations", zap.Error(err))
		}

		if err := inspector.ExportServiceGraph(configs, *output); err != nil {
			logger.Fatal("Failed to export service graph", zap.Error(err))
		}

		fmt.Printf("Exported service graph: %s\n", *output)

	default:
		fmt.Fprintf(os.Stderr, "Unknown mode: %s\n", *mode)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("Dynamic Service Registration Inspector")
	fmt.Println("=====================================")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("  registry-inspect-dynamic [flags]")
	fmt.Println()
	fmt.Println("MODES:")
	fmt.Println("  services    - List all registered services")
	fmt.Println("  events      - List all registered events")
	fmt.Println("  health      - Check health of all services")
	fmt.Println("  generate    - Generate service registration config from proto files")
	fmt.Println("  inspect     - Inspect a specific service (requires -service)")
	fmt.Println("  validate    - Validate service registration (requires -service)")
	fmt.Println("  compare     - Compare two services (requires -service and -compare)")
	fmt.Println("  graph       - Export service dependency graph")
	fmt.Println("  watch       - Watch for proto changes and auto-regenerate config")
	fmt.Println("  help        - Show this help message")
	fmt.Println()
	fmt.Println("FLAGS:")
	fmt.Println("  -mode string")
	fmt.Println("        Operation mode (default \"services\")")
	fmt.Println("  -service string")
	fmt.Println("        Service name for specific operations")
	fmt.Println("  -compare string")
	fmt.Println("        Second service name for comparison")
	fmt.Println("  -output string")
	fmt.Println("        Output file path")
	fmt.Println("  -format string")
	fmt.Println("        Output format: json, yaml, table (default \"json\")")
	fmt.Println("  -proto-path string")
	fmt.Println("        Path to proto files (default \"api/protos\")")
	fmt.Println("  -src-path string")
	fmt.Println("        Path to source files (default \".\")")
	fmt.Println("  -watch")
	fmt.Println("        Enable watch mode for continuous regeneration")
	fmt.Println("  -debounce int")
	fmt.Println("        Debounce time in milliseconds for watch mode (default 1000)")
	fmt.Println("  -monitor")
	fmt.Println("        Enable continuous health monitoring")
	fmt.Println("  -interval int")
	fmt.Println("        Health check interval in seconds (default 30)")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  # List all services")
	fmt.Println("  registry-inspect-dynamic -mode services")
	fmt.Println()
	fmt.Println("  # Check health of all services")
	fmt.Println("  registry-inspect-dynamic -mode health")
	fmt.Println()
	fmt.Println("  # Monitor health continuously")
	fmt.Println("  registry-inspect-dynamic -mode health -monitor -interval 60")
	fmt.Println()
	fmt.Println("  # Generate config from proto files")
	fmt.Println("  registry-inspect-dynamic -mode generate -output config/my_services.json")
	fmt.Println()
	fmt.Println("  # Watch for changes and auto-regenerate")
	fmt.Println("  registry-inspect-dynamic -mode watch")
	fmt.Println()
	fmt.Println("  # Inspect a specific service")
	fmt.Println("  registry-inspect-dynamic -mode inspect -service user -format table")
	fmt.Println()
	fmt.Println("  # Compare two services")
	fmt.Println("  registry-inspect-dynamic -mode compare -service user -compare admin")
	fmt.Println()
	fmt.Println("  # Export service dependency graph")
	fmt.Println("  registry-inspect-dynamic -mode graph -output amadeus/service_graph.json")
}

func checkHealth(enableMonitor bool, intervalSeconds int) {
	configs, err := loadServiceConfigs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load service configurations: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	logger, _ := zap.NewDevelopment()
	checker := registration.NewHealthChecker(logger)

	if enableMonitor {
		// Continuous monitoring mode
		intervalDuration := time.Duration(intervalSeconds) * time.Second
		healthMonitor := registration.NewHealthMonitor(logger, configs, intervalDuration)

		fmt.Printf("Starting health monitoring (interval: %s)...\n", intervalDuration)
		resultChan := healthMonitor.Start(ctx)

		for result := range resultChan {
			printHealthResult(result)
		}
	} else {
		// One-time health check
		result := checker.CheckAllServices(ctx, configs)
		printHealthResult(result)
	}
}

func printHealthResult(result *registration.HealthCheckResult) {
	fmt.Printf("Health Check Results (%s)\n", result.CheckedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("========================\n")
	fmt.Printf("Total Services: %d\n", result.TotalServices)
	fmt.Printf("Healthy: %d\n", result.HealthyServices)
	fmt.Printf("Unhealthy: %d\n", result.UnhealthyServices)
	fmt.Printf("Success Rate: %.1f%%\n", float64(result.HealthyServices)/float64(result.TotalServices)*100)
	fmt.Println()

	for name, status := range result.Services {
		statusIcon := "✓"
		if !status.IsHealthy {
			statusIcon = "✗"
		}

		fmt.Printf("%s %s", statusIcon, name)
		if status.ResponseTime > 0 {
			fmt.Printf(" (%s)", status.ResponseTime.Round(time.Millisecond))
		}
		if status.Error != "" {
			fmt.Printf(" - %s", status.Error)
		}
		fmt.Println()
	}
	fmt.Println()
}

func printServices() {
	for name, svc := range registry.GetServiceRegistry() {
		fmt.Printf("Service: %s (v%s)\n", name, svc.Version)
		fmt.Printf("  Description: %s\n", svc.Description)
		fmt.Printf("  Registered: %s\n", svc.RegisteredAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Methods:\n")
		for _, method := range svc.Methods {
			fmt.Printf("    - %s", method.Name)
			if len(method.Parameters) > 0 {
				fmt.Printf(" (params: %v)", method.Parameters)
			}
			if method.Description != "" {
				fmt.Printf(" - %s", method.Description)
			}
			fmt.Println()
		}
		fmt.Println()
	}
}

func printEvents() {
	for name, evt := range registry.GetEventRegistry() {
		fmt.Printf("Event: %s (v%s)\n", name, evt.Version)
		fmt.Printf("  Description: %s\n", evt.Description)
		fmt.Printf("  Registered: %s\n", evt.RegisteredAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Parameters: %v\n", evt.Parameters)
		fmt.Printf("  Required Fields: %v\n", evt.RequiredFields)
		fmt.Println()
	}
}

func printInspectionResult(result *registration.ServiceInspectionResult, format string) {
	switch format {
	case "table":
		fmt.Printf("Service: %s\n", result.ServiceName)
		fmt.Printf("Registered: %t\n", result.IsRegistered)
		fmt.Printf("Methods: %d\n", len(result.Methods))

		if result.RuntimeInfo != nil {
			fmt.Printf("Runtime Info:\n")
			fmt.Printf("  Go Version: %s\n", result.RuntimeInfo.GoVersion)
			fmt.Printf("  Goroutines: %d\n", result.RuntimeInfo.NumGoroutines)
		}

	default: // json
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
	}
}

func printValidationResult(result *registration.ValidationResult) {
	fmt.Printf("Service: %s\n", result.ServiceName)
	fmt.Printf("Valid: %t\n", result.IsValid)

	if len(result.Issues) > 0 {
		fmt.Println("Issues:")
		for _, issue := range result.Issues {
			fmt.Printf("  - %s\n", issue)
		}
	}

	if len(result.Suggestions) > 0 {
		fmt.Println("Suggestions:")
		for _, suggestion := range result.Suggestions {
			fmt.Printf("  - %s\n", suggestion)
		}
	}
}

func printComparisonResult(result *registration.ComparisonResult) {
	fmt.Printf("Comparison: %s vs %s\n", result.Service1, result.Service2)

	if len(result.Similarities) > 0 {
		fmt.Println("Similarities:")
		for _, similarity := range result.Similarities {
			fmt.Printf("  + %s\n", similarity)
		}
	}

	if len(result.Differences) > 0 {
		fmt.Println("Differences:")
		for _, difference := range result.Differences {
			fmt.Printf("  - %s\n", difference)
		}
	}
}

func loadServiceConfigs() ([]registration.ServiceRegistrationConfig, error) {
	// Try to load from generated config first, then fallback to manual config
	configs, err := loadConfigFile("config/service_registration_dynamic.json")
	if err != nil {
		configs, err = loadConfigFile("config/service_registration.json")
		if err != nil {
			return nil, err
		}
	}
	return configs, nil
}

func loadConfigFile(path string) ([]registration.ServiceRegistrationConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var configs []registration.ServiceRegistrationConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return nil, err
	}

	return configs, nil
}
