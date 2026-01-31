package generator

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// PlatformGenerator generates platform YAML with environment profiles
type PlatformGenerator struct{}

// PlatformConfig represents platform configuration
type PlatformConfig struct {
	Name         string
	Organization string
	Type         string   // kubernetes, serverless, hybrid
	Environments []string
}

// NewPlatformGenerator creates a new platform generator
func NewPlatformGenerator() *PlatformGenerator {
	return &PlatformGenerator{}
}

// Generate generates platform YAML with environment profiles
func (g *PlatformGenerator) Generate(config PlatformConfig) (string, error) {
	var documents []string

	// 1. Generate base platform definition
	platform := map[string]interface{}{
		"apiVersion": "platformfoundry.io/v1",
		"kind":       "Platform",
		"metadata": map[string]interface{}{
			"name":         config.Name,
			"organization": config.Organization,
		},
		"spec": g.generatePlatformSpec(config.Type),
	}

	platformYAML, err := yaml.Marshal(platform)
	if err != nil {
		return "", fmt.Errorf("failed to marshal platform YAML: %w", err)
	}
	documents = append(documents, string(platformYAML))

	// 2. Generate environment profiles for each environment
	for _, env := range config.Environments {
		envProfile := g.generateEnvironmentProfile(config.Name, env, config.Organization)
		envYAML, err := yaml.Marshal(envProfile)
		if err != nil {
			return "", fmt.Errorf("failed to marshal environment YAML: %w", err)
		}
		documents = append(documents, string(envYAML))
	}

	// Join with YAML document separator
	return strings.Join(documents, "---\n"), nil
}

// generatePlatformSpec generates platform spec based on type
func (g *PlatformGenerator) generatePlatformSpec(platformType string) map[string]interface{} {
	switch platformType {
	case "kubernetes":
		return map[string]interface{}{
			"components": map[string]interface{}{
				"infrastructure": "base-infra",
				"orchestrator":   "base-orchestrator",
				"observability":  "base-observability",
			},
			"global": map[string]interface{}{
				"region": "us-east-1",
				"tags": map[string]string{
					"managed-by": "platformfoundry",
				},
			},
		}
	default:
		return map[string]interface{}{
			"components": map[string]interface{}{},
		}
	}
}

// generateEnvironmentProfile generates an environment profile
func (g *PlatformGenerator) generateEnvironmentProfile(platformName, envName, org string) map[string]interface{} {
	envType := "development"
	if envName == "staging" {
		envType = "staging"
	} else if envName == "prod" || envName == "production" {
		envType = "production"
	}

	profile := map[string]interface{}{
		"apiVersion": "platformfoundry.io/v1",
		"kind":       "Environment",
		"metadata": map[string]interface{}{
			"name":         fmt.Sprintf("%s-%s", platformName, envName),
			"organization": org,
		},
		"spec": map[string]interface{}{
			"type":        envType,
			"platformRef": platformName,
			"overrides":   g.generateEnvironmentOverrides(envType),
		},
	}

	return profile
}

// generateEnvironmentOverrides generates environment-specific overrides
func (g *PlatformGenerator) generateEnvironmentOverrides(envType string) map[string]interface{} {
	overrides := map[string]interface{}{}

	switch envType {
	case "development":
		overrides["infrastructure"] = map[string]interface{}{
			"nodeCount": 2,
			"nodeType":  "t3.medium",
		}
		overrides["tags"] = map[string]string{
			"environment": "dev",
			"cost-center": "development",
		}

	case "staging":
		overrides["infrastructure"] = map[string]interface{}{
			"nodeCount": 3,
			"nodeType":  "t3.large",
		}
		overrides["tags"] = map[string]string{
			"environment": "staging",
			"cost-center": "qa",
		}

	case "production":
		overrides["infrastructure"] = map[string]interface{}{
			"nodeCount": 5,
			"nodeType":  "t3.xlarge",
		}
		overrides["observability"] = map[string]interface{}{
			"prometheus": map[string]interface{}{
				"retention": "30d",
			},
		}
		overrides["tags"] = map[string]string{
			"environment": "production",
			"cost-center": "production",
		}
	}

	return overrides
}
