package lifecycle

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// BackgroundWorker provides generic management for goroutines and background tasks
type BackgroundWorker struct {
	name     string
	workFunc func(ctx context.Context) error
	interval time.Duration
	log      *zap.Logger
	stopCh   chan struct{}
	wg       sync.WaitGroup
	started  bool
	mu       sync.Mutex
}

// NewBackgroundWorker creates a new background worker
func NewBackgroundWorker(name string, workFunc func(ctx context.Context) error, interval time.Duration, log *zap.Logger) *BackgroundWorker {
	return &BackgroundWorker{
		name:     name,
		workFunc: workFunc,
		interval: interval,
		log:      log,
		stopCh:   make(chan struct{}),
	}
}

// Name returns the worker name
func (w *BackgroundWorker) Name() string {
	return w.name
}

// Start begins the background worker
func (w *BackgroundWorker) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.started {
		return nil
	}

	w.wg.Add(1)
	go w.run(ctx)
	w.started = true

	w.log.Info("Background worker started", zap.String("worker", w.name))
	return nil
}

// Stop gracefully stops the background worker
func (w *BackgroundWorker) Stop(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.started {
		return nil
	}

	close(w.stopCh)

	// Wait for worker to stop with timeout
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		w.log.Info("Background worker stopped", zap.String("worker", w.name))
		return nil
	case <-ctx.Done():
		w.log.Warn("Background worker stop timeout", zap.String("worker", w.name))
		return ctx.Err()
	}
}

// Health checks if the worker is running
func (w *BackgroundWorker) Health() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.started {
		return &HealthError{Resource: w.name, Message: "worker not started"}
	}
	return nil
}

// run is the main worker loop
func (w *BackgroundWorker) run(ctx context.Context) {
	defer w.wg.Done()

	if w.interval > 0 {
		w.runPeriodic(ctx)
	} else {
		w.runOnce(ctx)
	}
}

// runPeriodic runs the work function at regular intervals
func (w *BackgroundWorker) runPeriodic(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.log.Debug("Background worker context cancelled", zap.String("worker", w.name))
			return
		case <-w.stopCh:
			w.log.Debug("Background worker stop signal received", zap.String("worker", w.name))
			return
		case <-ticker.C:
			if err := w.workFunc(ctx); err != nil {
				w.log.Error("Background worker execution failed",
					zap.String("worker", w.name),
					zap.Error(err))
			}
		}
	}
}

// runOnce runs the work function once
func (w *BackgroundWorker) runOnce(ctx context.Context) {
	if err := w.workFunc(ctx); err != nil {
		w.log.Error("Background worker execution failed",
			zap.String("worker", w.name),
			zap.Error(err))
	}
}

// PoolManager provides generic management for resource pools
type PoolManager struct {
	name     string
	pools    map[string]interface{}
	cleaners map[string]func()
	mu       sync.RWMutex
	log      *zap.Logger
}

// NewPoolManager creates a new pool manager
func NewPoolManager(name string, log *zap.Logger) *PoolManager {
	return &PoolManager{
		name:     name,
		pools:    make(map[string]interface{}),
		cleaners: make(map[string]func()),
		log:      log,
	}
}

// Name returns the pool manager name
func (p *PoolManager) Name() string {
	return p.name
}

// Start initializes the pool manager
func (p *PoolManager) Start(ctx context.Context) error {
	p.log.Info("Pool manager started", zap.String("manager", p.name))
	return nil
}

// Stop cleans up all pools
func (p *PoolManager) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for name, cleaner := range p.cleaners {
		p.log.Debug("Cleaning pool", zap.String("pool", name))
		cleaner()
	}

	p.log.Info("Pool manager stopped", zap.String("manager", p.name))
	return nil
}

// Health checks pool status
func (p *PoolManager) Health() error {
	return nil
}

// RegisterPool adds a pool with optional cleanup function
func (p *PoolManager) RegisterPool(name string, pool interface{}, cleaner func()) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.pools[name] = pool
	if cleaner != nil {
		p.cleaners[name] = cleaner
	}
}

// GetPool retrieves a pool by name
func (p *PoolManager) GetPool(name string) (interface{}, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	pool, exists := p.pools[name]
	return pool, exists
}

// ConnectionManager provides generic connection lifecycle management
type ConnectionManager struct {
	name        string
	connections map[string]Connection
	mu          sync.RWMutex
	log         *zap.Logger
}

// Connection represents any connection that can be closed
type Connection interface {
	Close() error
	Ping() error
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(name string, log *zap.Logger) *ConnectionManager {
	return &ConnectionManager{
		name:        name,
		connections: make(map[string]Connection),
		log:         log,
	}
}

// Name returns the connection manager name
func (c *ConnectionManager) Name() string {
	return c.name
}

// Start initializes the connection manager
func (c *ConnectionManager) Start(ctx context.Context) error {
	c.log.Info("Connection manager started", zap.String("manager", c.name))
	return nil
}

// Stop closes all connections
func (c *ConnectionManager) Stop(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for name, conn := range c.connections {
		c.log.Debug("Closing connection", zap.String("connection", name))
		if err := conn.Close(); err != nil {
			c.log.Error("Failed to close connection",
				zap.String("connection", name),
				zap.Error(err))
		}
	}

	c.log.Info("Connection manager stopped", zap.String("manager", c.name))
	return nil
}

// Health checks all connections
func (c *ConnectionManager) Health() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for name, conn := range c.connections {
		if err := conn.Ping(); err != nil {
			return &HealthError{
				Resource: c.name,
				Message:  fmt.Sprintf("connection %s unhealthy: %v", name, err),
			}
		}
	}
	return nil
}

// RegisterConnection adds a connection to be managed
func (c *ConnectionManager) RegisterConnection(name string, conn Connection) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connections[name] = conn
}

// GetConnection retrieves a connection by name
func (c *ConnectionManager) GetConnection(name string) (Connection, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	conn, exists := c.connections[name]
	return conn, exists
}
