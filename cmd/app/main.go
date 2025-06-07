package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nmxmxh/master-ovasabi/internal/config"
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

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Error("Failed to load configuration", zap.Error(err))
		return
	}

	// Create DI container
	container := di.New()

	// Create HTTP server
	httpServer := &http.Server{
		Addr:              ":" + cfg.AppPort,
		ReadHeaderTimeout: 10 * time.Second, // Mitigate Slowloris attacks
	}

	// Create server instance
	srv := server.NewServer(container, log.GetZapLogger(), httpServer)

	// Start server in a goroutine
	go func() {
		if err := srv.Start(); err != nil {
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
