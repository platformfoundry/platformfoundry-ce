package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/platformfoundry/pf-ce/internal/policy"
	"github.com/platformfoundry/pf-ce/pkg/types"
)

func TestPolicy_LocalEngineIntegration(t *testing.T) {
	t.Skip("Skipping integration test requiring OPA engine - TODO: setup proper test environment")
	// Create temporary directory for policy files
	tmpDir, err := os.MkdirTemp("", "pf-policy-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a simple policy that requires a specific label
	policyContent := `package test

default allow = false

allow {
    input.metadata.labels["required-label"] == "present"
}

deny[msg] {
    not input.metadata.labels["required-label"]
    msg = "Missing required label: required-label"
}
`

	policyFile := filepath.Join(tmpDir, "test-policy.rego")
	if err := os.WriteFile(policyFile, []byte(policyContent), 0644); err != nil {
		t.Fatalf("Failed to write policy file: %v", err)
	}

	// Initialize local policy engine
	config := policy.DefaultConfig()
	config.Type = "local"
	engine, err := policy.NewLocalEngine(config)
	if err != nil {
		t.Fatalf("Failed to create local engine: %v", err)
	}

	ctx := context.Background()

	// Test 1: Load policy
	t.Run("Load Policy", func(t *testing.T) {
		err := engine.LoadPolicy(ctx, "test-policy", policyFile)
		if err != nil {
			t.Fatalf("Failed to load policy: %v", err)
		}
	})

	// Test 2: Evaluate valid resource
	t.Run("Evaluate Valid Resource", func(t *testing.T) {
		validResource := map[string]interface{}{
			"apiVersion": "platformfoundry.io/v1",
			"kind":       "Platform",
			"metadata": map[string]interface{}{
				"name": "test-platform",
				"labels": map[string]interface{}{
					"required-label": "present",
				},
			},
		}

		result, err := engine.Evaluate(ctx, "test-policy", validResource)
		if err != nil {
			t.Fatalf("Failed to evaluate policy: %v", err)
		}

		if !result.Allowed {
			t.Errorf("Expected resource to be allowed, got denied with: %v", result.Reasons)
		}

		if len(result.Reasons) > 0 {
			t.Errorf("Expected no reasons, got: %v", result.Reasons)
		}
	})

	// Test 3: Evaluate invalid resource
	t.Run("Evaluate Invalid Resource", func(t *testing.T) {
		invalidResource := map[string]interface{}{
			"apiVersion": "platformfoundry.io/v1",
			"kind":       "Platform",
			"metadata": map[string]interface{}{
				"name": "test-platform",
				"labels": map[string]interface{}{
					"other-label": "value",
				},
			},
		}

		result, err := engine.Evaluate(ctx, "test-policy", invalidResource)
		if err != nil {
			t.Fatalf("Failed to evaluate policy: %v", err)
		}

		if result.Allowed {
			t.Error("Expected resource to be denied, but it was allowed")
		}

		if len(result.Reasons) == 0 {
			t.Error("Expected reasons, got none")
		}

		if len(result.Reasons) > 0 {
			expectedMsg := "Missing required label: required-label"
			found := false
			for _, v := range result.Reasons {
				if v == expectedMsg {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected reason message '%s', got: %v", expectedMsg, result.Reasons)
			}
		}
	})
}

func TestPolicy_NamingConvention(t *testing.T) {
	t.Skip("Skipping integration test requiring OPA engine - TODO: setup proper test environment")
	// Test the resource naming policy
	tmpDir, err := os.MkdirTemp("", "pf-naming-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Copy the actual naming policy
	namingPolicyContent := `package naming

import future.keywords.if
import future.keywords.in

# Default deny
default allow = false

# Valid name pattern: lowercase alphanumeric with hyphens
valid_name_pattern := "^[a-z0-9-]+$"

# Maximum name length
max_name_length := 63

# Environment-specific prefixes
valid_prefix if {
    input.metadata.labels.environment == "production"
    startswith(input.metadata.name, "prod-")
}

valid_prefix if {
    input.metadata.labels.environment == "staging"
    startswith(input.metadata.name, "staging-")
}

valid_prefix if {
    input.metadata.labels.environment == "development"
    startswith(input.metadata.name, "dev-")
}

# Allow if all checks pass
allow if {
    regex.match(valid_name_pattern, input.metadata.name)
    count(input.metadata.name) <= max_name_length
    valid_prefix
}

# Deny rules
deny[msg] if {
    not regex.match(valid_name_pattern, input.metadata.name)
    msg = sprintf("Resource name '%s' must contain only lowercase alphanumeric characters and hyphens", [input.metadata.name])
}

deny[msg] if {
    count(input.metadata.name) > max_name_length
    msg = sprintf("Resource name '%s' exceeds maximum length of %d characters", [input.metadata.name, max_name_length])
}

deny[msg] if {
    not valid_prefix
    env := input.metadata.labels.environment
    msg = sprintf("Resource name must start with '%s-' for %s environment", [env, env])
}
`

	policyFile := filepath.Join(tmpDir, "naming.rego")
	if err := os.WriteFile(policyFile, []byte(namingPolicyContent), 0644); err != nil {
		t.Fatalf("Failed to write policy file: %v", err)
	}

	config := policy.DefaultConfig()
	config.Type = "local"
	engine, err := policy.NewLocalEngine(config)
	if err != nil {
		t.Fatalf("Failed to create local engine: %v", err)
	}

	ctx := context.Background()

	if err := engine.LoadPolicy(ctx, "naming", policyFile); err != nil {
		t.Fatalf("Failed to load naming policy: %v", err)
	}

	tests := []struct {
		name        string
		resource    map[string]interface{}
		shouldAllow bool
		description string
	}{
		{
			name: "Valid production name",
			resource: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "prod-my-platform",
					"labels": map[string]interface{}{
						"environment": "production",
					},
				},
			},
			shouldAllow: true,
			description: "Resource with valid prod- prefix",
		},
		{
			name: "Invalid production name - wrong prefix",
			resource: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "my-platform",
					"labels": map[string]interface{}{
						"environment": "production",
					},
				},
			},
			shouldAllow: false,
			description: "Production resource without prod- prefix",
		},
		{
			name: "Valid dev name",
			resource: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "dev-test-cluster",
					"labels": map[string]interface{}{
						"environment": "development",
					},
				},
			},
			shouldAllow: true,
			description: "Resource with valid dev- prefix",
		},
		{
			name: "Invalid name - uppercase",
			resource: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "Dev-Platform",
					"labels": map[string]interface{}{
						"environment": "development",
					},
				},
			},
			shouldAllow: false,
			description: "Resource with uppercase characters",
		},
		{
			name: "Invalid name - too long",
			resource: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "dev-this-is-an-extremely-long-platform-name-that-exceeds-the-maximum-allowed-length",
					"labels": map[string]interface{}{
						"environment": "development",
					},
				},
			},
			shouldAllow: false,
			description: "Resource name exceeding 63 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Evaluate(ctx, "naming", tt.resource)
			if err != nil {
				t.Fatalf("Failed to evaluate policy: %v", err)
			}

			if tt.shouldAllow && !result.Allowed {
				t.Errorf("%s: Expected allowed, got denied with: %v", tt.description, result.Reasons)
			}

			if !tt.shouldAllow && result.Allowed {
				t.Errorf("%s: Expected denied, but was allowed", tt.description)
			}
		})
	}
}

