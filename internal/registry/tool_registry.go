// Package registry provides tool registration, compatibility checking,
// and scaffold management for platform components.
package registry

import (
	"fmt"
	"sync"
	"time"
)

// ToolRegistry manages all available tools
type ToolRegistry struct {
	tools         map[string]*RegisteredTool
	categories    map[string][]string
	compatibility *CompatibilityMatrix
	mu            sync.RWMutex
}

// RegisteredTool represents a tool in the registry
type RegisteredTool struct {
	// Identity
	Name        string
	DisplayName string
	Description string
	Category    string

	// Versions
	Versions      []ToolVersion
	LatestVersion string

	// Scaffolds
	Scaffolds []Scaffold

	// Documentation
	Documentation ToolDocumentation

	// Requirements
	Requirements []Requirement

	// Metadata
	Tags       []string
	Maintainer string
	Repository string
	License    string
}

// ToolVersion represents a specific version
type ToolVersion struct {
	Version        string
	ReleaseDate    time.Time
	Deprecated     bool
	MinKubeVersion string
	MaxKubeVersion string
	Breaking       bool
	Changelog      string
}

// Scaffold represents a scaffold template
type Scaffold struct {
	Name        string
	Description string
	Category    string // basic, advanced, production

	// Template
	Files      []ScaffoldFile
	Parameters []ScaffoldParameter

	// Hooks
	PreGenerate  string
	PostGenerate string

	// Validation
	ValidateScript string
}

// ScaffoldFile represents a file to generate
type ScaffoldFile struct {
	Path      string
	Template  string
	Condition string
	Mode      uint32
}

// ScaffoldParameter represents a template parameter
type ScaffoldParameter struct {
	Name           string
	DisplayName    string
	Description    string
	Type           string // string, int, bool, select, multiselect
	Default        interface{}
	Required       bool
	Pattern        string
	Min            *int
	Max            *int
	Options        []string
	DependsOn      string
	DependsOnValue interface{}
}

// ToolDocumentation holds documentation references
type ToolDocumentation struct {
	Overview   string
	QuickStart string
	LearnMore  string
	Tutorials  []Tutorial
}

// Tutorial represents a learning tutorial
type Tutorial struct {
	Name        string
	Description string
	Duration    string
	Content     string
}

// Requirement represents a tool requirement
type Requirement struct {
	Type       string // kubernetes, cloud, tool
	MinVersion string
	MaxVersion string
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() *ToolRegistry {
	registry := &ToolRegistry{
		tools:         make(map[string]*RegisteredTool),
		categories:    make(map[string][]string),
		compatibility: NewCompatibilityMatrix(),
	}

	// Register built-in tools
	registry.registerBuiltinTools()

	return registry
}

// Register adds a tool to the registry
func (r *ToolRegistry) Register(tool *RegisteredTool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[tool.Name]; exists {
		return fmt.Errorf("tool %s already registered", tool.Name)
	}

	r.tools[tool.Name] = tool
	r.categories[tool.Category] = append(r.categories[tool.Category], tool.Name)

	return nil
}

// Get returns a tool by name
func (r *ToolRegistry) Get(name string) (*RegisteredTool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[name]
	return tool, ok
}

// GetByCategory returns all tools in a category
func (r *ToolRegistry) GetByCategory(category string) []*RegisteredTool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tools []*RegisteredTool
	for _, name := range r.categories[category] {
		if tool, ok := r.tools[name]; ok {
			tools = append(tools, tool)
		}
	}
	return tools
}

// GetScaffold returns a specific scaffold for a tool
func (r *ToolRegistry) GetScaffold(toolName, scaffoldName string) (*Scaffold, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, ok := r.tools[toolName]
	if !ok {
		return nil, fmt.Errorf("tool %s not found", toolName)
	}

	for _, scaffold := range tool.Scaffolds {
		if scaffold.Name == scaffoldName {
			return &scaffold, nil
		}
	}

	return nil, fmt.Errorf("scaffold %s not found for tool %s", scaffoldName, toolName)
}

// List returns all registered tools
func (r *ToolRegistry) List() []*RegisteredTool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]*RegisteredTool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// ListCategories returns all categories
func (r *ToolRegistry) ListCategories() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	categories := make([]string, 0, len(r.categories))
	for cat := range r.categories {
		categories = append(categories, cat)
	}
	return categories
}

// GetCompatibility returns the compatibility matrix
func (r *ToolRegistry) GetCompatibility() *CompatibilityMatrix {
	return r.compatibility
}

