// Package config provides configuration management for CYP-Docker-Registry.
package config

import (
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Watcher watches configuration file for changes and reloads automatically.
type Watcher struct {
	path       string
	interval   time.Duration
	lastMod    time.Time
	callbacks  []func(*Config)
	stopCh     chan struct{}
	logger     *zap.Logger
	mu         sync.RWMutex
	isRunning  bool
}

// NewWatcher creates a new configuration watcher.
func NewWatcher(path string, interval time.Duration, logger *zap.Logger) *Watcher {
	return &Watcher{
		path:      path,
		interval:  interval,
		callbacks: make([]func(*Config), 0),
		stopCh:    make(chan struct{}),
		logger:    logger,
	}
}

// OnReload registers a callback to be called when configuration is reloaded.
func (w *Watcher) OnReload(callback func(*Config)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.callbacks = append(w.callbacks, callback)
}

// Start starts watching the configuration file.
func (w *Watcher) Start() error {
	w.mu.Lock()
	if w.isRunning {
		w.mu.Unlock()
		return nil
	}
	w.isRunning = true
	w.mu.Unlock()

	// Get initial modification time
	info, err := os.Stat(w.path)
	if err != nil {
		return err
	}
	w.lastMod = info.ModTime()

	go w.watch()

	if w.logger != nil {
		w.logger.Info("Configuration watcher started",
			zap.String("path", w.path),
			zap.Duration("interval", w.interval),
		)
	}

	return nil
}

// Stop stops watching the configuration file.
func (w *Watcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.isRunning {
		return
	}

	close(w.stopCh)
	w.isRunning = false

	if w.logger != nil {
		w.logger.Info("Configuration watcher stopped")
	}
}

// watch is the main watch loop.
func (w *Watcher) watch() {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.checkAndReload()
		}
	}
}

// checkAndReload checks if the configuration file has changed and reloads it.
func (w *Watcher) checkAndReload() {
	info, err := os.Stat(w.path)
	if err != nil {
		if w.logger != nil {
			w.logger.Error("Failed to stat config file",
				zap.String("path", w.path),
				zap.Error(err),
			)
		}
		return
	}

	if info.ModTime().After(w.lastMod) {
		w.lastMod = info.ModTime()
		w.reload()
	}
}

// reload reloads the configuration and notifies callbacks.
func (w *Watcher) reload() {
	config, err := Load(w.path)
	if err != nil {
		if w.logger != nil {
			w.logger.Error("Failed to reload config",
				zap.String("path", w.path),
				zap.Error(err),
			)
		}
		return
	}

	if w.logger != nil {
		w.logger.Info("Configuration reloaded",
			zap.String("path", w.path),
		)
	}

	// Notify callbacks
	w.mu.RLock()
	callbacks := make([]func(*Config), len(w.callbacks))
	copy(callbacks, w.callbacks)
	w.mu.RUnlock()

	for _, callback := range callbacks {
		go callback(config)
	}
}

// ForceReload forces a configuration reload.
func (w *Watcher) ForceReload() error {
	config, err := Load(w.path)
	if err != nil {
		return err
	}

	// Notify callbacks
	w.mu.RLock()
	callbacks := make([]func(*Config), len(w.callbacks))
	copy(callbacks, w.callbacks)
	w.mu.RUnlock()

	for _, callback := range callbacks {
		go callback(config)
	}

	return nil
}

// IsRunning returns whether the watcher is running.
func (w *Watcher) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.isRunning
}