func TestPolicy_CostLimits(t *testing.T) {
	t.Skip("Skipping integration test requiring OPA engine - TODO: setup proper test environment")
	tmpDir, err := os.MkdirTemp("", "pf-cost-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Simplified cost limits policy
	costPolicyContent := `package cost

import future.keywords.if
import future.keywords.in

default allow = false

# Budget limits (monthly USD)
budget_limits := {
    "development": 500,
    "staging": 2000,
    "production": 10000
}

# Instance type costs (monthly USD)
instance_costs := {
    "t3.micro": 7,
    "t3.small": 15,
    "t3.medium": 30,
    "t3.large": 60,
    "t3.xlarge": 120
}

# Calculate estimated cost
estimated_cost := cost if {
    infrastructure := input.spec.infrastructure
    instance_type := infrastructure.instanceType
    node_count := infrastructure.nodeCount
    cost = instance_costs[instance_type] * node_count
}

# Get budget for environment
budget := budget_limits[input.metadata.labels.environment]

# Allow if within budget
allow if {
    estimated_cost <= budget
}

# Deny if exceeds budget
deny[msg] if {
    estimated_cost > budget
    msg = sprintf("Estimated cost $%d/month exceeds budget of $%d/month", [estimated_cost, budget])
}

# Warning if approaching budget
warnings[msg] if {
    estimated_cost > budget * 0.8
    estimated_cost <= budget
    msg = sprintf("Cost $%d/month is approaching budget limit of $%d/month", [estimated_cost, budget])
}
`

	policyFile := filepath.Join(tmpDir, "cost.rego")
	if err := os.WriteFile(policyFile, []byte(costPolicyContent), 0644); err != nil {
		t.Fatalf("Failed to write policy file: %v", err)
	}

	config := policy.DefaultConfig()
	config.Type = "local"
	engine, err := policy.NewLocalEngine(config)
	if err != nil {
		t.Fatalf("Failed to create local engine: %v", err)
	}

	ctx := context.Background()

	if err := engine.LoadPolicy(ctx, "cost", policyFile); err != nil {
		t.Fatalf("Failed to load cost policy: %v", err)
	}

	tests := []struct {
		name        string
		resource    map[string]interface{}
		shouldAllow bool
		shouldWarn  bool
		description string
	}{
		{
			name: "Development within budget",
			resource: map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": map[string]interface{}{
						"environment": "development",
					},
				},
				"spec": map[string]interface{}{
					"infrastructure": map[string]interface{}{
						"nodeCount":    2.0,
						"instanceType": "t3.micro",
					},
				},
			},
			shouldAllow: true,
			shouldWarn:  false,
			description: "Dev environment with 2 t3.micro instances ($14/mo < $500 budget)",
		},
		{
			name: "Production within budget",
			resource: map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": map[string]interface{}{
						"environment": "production",
					},
				},
				"spec": map[string]interface{}{
					"infrastructure": map[string]interface{}{
						"nodeCount":    5.0,
						"instanceType": "t3.large",
					},
				},
			},
			shouldAllow: true,
			shouldWarn:  false,
			description: "Prod environment with 5 t3.large instances ($300/mo < $10000 budget)",
		},
		{
			name: "Development exceeding budget",
			resource: map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": map[string]interface{}{
						"environment": "development",
					},
				},
				"spec": map[string]interface{}{
					"infrastructure": map[string]interface{}{
						"nodeCount":    10.0,
						"instanceType": "t3.xlarge",
					},
				},
			},
			shouldAllow: false,
			shouldWarn:  false,
			description: "Dev environment with 10 t3.xlarge instances ($1200/mo > $500 budget)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Evaluate(ctx, "cost", tt.resource)
			if err != nil {
				t.Fatalf("Failed to evaluate policy: %v", err)
			}

			if tt.shouldAllow && !result.Allowed {
				t.Errorf("%s: Expected allowed, got denied with: %v", tt.description, result.Reasons)
			}

			if !tt.shouldAllow && result.Allowed {
				t.Errorf("%s: Expected denied, but was allowed", tt.description)
			}
		})
	}
}

