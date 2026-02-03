package multicloud

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/platformfoundry/pf-ce/pkg/types"
)

// Manager handles multi-cloud operations
type Manager struct {
	accounts  map[string]*types.CloudAccount
	providers map[types.CloudProvider]Provider
	resources map[string]*types.UnifiedResource
	mappings  []types.ResourceMapping
	mu        sync.RWMutex
}

// Provider interface for cloud provider operations
type Provider interface {
	Name() types.CloudProvider
	Connect(ctx context.Context, account *types.CloudAccount) error
	Disconnect(ctx context.Context) error

	// Resource operations
	CreateResource(ctx context.Context, spec types.UnifiedResourceSpec) (*types.UnifiedResourceStatus, error)
	UpdateResource(ctx context.Context, id string, spec types.UnifiedResourceSpec) (*types.UnifiedResourceStatus, error)
	DeleteResource(ctx context.Context, id string) error
	GetResource(ctx context.Context, id string) (*types.UnifiedResourceStatus, error)
	ListResources(ctx context.Context, resourceType types.UnifiedResourceType) ([]types.UnifiedResourceStatus, error)

	// Cost operations
	GetCosts(ctx context.Context, start, end time.Time) (*types.CloudCost, error)

	// Region operations
	ListRegions(ctx context.Context) ([]types.RegionStatus, error)
}

// NewManager creates a new multi-cloud manager
func NewManager() *Manager {
	m := &Manager{
		accounts:  make(map[string]*types.CloudAccount),
		providers: make(map[types.CloudProvider]Provider),
		resources: make(map[string]*types.UnifiedResource),
		mappings:  defaultResourceMappings(),
	}
	return m
}

// RegisterProvider registers a cloud provider implementation
func (m *Manager) RegisterProvider(provider Provider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.providers[provider.Name()] = provider
}

// AddAccount adds a cloud account
func (m *Manager) AddAccount(ctx context.Context, account *types.CloudAccount) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if account.Metadata.Name == "" {
		return fmt.Errorf("account name is required")
	}

	// Verify provider is registered
	provider, ok := m.providers[account.Spec.Provider]
	if !ok {
		return fmt.Errorf("provider %s is not registered", account.Spec.Provider)
	}

	// Connect to verify credentials
	if err := provider.Connect(ctx, account); err != nil {
		account.Status = &types.CloudAccountStatus{
			Connected: false,
			Message:   err.Error(),
		}
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Get available regions
	regions, _ := provider.ListRegions(ctx)

	now := time.Now()
	account.Status = &types.CloudAccountStatus{
		Connected:    true,
		LastVerified: &now,
		Regions:      regions,
	}

	m.accounts[account.Metadata.Name] = account
	return nil
}

// GetAccount retrieves an account by name
func (m *Manager) GetAccount(name string) (*types.CloudAccount, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	account, ok := m.accounts[name]
	if !ok {
		return nil, fmt.Errorf("account not found: %s", name)
	}
	return account, nil
}

// ListAccounts returns all registered accounts
func (m *Manager) ListAccounts() []*types.CloudAccount {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*types.CloudAccount, 0, len(m.accounts))
	for _, a := range m.accounts {
		result = append(result, a)
	}
	return result
}

// CreateResource creates a cloud-agnostic resource
func (m *Manager) CreateResource(ctx context.Context, resource *types.UnifiedResource) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get the account
	account, ok := m.accounts[resource.Spec.AccountRef]
	if !ok {
		return fmt.Errorf("account not found: %s", resource.Spec.AccountRef)
	}

	// Get the provider
	provider, ok := m.providers[account.Spec.Provider]
	if !ok {
		return fmt.Errorf("provider %s is not registered", account.Spec.Provider)
	}

	// Apply provider-specific overrides
	spec := resource.Spec
	if overrides, ok := spec.ProviderOverrides[account.Spec.Provider]; ok {
		for k, v := range overrides {
			spec.Config[k] = v
		}
	}

	// Create the resource
	status, err := provider.CreateResource(ctx, spec)
	if err != nil {
		resource.Status = &types.UnifiedResourceStatus{
			State: "failed",
		}
		return fmt.Errorf("failed to create resource: %w", err)
	}

	resource.Status = status
	m.resources[resource.Metadata.Name] = resource
	return nil
}

