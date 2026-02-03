package types

import (
	"testing"
)

func TestDevEx_Validate(t *testing.T) {
	tests := []struct {
		name    string
		devex   DevEx
		wantErr bool
		errType error
	}{
		{
			name: "valid devex with Backstage",
			devex: DevEx{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "DevEx",
				Metadata: Metadata{
					Name: "test-devex",
				},
				Spec: DevExSpec{
					Provider:   "backstage",
					ClusterRef: "my-cluster",
					Portal: &PortalConfig{
						Features: []PortalFeature{
							{Name: "catalog", Enabled: true},
							{Name: "templates", Enabled: true},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid devex with Port",
			devex: DevEx{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "DevEx",
				Metadata: Metadata{
					Name: "port-devex",
				},
				Spec: DevExSpec{
					Provider:   "port",
					ClusterRef: "my-cluster",
				},
			},
			wantErr: false,
		},
		{
			name: "valid devex with intelligent generation",
			devex: DevEx{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "DevEx",
				Metadata: Metadata{
					Name: "ai-devex",
				},
				Spec: DevExSpec{
					Provider:   "backstage",
					ClusterRef: "my-cluster",
					IntelligentGeneration: &IntelligentGenerationConfig{
						Enabled:  true,
						Strategy: "llm",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid devex with customization",
			devex: DevEx{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "DevEx",
				Metadata: Metadata{
					Name: "custom-devex",
				},
				Spec: DevExSpec{
					Provider:   "backstage",
					ClusterRef: "my-cluster",
					Customization: &CustomizationConfig{
						Branding: &BrandingConfig{
							Name: "Acme Platform",
							Logo: "https://example.com/logo.png",
						},
						Theme: &ThemeConfig{
							PrimaryColor: "#007bff",
							Mode:         "dark",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid devex with integrations",
			devex: DevEx{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "DevEx",
				Metadata: Metadata{
					Name: "integrated-devex",
				},
				Spec: DevExSpec{
					Provider:   "backstage",
					ClusterRef: "my-cluster",
					Portal: &PortalConfig{
						Integrations: []Integration{
							{
								Type:    "github",
								Enabled: true,
								Config: map[string]interface{}{
									"org": "my-org",
								},
							},
							{
								Type:    "argocd",
								Enabled: true,
								Config: map[string]interface{}{
									"url": "https://argocd.example.com",
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing apiVersion",
			devex: DevEx{
				Kind: "DevEx",
				Metadata: Metadata{
					Name: "test",
				},
				Spec: DevExSpec{
					Provider:   "backstage",
					ClusterRef: "my-cluster",
				},
			},
			wantErr: true,
			errType: ErrMissingAPIVersion,
		},
		{
			name: "invalid kind",
			devex: DevEx{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "InvalidKind",
				Metadata: Metadata{
					Name: "test",
				},
				Spec: DevExSpec{
					Provider:   "backstage",
					ClusterRef: "my-cluster",
				},
			},
			wantErr: true,
			errType: ErrInvalidKind,
		},
		{
			name: "missing name",
			devex: DevEx{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "DevEx",
				Metadata:   Metadata{},
				Spec: DevExSpec{
					Provider:   "backstage",
					ClusterRef: "my-cluster",
				},
			},
			wantErr: true,
			errType: ErrMissingName,
		},
		{
			name: "missing provider",
			devex: DevEx{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "DevEx",
				Metadata: Metadata{
					Name: "test",
				},
				Spec: DevExSpec{
					ClusterRef: "my-cluster",
				},
			},
			wantErr: true,
			errType: ErrInvalidProvider,
		},
		{
			name: "missing clusterRef",
			devex: DevEx{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "DevEx",
				Metadata: Metadata{
					Name: "test",
				},
				Spec: DevExSpec{
					Provider: "backstage",
				},
			},
			wantErr: true,
			errType: ErrMissingClusterRef,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.devex.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("DevEx.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != tt.errType {
				t.Errorf("DevEx.Validate() error = %v, want %v", err, tt.errType)
			}
		})
	}
}

func TestIntelligentGenerationConfig(t *testing.T) {
	tests := []struct {
		name         string
		config       *IntelligentGenerationConfig
		wantEnabled  bool
		wantStrategy string
	}{
		{
			name: "rule-based strategy",
			config: &IntelligentGenerationConfig{
				Enabled:  true,
				Strategy: "rule-based",
			},
			wantEnabled:  true,
			wantStrategy: "rule-based",
		},
		{
			name: "similarity strategy",
			config: &IntelligentGenerationConfig{
				Enabled:  true,
				Strategy: "similarity",
			},
			wantEnabled:  true,
			wantStrategy: "similarity",
		},
		{
			name: "llm strategy",
			config: &IntelligentGenerationConfig{
				Enabled:  true,
				Strategy: "llm",
			},
			wantEnabled:  true,
			wantStrategy: "llm",
		},
		{
			name: "disabled",
			config: &IntelligentGenerationConfig{
				Enabled: false,
			},
			wantEnabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.Enabled != tt.wantEnabled {
				t.Errorf("IntelligentGenerationConfig.Enabled = %v, want %v", tt.config.Enabled, tt.wantEnabled)
			}
			if tt.wantEnabled && tt.config.Strategy != tt.wantStrategy {
				t.Errorf("IntelligentGenerationConfig.Strategy = %v, want %v", tt.config.Strategy, tt.wantStrategy)
			}
		})
	}
}

func TestPortalFeature(t *testing.T) {
	tests := []struct {
		name        string
		feature     PortalFeature
		wantName    string
		wantEnabled bool
	}{
		{
			name:        "catalog enabled",
			feature:     PortalFeature{Name: "catalog", Enabled: true},
			wantName:    "catalog",
			wantEnabled: true,
		},
		{
			name:        "templates disabled",
			feature:     PortalFeature{Name: "templates", Enabled: false},
			wantName:    "templates",
			wantEnabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.feature.Name != tt.wantName {
				t.Errorf("PortalFeature.Name = %v, want %v", tt.feature.Name, tt.wantName)
			}
			if tt.feature.Enabled != tt.wantEnabled {
				t.Errorf("PortalFeature.Enabled = %v, want %v", tt.feature.Enabled, tt.wantEnabled)
			}
		})
	}
}

func TestIntegration(t *testing.T) {
	tests := []struct {
		name        string
		integration Integration
		wantType    string
		wantEnabled bool
	}{
		{
			name: "github integration",
			integration: Integration{
				Type:    "github",
				Enabled: true,
				Config: map[string]interface{}{
					"org": "my-org",
				},
			},
			wantType:    "github",
			wantEnabled: true,
		},
		{
			name: "argocd integration",
			integration: Integration{
				Type:    "argocd",
				Enabled: true,
				Config: map[string]interface{}{
					"url": "https://argocd.example.com",
				},
			},
			wantType:    "argocd",
			wantEnabled: true,
		},
		{
			name: "disabled integration",
			integration: Integration{
				Type:    "jenkins",
				Enabled: false,
			},
			wantType:    "jenkins",
			wantEnabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.integration.Type != tt.wantType {
				t.Errorf("Integration.Type = %v, want %v", tt.integration.Type, tt.wantType)
			}
			if tt.integration.Enabled != tt.wantEnabled {
				t.Errorf("Integration.Enabled = %v, want %v", tt.integration.Enabled, tt.wantEnabled)
			}
		})
	}
}

func TestCustomizationConfig(t *testing.T) {
	config := &CustomizationConfig{
		Branding: &BrandingConfig{
			Name: "Acme Platform",
			Logo: "https://example.com/logo.png",
		},
		Theme: &ThemeConfig{
			PrimaryColor: "#007bff",
			Mode:         "dark",
		},
	}

	if config.Branding.Name != "Acme Platform" {
		t.Errorf("BrandingConfig.Name = %v, want Acme Platform", config.Branding.Name)
	}
	if config.Theme.Mode != "dark" {
		t.Errorf("ThemeConfig.Mode = %v, want dark", config.Theme.Mode)
	}
	if config.Theme.PrimaryColor != "#007bff" {
		t.Errorf("ThemeConfig.PrimaryColor = %v, want #007bff", config.Theme.PrimaryColor)
	}
}

func TestBrandingConfig(t *testing.T) {
	branding := &BrandingConfig{
		Name: "My Platform",
		Logo: "https://example.com/logo.png",
	}

	if branding.Name != "My Platform" {
		t.Errorf("BrandingConfig.Name = %v, want My Platform", branding.Name)
	}
	if branding.Logo != "https://example.com/logo.png" {
		t.Errorf("BrandingConfig.Logo = %v, want https://example.com/logo.png", branding.Logo)
	}
}

func TestThemeConfig(t *testing.T) {
	tests := []struct {
		name      string
		theme     *ThemeConfig
		wantColor string
		wantMode  string
	}{
		{
			name:      "light theme",
			theme:     &ThemeConfig{PrimaryColor: "#007bff", Mode: "light"},
			wantColor: "#007bff",
			wantMode:  "light",
		},
		{
			name:      "dark theme",
			theme:     &ThemeConfig{PrimaryColor: "#28a745", Mode: "dark"},
			wantColor: "#28a745",
			wantMode:  "dark",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.theme.PrimaryColor != tt.wantColor {
				t.Errorf("ThemeConfig.PrimaryColor = %v, want %v", tt.theme.PrimaryColor, tt.wantColor)
			}
			if tt.theme.Mode != tt.wantMode {
				t.Errorf("ThemeConfig.Mode = %v, want %v", tt.theme.Mode, tt.wantMode)
			}
		})
	}
}

func TestPortalRecommendation(t *testing.T) {
	rec := &PortalRecommendation{
		Template:   "microservices-template",
		Confidence: 0.95,
		Reason:     "Based on ArgoCD and Prometheus usage",
	}

	if rec.Template != "microservices-template" {
		t.Errorf("PortalRecommendation.Template = %v, want microservices-template", rec.Template)
	}
	if rec.Confidence != 0.95 {
		t.Errorf("PortalRecommendation.Confidence = %v, want 0.95", rec.Confidence)
	}
	if rec.Reason == "" {
		t.Error("Expected non-empty reason")
	}
}

func TestDevExStatus(t *testing.T) {
	devex := DevEx{
		Status: DevExStatus{
			Phase:     PhaseReady,
			Message:   "Backstage installed successfully",
			PortalURL: "https://backstage.example.com",
			Recommendation: &PortalRecommendation{
				Template:   "default-template",
				Confidence: 0.9,
				Reason:     "Standard configuration",
			},
		},
	}

	if devex.Status.Phase != PhaseReady {
		t.Errorf("Expected phase %s, got %s", PhaseReady, devex.Status.Phase)
	}

	if devex.Status.PortalURL == "" {
		t.Error("Expected non-empty portal URL")
	}

	if devex.Status.Recommendation == nil {
		t.Error("Expected non-nil recommendation")
	}
}
