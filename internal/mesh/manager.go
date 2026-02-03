package mesh

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Manager handles service mesh operations
type Manager struct {
	meshes          map[string]*ServiceMesh
	virtualServices map[string]*VirtualService
	destRules       map[string]*DestinationRule
	mu              sync.RWMutex
}

// NewManager creates a new service mesh manager
func NewManager() *Manager {
	m := &Manager{
		meshes:          make(map[string]*ServiceMesh),
		virtualServices: make(map[string]*VirtualService),
		destRules:       make(map[string]*DestinationRule),
	}

	// Initialize with a default mesh configuration
	m.initializeDefaultMesh()

	return m
}

// initializeDefaultMesh creates a default mesh configuration
func (m *Manager) initializeDefaultMesh() {
	defaultMesh := &ServiceMesh{
		APIVersion: "platformfoundry.io/v1",
		Kind:       "ServiceMesh",
		Metadata: MeshMetadata{
			Name:      "production-mesh",
			Namespace: "istio-system",
			CreatedAt: time.Now(),
		},
		Spec: ServiceMeshSpec{
			Provider: MeshProviderIstio,
			MTLS: MTLSConfig{
				Mode:            MTLSModeStrict,
				MinTLSVersion:   "TLSv1_2",
				WorkloadCertTTL: "24h",
			},
			Traffic: TrafficConfig{
				Retries: RetryConfig{
					Attempts:      3,
					PerTryTimeout: "2s",
					RetryOn:       []string{"5xx", "gateway-error", "connect-failure"},
				},
				CircuitBreaker: CircuitBreakerConfig{
					ConsecutiveErrors:  5,
					Interval:           "30s",
					BaseEjectionTime:   "30s",
					MaxEjectionPercent: 50,
				},
				Timeout: TimeoutConfig{
					Request: "15s",
					Idle:    "1h",
				},
				LoadBalancing: LoadBalancingConfig{
					Algorithm:     "round_robin",
					LocalityAware: true,
				},
			},
			Observability: ObservabilityConfig{
				Tracing: TracingConfig{
					Enabled:  true,
					Sampling: 1.0, // 1%
					Provider: "tempo",
				},
				Metrics: MetricsConfig{
					Enabled:        true,
					PrometheusPort: 15090,
				},
				Logging: LoggingConfig{
					Enabled: true,
					Format:  "json",
					Level:   "info",
				},
			},
		},
		Status: &ServiceMeshStatus{
			Phase:          MeshPhaseHealthy,
			LastUpdated:    time.Now(),
			Services:       25,
			ProxiesHealthy: 48,
			ProxiesTotal:   50,
			Conditions: []MeshCondition{
				{
					Type:               "Ready",
					Status:             "True",
					LastTransitionTime: time.Now(),
					Reason:             "MeshHealthy",
					Message:            "All mesh components are healthy",
				},
			},
		},
	}

	m.meshes[defaultMesh.Metadata.Name] = defaultMesh
}

// RegisterMesh registers a service mesh configuration
func (m *Manager) RegisterMesh(ctx context.Context, mesh *ServiceMesh) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if mesh.Metadata.Name == "" {
		return fmt.Errorf("mesh name is required")
	}

	if mesh.APIVersion == "" {
		mesh.APIVersion = "platformfoundry.io/v1"
	}
	if mesh.Kind == "" {
		mesh.Kind = "ServiceMesh"
	}

	mesh.Metadata.CreatedAt = time.Now()

	// Initialize status
	mesh.Status = &ServiceMeshStatus{
		Phase:       MeshPhaseInstalling,
		LastUpdated: time.Now(),
		Conditions: []MeshCondition{
			{
				Type:               "Installing",
				Status:             "True",
				LastTransitionTime: time.Now(),
				Reason:             "MeshRegistered",
				Message:            "Mesh configuration registered",
			},
		},
	}

	m.meshes[mesh.Metadata.Name] = mesh
	return nil
}

// GetMesh retrieves a mesh by name
func (m *Manager) GetMesh(name string) (*ServiceMesh, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mesh, ok := m.meshes[name]
	if !ok {
		return nil, fmt.Errorf("mesh not found: %s", name)
	}
	return mesh, nil
}

// ListMeshes returns all meshes
func (m *Manager) ListMeshes() []*ServiceMesh {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*ServiceMesh, 0, len(m.meshes))
	for _, mesh := range m.meshes {
		result = append(result, mesh)
	}
	return result
}

// DeleteMesh removes a mesh
func (m *Manager) DeleteMesh(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.meshes[name]; !ok {
		return fmt.Errorf("mesh not found: %s", name)
	}

	delete(m.meshes, name)
	return nil
}

// ApplyVirtualService applies a virtual service configuration
func (m *Manager) ApplyVirtualService(ctx context.Context, vs *VirtualService) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if vs.Metadata.Name == "" {
		return fmt.Errorf("virtual service name is required")
	}

	if vs.APIVersion == "" {
		vs.APIVersion = "networking.istio.io/v1beta1"
	}
	if vs.Kind == "" {
		vs.Kind = "VirtualService"
	}

	vs.Metadata.CreatedAt = time.Now()

	m.virtualServices[vs.Metadata.Name] = vs
	return nil
}

// GetVirtualService retrieves a virtual service
func (m *Manager) GetVirtualService(name string) (*VirtualService, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	vs, ok := m.virtualServices[name]
	if !ok {
		return nil, fmt.Errorf("virtual service not found: %s", name)
	}
	return vs, nil
}

