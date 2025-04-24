package feature

import (
	"os"
	"sync"
)

// FeatureFlag represents a feature toggle
type FeatureFlag struct {
	Name    string
	Enabled bool
	Env     string
	mu      sync.RWMutex
}

// NewFeatureFlag creates a new feature flag
func NewFeatureFlag(name string, defaultValue bool) *FeatureFlag {
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "development"
	}
	return &FeatureFlag{
		Name:    name,
		Enabled: defaultValue,
		Env:     env,
	}
}

// IsEnabled checks if the feature is enabled
func (f *FeatureFlag) IsEnabled() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.Enabled
}

// Enable turns on the feature flag
func (f *FeatureFlag) Enable() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Enabled = true
}

// Disable turns off the feature flag
func (f *FeatureFlag) Disable() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Enabled = false
}

// FeatureManager manages all feature flags
type FeatureManager struct {
	flags map[string]*FeatureFlag
	mu    sync.RWMutex
}

// NewFeatureManager creates a new feature manager
func NewFeatureManager() *FeatureManager {
	return &FeatureManager{
		flags: make(map[string]*FeatureFlag),
	}
}

// RegisterFeature adds a new feature flag
func (fm *FeatureManager) RegisterFeature(name string, defaultValue bool) *FeatureFlag {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	flag := NewFeatureFlag(name, defaultValue)
	fm.flags[name] = flag
	return flag
}

// GetFeature retrieves a feature flag
func (fm *FeatureManager) GetFeature(name string) (*FeatureFlag, bool) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	flag, exists := fm.flags[name]
	return flag, exists
}
