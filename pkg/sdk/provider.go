// Package sdk provides the Plugin SDK for Platform Foundry.
package sdk

import (
	"context"

	"github.com/platformfoundry/pf-ce/pkg/contracts/ppi"
)

// ProviderBase provides a base implementation of the Provider interface.
// Plugin developers can embed this in their provider implementations
// to get default behavior for optional methods.
type ProviderBase struct {
	schema      *ppi.Schema
	resources   map[string]Resource
	dataSources map[string]DataSource
	configured  bool
}

// NewProviderBase creates a new ProviderBase
func NewProviderBase() *ProviderBase {
	return &ProviderBase{
		resources:   make(map[string]Resource),
		dataSources: make(map[string]DataSource),
	}
}

// SetSchema sets the provider schema
func (p *ProviderBase) SetSchema(schema *ppi.Schema) {
	p.schema = schema
}

// RegisterResource registers a resource with the provider
func (p *ProviderBase) RegisterResource(name string, resource Resource) {
	p.resources[name] = resource
}

// RegisterDataSource registers a data source with the provider
func (p *ProviderBase) RegisterDataSource(name string, ds DataSource) {
	p.dataSources[name] = ds
}

// GetSchema returns the provider schema
func (p *ProviderBase) GetSchema() *ppi.Schema {
	return p.schema
}

// GetResources returns all registered resource types
func (p *ProviderBase) GetResources() []ppi.ResourceType {
	types := make([]ppi.ResourceType, 0, len(p.resources))
	for name, r := range p.resources {
		types = append(types, ppi.ResourceType{
			Name:        name,
			Description: r.Description(),
			Schema:      r.Schema(),
		})
	}
	return types
}

// GetDataSources returns all registered data source types
func (p *ProviderBase) GetDataSources() []ppi.DataSourceType {
	types := make([]ppi.DataSourceType, 0, len(p.dataSources))
	for name, ds := range p.dataSources {
		types = append(types, ppi.DataSourceType{
			Name:        name,
			Description: ds.Description(),
			Schema:      ds.Schema(),
		})
	}
	return types
}

// ValidateResource validates a resource configuration
func (p *ProviderBase) ValidateResource(ctx context.Context, typeName string, config *ppi.ResourceConfig) (*ppi.Diagnostics, error) {
	resource, ok := p.resources[typeName]
	if !ok {
		diags := &ppi.Diagnostics{}
		diags.AddError("Unknown resource type", "Resource type %s is not supported", typeName)
		return diags, nil
	}
	return resource.Validate(ctx, config)
}

// PlanResource creates a plan for a resource change
func (p *ProviderBase) PlanResource(ctx context.Context, typeName string, prior, proposed *ppi.ResourceState) (*ppi.Plan, error) {
	resource, ok := p.resources[typeName]
	if !ok {
		return nil, &ResourceError{
			TypeName: typeName,
			Message:  "unknown resource type",
		}
	}
	return resource.Plan(ctx, prior, proposed)
}

// ApplyResource applies a resource plan
func (p *ProviderBase) ApplyResource(ctx context.Context, typeName string, plan *ppi.Plan) (*ppi.ResourceState, error) {
	resource, ok := p.resources[typeName]
	if !ok {
		return nil, &ResourceError{
			TypeName: typeName,
			Message:  "unknown resource type",
		}
	}

	switch plan.Action {
	case ppi.PlanActionCreate:
		return resource.Create(ctx, plan)
	case ppi.PlanActionUpdate:
		return resource.Update(ctx, plan)
	case ppi.PlanActionDelete:
		return nil, resource.Delete(ctx, plan.Prior)
	case ppi.PlanActionReplace:
		if err := resource.Delete(ctx, plan.Prior); err != nil {
			return nil, err
		}
		return resource.Create(ctx, plan)
	case ppi.PlanActionNoop:
		return plan.Prior, nil
	default:
		return nil, &ResourceError{
			TypeName: typeName,
			Message:  "unknown plan action: " + string(plan.Action),
		}
	}
}

// ImportResource imports an existing resource
func (p *ProviderBase) ImportResource(ctx context.Context, typeName string, id string) (*ppi.ResourceState, error) {
	resource, ok := p.resources[typeName]
	if !ok {
		return nil, &ResourceError{
			TypeName: typeName,
			Message:  "unknown resource type",
		}
	}
	return resource.Import(ctx, id)
}

// DestroyResource destroys a resource
func (p *ProviderBase) DestroyResource(ctx context.Context, typeName string, state *ppi.ResourceState) error {
	resource, ok := p.resources[typeName]
	if !ok {
		return &ResourceError{
			TypeName: typeName,
			Message:  "unknown resource type",
		}
	}
	return resource.Delete(ctx, state)
}

// ReadDataSource reads a data source
func (p *ProviderBase) ReadDataSource(ctx context.Context, typeName string, config *ppi.DataSourceConfig) (*ppi.DataSourceState, error) {
	ds, ok := p.dataSources[typeName]
	if !ok {
		return nil, &ResourceError{
			TypeName: typeName,
			Message:  "unknown data source type",
		}
	}
	return ds.Read(ctx, config)
}

// IsConfigured returns whether the provider has been configured
func (p *ProviderBase) IsConfigured() bool {
	return p.configured
}

// SetConfigured marks the provider as configured
func (p *ProviderBase) SetConfigured(configured bool) {
	p.configured = configured
}

// Resource represents a resource that can be managed by a provider
type Resource interface {
	// Schema returns the resource schema
	Schema() *ppi.Schema

	// Description returns a description of the resource
	Description() string

	// Validate validates the resource configuration
	Validate(ctx context.Context, config *ppi.ResourceConfig) (*ppi.Diagnostics, error)

	// Plan creates a plan for resource changes
	Plan(ctx context.Context, prior, proposed *ppi.ResourceState) (*ppi.Plan, error)

	// Create creates a new resource
	Create(ctx context.Context, plan *ppi.Plan) (*ppi.ResourceState, error)

	// Read reads the current state of a resource
	Read(ctx context.Context, state *ppi.ResourceState) (*ppi.ResourceState, error)

	// Update updates an existing resource
	Update(ctx context.Context, plan *ppi.Plan) (*ppi.ResourceState, error)

	// Delete deletes a resource
	Delete(ctx context.Context, state *ppi.ResourceState) error

	// Import imports an existing resource by ID
	Import(ctx context.Context, id string) (*ppi.ResourceState, error)
}

// DataSource represents a read-only data source
type DataSource interface {
	// Schema returns the data source schema
	Schema() *ppi.Schema

	// Description returns a description of the data source
	Description() string

	// Read reads the data source
	Read(ctx context.Context, config *ppi.DataSourceConfig) (*ppi.DataSourceState, error)
}

// ResourceError represents an error related to a resource operation
type ResourceError struct {
	TypeName string
	ID       string
	Message  string
	Cause    error
}

func (e *ResourceError) Error() string {
	msg := "resource error"
	if e.TypeName != "" {
		msg += " [" + e.TypeName + "]"
	}
	if e.ID != "" {
		msg += " (id: " + e.ID + ")"
	}
	msg += ": " + e.Message
	if e.Cause != nil {
		msg += ": " + e.Cause.Error()
	}
	return msg
}

func (e *ResourceError) Unwrap() error {
	return e.Cause
}
