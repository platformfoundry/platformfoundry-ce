package drivers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// HTTPDriverConfig contains configuration for HTTP-based drivers
type HTTPDriverConfig struct {
	Name         string            `json:"name"`
	ResourceType string            `json:"resourceType"`
	BaseURL      string            `json:"baseUrl"`
	Headers      map[string]string `json:"headers,omitempty"`
	Timeout      time.Duration     `json:"timeout,omitempty"`
}

// HTTPDriver is a generic HTTP-based resource driver
type HTTPDriver struct {
	config HTTPDriverConfig
	client *http.Client
}

// NewHTTPDriver creates a new HTTP driver
func NewHTTPDriver(config HTTPDriverConfig) *HTTPDriver {
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &HTTPDriver{
		config: config,
		client: &http.Client{Timeout: timeout},
	}
}

func (d *HTTPDriver) Name() string         { return d.config.Name }
func (d *HTTPDriver) ResourceType() string { return d.config.ResourceType }

func (d *HTTPDriver) Provision(ctx context.Context, spec map[string]interface{}) (*ProvisionResult, error) {
	// In production, this would make an HTTP POST to create the resource
	return &ProvisionResult{
		ID:        fmt.Sprintf("%s-%d", d.config.ResourceType, time.Now().UnixNano()),
		Status:    "ready",
		CreatedAt: time.Now(),
		Outputs:   spec,
	}, nil
}

func (d *HTTPDriver) Update(ctx context.Context, id string, spec map[string]interface{}) error {
	return nil
}

func (d *HTTPDriver) Delete(ctx context.Context, id string) error {
	return nil
}

func (d *HTTPDriver) GetStatus(ctx context.Context, id string) (*ResourceStatus, error) {
	return &ResourceStatus{
		ID:        id,
		Status:    "ready",
		Health:    "healthy",
		UpdatedAt: time.Now(),
	}, nil
}

func (d *HTTPDriver) GetOutputs(ctx context.Context, id string) (map[string]interface{}, error) {
	return map[string]interface{}{"id": id}, nil
}

func (d *HTTPDriver) Validate(ctx context.Context, spec map[string]interface{}) error {
	return nil
}

// DatabaseDriver handles database resource provisioning
type DatabaseDriver struct {
	name     string
	provider string // postgres, mysql, mongodb
}

// NewDatabaseDriver creates a new database driver
func NewDatabaseDriver(name, provider string) *DatabaseDriver {
	return &DatabaseDriver{name: name, provider: provider}
}

func (d *DatabaseDriver) Name() string         { return d.name }
func (d *DatabaseDriver) ResourceType() string { return "database" }

func (d *DatabaseDriver) Provision(ctx context.Context, spec map[string]interface{}) (*ProvisionResult, error) {
	dbName, _ := spec["name"].(string)
	if dbName == "" {
		return nil, fmt.Errorf("database name is required")
	}

	// In production, this would create the actual database
	return &ProvisionResult{
		ID:        fmt.Sprintf("db-%s-%d", dbName, time.Now().Unix()),
		Status:    "ready",
		CreatedAt: time.Now(),
		Outputs: map[string]interface{}{
			"host":     fmt.Sprintf("%s.database.local", dbName),
			"port":     5432,
			"database": dbName,
			"provider": d.provider,
		},
	}, nil
}

func (d *DatabaseDriver) Update(ctx context.Context, id string, spec map[string]interface{}) error {
	return nil
}

func (d *DatabaseDriver) Delete(ctx context.Context, id string) error {
	return nil
}

func (d *DatabaseDriver) GetStatus(ctx context.Context, id string) (*ResourceStatus, error) {
	return &ResourceStatus{
		ID:        id,
		Status:    "ready",
		Health:    "healthy",
		UpdatedAt: time.Now(),
	}, nil
}

func (d *DatabaseDriver) GetOutputs(ctx context.Context, id string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"id":       id,
		"provider": d.provider,
	}, nil
}

func (d *DatabaseDriver) Validate(ctx context.Context, spec map[string]interface{}) error {
	if _, ok := spec["name"]; !ok {
		return fmt.Errorf("database name is required")
	}
	return nil
}

