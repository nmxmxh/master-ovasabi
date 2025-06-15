package bridge

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
)

// Canonical Event Bus Pattern: All orchestration uses eventBusImpl for event emission and logging.

// Container provides the bridge container orchestration and health endpoints.
type Container struct {
	bridgeService *Service
	adapters      []Adapter
	eventBus      EventBus
	healthStatus  HealthStatus
	shutdown      chan struct{}
	wg            sync.WaitGroup
}

// NewBridgeContainer creates and initializes the bridge container.
func NewBridgeContainer(rules []RoutingRule, adapters []Adapter, logger *zap.Logger, redisCache *redis.Cache) *Container {
	// Instantiate canonical event bus with logger and redis
	eventBus := NewEventBusWithRedis(logger, redisCache)
	for _, adapter := range adapters {
		RegisterAdapter(adapter)
	}
	bridgeService := NewBridgeService(rules, eventBus, logger)
	return &Container{
		bridgeService: bridgeService,
		adapters:      adapters,
		eventBus:      eventBus,
		healthStatus:  HealthStatus{Status: "UP", Timestamp: time.Now()},
		shutdown:      make(chan struct{}),
	}
}

// Start runs the bridge container, serving health and metrics endpoints and handling graceful shutdown.
func (c *Container) Start() {
	c.wg.Add(1)
	go c.serveHealth()
	log.Println("BridgeContainer started. Serving health at /healthz")

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Println("BridgeContainer shutting down...")
	close(c.shutdown)
	c.wg.Wait()
	for _, adapter := range c.adapters {
		if err := adapter.Close(); err != nil {
			zap.L().Warn("Failed to close adapter in container", zap.Error(err))
		}
	}
	log.Println("BridgeContainer stopped.")
}

// serveHealth exposes a simple health endpoint for monitoring.
func (c *Container) serveHealth() {
	defer c.wg.Done()
	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"status":"` + c.healthStatus.Status + `"}`)); err != nil {
			log.Printf("Failed to write health response: %v", err)
		}
	})
	srv := &http.Server{
		Addr:         ":8091",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Printf("Failed to start health endpoint: %v", err)
	}
}
