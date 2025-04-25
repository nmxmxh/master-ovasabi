package feature

import (
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFeatureFlag(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue bool
		expected     string
	}{
		{
			name:         "with environment set",
			envValue:     "production",
			defaultValue: true,
			expected:     "production",
		},
		{
			name:         "without environment",
			envValue:     "",
			defaultValue: false,
			expected:     "development",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			if tt.envValue != "" {
				if err := os.Setenv("ENVIRONMENT", tt.envValue); err != nil {
					t.Fatalf("Failed to set ENVIRONMENT: %v", err)
				}
				defer func() {
					_ = os.Unsetenv("ENVIRONMENT")
				}()
			} else {
				if err := os.Unsetenv("ENVIRONMENT"); err != nil {
					t.Fatalf("Failed to unset ENVIRONMENT: %v", err)
				}
			}

			flag := NewFeatureFlag("test-feature", tt.defaultValue)
			assert.Equal(t, "test-feature", flag.Name)
			assert.Equal(t, tt.defaultValue, flag.Enabled)
			assert.Equal(t, tt.expected, flag.Env)
		})
	}
}

func TestFeatureFlag_EnableDisable(t *testing.T) {
	flag := NewFeatureFlag("test-feature", false)

	// Test initial state
	assert.False(t, flag.IsEnabled())

	// Test Enable
	flag.Enable()
	assert.True(t, flag.IsEnabled())

	// Test Disable
	flag.Disable()
	assert.False(t, flag.IsEnabled())
}

func TestFeatureFlag_ConcurrentAccess(t *testing.T) {
	flag := NewFeatureFlag("test-feature", false)
	var wg sync.WaitGroup
	iterations := 1000

	// Concurrent reads and writes
	for i := 0; i < iterations; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			flag.Enable()
			flag.Disable()
		}()
		go func() {
			defer wg.Done()
			_ = flag.IsEnabled()
		}()
	}

	wg.Wait()
}

func TestFeatureManager_RegisterFeature(t *testing.T) {
	manager := NewFeatureManager()
	flag := manager.RegisterFeature("test-feature", true)

	assert.NotNil(t, flag)
	assert.Equal(t, "test-feature", flag.Name)
	assert.True(t, flag.Enabled)

	// Test duplicate registration
	flag2 := manager.RegisterFeature("test-feature", false)
	assert.NotNil(t, flag2)
	assert.Equal(t, "test-feature", flag2.Name)
	assert.False(t, flag2.Enabled) // The new value should override the old one
}

func TestFeatureManager_GetFeature(t *testing.T) {
	manager := NewFeatureManager()

	// Register a feature
	original := manager.RegisterFeature("test-feature", true)
	require.NotNil(t, original)

	// Get the feature
	retrieved, exists := manager.GetFeature("test-feature")
	assert.True(t, exists)
	assert.Equal(t, original, retrieved)

	// Try to get a non-existent feature
	retrieved, exists = manager.GetFeature("non-existent")
	assert.False(t, exists)
	assert.Nil(t, retrieved)
}

func TestFeatureManager_ConcurrentAccess(t *testing.T) {
	manager := NewFeatureManager()
	var wg sync.WaitGroup
	iterations := 100
	features := []string{"feature1", "feature2", "feature3", "feature4", "feature5"}

	// Concurrent registration and retrieval
	for i := 0; i < iterations; i++ {
		for _, feature := range features {
			wg.Add(2)
			go func(name string) {
				defer wg.Done()
				manager.RegisterFeature(name, true)
			}(feature)
			go func(name string) {
				defer wg.Done()
				_, _ = manager.GetFeature(name)
			}(feature)
		}
	}

	wg.Wait()

	// Verify all features are registered
	for _, feature := range features {
		flag, exists := manager.GetFeature(feature)
		assert.True(t, exists)
		assert.NotNil(t, flag)
		assert.Equal(t, feature, flag.Name)
	}
}

func TestFeatureFlag_Environment(t *testing.T) {
	environments := []string{"development", "staging", "production", "test"}

	for _, env := range environments {
		t.Run(env, func(t *testing.T) {
			if err := os.Setenv("ENVIRONMENT", env); err != nil {
				t.Fatalf("Failed to set ENVIRONMENT: %v", err)
			}
			defer func() {
				_ = os.Unsetenv("ENVIRONMENT")
			}()

			flag := NewFeatureFlag("test-feature", true)
			assert.Equal(t, env, flag.Env)
		})
	}
}

func TestFeatureManager_MultipleFeatures(t *testing.T) {
	manager := NewFeatureManager()
	features := map[string]bool{
		"feature1": true,
		"feature2": false,
		"feature3": true,
		"feature4": false,
		"feature5": true,
	}

	// Register all features
	for name, enabled := range features {
		flag := manager.RegisterFeature(name, enabled)
		assert.NotNil(t, flag)
		assert.Equal(t, name, flag.Name)
		assert.Equal(t, enabled, flag.IsEnabled())
	}

	// Verify all features
	for name, enabled := range features {
		flag, exists := manager.GetFeature(name)
		assert.True(t, exists)
		assert.NotNil(t, flag)
		assert.Equal(t, name, flag.Name)
		assert.Equal(t, enabled, flag.IsEnabled())
	}
}
