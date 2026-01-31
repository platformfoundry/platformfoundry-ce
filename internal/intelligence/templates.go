package intelligence

import (
	"encoding/json"
	"fmt"
	"os"
)

// Template represents a developer portal template
type Template struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	TechStack   []string               `json:"tech_stack"`    // Required tech stack patterns
	Features    []string               `json:"features"`      // Portal features included
	Plugins     []string               `json:"plugins"`       // Backstage plugins
	Config      map[string]interface{} `json:"config"`        // Template-specific config
	Tags        []string               `json:"tags"`          // Searchable tags
}

// TemplateRepository manages portal templates
type TemplateRepository struct {
	templates map[string]Template
}

// NewTemplateRepository creates a new template repository
func NewTemplateRepository() *TemplateRepository {
	return &TemplateRepository{
		templates: make(map[string]Template),
	}
}

// LoadFromFile loads templates from a JSON file
func (tr *TemplateRepository) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read templates file: %w", err)
	}

	var templatesData struct {
		Templates []Template `json:"templates"`
	}
	if err := json.Unmarshal(data, &templatesData); err != nil {
		return fmt.Errorf("failed to parse templates JSON: %w", err)
	}

	for _, template := range templatesData.Templates {
		tr.templates[template.ID] = template
	}

	return nil
}

// LoadDefaults loads default embedded templates
func (tr *TemplateRepository) LoadDefaults() {
	templates := getDefaultTemplates()
	for _, template := range templates {
		tr.templates[template.ID] = template
	}
}

// Get retrieves a template by ID
func (tr *TemplateRepository) Get(id string) (*Template, error) {
	template, ok := tr.templates[id]
	if !ok {
		return nil, fmt.Errorf("template not found: %s", id)
	}
	return &template, nil
}

// List returns all templates
func (tr *TemplateRepository) List() []Template {
	templates := make([]Template, 0, len(tr.templates))
	for _, template := range tr.templates {
		templates = append(templates, template)
	}
	return templates
}

// Search finds templates by tag or tech stack
func (tr *TemplateRepository) Search(query string) []Template {
	results := make([]Template, 0)

	for _, template := range tr.templates {
		// Search in tags
		for _, tag := range template.Tags {
			if tag == query {
				results = append(results, template)
				break
			}
		}

		// Search in tech stack
		for _, tech := range template.TechStack {
			if tech == query {
				// Check if not already added
				found := false
				for _, r := range results {
					if r.ID == template.ID {
						found = true
						break
					}
				}
				if !found {
					results = append(results, template)
				}
				break
			}
		}
	}

	return results
}

