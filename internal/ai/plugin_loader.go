package ai

import (
	"sync"
)

type PluginLoader struct {
	plugins map[string]Plugin
	mu      sync.RWMutex
}

func NewPluginLoader() *PluginLoader {
	return &PluginLoader{
		plugins: make(map[string]Plugin),
	}
}

func (pl *PluginLoader) LoadGoPlugin(name string, plugin Plugin) {
	pl.mu.Lock()
	defer pl.mu.Unlock()
	pl.plugins[name] = plugin
}

func (pl *PluginLoader) LoadWASMPlugin(name string, loader func() Plugin) {
	pl.mu.Lock()
	defer pl.mu.Unlock()
	pl.plugins[name] = loader()
}

func (pl *PluginLoader) GetPlugin(name string) (Plugin, bool) {
	pl.mu.RLock()
	defer pl.mu.RUnlock()
	plugin, ok := pl.plugins[name]
	return plugin, ok
}

func (pl *PluginLoader) ListPlugins() []string {
	pl.mu.RLock()
	defer pl.mu.RUnlock()
	names := make([]string, 0, len(pl.plugins))
	for name := range pl.plugins {
		names = append(names, name)
	}
	return names
}

// Example: Register a multithreaded WASM plugin
// loaderFunc should return an AIPlugin instance that wraps the WASM module with multithreaded support.
func (pl *PluginLoader) RegisterMultithreadedWASMPlugin(name string, loaderFunc func() Plugin) {
	pl.LoadWASMPlugin(name, loaderFunc)
}

// RegisterLearningWASMPlugin registers a WASM plugin that only learns (does not act).
func (pl *PluginLoader) RegisterLearningWASMPlugin(name string, plugin Plugin) {
	pl.LoadGoPlugin(name, plugin)
}
