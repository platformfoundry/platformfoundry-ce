package intelligence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Recommendation represents a portal configuration recommendation
type Recommendation struct {
	Template     string                 `json:"template"`
	Features     []string               `json:"features"`
	Integrations []string               `json:"integrations"`
	Reason       string                 `json:"reason"`
	Confidence   float64                `json:"confidence"` // 0.0 to 1.0
	Details      map[string]interface{} `json:"details"`
}

// Rule represents a recommendation rule
type Rule struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Conditions  map[string]interface{} `json:"conditions"`
	Actions     map[string]interface{} `json:"actions"`
	Priority    int                    `json:"priority"`
	Confidence  float64                `json:"confidence"`
}

// Recommender provides intelligent portal recommendations based on tech stack
type Recommender struct {
	rules []Rule
}

// NewRecommender creates a new recommender with loaded rules
func NewRecommender(rulesPath string) (*Recommender, error) {
	r := &Recommender{
		rules: make([]Rule, 0),
	}

	if rulesPath == "" {
		// Use default embedded rules
		r.rules = getDefaultRules()
		return r, nil
	}

	// Load rules from file
	data, err := os.ReadFile(rulesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read rules file: %w", err)
	}

	var rulesData struct {
		Rules []Rule `json:"rules"`
	}
	if err := json.Unmarshal(data, &rulesData); err != nil {
		return nil, fmt.Errorf("failed to parse rules JSON: %w", err)
	}

	r.rules = rulesData.Rules
	return r, nil
}

// Recommend generates recommendations based on tech stack analysis
func (r *Recommender) Recommend(ts *TechStack) *Recommendation {
	rec := &Recommendation{
		Features:     make([]string, 0),
		Integrations: make([]string, 0),
		Details:      make(map[string]interface{}),
	}

	// Match tech stack against rules
	matchedRules := r.matchRules(ts)

	if len(matchedRules) == 0 {
		// No rules matched, return minimal recommendation
		rec.Template = "minimal"
		rec.Features = []string{"catalog", "docs"}
		rec.Reason = "No specific tech stack pattern detected, recommending minimal setup"
		rec.Confidence = 0.5
		return rec
	}

	// Use highest priority/confidence rule
	bestRule := matchedRules[0]
	rec.Template = bestRule.Actions["template"].(string)
	rec.Confidence = bestRule.Confidence

	// Extract features
	if features, ok := bestRule.Actions["features"].([]interface{}); ok {
		for _, f := range features {
			rec.Features = append(rec.Features, f.(string))
		}
	}

	// Extract integrations
	if integrations, ok := bestRule.Actions["integrations"].([]interface{}); ok {
		for _, i := range integrations {
			rec.Integrations = append(rec.Integrations, i.(string))
		}
	}

	// Build reason
	rec.Reason = buildReason(ts, bestRule)

	// Add details
	rec.Details["cloud"] = ts.CloudProvider
	rec.Details["orchestrator"] = ts.Orchestrator
	rec.Details["observability"] = ts.ObservabilityTools
	rec.Details["matched_rule"] = bestRule.Name

	return rec
}

// matchRules matches tech stack against all rules and returns matched rules sorted by priority
func (r *Recommender) matchRules(ts *TechStack) []Rule {
	matched := make([]Rule, 0)

	for _, rule := range r.rules {
		if r.evaluateConditions(rule.Conditions, ts) {
			matched = append(matched, rule)
		}
	}

	// Sort by priority (higher first), then by confidence
	for i := 0; i < len(matched)-1; i++ {
		for j := i + 1; j < len(matched); j++ {
			if matched[i].Priority < matched[j].Priority ||
				(matched[i].Priority == matched[j].Priority && matched[i].Confidence < matched[j].Confidence) {
				matched[i], matched[j] = matched[j], matched[i]
			}
		}
	}

	return matched
}

// evaluateConditions evaluates rule conditions against tech stack
func (r *Recommender) evaluateConditions(conditions map[string]interface{}, ts *TechStack) bool {
	for key, value := range conditions {
		switch key {
		case "cloud":
			if !matchString(value.(string), ts.CloudProvider) {
				return false
			}
		case "orchestrator":
			if !matchString(value.(string), ts.Orchestrator) {
				return false
			}
		case "has_monitoring":
			if value.(bool) != ts.HasMonitoring {
				return false
			}
		case "has_logging":
			if value.(bool) != ts.HasLogging {
				return false
			}
		case "has_tracing":
			if value.(bool) != ts.HasTracing {
				return false
			}
		case "observability_tools":
			tools := value.([]interface{})
			if !matchAny(tools, ts.ObservabilityTools) {
				return false
			}
		}
	}
	return true
}

