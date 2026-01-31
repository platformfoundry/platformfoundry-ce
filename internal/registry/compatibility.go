package registry

import (
	"sync"
)

// CompatibilityMatrix tracks which tools work together
type CompatibilityMatrix struct {
	rules []CompatibilityRule
	mu    sync.RWMutex
}

// CompatibilityRule defines a compatibility relationship
type CompatibilityRule struct {
	ToolA      string
	ToolB      string
	Compatible bool
	Notes      string
	Conditions []Condition
}

// Condition for conditional compatibility
type Condition struct {
	Type  string // version, config, cloud
	Key   string
	Value string
	Op    string // eq, ne, gt, lt, gte, lte
}

// CompatibilityIssue represents a compatibility problem
type CompatibilityIssue struct {
	ToolA    string
	ToolB    string
	Message  string
	Severity string // warning, error
}

// NewCompatibilityMatrix creates a new compatibility matrix with built-in rules
func NewCompatibilityMatrix() *CompatibilityMatrix {
	m := &CompatibilityMatrix{
		rules: []CompatibilityRule{},
	}
	m.loadBuiltinRules()
	return m
}

// loadBuiltinRules adds built-in compatibility rules
func (m *CompatibilityMatrix) loadBuiltinRules() {
	m.rules = []CompatibilityRule{
		// GitOps tools - usually pick one
		{
			ToolA:      "argocd",
			ToolB:      "flux",
			Compatible: false,
			Notes:      "Both are GitOps tools, use one or the other to avoid conflicts",
		},

		// ArgoCD + various CI tools
		{
			ToolA:      "argocd",
			ToolB:      "github-actions",
			Compatible: true,
			Notes:      "Excellent combination - CI builds images, ArgoCD deploys",
		},
		{
			ToolA:      "argocd",
			ToolB:      "jenkins",
			Compatible: true,
			Notes:      "Works well together for CI/CD pipeline",
		},
		{
			ToolA:      "argocd",
			ToolB:      "tekton",
			Compatible: true,
			Notes:      "Cloud-native CI/CD combination",
		},

		// Flux + CI tools
		{
			ToolA:      "flux",
			ToolB:      "github-actions",
			Compatible: true,
			Notes:      "Works well with GitHub Actions for CI",
		},

		// Observability stack
		{
			ToolA:      "prometheus",
			ToolB:      "grafana",
			Compatible: true,
			Notes:      "Standard combination, Grafana visualizes Prometheus metrics",
		},
		{
			ToolA:      "prometheus",
			ToolB:      "loki",
			Compatible: true,
			Notes:      "Prometheus for metrics, Loki for logs - great combination",
		},
		{
			ToolA:      "prometheus",
			ToolB:      "datadog",
			Compatible: true,
			Notes:      "Can use both, but may have overlapping functionality and costs",
		},
		{
			ToolA:      "grafana",
			ToolB:      "loki",
			Compatible: true,
			Notes:      "Grafana has native support for Loki log queries",
		},

		// Secrets management
		{
			ToolA:      "vault",
			ToolB:      "external-secrets",
			Compatible: true,
			Notes:      "External Secrets can sync from Vault to Kubernetes secrets",
		},
		{
			ToolA:      "vault",
			ToolB:      "argocd",
			Compatible: true,
			Notes:      "ArgoCD can use Vault for secrets via plugins",
		},

		// Service mesh - pick one
		{
			ToolA:      "istio",
			ToolB:      "linkerd",
			Compatible: false,
			Notes:      "Both are service meshes, use one or the other",
		},
		{
			ToolA:      "istio",
			ToolB:      "consul-connect",
			Compatible: false,
			Notes:      "Both provide service mesh functionality",
		},

		// Ingress controllers - pick one (usually)
		{
			ToolA:      "nginx-ingress",
			ToolB:      "traefik",
			Compatible: true,
			Notes:      "Can coexist but may cause confusion; typically pick one",
		},

		// Policy engines
		{
			ToolA:      "opa-gatekeeper",
			ToolB:      "kyverno",
			Compatible: true,
			Notes:      "Can coexist but typically pick one for consistency",
		},

		// DevEx + Orchestration
		{
			ToolA:      "backstage",
			ToolB:      "argocd",
			Compatible: true,
			Notes:      "Backstage can integrate with ArgoCD for deployment visibility",
		},
		{
			ToolA:      "backstage",
			ToolB:      "prometheus",
			Compatible: true,
			Notes:      "Backstage can show Prometheus metrics in scorecards",
		},

		// Infrastructure
		{
			ToolA:      "terraform",
			ToolB:      "crossplane",
			Compatible: true,
			Notes:      "Can coexist - Terraform for cloud, Crossplane for K8s-native IaC",
		},
	}
}

