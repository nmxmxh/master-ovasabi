package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/nmxmxh/master-ovasabi/internal/server"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Service struct {
		Name        string `yaml:"name"`
		Version     string `yaml:"version"`
		Environment string `yaml:"environment"`
		LogLevel    string `yaml:"log_level"`
	} `yaml:"service"`

	Redis struct {
		Host     string `yaml:"host"`
		Port     string `yaml:"port"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
		PoolSize int    `yaml:"pool_size"`
	} `yaml:"redis"`

	Health struct {
		Port          int           `yaml:"port"`
		CheckInterval time.Duration `yaml:"check_interval"`
		Timeout       time.Duration `yaml:"timeout"`
		Retries       int           `yaml:"retries"`
	} `yaml:"health"`

	Metrics struct {
		Enabled bool `yaml:"enabled"`
		Port    int  `yaml:"port"`
	} `yaml:"metrics"`
}

func main() {
	// Load configuration
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "/config/kg-config.yaml"
	}

	var config Config
	configFile, err := os.ReadFile(configPath)
	if err != nil {
		panic(fmt.Sprintf("Failed to read config file: %v", err))
	}

	if err := yaml.Unmarshal(configFile, &config); err != nil {
		panic(fmt.Sprintf("Failed to parse config: %v", err))
	}

	// Initialize logger
	logConfig := zap.NewProductionConfig()
	if config.Service.Environment == "development" {
		logConfig = zap.NewDevelopmentConfig()
	}
	logger, err := logConfig.Build()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}
	defer func() {
		if err := logger.Sync(); err != nil {
			fmt.Printf("Failed to sync logger: %v\n", err)
		}
	}()

	// Create context for service lifecycle
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", config.Redis.Host, config.Redis.Port),
		Password: config.Redis.Password,
		DB:       config.Redis.DB,
		PoolSize: config.Redis.PoolSize,
	})

	// Test Redis connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			logger.Error("Failed to close Redis client", zap.Error(err))
		}
	}()

	// Create knowledge graph service
	kgService := server.NewKGService(redisClient, logger)

	// Start the service
	if err := kgService.Start(); err != nil {
		logger.Fatal("Failed to start knowledge graph service", zap.Error(err))
	}

	// Set up health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := redisClient.Ping(r.Context()).Err(); err != nil {
			logger.Error("Health check failed", zap.Error(err))
			http.Error(w, "Service unhealthy", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			logger.Error("Failed to write health check response", zap.Error(err))
		}
	})

	// Start health check server
	healthServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Health.Port),
		Handler:      http.DefaultServeMux,
		ReadTimeout:  config.Health.Timeout,
		WriteTimeout: config.Health.Timeout,
	}

	go func() {
		logger.Info("Starting health check server", zap.Int("port", config.Health.Port))
		if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Health check server failed", zap.Error(err))
		}
	}()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	logger.Info("Shutting down knowledge graph service...")

	// Create shutdown context
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Shutdown health check server
	if err := healthServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("Failed to shutdown health check server", zap.Error(err))
	}

	// Stop the knowledge graph service
	cancel() // Cancel the main context to stop the service
	kgService.Stop()

	logger.Info("Knowledge graph service stopped")
}