func TestPolicy_WithPlatformType(t *testing.T) {
	t.Skip("Skipping integration test requiring OPA engine - TODO: setup proper test environment")
	// Test policy evaluation with actual Platform type
	config := policy.DefaultConfig()
	config.Type = "local"
	engine, err := policy.NewLocalEngine(config)
	if err != nil {
		t.Fatalf("Failed to create local engine: %v", err)
	}

	// Create a simple policy for testing with Platform
	tmpDir, err := os.MkdirTemp("", "pf-platform-policy-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	policyContent := `package platform_test

default allow = true

# Require team label
deny[msg] {
    not input.metadata.labels.team
    msg = "Platform must have a 'team' label"
}

# Require at least one component
deny[msg] {
    not input.spec.components
    msg = "Platform must define at least one component"
}

# Deny if any deny rules fire
allow = false {
    count(deny) > 0
}
`

	policyFile := filepath.Join(tmpDir, "platform.rego")
	if err := os.WriteFile(policyFile, []byte(policyContent), 0644); err != nil {
		t.Fatalf("Failed to write policy file: %v", err)
	}

	ctx := context.Background()

	if err := engine.LoadPolicy(ctx, "platform_test", policyFile); err != nil {
		t.Fatalf("Failed to load platform policy: %v", err)
	}

	t.Run("Valid Platform", func(t *testing.T) {
		platform := &types.Platform{
			APIVersion: "platformfoundry.io/v1",
			Kind:       "Platform",
			Metadata: types.Metadata{
				Name: "test-platform",
				Labels: map[string]string{
					"team": "platform-engineering",
				},
			},
			Spec: types.PlatformSpec{
				Components: types.ComponentReferences{
					Infrastructure: "aws-infra",
				},
			},
		}

		result, err := engine.Evaluate(ctx, "platform_test", platform)
		if err != nil {
			t.Fatalf("Failed to evaluate policy: %v", err)
		}

		if !result.Allowed {
			t.Errorf("Expected platform to be allowed, got denied with: %v", result.Reasons)
		}
	})

	t.Run("Invalid Platform - Missing Team Label", func(t *testing.T) {
		platform := &types.Platform{
			APIVersion: "platformfoundry.io/v1",
			Kind:       "Platform",
			Metadata: types.Metadata{
				Name: "test-platform",
			},
			Spec: types.PlatformSpec{
				Components: types.ComponentReferences{
					Infrastructure: "aws-infra",
				},
			},
		}

		result, err := engine.Evaluate(ctx, "platform_test", platform)
		if err != nil {
			t.Fatalf("Failed to evaluate policy: %v", err)
		}

		if result.Allowed {
			t.Error("Expected platform to be denied, but it was allowed")
		}

		if len(result.Reasons) == 0 {
			t.Error("Expected violations for missing team label")
		}
	})
}
