package di

import (
	"fmt"
	"reflect"
	"sync"

	errs "github.com/nmxmxh/master-ovasabi/pkg/errors"
)

// Factory is a function that creates an instance of a service.
type Factory func(*Container) (interface{}, error)

// Container manages dependency injection.
type Container struct {
	mu        sync.RWMutex
	services  map[reflect.Type]interface{}
	mocks     map[reflect.Type]interface{}
	configs   map[string]interface{}
	factories map[reflect.Type]Factory
}

// New creates a new DI container.
func New() *Container {
	return &Container{
		services:  make(map[reflect.Type]interface{}),
		mocks:     make(map[reflect.Type]interface{}),
		configs:   make(map[string]interface{}),
		factories: make(map[reflect.Type]Factory),
	}
}

// Register registers a service factory.
func (c *Container) Register(iface interface{}, factory Factory) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	t := reflect.TypeOf(iface)
	if t.Kind() != reflect.Ptr {
		return errs.ErrInterfaceMustBePointer
	}
	elem := t.Elem()
	var key reflect.Type
	if elem.Kind() == reflect.Interface {
		key = elem
	} else {
		// pointer to concrete type
		key = t
	}
	c.factories[key] = factory
	return nil
}

// isPointer checks if the interface is a pointer type.
func isPointer(iface interface{}) bool {
	t := reflect.TypeOf(iface)
	return t.Kind() == reflect.Ptr
}

// implementsInterface checks if mock implements the interface.
func implementsInterface(mock, iface interface{}) bool {
	t := reflect.TypeOf(iface)
	if t.Kind() != reflect.Ptr {
		return false
	}
	elem := t.Elem()
	if elem.Kind() != reflect.Interface {
		return false
	}
	mockType := reflect.TypeOf(mock)
	return mockType.Implements(elem)
}

// RegisterMock registers a mock implementation for testing.
func (c *Container) RegisterMock(iface, mock interface{}) error {
	if !isPointer(iface) {
		return errs.ErrInterfaceMustBePointer
	}

	if !implementsInterface(mock, iface) {
		return errs.ErrMockDoesNotImplement
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	key := reflect.TypeOf(iface)
	c.mocks[key] = mock

	return nil
}

// RegisterConfig registers a configuration value.
func (c *Container) RegisterConfig(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.configs[key] = value
}

// GetConfig retrieves a configuration value.
func (c *Container) GetConfig(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	value, ok := c.configs[key]
	return value, ok
}

// GetString retrieves the configuration value as a string.
func (c *Container) GetString(key string) (string, bool) {
	v, ok := c.GetConfig(key)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// GetInt retrieves the configuration value as an int.
func (c *Container) GetInt(key string) (int, bool) {
	v, ok := c.GetConfig(key)
	if !ok {
		return 0, false
	}
	i, ok := v.(int)
	return i, ok
}

// Resolve resolves a service instance.
func (c *Container) Resolve(target interface{}) error {
	targetType := reflect.TypeOf(target)
	if targetType.Kind() != reflect.Ptr {
		return errs.ErrTargetMustBePointer
	}

	elemType := targetType.Elem()

	c.mu.RLock()
	if mock, ok := c.mocks[elemType]; ok {
		reflect.ValueOf(target).Elem().Set(reflect.ValueOf(mock))
		c.mu.RUnlock()
		return nil
	}

	if service, ok := c.services[elemType]; ok {
		reflect.ValueOf(target).Elem().Set(reflect.ValueOf(service))
		c.mu.RUnlock()
		return nil
	}

	factory, ok := c.factories[elemType]
	if !ok {
		c.mu.RUnlock()
		return fmt.Errorf("%w for type %v", errs.ErrNoFactoryRegistered, elemType)
	}
	c.mu.RUnlock()

	// Create instance outside of lock
	instance, err := factory(c)
	if err != nil {
		return fmt.Errorf("%w: %w", errs.ErrFactoryFailed, err)
	}

	// Lock again to store the instance
	c.mu.Lock()
	c.services[elemType] = instance
	c.mu.Unlock()

	reflect.ValueOf(target).Elem().Set(reflect.ValueOf(instance))
	return nil
}

// MustResolve resolves a service instance or returns an error.
func (c *Container) MustResolve(target interface{}) error {
	if err := c.Resolve(target); err != nil {
		return fmt.Errorf("failed to resolve dependency: %w", err)
	}
	return nil
}

// Reset clears all registered services and mocks.
func (c *Container) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.services = make(map[reflect.Type]interface{})
	c.mocks = make(map[reflect.Type]interface{})
	c.configs = make(map[string]interface{})
}

// Clear removes a specific service or mock.
func (c *Container) Clear(iface interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	t := reflect.TypeOf(iface)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	delete(c.services, t)
	delete(c.mocks, t)
}