// CacheDriver handles cache resource provisioning (Redis, Memcached)
type CacheDriver struct {
	name     string
	provider string
}

// NewCacheDriver creates a new cache driver
func NewCacheDriver(name, provider string) *CacheDriver {
	return &CacheDriver{name: name, provider: provider}
}

func (d *CacheDriver) Name() string         { return d.name }
func (d *CacheDriver) ResourceType() string { return "cache" }

func (d *CacheDriver) Provision(ctx context.Context, spec map[string]interface{}) (*ProvisionResult, error) {
	cacheName, _ := spec["name"].(string)
	if cacheName == "" {
		return nil, fmt.Errorf("cache name is required")
	}

	return &ProvisionResult{
		ID:        fmt.Sprintf("cache-%s-%d", cacheName, time.Now().Unix()),
		Status:    "ready",
		CreatedAt: time.Now(),
		Outputs: map[string]interface{}{
			"host":     fmt.Sprintf("%s.cache.local", cacheName),
			"port":     6379,
			"provider": d.provider,
		},
	}, nil
}

func (d *CacheDriver) Update(ctx context.Context, id string, spec map[string]interface{}) error {
	return nil
}

func (d *CacheDriver) Delete(ctx context.Context, id string) error {
	return nil
}

func (d *CacheDriver) GetStatus(ctx context.Context, id string) (*ResourceStatus, error) {
	return &ResourceStatus{
		ID:        id,
		Status:    "ready",
		Health:    "healthy",
		UpdatedAt: time.Now(),
	}, nil
}

func (d *CacheDriver) GetOutputs(ctx context.Context, id string) (map[string]interface{}, error) {
	return map[string]interface{}{"id": id, "provider": d.provider}, nil
}

func (d *CacheDriver) Validate(ctx context.Context, spec map[string]interface{}) error {
	if _, ok := spec["name"]; !ok {
		return fmt.Errorf("cache name is required")
	}
	return nil
}

// StorageDriver handles storage resource provisioning (S3, GCS, Azure Blob)
type StorageDriver struct {
	name     string
	provider string
}

// NewStorageDriver creates a new storage driver
func NewStorageDriver(name, provider string) *StorageDriver {
	return &StorageDriver{name: name, provider: provider}
}

func (d *StorageDriver) Name() string         { return d.name }
func (d *StorageDriver) ResourceType() string { return "storage" }

func (d *StorageDriver) Provision(ctx context.Context, spec map[string]interface{}) (*ProvisionResult, error) {
	bucketName, _ := spec["name"].(string)
	if bucketName == "" {
		return nil, fmt.Errorf("bucket name is required")
	}

	return &ProvisionResult{
		ID:        fmt.Sprintf("bucket-%s-%d", bucketName, time.Now().Unix()),
		Status:    "ready",
		CreatedAt: time.Now(),
		Outputs: map[string]interface{}{
			"bucket":   bucketName,
			"endpoint": fmt.Sprintf("https://%s.storage.local", bucketName),
			"provider": d.provider,
		},
	}, nil
}

func (d *StorageDriver) Update(ctx context.Context, id string, spec map[string]interface{}) error {
	return nil
}

func (d *StorageDriver) Delete(ctx context.Context, id string) error {
	return nil
}

func (d *StorageDriver) GetStatus(ctx context.Context, id string) (*ResourceStatus, error) {
	return &ResourceStatus{
		ID:        id,
		Status:    "ready",
		Health:    "healthy",
		UpdatedAt: time.Now(),
	}, nil
}

func (d *StorageDriver) GetOutputs(ctx context.Context, id string) (map[string]interface{}, error) {
	return map[string]interface{}{"id": id, "provider": d.provider}, nil
}

func (d *StorageDriver) Validate(ctx context.Context, spec map[string]interface{}) error {
	if _, ok := spec["name"]; !ok {
		return fmt.Errorf("bucket name is required")
	}
	return nil
}

