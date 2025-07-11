package registration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

// ConfigWatcher watches for changes in proto files and automatically
// regenerates service registration configs
type ConfigWatcher struct {
	logger    *zap.Logger
	generator *DynamicServiceRegistrationGenerator
	watcher   *fsnotify.Watcher

	protoPath  string
	outputPath string
	debounceMs int
}

// NewConfigWatcher creates a new configuration watcher
func NewConfigWatcher(logger *zap.Logger, generator *DynamicServiceRegistrationGenerator,
	protoPath, outputPath string) (*ConfigWatcher, error) {

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	return &ConfigWatcher{
		logger:     logger,
		generator:  generator,
		watcher:    watcher,
		protoPath:  protoPath,
		outputPath: outputPath,
		debounceMs: 1000, // 1 second debounce
	}, nil
}

// Start begins watching for file changes
func (cw *ConfigWatcher) Start(ctx context.Context) error {
	// Walk through proto directory and add all directories to watch
	err := filepath.Walk(cw.protoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			cw.logger.Debug("Adding directory to watch", zap.String("path", path))
			return cw.watcher.Add(path)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to add directories to watch: %w", err)
	}

	cw.logger.Info("Started watching for proto file changes",
		zap.String("protoPath", cw.protoPath),
		zap.String("outputPath", cw.outputPath))

	// Channel for debouncing changes
	debounceTimer := time.NewTimer(0)
	<-debounceTimer.C // drain the timer

	go func() {
		for {
			select {
			case event, ok := <-cw.watcher.Events:
				if !ok {
					return
				}

				if cw.shouldProcessEvent(event) {
					cw.logger.Debug("File change detected",
						zap.String("file", event.Name),
						zap.String("op", event.Op.String()))

					// Debounce: reset timer on each event
					debounceTimer.Reset(time.Duration(cw.debounceMs) * time.Millisecond)
				}

			case err, ok := <-cw.watcher.Errors:
				if !ok {
					return
				}
				cw.logger.Error("Watcher error", zap.Error(err))

			case <-debounceTimer.C:
				// Timer expired, regenerate config
				cw.regenerateConfig(ctx)

			case <-ctx.Done():
				cw.logger.Info("Stopping config watcher")
				return
			}
		}
	}()

	return nil
}

// Stop stops the watcher
func (cw *ConfigWatcher) Stop() error {
	if cw.watcher != nil {
		return cw.watcher.Close()
	}
	return nil
}

// shouldProcessEvent determines if a file system event should trigger regeneration
func (cw *ConfigWatcher) shouldProcessEvent(event fsnotify.Event) bool {
	// Only process create, write, and remove events
	if event.Op&fsnotify.Create == 0 &&
		event.Op&fsnotify.Write == 0 &&
		event.Op&fsnotify.Remove == 0 {
		return false
	}

	// Only process .proto files
	if !strings.HasSuffix(event.Name, ".proto") {
		return false
	}

	return true
}

// regenerateConfig regenerates the service registration config
func (cw *ConfigWatcher) regenerateConfig(ctx context.Context) {
	cw.logger.Info("Regenerating service registration config...")

	start := time.Now()
	if err := cw.generator.GenerateAndSaveConfig(ctx, cw.outputPath); err != nil {
		cw.logger.Error("Failed to regenerate config", zap.Error(err))
		return
	}

	duration := time.Since(start)
	cw.logger.Info("Config regenerated successfully",
		zap.Duration("duration", duration),
		zap.String("outputPath", cw.outputPath))
}

// WatcherConfig holds configuration for the config watcher
type WatcherConfig struct {
	ProtoPath  string
	OutputPath string
	DebounceMs int
	AutoReload bool
	NotifyCmd  string // Command to run after regeneration
}

// NewConfigWatcherWithConfig creates a watcher with custom configuration
func NewConfigWatcherWithConfig(logger *zap.Logger, generator *DynamicServiceRegistrationGenerator,
	config WatcherConfig) (*ConfigWatcher, error) {

	watcher, err := NewConfigWatcher(logger, generator, config.ProtoPath, config.OutputPath)
	if err != nil {
		return nil, err
	}

	if config.DebounceMs > 0 {
		watcher.debounceMs = config.DebounceMs
	}

	return watcher, nil
}
