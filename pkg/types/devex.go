package types

// DevEx represents developer experience portal
// Implements US-1.5: DevEx Resource Definition
type DevEx struct {
	APIVersion string      `yaml:"apiVersion" json:"apiVersion"`
	Kind       string      `yaml:"kind" json:"kind"`
	Metadata   Metadata    `yaml:"metadata" json:"metadata"`
	Spec       DevExSpec   `yaml:"spec" json:"spec"`
	Status     DevExStatus `yaml:"status,omitempty" json:"status,omitempty"`
}

// DevExSpec defines DevEx specification
type DevExSpec struct {
	Provider              string                           `yaml:"provider" json:"provider"` // backstage, port
	ClusterRef            string                           `yaml:"clusterRef" json:"clusterRef"`
	IntelligentGeneration *IntelligentGenerationConfig     `yaml:"intelligentGeneration,omitempty" json:"intelligentGeneration,omitempty"`
	Portal                *PortalConfig                    `yaml:"portal,omitempty" json:"portal,omitempty"`
	Customization         *CustomizationConfig             `yaml:"customization,omitempty" json:"customization,omitempty"`
}

// IntelligentGenerationConfig defines intelligent generation settings
type IntelligentGenerationConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	Strategy string `yaml:"strategy,omitempty" json:"strategy,omitempty"` // rule-based, similarity, llm
}

// PortalConfig defines portal configuration
type PortalConfig struct {
	Features     []PortalFeature `yaml:"features,omitempty" json:"features,omitempty"`
	Integrations []Integration   `yaml:"integrations,omitempty" json:"integrations,omitempty"`
}

// PortalFeature defines a portal feature
type PortalFeature struct {
	Name    string `yaml:"name" json:"name"`
	Enabled bool   `yaml:"enabled" json:"enabled"`
}

// Integration defines an integration
type Integration struct {
	Type    string                 `yaml:"type" json:"type"`
	Enabled bool                   `yaml:"enabled" json:"enabled"`
	Config  map[string]interface{} `yaml:"config,omitempty" json:"config,omitempty"`
}

// CustomizationConfig defines portal customization
type CustomizationConfig struct{
	Branding *BrandingConfig `yaml:"branding,omitempty" json:"branding,omitempty"`
	Theme    *ThemeConfig    `yaml:"theme,omitempty" json:"theme,omitempty"`
}

// BrandingConfig defines branding
type BrandingConfig struct {
	Name string `yaml:"name,omitempty" json:"name,omitempty"`
	Logo string `yaml:"logo,omitempty" json:"logo,omitempty"`
}

// ThemeConfig defines theme
type ThemeConfig struct {
	PrimaryColor string `yaml:"primaryColor,omitempty" json:"primaryColor,omitempty"`
	Mode         string `yaml:"mode,omitempty" json:"mode,omitempty"` // light, dark
}

// DevExStatus represents DevEx status
type DevExStatus struct {
	Phase         Phase                  `json:"phase"`
	Message       string                 `json:"message,omitempty"`
	PortalURL     string                 `json:"portalUrl,omitempty"`
	Recommendation *PortalRecommendation `json:"recommendation,omitempty"`
}

// PortalRecommendation represents intelligent recommendation result
type PortalRecommendation struct {
	Template   string  `json:"template"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
}

// Validate validates the DevEx resource
func (d *DevEx) Validate() error {
	if d.APIVersion == "" {
		return ErrMissingAPIVersion
	}
	if d.Kind != "DevEx" {
		return ErrInvalidKind
	}
	if d.Metadata.Name == "" {
		return ErrMissingName
	}
	if d.Spec.Provider == "" {
		return ErrInvalidProvider
	}
	if d.Spec.ClusterRef == "" {
		return ErrMissingClusterRef
	}
	return nil
}
