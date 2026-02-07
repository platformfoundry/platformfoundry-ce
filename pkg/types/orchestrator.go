package types

// Orchestrator represents orchestrator resources
// Implements US-1.3: Orchestrator Resource Definition
type Orchestrator struct {
	APIVersion string             `yaml:"apiVersion" json:"apiVersion"`
	Kind       string             `yaml:"kind" json:"kind"`
	Metadata   Metadata           `yaml:"metadata" json:"metadata"`
	Spec       OrchestratorSpec   `yaml:"spec" json:"spec"`
	Status     OrchestratorStatus `yaml:"status,omitempty" json:"status,omitempty"`
}

// OrchestratorSpec defines orchestrator specification
type OrchestratorSpec struct {
	Provider     string        `yaml:"provider" json:"provider"` // argocd, flux, tekton
	ClusterRef   string        `yaml:"clusterRef" json:"clusterRef"`
	GitOps       *GitOpsConfig `yaml:"gitops,omitempty" json:"gitops,omitempty"`
	Applications []Application `yaml:"applications,omitempty" json:"applications,omitempty"`
}

// GitOpsConfig defines GitOps configuration
type GitOpsConfig struct {
	RepoURL    string      `yaml:"repoURL" json:"repoURL"`
	Branch     string      `yaml:"branch,omitempty" json:"branch,omitempty"`
	Path       string      `yaml:"path,omitempty" json:"path,omitempty"`
	SyncPolicy *SyncPolicy `yaml:"syncPolicy,omitempty" json:"syncPolicy,omitempty"`
}

// SyncPolicy defines sync policy
type SyncPolicy struct {
	Automated *AutomatedSync `yaml:"automated,omitempty" json:"automated,omitempty"`
}

// AutomatedSync defines automated sync configuration
type AutomatedSync struct {
	Prune    bool `yaml:"prune,omitempty" json:"prune,omitempty"`
	SelfHeal bool `yaml:"selfHeal,omitempty" json:"selfHeal,omitempty"`
}

// Application defines an application
type Application struct {
	Name      string `yaml:"name" json:"name"`
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	Path      string `yaml:"path,omitempty" json:"path,omitempty"`
}

// OrchestratorStatus represents orchestrator status
type OrchestratorStatus struct {
	Phase   Phase  `json:"phase"`
	Message string `json:"message,omitempty"`
}

// Validate validates the orchestrator resource
func (o *Orchestrator) Validate() error {
	if o.APIVersion == "" {
		return ErrMissingAPIVersion
	}
	if o.Kind != "Orchestrator" {
		return ErrInvalidKind
	}
	if o.Metadata.Name == "" {
		return ErrMissingName
	}
	if o.Spec.Provider == "" {
		return ErrInvalidProvider
	}
	if o.Spec.ClusterRef == "" {
		return ErrMissingClusterRef
	}
	return nil
}
