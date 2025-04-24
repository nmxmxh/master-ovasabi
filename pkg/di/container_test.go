package di

import (
	"errors"
	"fmt"
	"sync"
	"testing"
)

// Example interfaces and implementations for testing
type Logger interface {
	Log(msg string)
}

type Service interface {
	DoSomething() string
}

type RealLogger struct {
	messages []string
}

func (l *RealLogger) Log(msg string) {
	l.messages = append(l.messages, msg)
}

type MockLogger struct {
	LogCalled bool
	LastMsg   string
}

func (m *MockLogger) Log(msg string) {
	m.LogCalled = true
	m.LastMsg = msg
}

type RealService struct {
	logger Logger
}

func (s *RealService) DoSomething() string {
	s.logger.Log("DoSomething called")
	return "real service result"
}

type MockService struct {
	ReturnValue string
}

func (m *MockService) DoSomething() string {
	return m.ReturnValue
}

func TestContainer_Basic(t *testing.T) {
	c := New()

	// Register real logger
	err := c.Register((*Logger)(nil), func(c *Container) (interface{}, error) {
		return &RealLogger{messages: make([]string, 0)}, nil
	})
	if err != nil {
		t.Fatalf("Failed to register logger: %v", err)
	}

	// Register service that depends on logger
	err = c.Register((*Service)(nil), func(c *Container) (interface{}, error) {
		var logger Logger
		if err := c.Resolve(&logger); err != nil {
			return nil, err
		}
		return &RealService{logger: logger}, nil
	})
	if err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	// Resolve and use service
	var service Service
	if err := c.Resolve(&service); err != nil {
		t.Fatalf("Failed to resolve service: %v", err)
	}

	result := service.DoSomething()
	if result != "real service result" {
		t.Errorf("Expected 'real service result', got %q", result)
	}
}

func TestContainer_WithMocks(t *testing.T) {
	c := New()

	// Register mock service
	mockService := &MockService{ReturnValue: "mock result"}
	err := c.RegisterMock((*Service)(nil), mockService)
	if err != nil {
		t.Fatalf("Failed to register mock service: %v", err)
	}

	// Resolve and use mock service
	var service Service
	if err := c.Resolve(&service); err != nil {
		t.Fatalf("Failed to resolve service: %v", err)
	}

	result := service.DoSomething()
	if result != "mock result" {
		t.Errorf("Expected 'mock result', got %q", result)
	}
}

func TestContainer_WithConfig(t *testing.T) {
	c := New()

	// Register configuration
	c.RegisterConfig("app.name", "TestApp")
	c.RegisterConfig("app.version", "1.0.0")

	// Retrieve configuration
	name, ok := c.GetConfig("app.name")
	if !ok {
		t.Fatal("Expected app.name config to exist")
	}
	if name != "TestApp" {
		t.Errorf("Expected app.name to be 'TestApp', got %q", name)
	}

	version, ok := c.GetConfig("app.version")
	if !ok {
		t.Fatal("Expected app.version config to exist")
	}
	if version != "1.0.0" {
		t.Errorf("Expected app.version to be '1.0.0', got %q", version)
	}
}

func TestContainer_Reset(t *testing.T) {
	c := New()

	// Register and resolve service
	mockService := &MockService{ReturnValue: "mock result"}
	err := c.RegisterMock((*Service)(nil), mockService)
	if err != nil {
		t.Fatalf("Failed to register mock service: %v", err)
	}

	var service Service
	if err := c.Resolve(&service); err != nil {
		t.Fatalf("Failed to resolve service: %v", err)
	}

	// Reset container
	c.Reset()

	// Try to resolve service again (should fail)
	err = c.Resolve(&service)
	if err == nil {
		t.Error("Expected error after reset, got nil")
	}
}

func TestContainer_Clear(t *testing.T) {
	c := New()

	// Register two services
	mockService := &MockService{ReturnValue: "mock result"}
	mockLogger := &MockLogger{}

	err := c.RegisterMock((*Service)(nil), mockService)
	if err != nil {
		t.Fatalf("Failed to register mock service: %v", err)
	}

	err = c.RegisterMock((*Logger)(nil), mockLogger)
	if err != nil {
		t.Fatalf("Failed to register mock logger: %v", err)
	}

	// Clear only the service
	c.Clear((*Service)(nil))

	// Service should fail to resolve
	var service Service
	if err := c.Resolve(&service); err == nil {
		t.Error("Expected error resolving cleared service")
	}

	// Logger should still resolve
	var logger Logger
	if err := c.Resolve(&logger); err != nil {
		t.Errorf("Expected logger to still resolve, got error: %v", err)
	}
}

// Test registering a non-pointer interface should return an error
func TestContainer_RegisterErrorNonPointer(t *testing.T) {
	c := New()
	err := c.Register(123, nil)
	if err == nil {
		t.Error("Expected error when registering non-pointer interface, got nil")
	}
}

// Test registering a mock with a non-pointer interface should return an error
func TestContainer_RegisterMockErrorNonPointer(t *testing.T) {
	c := New()
	err := c.RegisterMock(123, &MockService{})
	if err == nil {
		t.Error("Expected error when registering mock with non-pointer interface, got nil")
	}
}

// Test registering a mock that does not implement the interface should return an error
func TestContainer_RegisterMockErrorNotImplement(t *testing.T) {
	c := New()
	err := c.RegisterMock((*Service)(nil), &RealLogger{})
	if err == nil {
		t.Error("Expected error when registering mock that does not implement interface, got nil")
	}
}

