package plugin

import (
	"fmt"

	"github.com/platformfoundry/platformfoundry-ce/pkg/plugin"
)

// Manager manages plugins
type Manager struct {
	plugins map[string]plugin.Plugin // key: "Kind:Provider" e.g., "Pipeline:jenkins"
}

// NewManager creates a new plugin manager
func NewManager() *Manager {
	return &Manager{
		plugins: make(map[string]plugin.Plugin),
	}
}

// Register registers a plugin
func (m *Manager) Register(p plugin.Plugin) error {
	key := m.getKey(p.Type(), p.Name())
	if _, exists := m.plugins[key]; exists {
		return fmt.Errorf("plugin %s already registered for type %s", p.Name(), p.Type())
	}
	m.plugins[key] = p
	return nil
}

// Get retrieves a plugin by type and provider name
func (m *Manager) Get(kind, provider string) (plugin.Plugin, error) {
	key := m.getKey(kind, provider)
	p, exists := m.plugins[key]
	if !exists {
		return nil, fmt.Errorf("plugin not found for %s:%s", kind, provider)
	}
	return p, nil
}

// List returns all registered plugins
func (m *Manager) List() []plugin.Plugin {
	plugins := make([]plugin.Plugin, 0, len(m.plugins))
	for _, p := range m.plugins {
		plugins = append(plugins, p)
	}
	return plugins
}

// getKey generates a unique key for a plugin
func (m *Manager) getKey(kind, provider string) string {
	return fmt.Sprintf("%s:%s", kind, provider)
}