// GetResource retrieves a resource by name
func (m *Manager) GetResource(name string) (*types.UnifiedResource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	resource, ok := m.resources[name]
	if !ok {
		return nil, fmt.Errorf("resource not found: %s", name)
	}
	return resource, nil
}

// ListResources returns resources, optionally filtered
func (m *Manager) ListResources(accountRef string, resourceType types.UnifiedResourceType) []*types.UnifiedResource {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*types.UnifiedResource, 0)
	for _, r := range m.resources {
		if accountRef != "" && r.Spec.AccountRef != accountRef {
			continue
		}
		if resourceType != "" && r.Spec.Type != resourceType {
			continue
		}
		result = append(result, r)
	}
	return result
}

// DeleteResource removes a resource
func (m *Manager) DeleteResource(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	resource, ok := m.resources[name]
	if !ok {
		return fmt.Errorf("resource not found: %s", name)
	}

	// Get the account
	account, ok := m.accounts[resource.Spec.AccountRef]
	if !ok {
		return fmt.Errorf("account not found: %s", resource.Spec.AccountRef)
	}

	// Get the provider
	provider, ok := m.providers[account.Spec.Provider]
	if !ok {
		return fmt.Errorf("provider %s is not registered", account.Spec.Provider)
	}

	// Delete from provider
	if resource.Status != nil && resource.Status.ProviderID != "" {
		if err := provider.DeleteResource(ctx, resource.Status.ProviderID); err != nil {
			return fmt.Errorf("failed to delete resource: %w", err)
		}
	}

	delete(m.resources, name)
	return nil
}

// GetCosts retrieves costs across all accounts
func (m *Manager) GetCosts(ctx context.Context, start, end time.Time) ([]types.CloudCost, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	costs := make([]types.CloudCost, 0)

	for name, account := range m.accounts {
		provider, ok := m.providers[account.Spec.Provider]
		if !ok {
			continue
		}

		cost, err := provider.GetCosts(ctx, start, end)
		if err != nil {
			fmt.Printf("Warning: failed to get costs for %s: %v\n", name, err)
			continue
		}

		cost.Account = name
		costs = append(costs, *cost)
	}

	return costs, nil
}

// GetProviderMapping returns the provider-specific type for a unified resource
func (m *Manager) GetProviderMapping(resourceType types.UnifiedResourceType, provider types.CloudProvider) (string, error) {
	for _, mapping := range m.mappings {
		if mapping.UnifiedType == resourceType {
			if providerType, ok := mapping.ProviderTypes[provider]; ok {
				return providerType, nil
			}
		}
	}
	return "", fmt.Errorf("no mapping found for %s on %s", resourceType, provider)
}

// defaultResourceMappings returns default resource type mappings
func defaultResourceMappings() []types.ResourceMapping {
	return []types.ResourceMapping{
		{
			UnifiedType: types.UnifiedResourceVM,
			ProviderTypes: map[types.CloudProvider]string{
				types.CloudProviderAWS:   "ec2:instance",
				types.CloudProviderAzure: "compute:virtualMachine",
				types.CloudProviderGCP:   "compute:instance",
			},
			SizeMapping: map[string]types.ProviderSizes{
				"small":  {AWS: "t3.small", Azure: "Standard_B1s", GCP: "e2-small"},
				"medium": {AWS: "t3.medium", Azure: "Standard_B2s", GCP: "e2-medium"},
				"large":  {AWS: "t3.large", Azure: "Standard_D2s_v3", GCP: "e2-standard-2"},
			},
		},
		{
			UnifiedType: types.UnifiedResourceKubernetes,
			ProviderTypes: map[types.CloudProvider]string{
				types.CloudProviderAWS:   "eks:cluster",
				types.CloudProviderAzure: "aks:managedCluster",
				types.CloudProviderGCP:   "gke:cluster",
			},
		},
		{
			UnifiedType: types.UnifiedResourceRDBMS,
			ProviderTypes: map[types.CloudProvider]string{
				types.CloudProviderAWS:   "rds:instance",
				types.CloudProviderAzure: "sql:database",
				types.CloudProviderGCP:   "cloudsql:instance",
			},
		},
		{
			UnifiedType: types.UnifiedResourceObjectStorage,
			ProviderTypes: map[types.CloudProvider]string{
				types.CloudProviderAWS:   "s3:bucket",
				types.CloudProviderAzure: "storage:blobContainer",
				types.CloudProviderGCP:   "storage:bucket",
			},
		},
		{
			UnifiedType: types.UnifiedResourceLoadBalancer,
			ProviderTypes: map[types.CloudProvider]string{
				types.CloudProviderAWS:   "elb:loadBalancer",
				types.CloudProviderAzure: "network:loadBalancer",
				types.CloudProviderGCP:   "compute:urlMap",
			},
		},
		{
			UnifiedType: types.UnifiedResourceCache,
			ProviderTypes: map[types.CloudProvider]string{
				types.CloudProviderAWS:   "elasticache:cacheCluster",
				types.CloudProviderAzure: "cache:redis",
				types.CloudProviderGCP:   "memorystore:instance",
			},
		},
	}
}