// ListVirtualServices returns all virtual services
func (m *Manager) ListVirtualServices() []*VirtualService {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*VirtualService, 0, len(m.virtualServices))
	for _, vs := range m.virtualServices {
		result = append(result, vs)
	}
	return result
}

// DeleteVirtualService removes a virtual service
func (m *Manager) DeleteVirtualService(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.virtualServices[name]; !ok {
		return fmt.Errorf("virtual service not found: %s", name)
	}

	delete(m.virtualServices, name)
	return nil
}

// ApplyDestinationRule applies a destination rule
func (m *Manager) ApplyDestinationRule(ctx context.Context, dr *DestinationRule) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if dr.Metadata.Name == "" {
		return fmt.Errorf("destination rule name is required")
	}

	if dr.APIVersion == "" {
		dr.APIVersion = "networking.istio.io/v1beta1"
	}
	if dr.Kind == "" {
		dr.Kind = "DestinationRule"
	}

	dr.Metadata.CreatedAt = time.Now()

	m.destRules[dr.Metadata.Name] = dr
	return nil
}

// GetDestinationRule retrieves a destination rule
func (m *Manager) GetDestinationRule(name string) (*DestinationRule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	dr, ok := m.destRules[name]
	if !ok {
		return nil, fmt.Errorf("destination rule not found: %s", name)
	}
	return dr, nil
}

// ListDestinationRules returns all destination rules
func (m *Manager) ListDestinationRules() []*DestinationRule {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*DestinationRule, 0, len(m.destRules))
	for _, dr := range m.destRules {
		result = append(result, dr)
	}
	return result
}

// DeleteDestinationRule removes a destination rule
func (m *Manager) DeleteDestinationRule(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.destRules[name]; !ok {
		return fmt.Errorf("destination rule not found: %s", name)
	}

	delete(m.destRules, name)
	return nil
}

// ConfigureTrafficSplit configures traffic split between versions
func (m *Manager) ConfigureTrafficSplit(ctx context.Context, service string, weights map[string]int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate weights sum to 100
	total := 0
	for _, w := range weights {
		total += w
	}
	if total != 100 {
		return fmt.Errorf("weights must sum to 100, got %d", total)
	}

	// Create or update virtual service
	routes := make([]HTTPRouteDestination, 0)
	for subset, weight := range weights {
		routes = append(routes, HTTPRouteDestination{
			Destination: Destination{
				Host:   service,
				Subset: subset,
			},
			Weight: weight,
		})
	}

	vs := &VirtualService{
		APIVersion: "networking.istio.io/v1beta1",
		Kind:       "VirtualService",
		Metadata: MeshMetadata{
			Name:      service + "-traffic-split",
			CreatedAt: time.Now(),
		},
		Spec: VirtualServiceSpec{
			Hosts: []string{service},
			HTTP: []HTTPRoute{
				{
					Name:  "traffic-split",
					Route: routes,
				},
			},
		},
	}

	m.virtualServices[vs.Metadata.Name] = vs
	return nil
}

// InjectFault injects a fault for testing
func (m *Manager) InjectFault(ctx context.Context, service string, fault FaultInjection) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	vs := &VirtualService{
		APIVersion: "networking.istio.io/v1beta1",
		Kind:       "VirtualService",
		Metadata: MeshMetadata{
			Name:      service + "-fault-injection",
			CreatedAt: time.Now(),
		},
		Spec: VirtualServiceSpec{
			Hosts: []string{service},
			HTTP: []HTTPRoute{
				{
					Name:  "fault-injection",
					Fault: &fault,
					Route: []HTTPRouteDestination{
						{
							Destination: Destination{
								Host: service,
							},
							Weight: 100,
						},
					},
				},
			},
		},
	}

	m.virtualServices[vs.Metadata.Name] = vs
	return nil
}

// RemoveFault removes fault injection
func (m *Manager) RemoveFault(ctx context.Context, service string) error {
	return m.DeleteVirtualService(service + "-fault-injection")
}

// GetMeshStatus returns current mesh status
func (m *Manager) GetMeshStatus(meshName string) (*ServiceMeshStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mesh, ok := m.meshes[meshName]
	if !ok {
		return nil, fmt.Errorf("mesh not found: %s", meshName)
	}

	return mesh.Status, nil
}

// UpdateMTLS updates mTLS configuration
func (m *Manager) UpdateMTLS(meshName string, mode MTLSMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	mesh, ok := m.meshes[meshName]
	if !ok {
		return fmt.Errorf("mesh not found: %s", meshName)
	}

	mesh.Spec.MTLS.Mode = mode
	return nil
}

// GetTrafficMetrics returns traffic metrics for a service (simulated)
func (m *Manager) GetTrafficMetrics(service string) map[string]interface{} {
	return map[string]interface{}{
		"requestsPerSecond": 125.5,
		"p50LatencyMs":      45.2,
		"p99LatencyMs":      230.5,
		"errorRate":         0.02,
		"successRate":       99.98,
		"activeConnections": 85,
		"bytesIn":           15234000,
		"bytesOut":          42156000,
	}
}

// GetServiceGraph returns service dependency graph (simulated)
func (m *Manager) GetServiceGraph() map[string][]string {
	return map[string][]string{
		"api-gateway":     {"auth-service", "user-service", "product-service"},
		"auth-service":    {"user-service", "redis"},
		"user-service":    {"postgres", "redis"},
		"product-service": {"postgres", "elasticsearch"},
		"order-service":   {"user-service", "product-service", "payment-service"},
		"payment-service": {"stripe-api"},
	}
}
