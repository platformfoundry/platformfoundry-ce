package generator

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// OrganizationGenerator generates organization YAML
type OrganizationGenerator struct{}

// OrganizationConfig represents organization configuration
type OrganizationConfig struct {
	Name        string
	DisplayName string
	Owner       string
	Email       string
}

// NewOrganizationGenerator creates a new organization generator
func NewOrganizationGenerator() *OrganizationGenerator {
	return &OrganizationGenerator{}
}

// Generate generates organization YAML
func (g *OrganizationGenerator) Generate(config OrganizationConfig) (string, error) {
	org := map[string]interface{}{
		"apiVersion": "platformfoundry.io/v1",
		"kind":       "Organization",
		"metadata": map[string]interface{}{
			"name": config.Name,
		},
		"spec": map[string]interface{}{
			"displayName": config.DisplayName,
			"owner":       config.Owner,
		},
	}

	if config.Email != "" {
		org["spec"].(map[string]interface{})["contact"] = map[string]interface{}{
			"email": config.Email,
		}
	}

	data, err := yaml.Marshal(org)
	if err != nil {
		return "", fmt.Errorf("failed to marshal YAML: %w", err)
	}

	return string(data), nil
}
