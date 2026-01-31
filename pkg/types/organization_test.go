package types

import (
	"strings"
	"testing"
)

func TestOrganization_Validate(t *testing.T) {
	tests := []struct {
		name    string
		org     Organization
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid organization",
			org: Organization{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Organization",
				Metadata: Metadata{
					Name: "acme-corp",
				},
				Spec: OrganizationSpec{
					DisplayName: "ACME Corporation",
					Description: "ACME Corp platform team",
					Owner:       "admin@acme.com",
					Contact: ContactInfo{
						Email: "platform@acme.com",
						Slack: "#platform",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid organization with quotas",
			org: Organization{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Organization",
				Metadata: Metadata{
					Name: "dev-team",
				},
				Spec: OrganizationSpec{
					DisplayName: "Development Team",
					Owner:       "dev-lead@example.com",
					Quotas: &ResourceQuotas{
						MaxPlatforms:    10,
						MaxClusters:     20,
						MaxEnvironments: 30,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid organization with settings",
			org: Organization{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Organization",
				Metadata: Metadata{
					Name: "qa-team",
				},
				Spec: OrganizationSpec{
					DisplayName: "QA Team",
					Owner:       "qa-lead@example.com",
					Settings: map[string]string{
						"region":      "us-east-1",
						"environment": "test",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing apiVersion",
			org: Organization{
				Kind: "Organization",
				Metadata: Metadata{
					Name: "test",
				},
				Spec: OrganizationSpec{
					DisplayName: "Test Org",
					Owner:       "owner@test.com",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid kind",
			org: Organization{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "InvalidKind",
				Metadata: Metadata{
					Name: "test",
				},
				Spec: OrganizationSpec{
					DisplayName: "Test Org",
					Owner:       "owner@test.com",
				},
			},
			wantErr: true,
		},
		{
			name: "missing name",
			org: Organization{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Organization",
				Metadata:   Metadata{},
				Spec: OrganizationSpec{
					DisplayName: "Test Org",
					Owner:       "owner@test.com",
				},
			},
			wantErr: true,
		},
		{
			name: "name too long",
			org: Organization{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Organization",
				Metadata: Metadata{
					Name: strings.Repeat("a", 64), // 64 chars
				},
				Spec: OrganizationSpec{
					DisplayName: "Test Org",
					Owner:       "owner@test.com",
				},
			},
			wantErr: true,
			errMsg:  "63 characters or less",
		},
		{
			name: "invalid name format (uppercase)",
			org: Organization{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Organization",
				Metadata: Metadata{
					Name: "AcmeCorp",
				},
				Spec: OrganizationSpec{
					DisplayName: "ACME Corp",
					Owner:       "owner@acme.com",
				},
			},
			wantErr: true,
			errMsg:  "lowercase alphanumeric",
		},
		{
			name: "invalid name format (underscore)",
			org: Organization{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Organization",
				Metadata: Metadata{
					Name: "acme_corp",
				},
				Spec: OrganizationSpec{
					DisplayName: "ACME Corp",
					Owner:       "owner@acme.com",
				},
			},
			wantErr: true,
			errMsg:  "lowercase alphanumeric",
		},
		{
			name: "invalid name format (starts with hyphen)",
			org: Organization{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Organization",
				Metadata: Metadata{
					Name: "-acme",
				},
				Spec: OrganizationSpec{
					DisplayName: "ACME Corp",
					Owner:       "owner@acme.com",
				},
			},
			wantErr: true,
			errMsg:  "lowercase alphanumeric",
		},
		{
			name: "missing display name",
			org: Organization{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Organization",
				Metadata: Metadata{
					Name: "test",
				},
				Spec: OrganizationSpec{
					Owner: "owner@test.com",
				},
			},
			wantErr: true,
		},
		{
			name: "display name too long",
			org: Organization{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Organization",
				Metadata: Metadata{
					Name: "test",
				},
				Spec: OrganizationSpec{
					DisplayName: strings.Repeat("a", 257), // 257 chars
					Owner:       "owner@test.com",
				},
			},
			wantErr: true,
			errMsg:  "256 characters or less",
		},
		{
			name: "missing owner",
			org: Organization{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Organization",
				Metadata: Metadata{
					Name: "test",
				},
				Spec: OrganizationSpec{
					DisplayName: "Test Org",
				},
			},
			wantErr: true,
		},
		{
			name: "owner name too long",
			org: Organization{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Organization",
				Metadata: Metadata{
					Name: "test",
				},
				Spec: OrganizationSpec{
					DisplayName: "Test Org",
					Owner:       strings.Repeat("a", 129), // 129 chars
				},
			},
			wantErr: true,
			errMsg:  "128 characters or less",
		},
		{
			name: "invalid email format",
			org: Organization{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Organization",
				Metadata: Metadata{
					Name: "test",
				},
				Spec: OrganizationSpec{
					DisplayName: "Test Org",
					Owner:       "owner@test.com",
					Contact: ContactInfo{
						Email: "invalid-email",
					},
				},
			},
			wantErr: true,
			errMsg:  "invalid email format",
		},
		{
			name: "description too long",
			org: Organization{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Organization",
				Metadata: Metadata{
					Name: "test",
				},
				Spec: OrganizationSpec{
					DisplayName: "Test Org",
					Owner:       "owner@test.com",
					Description: strings.Repeat("a", 1025), // 1025 chars
				},
			},
			wantErr: true,
			errMsg:  "1024 characters or less",
		},
		{
			name: "negative quotas",
			org: Organization{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Organization",
				Metadata: Metadata{
					Name: "test",
				},
				Spec: OrganizationSpec{
					DisplayName: "Test Org",
					Owner:       "owner@test.com",
					Quotas: &ResourceQuotas{
						MaxPlatforms: -1,
					},
				},
			},
			wantErr: true,
			errMsg:  "non-negative",
		},
		{
			name: "quotas exceed maximum",
			org: Organization{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Organization",
				Metadata: Metadata{
					Name: "test",
				},
				Spec: OrganizationSpec{
					DisplayName: "Test Org",
					Owner:       "owner@test.com",
					Quotas: &ResourceQuotas{
						MaxPlatforms: 1001,
					},
				},
			},
			wantErr: true,
			errMsg:  "exceed maximum allowed limits",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.org.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Organization.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Organization.Validate() error = %v, want error containing %v", err, tt.errMsg)
			}
		})
	}
}

func TestContactInfo(t *testing.T) {
	tests := []struct {
		name    string
		contact ContactInfo
		wantEmail string
		wantSlack string
	}{
		{
			name: "both email and slack",
			contact: ContactInfo{
				Email: "team@example.com",
				Slack: "#platform",
			},
			wantEmail: "team@example.com",
			wantSlack: "#platform",
		},
		{
			name: "email only",
			contact: ContactInfo{
				Email: "team@example.com",
			},
			wantEmail: "team@example.com",
			wantSlack: "",
		},
		{
			name: "slack only",
			contact: ContactInfo{
				Slack: "#engineering",
			},
			wantEmail: "",
			wantSlack: "#engineering",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.contact.Email != tt.wantEmail {
				t.Errorf("ContactInfo.Email = %v, want %v", tt.contact.Email, tt.wantEmail)
			}
			if tt.contact.Slack != tt.wantSlack {
				t.Errorf("ContactInfo.Slack = %v, want %v", tt.contact.Slack, tt.wantSlack)
			}
		})
	}
}

func TestResourceQuotas(t *testing.T) {
	tests := []struct {
		name            string
		quotas          *ResourceQuotas
		wantPlatforms   int
		wantClusters    int
		wantEnvironments int
	}{
		{
			name: "standard quotas",
			quotas: &ResourceQuotas{
				MaxPlatforms:    10,
				MaxClusters:     50,
				MaxEnvironments: 30,
			},
			wantPlatforms:   10,
			wantClusters:    50,
			wantEnvironments: 30,
		},
		{
			name: "high quotas",
			quotas: &ResourceQuotas{
				MaxPlatforms:    100,
				MaxClusters:     1000,
				MaxEnvironments: 500,
			},
			wantPlatforms:   100,
			wantClusters:    1000,
			wantEnvironments: 500,
		},
		{
			name: "unlimited (zero) quotas",
			quotas: &ResourceQuotas{
				MaxPlatforms:    0,
				MaxClusters:     0,
				MaxEnvironments: 0,
			},
			wantPlatforms:   0,
			wantClusters:    0,
			wantEnvironments: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.quotas.MaxPlatforms != tt.wantPlatforms {
				t.Errorf("ResourceQuotas.MaxPlatforms = %v, want %v", tt.quotas.MaxPlatforms, tt.wantPlatforms)
			}
			if tt.quotas.MaxClusters != tt.wantClusters {
				t.Errorf("ResourceQuotas.MaxClusters = %v, want %v", tt.quotas.MaxClusters, tt.wantClusters)
			}
			if tt.quotas.MaxEnvironments != tt.wantEnvironments {
				t.Errorf("ResourceQuotas.MaxEnvironments = %v, want %v", tt.quotas.MaxEnvironments, tt.wantEnvironments)
			}
		})
	}
}

func TestOrganizationStatus(t *testing.T) {
	org := Organization{
		Status: OrganizationStatus{
			Phase:         PhaseReady,
			MemberCount:   15,
			PlatformCount: 5,
		},
	}

	if org.Status.Phase != PhaseReady {
		t.Errorf("Expected phase %s, got %s", PhaseReady, org.Status.Phase)
	}

	if org.Status.MemberCount != 15 {
		t.Errorf("Expected 15 members, got %d", org.Status.MemberCount)
	}

	if org.Status.PlatformCount != 5 {
		t.Errorf("Expected 5 platforms, got %d", org.Status.PlatformCount)
	}
}

func TestValidEmailFormats(t *testing.T) {
	validEmails := []string{
		"user@example.com",
		"user.name@example.com",
		"user+tag@example.co.uk",
		"user_name@example-domain.com",
	}

	for _, email := range validEmails {
		org := Organization{
			APIVersion: "platformfoundry.io/v1",
			Kind:       "Organization",
			Metadata: Metadata{
				Name: "test",
			},
			Spec: OrganizationSpec{
				DisplayName: "Test",
				Owner:       "owner@test.com",
				Contact: ContactInfo{
					Email: email,
				},
			},
		}

		if err := org.Validate(); err != nil {
			t.Errorf("Valid email %s failed validation: %v", email, err)
		}
	}
}

func TestInvalidEmailFormats(t *testing.T) {
	invalidEmails := []string{
		"invalid",
		"@example.com",
		"user@",
		"user @example.com",
		"user@example",
	}

	for _, email := range invalidEmails {
		org := Organization{
			APIVersion: "platformfoundry.io/v1",
			Kind:       "Organization",
			Metadata: Metadata{
				Name: "test",
			},
			Spec: OrganizationSpec{
				DisplayName: "Test",
				Owner:       "owner@test.com",
				Contact: ContactInfo{
					Email: email,
				},
			},
		}

		if err := org.Validate(); err == nil {
			t.Errorf("Invalid email %s should have failed validation", email)
		}
	}
}
