package mock

import (
	"fmt"
	"sync"

	"github.com/platformfoundry/pf-ce/pkg/plugin"
)

// Registry manages mock plugins
type Registry struct {
	mocks        map[string]*MockPlugin
	mu           sync.RWMutex
	globalConfig *MockConfig
}

// NewRegistry creates a new mock registry
func NewRegistry(config *MockConfig) *Registry {
	if config == nil {
		config = DefaultMockConfig()
	}
	return &Registry{
		mocks:        make(map[string]*MockPlugin),
		globalConfig: config,
	}
}

// RegisterMock creates and registers a mock plugin
func (r *Registry) RegisterMock(tool, toolType string, config *MockConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cfg := r.globalConfig
	if config != nil {
		cfg = config
	}

	key := fmt.Sprintf("%s:%s", toolType, tool)
	r.mocks[key] = NewMockPlugin(tool, toolType, cfg)
}

// RegisterMockFromPlugin wraps a real plugin and registers it
func (r *Registry) RegisterMockFromPlugin(p plugin.Plugin, config *MockConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cfg := r.globalConfig
	if config != nil {
		cfg = config
	}

	key := fmt.Sprintf("%s:%s", p.Type(), p.Name())
	r.mocks[key] = WrapPlugin(p, cfg)
}

// Get returns a mock plugin by kind and name
func (r *Registry) Get(kind, name string) (*MockPlugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", kind, name)
	mock, ok := r.mocks[key]
	return mock, ok
}

// GetOrCreate returns an existing mock or creates a new one
func (r *Registry) GetOrCreate(kind, name string) *MockPlugin {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := fmt.Sprintf("%s:%s", kind, name)
	if mock, ok := r.mocks[key]; ok {
		return mock
	}

	mock := NewMockPlugin(name, kind, r.globalConfig)
	r.mocks[key] = mock
	return mock
}

// List returns all registered mock plugins
func (r *Registry) List() map[string]*MockPlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*MockPlugin)
	for k, v := range r.mocks {
		result[k] = v
	}
	return result
}

// ListByType returns mock plugins by type
func (r *Registry) ListByType(toolType string) []*MockPlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*MockPlugin
	for _, mock := range r.mocks {
		if mock.Type() == toolType {
			result = append(result, mock)
		}
	}
	return result
}

// SetGlobalConfig updates the global mock configuration
func (r *Registry) SetGlobalConfig(config *MockConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.globalConfig = config
}

// GetGlobalConfig returns the global mock configuration
func (r *Registry) GetGlobalConfig() *MockConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.globalConfig
}

// Clear removes all registered mocks
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.mocks = make(map[string]*MockPlugin)
}

// Remove removes a specific mock
func (r *Registry) Remove(kind, name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := fmt.Sprintf("%s:%s", kind, name)
	delete(r.mocks, key)
}

// Count returns the number of registered mocks
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.mocks)
}

// RegisterBuiltinMocks registers mock plugins for common tools
func (r *Registry) RegisterBuiltinMocks() {
	// Infrastructure
	r.RegisterMock("terraform", "Infrastructure", nil)
	r.RegisterMock("aws", "Infrastructure", nil)
	r.RegisterMock("gcp", "Infrastructure", nil)
	r.RegisterMock("azure", "Infrastructure", nil)

	// Orchestration
	r.RegisterMock("argocd", "Orchestrator", nil)
	r.RegisterMock("flux", "Orchestrator", nil)
	r.RegisterMock("rancher", "Orchestrator", nil)

	// Observability
	r.RegisterMock("prometheus", "Observability", nil)
	r.RegisterMock("prometheus-stack", "Observability", nil)
	r.RegisterMock("grafana", "Observability", nil)
	r.RegisterMock("loki", "Observability", nil)
	r.RegisterMock("datadog", "Observability", nil)

	// DevEx
	r.RegisterMock("backstage", "DevEx", nil)

	// Security
	r.RegisterMock("vault", "Security", nil)
	r.RegisterMock("external-secrets", "Security", nil)
	r.RegisterMock("opa-gatekeeper", "Security", nil)

	// Pipeline
	r.RegisterMock("jenkins", "Pipeline", nil)
	r.RegisterMock("tekton", "Pipeline", nil)
	r.RegisterMock("github-actions", "Pipeline", nil)
}

// DefaultRegistry is a global mock registry with builtin mocks
var DefaultRegistry = func() *Registry {
	r := NewRegistry(nil)
	r.RegisterBuiltinMocks()
	return r
}()
