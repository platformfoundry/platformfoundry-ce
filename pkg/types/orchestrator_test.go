package types

import (
	"testing"
)

func TestOrchestrator_Validate(t *testing.T) {
	tests := []struct {
		name    string
		orch    Orchestrator
		wantErr bool
		errType error
	}{
		{
			name: "valid orchestrator with ArgoCD",
			orch: Orchestrator{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Orchestrator",
				Metadata: Metadata{
					Name: "test-orch",
				},
				Spec: OrchestratorSpec{
					Provider:   "argocd",
					ClusterRef: "my-cluster",
					GitOps: &GitOpsConfig{
						RepoURL: "https://github.com/org/repo",
						Branch:  "main",
						Path:    "apps/",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid orchestrator with Flux",
			orch: Orchestrator{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Orchestrator",
				Metadata: Metadata{
					Name: "flux-orch",
				},
				Spec: OrchestratorSpec{
					Provider:   "flux",
					ClusterRef: "my-cluster",
					GitOps: &GitOpsConfig{
						RepoURL: "https://github.com/org/repo",
						Branch:  "develop",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid orchestrator with applications",
			orch: Orchestrator{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Orchestrator",
				Metadata: Metadata{
					Name: "app-orch",
				},
				Spec: OrchestratorSpec{
					Provider:   "argocd",
					ClusterRef: "my-cluster",
					GitOps: &GitOpsConfig{
						RepoURL: "https://github.com/org/repo",
						Branch:  "main",
					},
					Applications: []Application{
						{
							Name:      "app1",
							Namespace: "default",
							Path:      "apps/app1",
						},
						{
							Name:      "app2",
							Namespace: "production",
							Path:      "apps/app2",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid orchestrator with sync policy",
			orch: Orchestrator{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Orchestrator",
				Metadata: Metadata{
					Name: "sync-orch",
				},
				Spec: OrchestratorSpec{
					Provider:   "argocd",
					ClusterRef: "my-cluster",
					GitOps: &GitOpsConfig{
						RepoURL: "https://github.com/org/repo",
						Branch:  "main",
						SyncPolicy: &SyncPolicy{
							Automated: &AutomatedSync{
								Prune:    true,
								SelfHeal: true,
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing apiVersion",
			orch: Orchestrator{
				Kind: "Orchestrator",
				Metadata: Metadata{
					Name: "test",
				},
				Spec: OrchestratorSpec{
					Provider:   "argocd",
					ClusterRef: "my-cluster",
				},
			},
			wantErr: true,
			errType: ErrMissingAPIVersion,
		},
		{
			name: "invalid kind",
			orch: Orchestrator{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "InvalidKind",
				Metadata: Metadata{
					Name: "test",
				},
				Spec: OrchestratorSpec{
					Provider:   "argocd",
					ClusterRef: "my-cluster",
				},
			},
			wantErr: true,
			errType: ErrInvalidKind,
		},
		{
			name: "missing name",
			orch: Orchestrator{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Orchestrator",
				Metadata:   Metadata{},
				Spec: OrchestratorSpec{
					Provider:   "argocd",
					ClusterRef: "my-cluster",
				},
			},
			wantErr: true,
			errType: ErrMissingName,
		},
		{
			name: "missing provider",
			orch: Orchestrator{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Orchestrator",
				Metadata: Metadata{
					Name: "test",
				},
				Spec: OrchestratorSpec{
					ClusterRef: "my-cluster",
				},
			},
			wantErr: true,
			errType: ErrInvalidProvider,
		},
		{
			name: "missing clusterRef",
			orch: Orchestrator{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Orchestrator",
				Metadata: Metadata{
					Name: "test",
				},
				Spec: OrchestratorSpec{
					Provider: "argocd",
				},
			},
			wantErr: true,
			errType: ErrMissingClusterRef,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.orch.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Orchestrator.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != tt.errType {
				t.Errorf("Orchestrator.Validate() error = %v, want %v", err, tt.errType)
			}
		})
	}
}

func TestGitOpsConfig(t *testing.T) {
	tests := []struct {
		name       string
		gitops     *GitOpsConfig
		wantBranch string
	}{
		{
			name: "with branch",
			gitops: &GitOpsConfig{
				RepoURL: "https://github.com/org/repo",
				Branch:  "main",
				Path:    "apps/",
			},
			wantBranch: "main",
		},
		{
			name: "with different branch",
			gitops: &GitOpsConfig{
				RepoURL: "https://github.com/org/repo",
				Branch:  "develop",
			},
			wantBranch: "develop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.gitops.Branch != tt.wantBranch {
				t.Errorf("GitOpsConfig.Branch = %v, want %v", tt.gitops.Branch, tt.wantBranch)
			}
		})
	}
}

func TestSyncPolicy(t *testing.T) {
	tests := []struct {
		name         string
		policy       *SyncPolicy
		wantPrune    bool
		wantSelfHeal bool
	}{
		{
			name: "automated with prune and self-heal",
			policy: &SyncPolicy{
				Automated: &AutomatedSync{
					Prune:    true,
					SelfHeal: true,
				},
			},
			wantPrune:    true,
			wantSelfHeal: true,
		},
		{
			name: "automated with prune only",
			policy: &SyncPolicy{
				Automated: &AutomatedSync{
					Prune:    true,
					SelfHeal: false,
				},
			},
			wantPrune:    true,
			wantSelfHeal: false,
		},
		{
			name: "automated with self-heal only",
			policy: &SyncPolicy{
				Automated: &AutomatedSync{
					Prune:    false,
					SelfHeal: true,
				},
			},
			wantPrune:    false,
			wantSelfHeal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.policy.Automated.Prune != tt.wantPrune {
				t.Errorf("SyncPolicy.Automated.Prune = %v, want %v", tt.policy.Automated.Prune, tt.wantPrune)
			}
			if tt.policy.Automated.SelfHeal != tt.wantSelfHeal {
				t.Errorf("SyncPolicy.Automated.SelfHeal = %v, want %v", tt.policy.Automated.SelfHeal, tt.wantSelfHeal)
			}
		})
	}
}

func TestApplications(t *testing.T) {
	app := Application{
		Name:      "test-app",
		Namespace: "production",
		Path:      "apps/test",
	}

	if app.Name != "test-app" {
		t.Errorf("Application.Name = %v, want test-app", app.Name)
	}
	if app.Namespace != "production" {
		t.Errorf("Application.Namespace = %v, want production", app.Namespace)
	}
	if app.Path != "apps/test" {
		t.Errorf("Application.Path = %v, want apps/test", app.Path)
	}
}

func TestOrchestratorStatus(t *testing.T) {
	orch := Orchestrator{
		Status: OrchestratorStatus{
			Phase:   PhaseReady,
			Message: "ArgoCD installed successfully",
		},
	}

	if orch.Status.Phase != PhaseReady {
		t.Errorf("Expected phase %s, got %s", PhaseReady, orch.Status.Phase)
	}

	if orch.Status.Message == "" {
		t.Error("Expected non-empty status message")
	}
}
