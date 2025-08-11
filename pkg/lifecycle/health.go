package lifecycle

import (
	"fmt"
	"time"
)

// HealthError represents a health check failure
type HealthError struct {
	Resource string
	Message  string
}

// Error implements the error interface
func (e *HealthError) Error() string {
	return fmt.Sprintf("health check failed for %s: %s", e.Resource, e.Message)
}

// GracefulShutdown provides a standardized shutdown sequence
type GracefulShutdown struct {
	phases []ShutdownPhase
}

// ShutdownPhase represents a phase in the shutdown process
type ShutdownPhase struct {
	Name     string
	Timeout  time.Duration
	Executor func() error
}

// NewGracefulShutdown creates a new graceful shutdown manager
func NewGracefulShutdown() *GracefulShutdown {
	return &GracefulShutdown{
		phases: make([]ShutdownPhase, 0),
	}
}

// AddPhase adds a shutdown phase
func (g *GracefulShutdown) AddPhase(name string, timeout time.Duration, executor func() error) {
	g.phases = append(g.phases, ShutdownPhase{
		Name:     name,
		Timeout:  timeout,
		Executor: executor,
	})
}

// Execute runs all shutdown phases in order
func (g *GracefulShutdown) Execute() error {
	for _, phase := range g.phases {
		done := make(chan error, 1)

		go func(p ShutdownPhase) {
			done <- p.Executor()
		}(phase)

		select {
		case err := <-done:
			if err != nil {
				return fmt.Errorf("shutdown phase %s failed: %w", phase.Name, err)
			}
		case <-time.After(phase.Timeout):
			return fmt.Errorf("shutdown phase %s timed out after %v", phase.Name, phase.Timeout)
		}
	}
	return nil
}