// CheckCompatibility checks if two tools are compatible
func (r *ToolRegistry) CheckCompatibility(toolA, toolB string) (bool, string) {
	return r.compatibility.Check(toolA, toolB)
}

// ValidateToolSet checks if a set of tools are all compatible
func (r *ToolRegistry) ValidateToolSet(tools []string) []CompatibilityIssue {
	return r.compatibility.ValidateSet(tools)
}

// registerBuiltinTools registers all built-in tools
func (r *ToolRegistry) registerBuiltinTools() {
	// Orchestration tools
	r.Register(&RegisteredTool{
		Name:        "argocd",
		DisplayName: "Argo CD",
		Description: "Declarative GitOps continuous delivery tool for Kubernetes",
		Category:    "orchestration",
		Versions: []ToolVersion{
			{Version: "2.9.3", ReleaseDate: time.Now()},
			{Version: "2.8.7", ReleaseDate: time.Now().AddDate(0, -2, 0)},
		},
		LatestVersion: "2.9.3",
		Scaffolds: []Scaffold{
			{
				Name:        "basic",
				Description: "Basic ArgoCD installation",
				Category:    "basic",
				Parameters: []ScaffoldParameter{
					{Name: "namespace", Default: "argocd", Type: "string"},
					{Name: "ha", Default: false, Type: "bool", DisplayName: "High Availability"},
				},
			},
			{
				Name:        "with-sso",
				Description: "ArgoCD with SSO configuration",
				Category:    "advanced",
				Parameters: []ScaffoldParameter{
					{Name: "namespace", Default: "argocd", Type: "string"},
					{Name: "sso_provider", Type: "select", Options: []string{"oidc", "github", "gitlab", "google"}},
					{Name: "sso_client_id", Type: "string", Required: true},
				},
			},
		},
		Documentation: ToolDocumentation{
			Overview:   "ArgoCD is a declarative, GitOps continuous delivery tool for Kubernetes.",
			QuickStart: "Push changes to Git, ArgoCD automatically syncs to cluster",
			LearnMore:  "https://argo-cd.readthedocs.io/",
		},
		Requirements: []Requirement{
			{Type: "kubernetes", MinVersion: "1.25"},
		},
		Tags: []string{"gitops", "cd", "kubernetes"},
	})

	r.Register(&RegisteredTool{
		Name:        "flux",
		DisplayName: "Flux CD",
		Description: "Open and extensible continuous delivery solution for Kubernetes",
		Category:    "orchestration",
		Versions: []ToolVersion{
			{Version: "2.2.0", ReleaseDate: time.Now()},
		},
		LatestVersion: "2.2.0",
		Scaffolds: []Scaffold{
			{
				Name:        "basic",
				Description: "Basic Flux installation",
				Category:    "basic",
				Parameters: []ScaffoldParameter{
					{Name: "namespace", Default: "flux-system", Type: "string"},
				},
			},
		},
		Documentation: ToolDocumentation{
			Overview:  "Flux is a set of continuous and progressive delivery solutions for Kubernetes.",
			LearnMore: "https://fluxcd.io/docs/",
		},
		Tags: []string{"gitops", "cd", "kubernetes"},
	})

	// Observability tools
	r.Register(&RegisteredTool{
		Name:        "prometheus",
		DisplayName: "Prometheus",
		Description: "Monitoring and alerting toolkit",
		Category:    "observability",
		Versions: []ToolVersion{
			{Version: "2.47.0", ReleaseDate: time.Now()},
		},
		LatestVersion: "2.47.0",
		Scaffolds: []Scaffold{
			{
				Name:        "basic",
				Description: "Basic Prometheus installation",
				Category:    "basic",
				Parameters: []ScaffoldParameter{
					{Name: "retention", Default: "15d", Type: "string"},
					{Name: "storage_size", Default: "50Gi", Type: "string"},
				},
			},
		},
		Documentation: ToolDocumentation{
			Overview:  "Prometheus is an open-source systems monitoring and alerting toolkit.",
			LearnMore: "https://prometheus.io/docs/",
		},
		Tags: []string{"monitoring", "metrics", "alerting"},
	})

	r.Register(&RegisteredTool{
		Name:        "grafana",
		DisplayName: "Grafana",
		Description: "Analytics and interactive visualization web application",
		Category:    "observability",
		Versions: []ToolVersion{
			{Version: "10.2.0", ReleaseDate: time.Now()},
		},
		LatestVersion: "10.2.0",
		Scaffolds: []Scaffold{
			{
				Name:        "basic",
				Description: "Basic Grafana installation",
				Category:    "basic",
				Parameters: []ScaffoldParameter{
					{Name: "admin_user", Default: "admin", Type: "string"},
					{Name: "persistence", Default: true, Type: "bool"},
				},
			},
		},
		Documentation: ToolDocumentation{
			Overview:  "Grafana is the open source analytics & monitoring solution.",
			LearnMore: "https://grafana.com/docs/",
		},
		Tags: []string{"visualization", "dashboards", "monitoring"},
	})

	// DevEx tools
	r.Register(&RegisteredTool{
		Name:        "backstage",
		DisplayName: "Backstage",
		Description: "Open platform for building developer portals",
		Category:    "devex",
		Versions: []ToolVersion{
			{Version: "1.20.0", ReleaseDate: time.Now()},
		},
		LatestVersion: "1.20.0",
		Scaffolds: []Scaffold{
			{
				Name:        "basic",
				Description: "Basic Backstage installation",
				Category:    "basic",
				Parameters: []ScaffoldParameter{
					{Name: "namespace", Default: "backstage", Type: "string"},
					{Name: "database_type", Default: "postgresql", Type: "select", Options: []string{"postgresql", "sqlite"}},
				},
			},
		},
		Documentation: ToolDocumentation{
			Overview:  "Backstage is an open platform for building developer portals.",
			LearnMore: "https://backstage.io/docs/",
		},
		Tags: []string{"developer-portal", "catalog", "devex"},
	})

	// Security tools
	r.Register(&RegisteredTool{
		Name:        "vault",
		DisplayName: "HashiCorp Vault",
		Description: "Secrets management and data protection",
		Category:    "security",
		Versions: []ToolVersion{
			{Version: "1.15.0", ReleaseDate: time.Now()},
		},
		LatestVersion: "1.15.0",
		Scaffolds: []Scaffold{
			{
				Name:        "basic",
				Description: "Basic Vault installation",
				Category:    "basic",
				Parameters: []ScaffoldParameter{
					{Name: "ha", Default: false, Type: "bool"},
					{Name: "auto_unseal", Default: false, Type: "bool"},
				},
			},
		},
		Documentation: ToolDocumentation{
			Overview:  "Vault secures, stores, and controls access to tokens, passwords, certificates, and encryption keys.",
			LearnMore: "https://www.vaultproject.io/docs",
		},
		Tags: []string{"secrets", "security", "encryption"},
	})

	r.Register(&RegisteredTool{
		Name:        "external-secrets",
		DisplayName: "External Secrets Operator",
		Description: "Synchronizes secrets from external APIs into Kubernetes",
		Category:    "security",
		Versions: []ToolVersion{
			{Version: "0.9.0", ReleaseDate: time.Now()},
		},
		LatestVersion: "0.9.0",
		Scaffolds: []Scaffold{
			{
				Name:        "basic",
				Description: "Basic External Secrets installation",
				Category:    "basic",
				Parameters: []ScaffoldParameter{
					{Name: "provider", Type: "select", Options: []string{"aws", "gcp", "azure", "vault"}},
				},
			},
		},
		Documentation: ToolDocumentation{
			Overview:  "External Secrets Operator synchronizes secrets from external APIs into Kubernetes.",
			LearnMore: "https://external-secrets.io/",
		},
		Tags: []string{"secrets", "kubernetes", "sync"},
	})

	// Infrastructure tools
	r.Register(&RegisteredTool{
		Name:        "terraform",
		DisplayName: "Terraform",
		Description: "Infrastructure as Code tool",
		Category:    "infrastructure",
		Versions: []ToolVersion{
			{Version: "1.6.0", ReleaseDate: time.Now()},
		},
		LatestVersion: "1.6.0",
		Scaffolds: []Scaffold{
			{
				Name:        "aws-eks",
				Description: "AWS EKS cluster",
				Category:    "infrastructure",
				Parameters: []ScaffoldParameter{
					{Name: "region", Default: "us-west-2", Type: "string"},
					{Name: "cluster_version", Default: "1.28", Type: "string"},
				},
			},
			{
				Name:        "gcp-gke",
				Description: "GCP GKE cluster",
				Category:    "infrastructure",
				Parameters: []ScaffoldParameter{
					{Name: "region", Default: "us-central1", Type: "string"},
					{Name: "project", Type: "string", Required: true},
				},
			},
		},
		Documentation: ToolDocumentation{
			Overview:  "Terraform enables infrastructure as code across cloud providers.",
			LearnMore: "https://www.terraform.io/docs",
		},
		Tags: []string{"iac", "infrastructure", "provisioning"},
	})
}
