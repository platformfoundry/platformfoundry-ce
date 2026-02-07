package types

import (
	"strings"
	"testing"
	"time"
)

func TestService_Validate(t *testing.T) {
	tests := []struct {
		name    string
		service Service
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid service",
			service: Service{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Service",
				Metadata: Metadata{
					Name:         "user-api",
					Organization: "acme-corp",
				},
				Spec: ServiceSpec{
					Type: ServiceTypeMicroservice,
					Owner: ServiceOwner{
						Team:  "platform-team",
						Email: "team@example.com",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing api version",
			service: Service{
				Kind: "Service",
				Metadata: Metadata{
					Name: "user-api",
				},
				Spec: ServiceSpec{
					Type: ServiceTypeMicroservice,
					Owner: ServiceOwner{
						Team: "platform-team",
					},
				},
			},
			wantErr: true,
			errMsg:  "apiVersion",
		},
		{
			name: "wrong kind",
			service: Service{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "WrongKind",
				Metadata: Metadata{
					Name: "user-api",
				},
				Spec: ServiceSpec{
					Type: ServiceTypeMicroservice,
					Owner: ServiceOwner{
						Team: "platform-team",
					},
				},
			},
			wantErr: true,
			errMsg:  "kind",
		},
		{
			name: "missing name",
			service: Service{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Service",
				Metadata:   Metadata{},
				Spec: ServiceSpec{
					Type: ServiceTypeMicroservice,
					Owner: ServiceOwner{
						Team: "platform-team",
					},
				},
			},
			wantErr: true,
			errMsg:  "name",
		},
		{
			name: "invalid service name - too short",
			service: Service{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Service",
				Metadata: Metadata{
					Name: "a",
				},
				Spec: ServiceSpec{
					Type: ServiceTypeMicroservice,
					Owner: ServiceOwner{
						Team: "platform-team",
					},
				},
			},
			wantErr: true,
			errMsg:  "between 2 and 63 characters",
		},
		{
			name: "invalid service name - uppercase",
			service: Service{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Service",
				Metadata: Metadata{
					Name: "User-API",
				},
				Spec: ServiceSpec{
					Type: ServiceTypeMicroservice,
					Owner: ServiceOwner{
						Team: "platform-team",
					},
				},
			},
			wantErr: true,
			errMsg:  "lowercase alphanumeric",
		},
		{
			name: "invalid service type",
			service: Service{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Service",
				Metadata: Metadata{
					Name: "user-api",
				},
				Spec: ServiceSpec{
					Type: "invalid-type",
					Owner: ServiceOwner{
						Team: "platform-team",
					},
				},
			},
			wantErr: true,
			errMsg:  "invalid service type",
		},
		{
			name: "missing owner team",
			service: Service{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Service",
				Metadata: Metadata{
					Name: "user-api",
				},
				Spec: ServiceSpec{
					Type:  ServiceTypeMicroservice,
					Owner: ServiceOwner{},
				},
			},
			wantErr: true,
			errMsg:  "owner team is required",
		},
		{
			name: "invalid email",
			service: Service{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Service",
				Metadata: Metadata{
					Name: "user-api",
				},
				Spec: ServiceSpec{
					Type: ServiceTypeMicroservice,
					Owner: ServiceOwner{
						Team:  "platform-team",
						Email: "invalid-email",
					},
				},
			},
			wantErr: true,
			errMsg:  "invalid owner email",
		},
		{
			name: "too many dependencies",
			service: Service{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Service",
				Metadata: Metadata{
					Name: "user-api",
				},
				Spec: ServiceSpec{
					Type: ServiceTypeMicroservice,
					Owner: ServiceOwner{
						Team: "platform-team",
					},
					Dependencies: make([]ServiceDependency, 101),
				},
			},
			wantErr: true,
			errMsg:  "too many dependencies",
		},
		{
			name: "invalid SLO availability",
			service: Service{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Service",
				Metadata: Metadata{
					Name: "user-api",
				},
				Spec: ServiceSpec{
					Type: ServiceTypeMicroservice,
					Owner: ServiceOwner{
						Team: "platform-team",
					},
					SLO: &SLOConfig{
						Availability: 150.0,
					},
				},
			},
			wantErr: true,
			errMsg:  "SLO availability must be between 0 and 100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.service.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Service.Validate() error = %v, should contain %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestService_GetFullName(t *testing.T) {
	tests := []struct {
		name     string
		service  Service
		expected string
	}{
		{
			name: "with organization",
			service: Service{
				Metadata: Metadata{
					Name:         "user-api",
					Organization: "acme-corp",
				},
			},
			expected: "acme-corp/user-api",
		},
		{
			name: "without organization",
			service: Service{
				Metadata: Metadata{
					Name: "user-api",
				},
			},
			expected: "user-api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.service.GetFullName()
			if got != tt.expected {
				t.Errorf("Service.GetFullName() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsValidServiceType(t *testing.T) {
	tests := []struct {
		name        string
		serviceType ServiceType
		want        bool
	}{
		{"microservice", ServiceTypeMicroservice, true},
		{"frontend", ServiceTypeFrontend, true},
		{"backend", ServiceTypeBackend, true},
		{"database", ServiceTypeDatabase, true},
		{"cache", ServiceTypeCache, true},
		{"queue", ServiceTypeQueue, true},
		{"api", ServiceTypeAPI, true},
		{"worker", ServiceTypeWorker, true},
		{"invalid", ServiceType("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidServiceType(tt.serviceType)
			if got != tt.want {
				t.Errorf("IsValidServiceType(%v) = %v, want %v", tt.serviceType, got, tt.want)
			}
		})
	}
}

func TestIsValidServiceState(t *testing.T) {
	tests := []struct {
		name  string
		state ServiceState
		want  bool
	}{
		{"draft", ServiceStateDraft, true},
		{"active", ServiceStateActive, true},
		{"deprecated", ServiceStateDeprecated, true},
		{"retired", ServiceStateRetired, true},
		{"running", ServiceStateRunning, true},
		{"stopped", ServiceStateStopped, true},
		{"failed", ServiceStateFailed, true},
		{"invalid", ServiceState("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidServiceState(tt.state)
			if got != tt.want {
				t.Errorf("IsValidServiceState(%v) = %v, want %v", tt.state, got, tt.want)
			}
		})
	}
}

func TestIsValidServiceHealth(t *testing.T) {
	tests := []struct {
		name   string
		health ServiceHealth
		want   bool
	}{
		{"healthy", ServiceHealthHealthy, true},
		{"degraded", ServiceHealthDegraded, true},
		{"down", ServiceHealthDown, true},
		{"unknown", ServiceHealthUnknown, true},
		{"invalid", ServiceHealth("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidServiceHealth(tt.health)
			if got != tt.want {
				t.Errorf("IsValidServiceHealth(%v) = %v, want %v", tt.health, got, tt.want)
			}
		})
	}
}

func TestService_ValidateWithCompleteSpec(t *testing.T) {
	now := time.Now()
	service := Service{
		APIVersion: "platformfoundry.io/v1",
		Kind:       "Service",
		Metadata: Metadata{
			Name:         "user-api",
			Organization: "acme-corp",
			Labels: map[string]string{
				"env": "production",
			},
			Annotations: map[string]string{
				"description": "User management API",
			},
		},
		Spec: ServiceSpec{
			Type:    ServiceTypeMicroservice,
			Runtime: "go-1.21",
			Owner: ServiceOwner{
				Team:  "platform-team",
				Email: "platform@example.com",
				Slack: "#platform",
			},
			Repository: &RepositoryConfig{
				URL:    "https://github.com/acme/user-api",
				Branch: "main",
				Path:   "/",
			},
			Dependencies: []ServiceDependency{
				{Name: "postgres-db", Type: "database", Required: true},
				{Name: "redis-cache", Type: "cache", Required: false},
			},
			Links: []ServiceLink{
				{Name: "Documentation", URL: "https://docs.example.com", Type: "docs"},
				{Name: "Dashboard", URL: "https://grafana.example.com", Type: "dashboard"},
			},
			Health: &HealthConfig{
				Endpoint:    "/health",
				Interval:    30 * time.Second,
				Timeout:     5 * time.Second,
				Retries:     3,
				StartPeriod: 10 * time.Second,
			},
			Resources: &ResourceRequirements{
				CPU:    "500m",
				Memory: "512Mi",
				Disk:   "10Gi",
			},
			SLO: &SLOConfig{
				Availability: 99.9,
				Latency: &LatencySLO{
					P50: 50 * time.Millisecond,
					P95: 120 * time.Millisecond,
					P99: 250 * time.Millisecond,
				},
				ErrorRate: 0.1,
			},
		},
		Status: ServiceStatus{
			State:        ServiceStateActive,
			Health:       ServiceHealthHealthy,
			LastDeployed: &now,
			Version:      "v1.2.3",
			Metrics: &ServiceMetrics{
				RequestRate: 1200,
				ErrorRate:   0.01,
				LatencyP50:  45,
				LatencyP95:  120,
				LatencyP99:  250,
				CPUUsage:    45,
				MemoryUsage: 536870912, // 512MB
			},
		},
	}

	err := service.Validate()
	if err != nil {
		t.Errorf("Service.Validate() with complete spec failed: %v", err)
	}
}

// Helper function to check if error message contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
