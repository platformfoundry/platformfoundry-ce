package types

import (
	"testing"
)

func TestPlatform_Validate(t *testing.T) {
	tests := []struct {
		name     string
		platform Platform
		wantErr  bool
		errType  error
	}{
		{
			name: "valid platform with infrastructure",
			platform: Platform{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Platform",
				Metadata: Metadata{
					Name: "test-platform",
				},
				Spec: PlatformSpec{
					Components: ComponentReferences{
						Infrastructure: "aws-infra",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing apiVersion",
			platform: Platform{
				Kind: "Platform",
				Metadata: Metadata{
					Name: "test",
				},
				Spec: PlatformSpec{
					Components: ComponentReferences{
						Infrastructure: "aws-infra",
					},
				},
			},
			wantErr: true,
			errType: ErrMissingAPIVersion,
		},
		{
			name: "invalid kind",
			platform: Platform{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "InvalidKind",
				Metadata: Metadata{
					Name: "test",
				},
				Spec: PlatformSpec{
					Components: ComponentReferences{
						Infrastructure: "aws-infra",
					},
				},
			},
			wantErr: true,
			errType: ErrInvalidKind,
		},
		{
			name: "missing name",
			platform: Platform{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Platform",
				Metadata:   Metadata{},
				Spec: PlatformSpec{
					Components: ComponentReferences{
						Infrastructure: "aws-infra",
					},
				},
			},
			wantErr: true,
			errType: ErrMissingName,
		},
		{
			name: "no components defined",
			platform: Platform{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Platform",
				Metadata: Metadata{
					Name: "test",
				},
				Spec: PlatformSpec{
					Components: ComponentReferences{},
				},
			},
			wantErr: true,
			errType: ErrNoComponents,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.platform.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Platform.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != tt.errType {
				t.Errorf("Platform.Validate() error = %v, want %v", err, tt.errType)
			}
		})
	}
}

func TestPlatform_GetComponentNames(t *testing.T) {
	platform := Platform{
		Spec: PlatformSpec{
			Components: ComponentReferences{
				Infrastructure: "aws-infra",
				Orchestrator:   "argocd",
				DevEx:          "backstage",
			},
		},
	}

	names := platform.GetComponentNames()
	if len(names) != 3 {
		t.Errorf("GetComponentNames() returned %d names, want 3", len(names))
	}

	expected := map[string]bool{
		"aws-infra": true,
		"argocd":    true,
		"backstage": true,
	}

	for _, name := range names {
		if !expected[name] {
			t.Errorf("GetComponentNames() returned unexpected name: %s", name)
		}
	}
}
