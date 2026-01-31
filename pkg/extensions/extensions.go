// Package extensions provides the extension system for Platform Foundry.
// This allows the Enterprise Edition to register additional features
// without modifying the core Community Edition codebase.
package extensions

import (
	"sync"

	"github.com/spf13/cobra"
)

// Extension represents a Platform Foundry extension
type Extension struct {
	Name        string
	Version     string
	Description string
	Commands    []*cobra.Command
	OnInit      func() error
	OnShutdown  func() error
}

// Registry holds all registered extensions
type Registry struct {
	mu         sync.RWMutex
	extensions map[string]*Extension
	commands   []*cobra.Command
	features   map[string]FeatureProvider
}

// FeatureProvider interface for pluggable features
type FeatureProvider interface {
	Name() string
	Initialize() error
	Shutdown() error
}

// Global registry instance
var globalRegistry = &Registry{
	extensions: make(map[string]*Extension),
	features:   make(map[string]FeatureProvider),
}

// Register adds an extension to the registry
func Register(ext *Extension) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	globalRegistry.extensions[ext.Name] = ext
	globalRegistry.commands = append(globalRegistry.commands, ext.Commands...)
}

// RegisterFeature adds a feature provider
func RegisterFeature(name string, provider FeatureProvider) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	globalRegistry.features[name] = provider
}

// GetCommands returns all registered extension commands
func GetCommands() []*cobra.Command {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()
	return globalRegistry.commands
}

// GetFeature returns a registered feature provider
func GetFeature(name string) (FeatureProvider, bool) {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()
	f, ok := globalRegistry.features[name]
	return f, ok
}

// GetExtension returns a registered extension by name
func GetExtension(name string) (*Extension, bool) {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()
	ext, ok := globalRegistry.extensions[name]
	return ext, ok
}

// ListExtensions returns all registered extension names
func ListExtensions() []string {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()
	names := make([]string, 0, len(globalRegistry.extensions))
	for name := range globalRegistry.extensions {
		names = append(names, name)
	}
	return names
}

// InitializeAll initializes all registered extensions
func InitializeAll() error {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()
	for _, ext := range globalRegistry.extensions {
		if ext.OnInit != nil {
			if err := ext.OnInit(); err != nil {
				return err
			}
		}
	}
	for _, f := range globalRegistry.features {
		if err := f.Initialize(); err != nil {
			return err
		}
	}
	return nil
}

// ShutdownAll shuts down all registered extensions
func ShutdownAll() error {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()
	for _, ext := range globalRegistry.extensions {
		if ext.OnShutdown != nil {
			if err := ext.OnShutdown(); err != nil {
				return err
			}
		}
	}
	for _, f := range globalRegistry.features {
		if err := f.Shutdown(); err != nil {
			return err
		}
	}
	return nil
}