// getDefaultTemplates returns embedded default templates
func getDefaultTemplates() []Template {
	return []Template{
		{
			ID:          "aws-k8s-full",
			Name:        "AWS Kubernetes Full Stack",
			Description: "Complete developer portal for AWS with Kubernetes, GitOps, and full observability",
			TechStack:   []string{"aws", "kubernetes", "argocd", "prometheus", "grafana"},
			Features: []string{
				"catalog",
				"docs",
				"scaffolder",
				"techdocs",
				"kubernetes",
				"cost-insights",
				"search",
			},
			Plugins: []string{
				"@backstage/plugin-kubernetes",
				"@backstage/plugin-catalog",
				"@backstage/plugin-scaffolder",
				"@backstage/plugin-techdocs",
				"@roadiehq/backstage-plugin-aws",
				"@roadiehq/backstage-plugin-argo-cd",
				"@backstage/plugin-cost-insights",
			},
			Config: map[string]interface{}{
				"kubernetes": map[string]interface{}{
					"clusterLocatorMethod": "catalog",
				},
				"argocd": map[string]interface{}{
					"appLocatorMethod": "config",
				},
				"costInsights": map[string]interface{}{
					"enabled": true,
					"provider": "aws",
				},
			},
			Tags: []string{"aws", "kubernetes", "gitops", "full-stack", "production"},
		},
		{
			ID:          "gcp-k8s-full",
			Name:        "GCP Kubernetes Full Stack",
			Description: "Complete developer portal for GCP with Kubernetes and observability",
			TechStack:   []string{"gcp", "kubernetes", "argocd", "prometheus", "grafana"},
			Features: []string{
				"catalog",
				"docs",
				"scaffolder",
				"techdocs",
				"kubernetes",
				"search",
			},
			Plugins: []string{
				"@backstage/plugin-kubernetes",
				"@backstage/plugin-catalog",
				"@backstage/plugin-scaffolder",
				"@backstage/plugin-techdocs",
				"@backstage/plugin-gcp-projects",
				"@roadiehq/backstage-plugin-argo-cd",
			},
			Config: map[string]interface{}{
				"kubernetes": map[string]interface{}{
					"clusterLocatorMethod": "catalog",
				},
				"argocd": map[string]interface{}{
					"appLocatorMethod": "config",
				},
			},
			Tags: []string{"gcp", "kubernetes", "gitops", "full-stack", "production"},
		},
		{
			ID:          "azure-k8s-full",
			Name:        "Azure Kubernetes Full Stack",
			Description: "Complete developer portal for Azure with AKS and observability",
			TechStack:   []string{"azure", "kubernetes", "argocd", "prometheus", "grafana"},
			Features: []string{
				"catalog",
				"docs",
				"scaffolder",
				"techdocs",
				"kubernetes",
				"search",
			},
			Plugins: []string{
				"@backstage/plugin-kubernetes",
				"@backstage/plugin-catalog",
				"@backstage/plugin-scaffolder",
				"@backstage/plugin-techdocs",
				"@backstage/plugin-azure-devops",
				"@roadiehq/backstage-plugin-argo-cd",
			},
			Config: map[string]interface{}{
				"kubernetes": map[string]interface{}{
					"clusterLocatorMethod": "catalog",
				},
				"argocd": map[string]interface{}{
					"appLocatorMethod": "config",
				},
			},
			Tags: []string{"azure", "kubernetes", "gitops", "full-stack", "production"},
		},
		{
			ID:          "multi-cloud",
			Name:        "Multi-Cloud Platform",
			Description: "Portal for multi-cloud platforms with unified observability",
			TechStack:   []string{"aws", "gcp", "azure", "kubernetes"},
			Features: []string{
				"catalog",
				"docs",
				"scaffolder",
				"techdocs",
				"kubernetes",
				"cost-insights",
				"search",
			},
			Plugins: []string{
				"@backstage/plugin-kubernetes",
				"@backstage/plugin-catalog",
				"@backstage/plugin-scaffolder",
				"@backstage/plugin-techdocs",
				"@roadiehq/backstage-plugin-aws",
				"@backstage/plugin-gcp-projects",
				"@backstage/plugin-azure-devops",
			},
			Config: map[string]interface{}{
				"kubernetes": map[string]interface{}{
					"clusterLocatorMethod": "catalog",
				},
			},
			Tags: []string{"multi-cloud", "kubernetes", "enterprise", "production"},
		},
		{
			ID:          "k8s-basic",
			Name:        "Basic Kubernetes Portal",
			Description: "Simple portal for Kubernetes platforms with essential features",
			TechStack:   []string{"kubernetes"},
			Features: []string{
				"catalog",
				"docs",
				"kubernetes",
			},
			Plugins: []string{
				"@backstage/plugin-kubernetes",
				"@backstage/plugin-catalog",
				"@backstage/plugin-techdocs",
			},
			Config: map[string]interface{}{
				"kubernetes": map[string]interface{}{
					"clusterLocatorMethod": "catalog",
				},
			},
			Tags: []string{"kubernetes", "basic", "starter"},
		},
		{
			ID:          "minimal",
			Name:        "Minimal Portal",
			Description: "Minimal developer portal with catalog and documentation",
			TechStack:   []string{},
			Features: []string{
				"catalog",
				"docs",
			},
			Plugins: []string{
				"@backstage/plugin-catalog",
				"@backstage/plugin-techdocs",
			},
			Config: map[string]interface{}{},
			Tags:    []string{"minimal", "starter", "basic"},
		},
	}
}

// SaveTemplates saves templates to a JSON file
func SaveTemplates(templates []Template, path string) error {
	data := struct {
		Templates []Template `json:"templates"`
	}{
		Templates: templates,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal templates: %w", err)
	}

	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write templates file: %w", err)
	}

	return nil
}