// matchString matches a pattern against a value (supports wildcards)
func matchString(pattern, value string) bool {
	if pattern == "*" {
		return value != ""
	}
	return strings.EqualFold(pattern, value)
}

// matchAny checks if any of the required values are in the actual values
func matchAny(required []interface{}, actual []string) bool {
	if len(required) == 0 {
		return true
	}
	for _, r := range required {
		reqStr := r.(string)
		for _, a := range actual {
			if strings.EqualFold(reqStr, a) {
				return true
			}
		}
	}
	return false
}

// buildReason builds a human-readable reason for the recommendation
func buildReason(ts *TechStack, rule Rule) string {
	parts := make([]string, 0)

	if ts.CloudProvider != "" {
		parts = append(parts, fmt.Sprintf("Detected %s cloud infrastructure", ts.CloudProvider))
	}

	if ts.Orchestrator != "" {
		parts = append(parts, fmt.Sprintf("using %s for GitOps", ts.Orchestrator))
	}

	if len(ts.ObservabilityTools) > 0 {
		parts = append(parts, fmt.Sprintf("with %s observability stack", strings.Join(ts.ObservabilityTools, ", ")))
	}

	reason := strings.Join(parts, ", ")
	if reason != "" {
		reason = "Recommended " + rule.Actions["template"].(string) + " template based on: " + reason
	} else {
		reason = "Recommended " + rule.Actions["template"].(string) + " template"
	}

	return reason
}

// getDefaultRules returns embedded default rules
func getDefaultRules() []Rule {
	return []Rule{
		{
			Name:        "AWS Kubernetes Full Stack",
			Description: "AWS with K8s, GitOps, and full observability",
			Conditions: map[string]interface{}{
				"cloud":          "aws",
				"has_monitoring": true,
			},
			Actions: map[string]interface{}{
				"template": "aws-k8s-full",
				"features": []interface{}{
					"catalog",
					"docs",
					"scaffolder",
					"techdocs",
					"kubernetes",
					"cost-insights",
				},
				"integrations": []interface{}{
					"github",
					"argocd",
					"prometheus",
					"grafana",
					"aws",
				},
			},
			Priority:   100,
			Confidence: 0.9,
		},
		{
			Name:        "GCP Kubernetes Full Stack",
			Description: "GCP with K8s, GitOps, and full observability",
			Conditions: map[string]interface{}{
				"cloud":          "gcp",
				"has_monitoring": true,
			},
			Actions: map[string]interface{}{
				"template": "gcp-k8s-full",
				"features": []interface{}{
					"catalog",
					"docs",
					"scaffolder",
					"techdocs",
					"kubernetes",
				},
				"integrations": []interface{}{
					"github",
					"argocd",
					"prometheus",
					"grafana",
					"gcp",
				},
			},
			Priority:   100,
			Confidence: 0.9,
		},
		{
			Name:        "Azure Kubernetes Full Stack",
			Description: "Azure with K8s, GitOps, and full observability",
			Conditions: map[string]interface{}{
				"cloud":          "azure",
				"has_monitoring": true,
			},
			Actions: map[string]interface{}{
				"template": "azure-k8s-full",
				"features": []interface{}{
					"catalog",
					"docs",
					"scaffolder",
					"techdocs",
					"kubernetes",
				},
				"integrations": []interface{}{
					"github",
					"argocd",
					"prometheus",
					"grafana",
					"azure",
				},
			},
			Priority:   100,
			Confidence: 0.9,
		},
		{
			Name:        "Basic Kubernetes Setup",
			Description: "Any cloud with basic K8s",
			Conditions: map[string]interface{}{
				"cloud": "*",
			},
			Actions: map[string]interface{}{
				"template": "k8s-basic",
				"features": []interface{}{
					"catalog",
					"docs",
					"kubernetes",
				},
				"integrations": []interface{}{
					"github",
					"argocd",
				},
			},
			Priority:   50,
			Confidence: 0.7,
		},
		{
			Name:        "Minimal Setup",
			Description: "Minimal portal for any platform",
			Conditions:  map[string]interface{}{},
			Actions: map[string]interface{}{
				"template": "minimal",
				"features": []interface{}{
					"catalog",
					"docs",
				},
				"integrations": []interface{}{
					"github",
				},
			},
			Priority:   10,
			Confidence: 0.5,
		},
	}
}

// SaveRules saves rules to a JSON file
func SaveRules(rules []Rule, path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data := struct {
		Rules []Rule `json:"rules"`
	}{
		Rules: rules,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal rules: %w", err)
	}

	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write rules file: %w", err)
	}

	return nil
}