// MockProvider implements Provider for testing
type MockProvider struct {
	name      types.CloudProvider
	connected bool
	resources map[string]*types.UnifiedResourceStatus
}

// NewMockProvider creates a mock provider
func NewMockProvider(name types.CloudProvider) *MockProvider {
	return &MockProvider{
		name:      name,
		resources: make(map[string]*types.UnifiedResourceStatus),
	}
}

func (p *MockProvider) Name() types.CloudProvider {
	return p.name
}

func (p *MockProvider) Connect(ctx context.Context, account *types.CloudAccount) error {
	p.connected = true
	return nil
}

func (p *MockProvider) Disconnect(ctx context.Context) error {
	p.connected = false
	return nil
}

func (p *MockProvider) CreateResource(ctx context.Context, spec types.UnifiedResourceSpec) (*types.UnifiedResourceStatus, error) {
	id := fmt.Sprintf("%s-%d", spec.Type, time.Now().UnixNano())
	now := time.Now()

	status := &types.UnifiedResourceStatus{
		State:      "running",
		ProviderID: id,
		CreatedAt:  &now,
		UpdatedAt:  &now,
		Endpoints: []types.Endpoint{
			{
				Type:    "public",
				Address: fmt.Sprintf("%s.%s.example.com", id, p.name),
			},
		},
	}

	p.resources[id] = status
	return status, nil
}

func (p *MockProvider) UpdateResource(ctx context.Context, id string, spec types.UnifiedResourceSpec) (*types.UnifiedResourceStatus, error) {
	status, ok := p.resources[id]
	if !ok {
		return nil, fmt.Errorf("resource not found: %s", id)
	}

	now := time.Now()
	status.UpdatedAt = &now
	return status, nil
}

func (p *MockProvider) DeleteResource(ctx context.Context, id string) error {
	delete(p.resources, id)
	return nil
}

func (p *MockProvider) GetResource(ctx context.Context, id string) (*types.UnifiedResourceStatus, error) {
	status, ok := p.resources[id]
	if !ok {
		return nil, fmt.Errorf("resource not found: %s", id)
	}
	return status, nil
}

func (p *MockProvider) ListResources(ctx context.Context, resourceType types.UnifiedResourceType) ([]types.UnifiedResourceStatus, error) {
	result := make([]types.UnifiedResourceStatus, 0)
	for _, status := range p.resources {
		result = append(result, *status)
	}
	return result, nil
}

func (p *MockProvider) GetCosts(ctx context.Context, start, end time.Time) (*types.CloudCost, error) {
	return &types.CloudCost{
		Provider: p.name,
		Period:   fmt.Sprintf("%s to %s", start.Format("2006-01-02"), end.Format("2006-01-02")),
		Total:    1250.50,
		Currency: "USD",
		ByService: map[string]float64{
			"compute":  650.00,
			"storage":  200.50,
			"network":  150.00,
			"database": 250.00,
		},
	}, nil
}

func (p *MockProvider) ListRegions(ctx context.Context) ([]types.RegionStatus, error) {
	switch p.name {
	case types.CloudProviderAWS:
		return []types.RegionStatus{
			{Name: "us-east-1", Available: true},
			{Name: "us-west-2", Available: true},
			{Name: "eu-west-1", Available: true},
		}, nil
	case types.CloudProviderAzure:
		return []types.RegionStatus{
			{Name: "eastus", Available: true},
			{Name: "westus2", Available: true},
			{Name: "westeurope", Available: true},
		}, nil
	case types.CloudProviderGCP:
		return []types.RegionStatus{
			{Name: "us-central1", Available: true},
			{Name: "us-west1", Available: true},
			{Name: "europe-west1", Available: true},
		}, nil
	default:
		return []types.RegionStatus{}, nil
	}
}