// AddRule adds a compatibility rule
func (m *CompatibilityMatrix) AddRule(rule CompatibilityRule) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rules = append(m.rules, rule)
}

// Check returns compatibility status between two tools
func (m *CompatibilityMatrix) Check(toolA, toolB string) (bool, string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, rule := range m.rules {
		if (rule.ToolA == toolA && rule.ToolB == toolB) ||
			(rule.ToolA == toolB && rule.ToolB == toolA) {
			return rule.Compatible, rule.Notes
		}
	}
	// Default to compatible if no rule exists
	return true, ""
}

// ValidateSet checks if a set of tools are all compatible
func (m *CompatibilityMatrix) ValidateSet(tools []string) []CompatibilityIssue {
	var issues []CompatibilityIssue

	for i := 0; i < len(tools); i++ {
		for j := i + 1; j < len(tools); j++ {
			compatible, notes := m.Check(tools[i], tools[j])
			if !compatible {
				issues = append(issues, CompatibilityIssue{
					ToolA:    tools[i],
					ToolB:    tools[j],
					Message:  notes,
					Severity: "error",
				})
			}
		}
	}

	return issues
}

// GetRules returns all compatibility rules
func (m *CompatibilityMatrix) GetRules() []CompatibilityRule {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rules := make([]CompatibilityRule, len(m.rules))
	copy(rules, m.rules)
	return rules
}

// GetRulesFor returns all rules involving a specific tool
func (m *CompatibilityMatrix) GetRulesFor(tool string) []CompatibilityRule {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var rules []CompatibilityRule
	for _, rule := range m.rules {
		if rule.ToolA == tool || rule.ToolB == tool {
			rules = append(rules, rule)
		}
	}
	return rules
}

// GetIncompatibleTools returns tools that are incompatible with the given tool
func (m *CompatibilityMatrix) GetIncompatibleTools(tool string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var incompatible []string
	for _, rule := range m.rules {
		if !rule.Compatible {
			if rule.ToolA == tool {
				incompatible = append(incompatible, rule.ToolB)
			} else if rule.ToolB == tool {
				incompatible = append(incompatible, rule.ToolA)
			}
		}
	}
	return incompatible
}

// GetCompatibleTools returns tools that are known to be compatible with the given tool
func (m *CompatibilityMatrix) GetCompatibleTools(tool string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var compatible []string
	for _, rule := range m.rules {
		if rule.Compatible {
			if rule.ToolA == tool {
				compatible = append(compatible, rule.ToolB)
			} else if rule.ToolB == tool {
				compatible = append(compatible, rule.ToolA)
			}
		}
	}
	return compatible
}

// SuggestAlternatives suggests alternative tools when incompatibility is found
func (m *CompatibilityMatrix) SuggestAlternatives(toolA, toolB string) []string {
	// Define tool categories and alternatives
	alternatives := map[string][]string{
		"argocd":         {"flux"},
		"flux":           {"argocd"},
		"istio":          {"linkerd", "consul-connect"},
		"linkerd":        {"istio", "consul-connect"},
		"prometheus":     {"datadog", "newrelic"},
		"datadog":        {"prometheus", "newrelic"},
		"nginx-ingress":  {"traefik", "kong"},
		"traefik":        {"nginx-ingress", "kong"},
		"opa-gatekeeper": {"kyverno"},
		"kyverno":        {"opa-gatekeeper"},
	}

	compatible, _ := m.Check(toolA, toolB)
	if compatible {
		return nil // No alternatives needed
	}

	// Return alternatives for the second tool
	if alts, ok := alternatives[toolB]; ok {
		return alts
	}

	return nil
}
