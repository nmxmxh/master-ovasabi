package main

import (
	"context"
	"log"

	"github.com/nmxmxh/master-ovasabi/pkg/registration"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewDevelopment()
	gen := registration.NewDynamicServiceRegistrationGenerator(
		logger,
		"api/protos", // proto path
		".",          // src path (repo root)
	)
	if err := gen.GenerateAndSaveConfig(context.Background(), "config/service_registration.json"); err != nil {
		log.Fatalf("Failed to generate service registration: %v", err)
	}
}
