package types

import (
	"strings"
	"testing"
)

func TestEnvironment_Validate(t *testing.T) {
	tests := []struct {
		name    string
		env     Environment
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid development environment",
			env: Environment{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Environment",
				Metadata: Metadata{
					Name: "dev",
				},
				Spec: EnvironmentSpec{
					Type:        EnvironmentDev,
					PlatformRef: "my-platform",
					Overrides: EnvironmentOverrides{
						Tags: map[string]string{
							"environment": "development",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid staging environment",
			env: Environment{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Environment",
				Metadata: Metadata{
					Name: "staging",
				},
				Spec: EnvironmentSpec{
					Type:        EnvironmentStaging,
					PlatformRef: "my-platform",
					Overrides: EnvironmentOverrides{
						Infrastructure: map[string]interface{}{
							"nodeCount": 3,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid production environment with promotion",
			env: Environment{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Environment",
				Metadata: Metadata{
					Name: "production",
				},
				Spec: EnvironmentSpec{
					Type:        EnvironmentProduction,
					PlatformRef: "my-platform",
					Promotion: &PromotionConfig{
						Auto:             false,
						PromotesTo:       "production-backup",
						RequiresApproval: true,
						Approvers: []string{
							"admin@example.com",
							"lead-engineer@example.com",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "environment with all overrides",
			env: Environment{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Environment",
				Metadata: Metadata{
					Name: "test-env",
				},
				Spec: EnvironmentSpec{
					Type:        EnvironmentDev,
					PlatformRef: "platform",
					Overrides: EnvironmentOverrides{
						Infrastructure: map[string]interface{}{
							"nodeCount": 2,
						},
						Orchestrator: map[string]interface{}{
							"syncInterval": "5m",
						},
						Observability: map[string]interface{}{
							"retention": "7d",
						},
						DevEx: map[string]interface{}{
							"enabled": true,
						},
						Pipeline: map[string]interface{}{
							"workers": 4,
						},
						Global: map[string]interface{}{
							"region": "us-east-1",
						},
						Tags: map[string]string{
							"cost-center": "engineering",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing apiVersion",
			env: Environment{
				Kind: "Environment",
				Metadata: Metadata{
					Name: "test",
				},
				Spec: EnvironmentSpec{
					Type:        EnvironmentDev,
					PlatformRef: "platform",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid kind",
			env: Environment{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "InvalidKind",
				Metadata: Metadata{
					Name: "test",
				},
				Spec: EnvironmentSpec{
					Type:        EnvironmentDev,
					PlatformRef: "platform",
				},
			},
			wantErr: true,
		},
		{
			name: "missing name",
			env: Environment{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Environment",
				Metadata:   Metadata{},
				Spec: EnvironmentSpec{
					Type:        EnvironmentDev,
					PlatformRef: "platform",
				},
			},
			wantErr: true,
		},
		{
			name: "name too long",
			env: Environment{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Environment",
				Metadata: Metadata{
					Name: strings.Repeat("a", 64), // 64 chars
				},
				Spec: EnvironmentSpec{
					Type:        EnvironmentDev,
					PlatformRef: "platform",
				},
			},
			wantErr: true,
			errMsg:  "63 characters or less",
		},
		{
			name: "invalid name format (uppercase)",
			env: Environment{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Environment",
				Metadata: Metadata{
					Name: "Production",
				},
				Spec: EnvironmentSpec{
					Type:        EnvironmentProduction,
					PlatformRef: "platform",
				},
			},
			wantErr: true,
			errMsg:  "lowercase alphanumeric",
		},
		{
			name: "invalid name format (underscore)",
			env: Environment{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Environment",
				Metadata: Metadata{
					Name: "dev_environment",
				},
				Spec: EnvironmentSpec{
					Type:        EnvironmentDev,
					PlatformRef: "platform",
				},
			},
			wantErr: true,
			errMsg:  "lowercase alphanumeric",
		},
		{
			name: "missing environment type",
			env: Environment{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Environment",
				Metadata: Metadata{
					Name: "test",
				},
				Spec: EnvironmentSpec{
					PlatformRef: "platform",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid environment type",
			env: Environment{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Environment",
				Metadata: Metadata{
					Name: "test",
				},
				Spec: EnvironmentSpec{
					Type:        "invalid-type",
					PlatformRef: "platform",
				},
			},
			wantErr: true,
		},
		{
			name: "missing platformRef",
			env: Environment{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Environment",
				Metadata: Metadata{
					Name: "test",
				},
				Spec: EnvironmentSpec{
					Type: EnvironmentDev,
				},
			},
			wantErr: true,
		},
		{
			name: "platformRef too long",
			env: Environment{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Environment",
				Metadata: Metadata{
					Name: "test",
				},
				Spec: EnvironmentSpec{
					Type:        EnvironmentDev,
					PlatformRef: strings.Repeat("a", 254), // 254 chars
				},
			},
			wantErr: true,
			errMsg:  "253 characters or less",
		},
		{
			name: "too many approvers",
			env: Environment{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Environment",
				Metadata: Metadata{
					Name: "prod",
				},
				Spec: EnvironmentSpec{
					Type:        EnvironmentProduction,
					PlatformRef: "platform",
					Promotion: &PromotionConfig{
						Approvers: make([]string, 51), // 51 approvers
					},
				},
			},
			wantErr: true,
			errMsg:  "too many approvers",
		},
		{
			name: "approver name too long",
			env: Environment{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Environment",
				Metadata: Metadata{
					Name: "prod",
				},
				Spec: EnvironmentSpec{
					Type:        EnvironmentProduction,
					PlatformRef: "platform",
					Promotion: &PromotionConfig{
						Approvers: []string{
							strings.Repeat("a", 129), // 129 chars
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "128 characters or less",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.env.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Environment.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Environment.Validate() error = %v, want error containing %v", err, tt.errMsg)
			}
		})
	}
}

func TestIsValidEnvironmentType(t *testing.T) {
	tests := []struct {
		name      string
		envType   EnvironmentType
		wantValid bool
	}{
		{"development is valid", EnvironmentDev, true},
		{"staging is valid", EnvironmentStaging, true},
		{"production is valid", EnvironmentProduction, true},
		{"invalid type", "invalid", false},
		{"empty type", "", false},
		{"testing is invalid", "testing", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidEnvironmentType(tt.envType); got != tt.wantValid {
				t.Errorf("IsValidEnvironmentType(%v) = %v, want %v", tt.envType, got, tt.wantValid)
			}
		})
	}
}

func TestPromotionConfig(t *testing.T) {
	tests := []struct {
		name         string
		config       *PromotionConfig
		wantAuto     bool
		wantApproval bool
	}{
		{
			name: "auto promotion without approval",
			config: &PromotionConfig{
				Auto:             true,
				PromotesTo:       "staging",
				RequiresApproval: false,
			},
			wantAuto:     true,
			wantApproval: false,
		},
		{
			name: "manual promotion with approval",
			config: &PromotionConfig{
				Auto:             false,
				PromotesTo:       "production",
				RequiresApproval: true,
				Approvers: []string{
					"admin1",
					"admin2",
				},
			},
			wantAuto:     false,
			wantApproval: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.Auto != tt.wantAuto {
				t.Errorf("PromotionConfig.Auto = %v, want %v", tt.config.Auto, tt.wantAuto)
			}
			if tt.config.RequiresApproval != tt.wantApproval {
				t.Errorf("PromotionConfig.RequiresApproval = %v, want %v", tt.config.RequiresApproval, tt.wantApproval)
			}
		})
	}
}

func TestEnvironmentOverrides(t *testing.T) {
	overrides := EnvironmentOverrides{
		Infrastructure: map[string]interface{}{
			"nodeCount": 3,
			"nodeType":  "t3.large",
		},
		Orchestrator: map[string]interface{}{
			"syncInterval": "5m",
		},
		Tags: map[string]string{
			"environment": "staging",
			"team":        "platform",
		},
	}

	if overrides.Infrastructure == nil {
		t.Error("Infrastructure overrides should not be nil")
	}
	if overrides.Orchestrator == nil {
		t.Error("Orchestrator overrides should not be nil")
	}
	if len(overrides.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(overrides.Tags))
	}
}

func TestEnvironmentStatus(t *testing.T) {
	env := Environment{
		Status: EnvironmentStatus{
			Phase:   PhaseReady,
			Message: "Environment configured successfully",
		},
	}

	if env.Status.Phase != PhaseReady {
		t.Errorf("Expected phase %s, got %s", PhaseReady, env.Status.Phase)
	}

	if env.Status.Message == "" {
		t.Error("Expected non-empty status message")
	}
}
