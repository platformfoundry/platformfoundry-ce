package types

import (
	"fmt"
	"regexp"
	"time"
)

// Organization represents a tenant organization
type Organization struct {
	APIVersion string             `yaml:"apiVersion" json:"apiVersion"`
	Kind       string             `yaml:"kind" json:"kind"`
	Metadata   Metadata           `yaml:"metadata" json:"metadata"`
	Spec       OrganizationSpec   `yaml:"spec" json:"spec"`
	Status     OrganizationStatus `yaml:"status,omitempty" json:"status,omitempty"`
}

// OrganizationSpec defines organization specification
type OrganizationSpec struct {
	DisplayName string            `yaml:"displayName" json:"displayName"`
	Description string            `yaml:"description,omitempty" json:"description,omitempty"`
	Owner       string            `yaml:"owner" json:"owner"`
	Contact     ContactInfo       `yaml:"contact,omitempty" json:"contact,omitempty"`
	Quotas      *ResourceQuotas   `yaml:"quotas,omitempty" json:"quotas,omitempty"`
	Settings    map[string]string `yaml:"settings,omitempty" json:"settings,omitempty"`
}

// ContactInfo represents organization contact details
type ContactInfo struct {
	Email string `yaml:"email,omitempty" json:"email,omitempty"`
	Slack string `yaml:"slack,omitempty" json:"slack,omitempty"`
}

// ResourceQuotas defines resource limits for an organization
type ResourceQuotas struct {
	MaxPlatforms    int `yaml:"maxPlatforms,omitempty" json:"maxPlatforms,omitempty"`
	MaxClusters     int `yaml:"maxClusters,omitempty" json:"maxClusters,omitempty"`
	MaxEnvironments int `yaml:"maxEnvironments,omitempty" json:"maxEnvironments,omitempty"`
}

// OrganizationStatus represents organization status
type OrganizationStatus struct {
	Phase        Phase     `json:"phase"`
	MemberCount  int       `json:"memberCount"`
	PlatformCount int      `json:"platformCount"`
	LastActivity time.Time `json:"lastActivity,omitempty"`
}

var (
	// organizationNameRegex validates organization names (alphanumeric, hyphens, max 63 chars)
	organizationNameRegex = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	// emailRegex basic email validation
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
)

// Validate validates the organization resource with security checks
func (o *Organization) Validate() error {
	if o.APIVersion == "" {
		return ErrMissingAPIVersion
	}
	if o.Kind != "Organization" {
		return ErrInvalidKind
	}
	if o.Metadata.Name == "" {
		return ErrMissingName
	}

	// Security: Validate organization name format
	if len(o.Metadata.Name) > 63 {
		return fmt.Errorf("organization name must be 63 characters or less")
	}
	if !organizationNameRegex.MatchString(o.Metadata.Name) {
		return fmt.Errorf("organization name must be lowercase alphanumeric with hyphens, starting and ending with alphanumeric")
	}

	if o.Spec.DisplayName == "" {
		return ErrMissingDisplayName
	}

	// Security: Limit display name length
	if len(o.Spec.DisplayName) > 256 {
		return fmt.Errorf("display name must be 256 characters or less")
	}

	if o.Spec.Owner == "" {
		return ErrMissingOwner
	}

	// Security: Validate owner name format
	if len(o.Spec.Owner) > 128 {
		return fmt.Errorf("owner name must be 128 characters or less")
	}

	// Validate contact email if provided
	if o.Spec.Contact.Email != "" {
		if !emailRegex.MatchString(o.Spec.Contact.Email) {
			return fmt.Errorf("invalid email format")
		}
	}

	// Security: Validate description length
	if len(o.Spec.Description) > 1024 {
		return fmt.Errorf("description must be 1024 characters or less")
	}

	// Validate quotas if provided
	if o.Spec.Quotas != nil {
		if o.Spec.Quotas.MaxPlatforms < 0 || o.Spec.Quotas.MaxClusters < 0 || o.Spec.Quotas.MaxEnvironments < 0 {
			return fmt.Errorf("quotas must be non-negative")
		}
		// Security: Set reasonable upper limits
		if o.Spec.Quotas.MaxPlatforms > 1000 || o.Spec.Quotas.MaxClusters > 10000 || o.Spec.Quotas.MaxEnvironments > 1000 {
			return fmt.Errorf("quota values exceed maximum allowed limits")
		}
	}

	return nil
}
