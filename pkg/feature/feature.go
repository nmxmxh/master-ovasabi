package feature

import (
	"os"
	"sync"
)

// Rule defines a feature flag rule.
type Rule struct {
	Type      string
	Value     interface{}
	Condition string
}

// Flag represents a feature flag configuration.
type Flag struct {
	Name        string
	Description string
	Enabled     bool
	Rules       []Rule
}

// Manager handles feature flag management.
type Manager struct {
	flags map[string]*Flag
	mu    sync.RWMutex
}

// Toggle represents a feature toggle.
type Toggle struct {
	Name    string
	Enabled bool
	Env     string
	mu      sync.RWMutex
}

// NewToggle creates a new feature toggle.
func NewToggle(name string, defaultValue bool) *Toggle {
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "development"
	}
	return &Toggle{
		Name:    name,
		Enabled: defaultValue,
		Env:     env,
	}
}

// IsEnabled checks if the feature is enabled.
func (f *Toggle) IsEnabled() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.Enabled
}

// Enable turns on the feature toggle.
func (f *Toggle) Enable() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Enabled = true
}

// Disable turns off the feature toggle.
func (f *Toggle) Disable() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Enabled = false
}

// NewManager creates a new feature manager.
func NewManager() *Manager {
	return &Manager{
		flags: make(map[string]*Flag),
	}
}

// RegisterFeature adds a new feature flag.
func (m *Manager) RegisterFeature(name string, defaultValue bool) *Flag {
	m.mu.Lock()
	defer m.mu.Unlock()

	flag := &Flag{
		Name:    name,
		Enabled: defaultValue,
	}
	m.flags[name] = flag
	return flag
}

// GetFeature retrieves a feature flag.
func (m *Manager) GetFeature(name string) (*Flag, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	flag, ok := m.flags[name]
	return flag, ok
}

// IsEnabled checks if a feature is enabled.
func (m *Manager) IsEnabled(name string) bool {
	if flag, ok := m.GetFeature(name); ok {
		return flag.Enabled
	}
	return false
}

// Enable enables a feature flag.
func (m *Manager) Enable(name string) {
	if flag, ok := m.GetFeature(name); ok {
		flag.Enabled = true
	}
}

// Disable disables a feature flag.
func (m *Manager) Disable(name string) {
	if flag, ok := m.GetFeature(name); ok {
		flag.Enabled = false
	}
}
