package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nmxmxh/master-ovasabi/internal/server"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	log, err := logger.NewDefault()
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := log.Sync(); err != nil {
			fmt.Printf("Failed to sync logger: %v\n", err)
		}
	}()

	// Create DI container
	container := di.New()

	// NOTE: WebSocket endpoints are now handled by the ws-gateway service at /ws and /ws/{campaign_id}/{user_id}.
	// This app only serves REST and gRPC endpoints. For WebSocket/event relay, use ws-gateway.

	// Use the correct arguments for NewServer and Start (httpPort, grpcPort)
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = ":8090" // fallback, but should be set in env
	}
	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "8080" // fallback, but should be set in env
	}

	srv := server.NewServer(container, log.GetZapLogger(), httpPort)

	// Start server in a goroutine
	go func() {
		if err := srv.Start(grpcPort); err != nil {
			log.Error("Server error", zap.Error(err))
			return
		}
	}()

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Stop(ctx); err != nil {
		log.Error("Error during shutdown", zap.Error(err))
		return
	}
}
