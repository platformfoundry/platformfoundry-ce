package types

import (
	"testing"
)

func TestInfrastructure_Validate(t *testing.T) {
	tests := []struct {
		name    string
		infra   Infrastructure
		wantErr bool
		errType error
	}{
		{
			name: "valid infrastructure with terraform provider",
			infra: Infrastructure{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Infrastructure",
				Metadata: Metadata{
					Name: "test-infra",
				},
				Spec: InfrastructureSpec{
					Provider: "terraform",
					Cloud: CloudConfig{
						Provider: "aws",
						Region:   "us-east-1",
						VPC: &VPCConfig{
							CIDR: "10.0.0.0/16",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid infrastructure with crossplane provider",
			infra: Infrastructure{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Infrastructure",
				Metadata: Metadata{
					Name: "crossplane-infra",
				},
				Spec: InfrastructureSpec{
					Provider: "crossplane",
					Cloud: CloudConfig{
						Provider: "gcp",
						Region:   "us-central1",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid infrastructure with clusters",
			infra: Infrastructure{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Infrastructure",
				Metadata: Metadata{
					Name: "multi-cluster-infra",
				},
				Spec: InfrastructureSpec{
					Provider: "terraform",
					Cloud: CloudConfig{
						Provider: "azure",
						Region:   "eastus",
					},
					Clusters: []ClusterConfig{
						{
							Name:    "prod-cluster",
							Type:    "aks",
							Version: "1.28",
						},
						{
							Name:    "dev-cluster",
							Type:    "aks",
							Version: "1.27",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid infrastructure with tags",
			infra: Infrastructure{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Infrastructure",
				Metadata: Metadata{
					Name: "tagged-infra",
				},
				Spec: InfrastructureSpec{
					Provider: "terraform",
					Cloud: CloudConfig{
						Provider: "aws",
						Region:   "us-west-2",
					},
					Tags: map[string]string{
						"Environment": "production",
						"Team":        "platform",
						"CostCenter":  "engineering",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing apiVersion",
			infra: Infrastructure{
				Kind: "Infrastructure",
				Metadata: Metadata{
					Name: "test",
				},
				Spec: InfrastructureSpec{
					Provider: "terraform",
				},
			},
			wantErr: true,
			errType: ErrMissingAPIVersion,
		},
		{
			name: "invalid kind",
			infra: Infrastructure{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "InvalidKind",
				Metadata: Metadata{
					Name: "test",
				},
				Spec: InfrastructureSpec{
					Provider: "terraform",
				},
			},
			wantErr: true,
			errType: ErrInvalidKind,
		},
		{
			name: "missing name",
			infra: Infrastructure{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Infrastructure",
				Metadata:   Metadata{},
				Spec: InfrastructureSpec{
					Provider: "terraform",
				},
			},
			wantErr: true,
			errType: ErrMissingName,
		},
		{
			name: "missing provider",
			infra: Infrastructure{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Infrastructure",
				Metadata: Metadata{
					Name: "test",
				},
				Spec: InfrastructureSpec{},
			},
			wantErr: true,
			errType: ErrInvalidProvider,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.infra.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Infrastructure.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != tt.errType {
				t.Errorf("Infrastructure.Validate() error = %v, want %v", err, tt.errType)
			}
		})
	}
}

func TestInfrastructureSpec_MultiCloud(t *testing.T) {
	// Test different cloud providers
	providers := []struct {
		name     string
		provider string
		region   string
	}{
		{"AWS", "aws", "us-east-1"},
		{"GCP", "gcp", "us-central1"},
		{"Azure", "azure", "eastus"},
	}

	for _, tt := range providers {
		t.Run(tt.name, func(t *testing.T) {
			infra := Infrastructure{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Infrastructure",
				Metadata: Metadata{
					Name: tt.name + "-infra",
				},
				Spec: InfrastructureSpec{
					Provider: "terraform",
					Cloud: CloudConfig{
						Provider: tt.provider,
						Region:   tt.region,
					},
				},
			}

			if err := infra.Validate(); err != nil {
				t.Errorf("%s infrastructure validation failed: %v", tt.name, err)
			}
		})
	}
}

func TestClusterConfig_Types(t *testing.T) {
	// Test different cluster types
	clusterTypes := []struct {
		name        string
		clusterType string
		provider    string
	}{
		{"EKS on AWS", "eks", "aws"},
		{"GKE on GCP", "gke", "gcp"},
		{"AKS on Azure", "aks", "azure"},
	}

	for _, tt := range clusterTypes {
		t.Run(tt.name, func(t *testing.T) {
			infra := Infrastructure{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Infrastructure",
				Metadata: Metadata{
					Name: "test-infra",
				},
				Spec: InfrastructureSpec{
					Provider: "terraform",
					Cloud: CloudConfig{
						Provider: tt.provider,
					},
					Clusters: []ClusterConfig{
						{
							Name:    "test-cluster",
							Type:    tt.clusterType,
							Version: "1.28",
						},
					},
				},
			}

			if err := infra.Validate(); err != nil {
				t.Errorf("%s cluster validation failed: %v", tt.name, err)
			}

			if len(infra.Spec.Clusters) != 1 {
				t.Errorf("Expected 1 cluster, got %d", len(infra.Spec.Clusters))
			}

			cluster := infra.Spec.Clusters[0]
			if cluster.Type != tt.clusterType {
				t.Errorf("Expected cluster type %s, got %s", tt.clusterType, cluster.Type)
			}
		})
	}
}

func TestVPCConfig(t *testing.T) {
	tests := []struct {
		name     string
		cidr     string
		wantCIDR string
	}{
		{"standard VPC", "10.0.0.0/16", "10.0.0.0/16"},
		{"large VPC", "10.0.0.0/8", "10.0.0.0/8"},
		{"small VPC", "10.0.0.0/24", "10.0.0.0/24"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vpc := &VPCConfig{
				CIDR: tt.cidr,
			}

			if vpc.CIDR != tt.wantCIDR {
				t.Errorf("VPC CIDR = %v, want %v", vpc.CIDR, tt.wantCIDR)
			}
		})
	}
}

func TestInfrastructureStatus(t *testing.T) {
	infra := Infrastructure{
		Status: InfrastructureStatus{
			Phase:   PhaseProvisioning,
			Message: "Creating VPC",
		},
	}

	if infra.Status.Phase != PhaseProvisioning {
		t.Errorf("Expected phase %s, got %s", PhaseProvisioning, infra.Status.Phase)
	}

	if infra.Status.Message == "" {
		t.Error("Expected non-empty status message")
	}
}
