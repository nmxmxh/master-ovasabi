package health

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// MockHealthCheck implements HealthCheck interface for testing
type MockHealthCheck struct {
	name    string
	err     error
	checked bool
}

func (m *MockHealthCheck) Check(ctx context.Context) error {
	m.checked = true
	return m.err
}

func (m *MockHealthCheck) Name() string {
	return m.name
}

func TestNewHealthChecker(t *testing.T) {
	hc := NewHealthChecker()
	assert.NotNil(t, hc)
	assert.Empty(t, hc.checks)
}

func TestHealthChecker_Register(t *testing.T) {
	hc := NewHealthChecker()
	check := &MockHealthCheck{name: "test"}

	hc.Register(check)
	assert.Len(t, hc.checks, 1)
	assert.Equal(t, check, hc.checks[0])
}

func TestHealthChecker_Check(t *testing.T) {
	hc := NewHealthChecker()
	ctx := context.Background()

	successCheck := &MockHealthCheck{name: "success"}
	failCheck := &MockHealthCheck{
		name: "fail",
		err:  errors.New("check failed"),
	}

	hc.Register(successCheck)
	hc.Register(failCheck)

	results := hc.Check(ctx)

	assert.Len(t, results, 2)
	assert.NoError(t, results["success"])
	assert.Error(t, results["fail"])
	assert.True(t, successCheck.checked)
	assert.True(t, failCheck.checked)
}

func TestDatabaseHealthCheck(t *testing.T) {
	check := NewDatabaseHealthCheck("db")
	assert.Equal(t, "db", check.Name())

	err := check.Check(context.Background())
	assert.NoError(t, err) // Currently returns nil as per implementation
}

func TestRedisHealthCheck(t *testing.T) {
	check := NewRedisHealthCheck("redis")
	assert.Equal(t, "redis", check.Name())

	err := check.Check(context.Background())
	assert.NoError(t, err) // Currently returns nil as per implementation
}

func TestHTTPHealthCheck(t *testing.T) {
	timeout := 5 * time.Second
	check := NewHTTPHealthCheck("api", "http://example.com", timeout)

	assert.Equal(t, "api", check.Name())
	err := check.Check(context.Background())
	assert.NoError(t, err) // Currently returns nil as per implementation
}

func TestConcurrentHealthChecks(t *testing.T) {
	hc := NewHealthChecker()
	ctx := context.Background()

	// Register multiple checks
	for i := 0; i < 10; i++ {
		check := &MockHealthCheck{name: fmt.Sprintf("check-%d", i)}
		hc.Register(check)
	}

	// Run health checks concurrently
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results := hc.Check(ctx)
			assert.Len(t, results, 10)
		}()
	}

	wg.Wait()
}

func TestHealthCheckerWithTimeout(t *testing.T) {
	hc := NewHealthChecker()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Register a check that respects context cancellation
	check := &MockHealthCheck{
		name: "timeout-check",
		err:  context.DeadlineExceeded,
	}
	hc.Register(check)

	results := hc.Check(ctx)
	assert.Error(t, results["timeout-check"])
	assert.Equal(t, context.DeadlineExceeded, results["timeout-check"])
}
