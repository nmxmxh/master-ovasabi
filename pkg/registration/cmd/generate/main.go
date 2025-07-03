package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nmxmxh/master-ovasabi/pkg/registration"
	"go.uber.org/zap"
)

func main() {
	var (
		protoPath   = flag.String("proto-path", "api/protos", "Path to proto files")
		srcPath     = flag.String("src-path", ".", "Path to source files")
		outputPath  = flag.String("output", "config/service_registration_generated.json", "Output file path")
		mode        = flag.String("mode", "generate", "Mode: 'generate' or 'inspect'")
		serviceName = flag.String("service", "", "Service name for inspection mode")
	)
	flag.Parse()

	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}

	// Get absolute paths
	absProtoPath, err := filepath.Abs(*protoPath)
	if err != nil {
		logger.Fatal("Failed to get absolute proto path", zap.Error(err))
	}

	absSrcPath, err := filepath.Abs(*srcPath)
	if err != nil {
		logger.Fatal("Failed to get absolute source path", zap.Error(err))
	}

	generator := registration.NewDynamicServiceRegistrationGenerator(
		logger,
		absProtoPath,
		absSrcPath,
	)

	ctx := context.Background()

	switch *mode {
	case "generate":
		if err := generator.GenerateAndSaveConfig(ctx, *outputPath); err != nil {
			logger.Fatal("Failed to generate service registration config", zap.Error(err))
		}
		logger.Info("Successfully generated service registration configuration")

	case "inspect":
		if *serviceName == "" {
			logger.Fatal("Service name is required for inspection mode")
		}

		// For inspection mode, we would need to load the actual service
		// This is more complex and would require reflection on loaded services
		logger.Info("Inspection mode", zap.String("service", *serviceName))
		fmt.Println("Inspection mode is not fully implemented yet.")
		fmt.Println("Use 'generate' mode to create service registration configs from proto files.")

	default:
		logger.Fatal("Invalid mode", zap.String("mode", *mode))
	}
}
