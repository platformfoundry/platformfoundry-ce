// Package testing provides test helpers for plugin development.
package testing

import (
	"context"
	"sync"

	"github.com/platformfoundry/pf-ce/pkg/contracts/ppi"
)

// MockProvider is a mock implementation of the Provider interface for testing
type MockProvider struct {
	mu sync.RWMutex

	// Schema to return from GetSchema
	SchemaValue *ppi.Schema

	// Resources to return from GetResources
	ResourcesValue []ppi.ResourceType

	// DataSources to return from GetDataSources
	DataSourcesValue []ppi.DataSourceType

	// Errors to return from Configure
	ConfigureError error

	// State storage for resources
	resources map[string]*ppi.ResourceState

	// Call tracking
	ConfigureCalls []configureCall
	ValidateCalls  []validateCall
	PlanCalls      []planCall
	ApplyCalls     []applyCall
	ImportCalls    []importCall
}

type configureCall struct {
	Config *ppi.ProviderConfig
}

type validateCall struct {
	TypeName string
	Config   *ppi.ResourceConfig
}

type planCall struct {
	TypeName string
	Prior    *ppi.ResourceState
	Proposed *ppi.ResourceState
}

type applyCall struct {
	TypeName string
	Plan     *ppi.Plan
}

type importCall struct {
	TypeName string
	ID       string
}

// NewMockProvider creates a new mock provider
func NewMockProvider() *MockProvider {
	return &MockProvider{
		resources: make(map[string]*ppi.ResourceState),
	}
}

// GetSchema returns the mock schema
func (p *MockProvider) GetSchema() *ppi.Schema {
	return p.SchemaValue
}

// GetResources returns the mock resources
func (p *MockProvider) GetResources() []ppi.ResourceType {
	return p.ResourcesValue
}

// GetDataSources returns the mock data sources
func (p *MockProvider) GetDataSources() []ppi.DataSourceType {
	return p.DataSourcesValue
}

// Configure records the configure call
func (p *MockProvider) Configure(ctx context.Context, config *ppi.ProviderConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ConfigureCalls = append(p.ConfigureCalls, configureCall{Config: config})
	return p.ConfigureError
}

// ValidateResource records the validate call
func (p *MockProvider) ValidateResource(ctx context.Context, typeName string, config *ppi.ResourceConfig) (*ppi.Diagnostics, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ValidateCalls = append(p.ValidateCalls, validateCall{TypeName: typeName, Config: config})
	return &ppi.Diagnostics{}, nil
}

// PlanResource records the plan call and returns a plan
func (p *MockProvider) PlanResource(ctx context.Context, typeName string, prior, proposed *ppi.ResourceState) (*ppi.Plan, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.PlanCalls = append(p.PlanCalls, planCall{TypeName: typeName, Prior: prior, Proposed: proposed})

	action := ppi.PlanActionCreate
	if prior != nil && proposed != nil {
		action = ppi.PlanActionUpdate
	} else if proposed == nil {
		action = ppi.PlanActionDelete
	}

	return &ppi.Plan{
		Action:   action,
		Prior:    prior,
		Proposed: proposed,
	}, nil
}

// ApplyResource records the apply call and updates state
func (p *MockProvider) ApplyResource(ctx context.Context, typeName string, plan *ppi.Plan) (*ppi.ResourceState, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ApplyCalls = append(p.ApplyCalls, applyCall{TypeName: typeName, Plan: plan})

	if plan.Action == ppi.PlanActionDelete {
		if plan.Prior != nil {
			delete(p.resources, plan.Prior.ID)
		}
		return nil, nil
	}

	state := plan.Proposed
	if state.ID == "" {
		state.ID = "mock-" + typeName + "-" + randomID()
	}
	p.resources[state.ID] = state
	return state, nil
}

// ImportResource records the import call
func (p *MockProvider) ImportResource(ctx context.Context, typeName string, id string) (*ppi.ResourceState, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ImportCalls = append(p.ImportCalls, importCall{TypeName: typeName, ID: id})

	state, ok := p.resources[id]
	if !ok {
		return nil, &ppi.Diagnostics{}
	}
	return state, nil
}

// DestroyResource destroys a resource
func (p *MockProvider) DestroyResource(ctx context.Context, typeName string, state *ppi.ResourceState) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.resources, state.ID)
	return nil
}

// ReadDataSource reads a data source
func (p *MockProvider) ReadDataSource(ctx context.Context, typeName string, config *ppi.DataSourceConfig) (*ppi.DataSourceState, error) {
	return &ppi.DataSourceState{
		ID:         "mock-ds-" + typeName,
		TypeName:   typeName,
		Attributes: config.Values,
	}, nil
}

// Close does nothing for the mock
func (p *MockProvider) Close() error {
	return nil
}

// GetResource returns a stored resource by ID
func (p *MockProvider) GetResource(id string) *ppi.ResourceState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.resources[id]
}

// SetResource stores a resource
func (p *MockProvider) SetResource(id string, state *ppi.ResourceState) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.resources[id] = state
}

// Reset clears all call tracking and resources
func (p *MockProvider) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ConfigureCalls = nil
	p.ValidateCalls = nil
	p.PlanCalls = nil
	p.ApplyCalls = nil
	p.ImportCalls = nil
	p.resources = make(map[string]*ppi.ResourceState)
}

var idCounter int
var idMu sync.Mutex

func randomID() string {
	idMu.Lock()
	defer idMu.Unlock()
	idCounter++
	return string(rune('a'+idCounter%26)) + string(rune('0'+idCounter/26))
}