// Test retrieving a missing config key returns ok=false
func TestContainer_GetConfigMissing(t *testing.T) {
	c := New()
	_, ok := c.GetConfig("no_such")
	if ok {
		t.Error("Expected no value for missing config key, got one")
	}
}

// Test resolving a non-pointer target should return an error
func TestContainer_ResolveErrorTargetNonPointer(t *testing.T) {
	c := New()
	err := c.Resolve(123)
	if err == nil || !errors.Is(err, ErrTargetMustBePointer) {
		t.Errorf("Expected ErrTargetMustBePointer, got %v", err)
	}
}

// Test MustResolve should panic when resolution fails
func TestContainer_MustResolvePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic from MustResolve, but none occurred")
		}
	}()
	c := New()
	var s Service
	c.MustResolve(&s)
}

// Test that Resolve caches the service so the factory is called only once
func TestContainer_ServiceCaching(t *testing.T) {
	c := New()
	calls := 0
	err := c.Register((*Service)(nil), func(_ *Container) (interface{}, error) {
		calls++
		return &MockService{ReturnValue: "value"}, nil
	})
	if err != nil {
		t.Fatalf("Failed to register: %v", err)
	}
	var s1 Service
	if err := c.Resolve(&s1); err != nil {
		t.Fatalf("Unexpected error on first resolve: %v", err)
	}
	var s2 Service
	if err := c.Resolve(&s2); err != nil {
		t.Fatalf("Unexpected error on second resolve: %v", err)
	}
	m1 := s1.(*MockService)
	m2 := s2.(*MockService)
	if m1 != m2 {
		t.Error("Expected same instance on second resolve")
	}
	if calls != 1 {
		t.Errorf("Expected factory to be called once, got %d", calls)
	}
}

// Test that Resolve wraps factory errors correctly
func TestContainer_ResolveFactoryError(t *testing.T) {
	c := New()
	err := c.Register((*Service)(nil), func(_ *Container) (interface{}, error) {
		return nil, fmt.Errorf("oops")
	})
	if err != nil {
		t.Fatalf("Failed to register: %v", err)
	}
	var s Service
	err = c.Resolve(&s)
	if err == nil || !errors.Is(err, ErrFactoryFailed) {
		t.Errorf("Expected ErrFactoryFailed, got %v", err)
	}
}

// TestContainer_GetString tests the typed string configuration getter
func TestContainer_GetString(t *testing.T) {
	c := New()
	// Valid string
	c.RegisterConfig("key", "value")
	val, ok := c.GetString("key")
	if !ok || val != "value" {
		t.Errorf("Expected GetString to return 'value', got '%s', ok=%v", val, ok)
	}
	// Missing key
	if _, ok2 := c.GetString("missing"); ok2 {
		t.Error("Expected GetString to return ok=false for missing key")
	}
	// Wrong type
	c.RegisterConfig("num", 123)
	if _, ok3 := c.GetString("num"); ok3 {
		t.Error("Expected GetString to fail type assertion for non-string")
	}
}

// TestContainer_GetInt tests the typed int configuration getter
func TestContainer_GetInt(t *testing.T) {
	c := New()
	// Valid int
	c.RegisterConfig("num", 42)
	i, ok := c.GetInt("num")
	if !ok || i != 42 {
		t.Errorf("Expected GetInt to return 42, got %d, ok=%v", i, ok)
	}
	// Missing key
	if _, ok2 := c.GetInt("missing"); ok2 {
		t.Error("Expected GetInt to return ok=false for missing key")
	}
	// Wrong type
	c.RegisterConfig("str", "value")
	if _, ok3 := c.GetInt("str"); ok3 {
		t.Error("Expected GetInt to fail type assertion for non-int")
	}
}

// TestContainer_ResolveConcurrent stress tests concurrent resolves for thread-safety
func TestContainer_ResolveConcurrent(t *testing.T) {
	c := New()
	err := c.Register((*Service)(nil), func(_ *Container) (interface{}, error) {
		return &MockService{ReturnValue: "val"}, nil
	})
	if err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}
	// Warm-up resolve
	var s Service
	if err := c.Resolve(&s); err != nil {
		t.Fatalf("Initial resolve failed: %v", err)
	}

	var wg sync.WaitGroup
	const goroutines = 50
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			var s2 Service
			if err := c.Resolve(&s2); err != nil {
				t.Errorf("Resolve in goroutine failed: %v", err)
			}
			if s2 != s {
				t.Errorf("Expected same instance, got %v and %v", s2, s)
			}
		}()
	}
	wg.Wait()
}

// BenchmarkContainer_ResolveParallel benchmarks concurrent Resolve calls for thread-safety.
func BenchmarkContainer_ResolveParallel(b *testing.B) {
	c := New()
	err := c.Register((*Service)(nil), func(_ *Container) (interface{}, error) {
		return &MockService{ReturnValue: "val"}, nil
	})
	if err != nil {
		b.Fatalf("Failed to register service: %v", err)
	}
	// Warm-up resolve
	var s Service
	if err := c.Resolve(&s); err != nil {
		b.Fatalf("Initial resolve failed: %v", err)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var s2 Service
			if err := c.Resolve(&s2); err != nil {
				b.Fatalf("Resolve in parallel failed: %v", err)
			}
		}
	})
}