// CustomDriver allows defining drivers via configuration
type CustomDriver struct {
	name         string
	resourceType string
	schema       map[string]interface{}
	hooks        DriverHooks
}

// DriverHooks defines hooks for custom driver lifecycle events
type DriverHooks struct {
	OnProvision func(ctx context.Context, spec map[string]interface{}) (*ProvisionResult, error)
	OnUpdate    func(ctx context.Context, id string, spec map[string]interface{}) error
	OnDelete    func(ctx context.Context, id string) error
	OnGetStatus func(ctx context.Context, id string) (*ResourceStatus, error)
	OnValidate  func(ctx context.Context, spec map[string]interface{}) error
}

// NewCustomDriver creates a new custom driver
func NewCustomDriver(name, resourceType string, schema map[string]interface{}, hooks DriverHooks) *CustomDriver {
	return &CustomDriver{
		name:         name,
		resourceType: resourceType,
		schema:       schema,
		hooks:        hooks,
	}
}

func (d *CustomDriver) Name() string         { return d.name }
func (d *CustomDriver) ResourceType() string { return d.resourceType }

func (d *CustomDriver) Provision(ctx context.Context, spec map[string]interface{}) (*ProvisionResult, error) {
	if d.hooks.OnProvision != nil {
		return d.hooks.OnProvision(ctx, spec)
	}
	return &ProvisionResult{
		ID:        fmt.Sprintf("%s-%d", d.resourceType, time.Now().UnixNano()),
		Status:    "ready",
		CreatedAt: time.Now(),
	}, nil
}

func (d *CustomDriver) Update(ctx context.Context, id string, spec map[string]interface{}) error {
	if d.hooks.OnUpdate != nil {
		return d.hooks.OnUpdate(ctx, id, spec)
	}
	return nil
}

func (d *CustomDriver) Delete(ctx context.Context, id string) error {
	if d.hooks.OnDelete != nil {
		return d.hooks.OnDelete(ctx, id)
	}
	return nil
}

func (d *CustomDriver) GetStatus(ctx context.Context, id string) (*ResourceStatus, error) {
	if d.hooks.OnGetStatus != nil {
		return d.hooks.OnGetStatus(ctx, id)
	}
	return &ResourceStatus{ID: id, Status: "ready", UpdatedAt: time.Now()}, nil
}

func (d *CustomDriver) GetOutputs(ctx context.Context, id string) (map[string]interface{}, error) {
	return map[string]interface{}{"id": id}, nil
}

func (d *CustomDriver) Validate(ctx context.Context, spec map[string]interface{}) error {
	if d.hooks.OnValidate != nil {
		return d.hooks.OnValidate(ctx, spec)
	}
	return d.validateAgainstSchema(spec)
}

func (d *CustomDriver) validateAgainstSchema(spec map[string]interface{}) error {
	if d.schema == nil {
		return nil
	}

	required, _ := d.schema["required"].([]interface{})
	for _, r := range required {
		field, _ := r.(string)
		if _, ok := spec[field]; !ok {
			return fmt.Errorf("required field %s is missing", field)
		}
	}

	return nil
}

// RegisterBuiltinDrivers registers all built-in drivers with the registry
func RegisterBuiltinDrivers(registry *Registry) {
	registry.Register(NewDatabaseDriver("postgres", "postgres"))
	registry.Register(NewDatabaseDriver("mysql", "mysql"))
	registry.Register(NewCacheDriver("redis", "redis"))
	registry.Register(NewStorageDriver("s3", "aws"))
}

// DriverFromJSON creates a custom driver from JSON configuration
func DriverFromJSON(data []byte) (*CustomDriver, error) {
	var config struct {
		Name         string                 `json:"name"`
		ResourceType string                 `json:"resourceType"`
		Schema       map[string]interface{} `json:"schema"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("invalid driver configuration: %w", err)
	}

	if config.Name == "" || config.ResourceType == "" {
		return nil, fmt.Errorf("driver name and resourceType are required")
	}

	return NewCustomDriver(config.Name, config.ResourceType, config.Schema, DriverHooks{}), nil
}
