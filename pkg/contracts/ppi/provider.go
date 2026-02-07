// Package ppi defines the Platform Provider Interface (PPI).
// This is inspired by Terraform's provider protocol and handles infrastructure provisioning.
package ppi

import (
	"context"
)

// Provider is the main interface that all providers must implement.
// It handles infrastructure provisioning operations.
type Provider interface {
	// Metadata returns information about this provider
	GetSchema() *Schema
	GetResources() []ResourceType
	GetDataSources() []DataSourceType

	// Lifecycle manages provider configuration
	Configure(ctx context.Context, config *ProviderConfig) error

	// Resource Operations handle CRUD operations on resources
	ValidateResource(ctx context.Context, typeName string, config *ResourceConfig) (*Diagnostics, error)
	PlanResource(ctx context.Context, typeName string, prior, proposed *ResourceState) (*Plan, error)
	ApplyResource(ctx context.Context, typeName string, plan *Plan) (*ResourceState, error)
	ImportResource(ctx context.Context, typeName string, id string) (*ResourceState, error)
	DestroyResource(ctx context.Context, typeName string, state *ResourceState) error

	// DataSource Operations handle read-only data lookups
	ReadDataSource(ctx context.Context, typeName string, config *DataSourceConfig) (*DataSourceState, error)

	// Cleanup releases provider resources
	Close() error
}

// ProviderConfig holds configuration for a provider instance
type ProviderConfig struct {
	// Raw contains the raw configuration values
	Raw map[string]interface{}
}

// GetString returns a string configuration value
func (c *ProviderConfig) GetString(key string) string {
	if v, ok := c.Raw[key].(string); ok {
		return v
	}
	return ""
}

// GetInt returns an integer configuration value
func (c *ProviderConfig) GetInt(key string) int {
	if v, ok := c.Raw[key].(int); ok {
		return v
	}
	return 0
}

// GetBool returns a boolean configuration value
func (c *ProviderConfig) GetBool(key string) bool {
	if v, ok := c.Raw[key].(bool); ok {
		return v
	}
	return false
}

// ResourceType represents a type of resource that a provider can manage
type ResourceType struct {
	Name        string
	Description string
	Schema      *Schema
}

// DataSourceType represents a type of data source that a provider can read
type DataSourceType struct {
	Name        string
	Description string
	Schema      *Schema
}

// ProviderMetadata contains metadata about a provider
type ProviderMetadata struct {
	Name        string
	Version     string
	Description string
	Source      string
}
