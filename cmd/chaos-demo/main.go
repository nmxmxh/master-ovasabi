package main

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"time"

	chaos "github.com/nmxmxh/master-ovasabi/pkg/chaos"
	thecat "github.com/nmxmxh/master-ovasabi/pkg/thecathasnoname"
)

func main() {
	logger := log.New(os.Stdout, "[chaos-demo] ", log.LstdFlags)
	cat := thecat.New(logger)

	// Load service registrations from JSON
	jsonData, err := ioutil.ReadFile("config/service_registration.json")
	if err != nil {
		logger.Fatalf("Failed to read service_registration.json: %v", err)
	}
	services, err := chaos.LoadServiceRegistrationsFromJSON(jsonData)
	if err != nil {
		logger.Fatalf("Failed to parse service registrations: %v", err)
	}

	orchestrator := chaos.NewChaosOrchestrator(logger, cat, services, 5, nil) // concurrency=5 for demo
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	orchestrator.RunChaosDemo(ctx)
}
