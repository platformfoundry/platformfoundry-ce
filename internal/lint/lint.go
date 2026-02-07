// Package lint provides configuration linting and best practices checks.
package lint

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Severity represents the severity of a lint issue
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// Issue represents a single lint issue
type Issue struct {
	Severity   Severity `json:"severity"`
	Rule       string   `json:"rule"`
	Message    string   `json:"message"`
	File       string   `json:"file,omitempty"`
	Line       int      `json:"line,omitempty"`
	Column     int      `json:"column,omitempty"`
	Suggestion string   `json:"suggestion,omitempty"`
}

// Result contains all lint issues for a file
type Result struct {
	File    string  `json:"file"`
	Issues  []Issue `json:"issues"`
	Summary struct {
		Errors   int `json:"errors"`
		Warnings int `json:"warnings"`
		Info     int `json:"info"`
	} `json:"summary"`
}

// Rule defines a linting rule
type Rule struct {
	ID          string
	Name        string
	Description string
	Severity    Severity
	Check       func(config map[string]interface{}, file string) []Issue
}

// Linter performs configuration linting
type Linter struct {
	rules []Rule
}

// New creates a new Linter with default rules
func New() *Linter {
	l := &Linter{
		rules: make([]Rule, 0),
	}

	// Register default rules
	l.RegisterRule(ruleRequireAPIVersion())
	l.RegisterRule(ruleRequireKind())
	l.RegisterRule(ruleRequireMetadata())
	l.RegisterRule(ruleValidateResourceName())
	l.RegisterRule(ruleCheckEmptySpec())
	l.RegisterRule(ruleReplicaCount())
	l.RegisterRule(ruleResourceLimits())
	l.RegisterRule(ruleImageTag())
	l.RegisterRule(ruleNamespaceExplicit())
	l.RegisterRule(ruleLabelsPresent())
	l.RegisterRule(ruleSecurityContext())
	l.RegisterRule(ruleHardcodedSecrets())

	return l
}

// RegisterRule adds a new rule to the linter
func (l *Linter) RegisterRule(rule Rule) {
	l.rules = append(l.rules, rule)
}

// Lint checks a configuration file content
func (l *Linter) Lint(content []byte, filename string) (*Result, error) {
	result := &Result{
		File:   filename,
		Issues: make([]Issue, 0),
	}

	var config map[string]interface{}
	if err := yaml.Unmarshal(content, &config); err != nil {
		result.Issues = append(result.Issues, Issue{
			Severity: SeverityError,
			Rule:     "yaml-parse",
			Message:  fmt.Sprintf("Failed to parse YAML: %v", err),
			File:     filename,
		})
		result.Summary.Errors++
		return result, nil
	}

	// Run all rules
	for _, rule := range l.rules {
		issues := rule.Check(config, filename)
		for _, issue := range issues {
			issue.File = filename
			result.Issues = append(result.Issues, issue)

			switch issue.Severity {
			case SeverityError:
				result.Summary.Errors++
			case SeverityWarning:
				result.Summary.Warnings++
			case SeverityInfo:
				result.Summary.Info++
			}
		}
	}

	return result, nil
}

// LintMultiple checks multiple documents in a single file
func (l *Linter) LintMultiple(content []byte, filename string) (*Result, error) {
	result := &Result{
		File:   filename,
		Issues: make([]Issue, 0),
	}

	// Split by YAML document separator
	docs := strings.Split(string(content), "\n---")

	for i, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		docResult, err := l.Lint([]byte(doc), fmt.Sprintf("%s[%d]", filename, i))
		if err != nil {
			return nil, err
		}

		result.Issues = append(result.Issues, docResult.Issues...)
		result.Summary.Errors += docResult.Summary.Errors
		result.Summary.Warnings += docResult.Summary.Warnings
		result.Summary.Info += docResult.Summary.Info
	}

	return result, nil
}

// Format returns a formatted string representation of lint results
func (r *Result) Format() string {
	var sb strings.Builder

	if len(r.Issues) == 0 {
		sb.WriteString(fmt.Sprintf("%s: No issues found\n", r.File))
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("%s:\n", r.File))

	for _, issue := range r.Issues {
		icon := getIssueIcon(issue.Severity)
		location := ""
		if issue.Line > 0 {
			location = fmt.Sprintf(":%d", issue.Line)
			if issue.Column > 0 {
				location += fmt.Sprintf(":%d", issue.Column)
			}
		}

		sb.WriteString(fmt.Sprintf("  %s [%s] %s%s\n", icon, issue.Rule, issue.Message, location))

		if issue.Suggestion != "" {
			sb.WriteString(fmt.Sprintf("     Suggestion: %s\n", issue.Suggestion))
		}
	}

	sb.WriteString(fmt.Sprintf("\n  Summary: %d errors, %d warnings, %d info\n",
		r.Summary.Errors, r.Summary.Warnings, r.Summary.Info))

	return sb.String()
}

func getIssueIcon(severity Severity) string {
	switch severity {
	case SeverityError:
		return "[ERR]"
	case SeverityWarning:
		return "[WARN]"
	case SeverityInfo:
		return "[INFO]"
	default:
		return "[?]"
	}
}

// Default rules

func ruleRequireAPIVersion() Rule {
	return Rule{
		ID:          "require-api-version",
		Name:        "Require API Version",
		Description: "All resources must specify an apiVersion",
		Severity:    SeverityError,
		Check: func(config map[string]interface{}, file string) []Issue {
			if _, ok := config["apiVersion"]; !ok {
				return []Issue{{
					Severity:   SeverityError,
					Rule:       "require-api-version",
					Message:    "Missing required field 'apiVersion'",
					Suggestion: "Add 'apiVersion: platform.io/v1' to your configuration",
				}}
			}
			return nil
		},
	}
}

func ruleRequireKind() Rule {
	return Rule{
		ID:          "require-kind",
		Name:        "Require Kind",
		Description: "All resources must specify a kind",
		Severity:    SeverityError,
		Check: func(config map[string]interface{}, file string) []Issue {
			if _, ok := config["kind"]; !ok {
				return []Issue{{
					Severity:   SeverityError,
					Rule:       "require-kind",
					Message:    "Missing required field 'kind'",
					Suggestion: "Add 'kind: <ResourceType>' to specify the resource type",
				}}
			}
			return nil
		},
	}
}

func ruleRequireMetadata() Rule {
	return Rule{
		ID:          "require-metadata",
		Name:        "Require Metadata",
		Description: "All resources should have metadata with a name",
		Severity:    SeverityError,
		Check: func(config map[string]interface{}, file string) []Issue {
			metadata, ok := config["metadata"].(map[string]interface{})
			if !ok {
				return []Issue{{
					Severity:   SeverityError,
					Rule:       "require-metadata",
					Message:    "Missing required field 'metadata'",
					Suggestion: "Add 'metadata: { name: <resource-name> }'",
				}}
			}

			if _, ok := metadata["name"]; !ok {
				return []Issue{{
					Severity:   SeverityError,
					Rule:       "require-metadata",
					Message:    "Missing required field 'metadata.name'",
					Suggestion: "Add 'name' field under metadata",
				}}
			}

			return nil
		},
	}
}

func ruleValidateResourceName() Rule {
	validName := regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$|^[a-z0-9]$`)

	return Rule{
		ID:          "valid-resource-name",
		Name:        "Valid Resource Name",
		Description: "Resource names should follow naming conventions",
		Severity:    SeverityWarning,
		Check: func(config map[string]interface{}, file string) []Issue {
			metadata, ok := config["metadata"].(map[string]interface{})
			if !ok {
				return nil
			}

			name, ok := metadata["name"].(string)
			if !ok {
				return nil
			}

			if !validName.MatchString(name) {
				return []Issue{{
					Severity:   SeverityWarning,
					Rule:       "valid-resource-name",
					Message:    fmt.Sprintf("Resource name '%s' doesn't follow naming conventions", name),
					Suggestion: "Use lowercase letters, numbers, and hyphens. Must start and end with alphanumeric",
				}}
			}

			if len(name) > 63 {
				return []Issue{{
					Severity:   SeverityWarning,
					Rule:       "valid-resource-name",
					Message:    fmt.Sprintf("Resource name '%s' exceeds 63 characters", name),
					Suggestion: "Shorten the resource name to 63 characters or less",
				}}
			}

			return nil
		},
	}
}

func ruleCheckEmptySpec() Rule {
	return Rule{
		ID:          "empty-spec",
		Name:        "Non-Empty Spec",
		Description: "Resources should have a non-empty spec",
		Severity:    SeverityWarning,
		Check: func(config map[string]interface{}, file string) []Issue {
			spec, ok := config["spec"].(map[string]interface{})
			if !ok {
				// Some resources may not need spec
				return nil
			}

			if len(spec) == 0 {
				return []Issue{{
					Severity:   SeverityWarning,
					Rule:       "empty-spec",
					Message:    "Resource has an empty 'spec' section",
					Suggestion: "Add configuration under 'spec' or remove the empty section",
				}}
			}

			return nil
		},
	}
}

func ruleReplicaCount() Rule {
	return Rule{
		ID:          "replica-count",
		Name:        "Replica Count",
		Description: "Production deployments should have multiple replicas",
		Severity:    SeverityInfo,
		Check: func(config map[string]interface{}, file string) []Issue {
			kind, _ := config["kind"].(string)
			if kind != "Deployment" {
				return nil
			}

			spec, ok := config["spec"].(map[string]interface{})
			if !ok {
				return nil
			}

			replicas, ok := spec["replicas"]
			if !ok {
				return []Issue{{
					Severity:   SeverityInfo,
					Rule:       "replica-count",
					Message:    "Deployment does not specify replica count",
					Suggestion: "Consider setting 'replicas' for high availability",
				}}
			}

			if r, ok := replicas.(int); ok && r < 2 {
				return []Issue{{
					Severity:   SeverityInfo,
					Rule:       "replica-count",
					Message:    fmt.Sprintf("Deployment has only %d replica", r),
					Suggestion: "Consider using 2+ replicas for high availability",
				}}
			}

			return nil
		},
	}
}

func ruleResourceLimits() Rule {
	return Rule{
		ID:          "resource-limits",
		Name:        "Resource Limits",
		Description: "Containers should have resource limits defined",
		Severity:    SeverityWarning,
		Check: func(config map[string]interface{}, file string) []Issue {
			kind, _ := config["kind"].(string)
			if kind != "Deployment" && kind != "Pod" {
				return nil
			}

			spec, ok := config["spec"].(map[string]interface{})
			if !ok {
				return nil
			}

			// Check for resources section
			if _, ok := spec["resources"]; !ok {
				return []Issue{{
					Severity:   SeverityWarning,
					Rule:       "resource-limits",
					Message:    "No resource limits defined",
					Suggestion: "Add 'resources.limits' to prevent resource exhaustion",
				}}
			}

			return nil
		},
	}
}

func ruleImageTag() Rule {
	return Rule{
		ID:          "image-tag",
		Name:        "Image Tag",
		Description: "Container images should use specific tags, not 'latest'",
		Severity:    SeverityWarning,
		Check: func(config map[string]interface{}, file string) []Issue {
			spec, ok := config["spec"].(map[string]interface{})
			if !ok {
				return nil
			}

			image, ok := spec["image"].(string)
			if !ok {
				return nil
			}

			if strings.HasSuffix(image, ":latest") || !strings.Contains(image, ":") {
				return []Issue{{
					Severity:   SeverityWarning,
					Rule:       "image-tag",
					Message:    fmt.Sprintf("Image '%s' uses 'latest' or no tag", image),
					Suggestion: "Use a specific version tag for reproducible deployments",
				}}
			}

			return nil
		},
	}
}

func ruleNamespaceExplicit() Rule {
	return Rule{
		ID:          "namespace-explicit",
		Name:        "Explicit Namespace",
		Description: "Resources should explicitly specify a namespace",
		Severity:    SeverityInfo,
		Check: func(config map[string]interface{}, file string) []Issue {
			metadata, ok := config["metadata"].(map[string]interface{})
			if !ok {
				return nil
			}

			if _, ok := metadata["namespace"]; !ok {
				return []Issue{{
					Severity:   SeverityInfo,
					Rule:       "namespace-explicit",
					Message:    "Resource does not specify a namespace",
					Suggestion: "Add 'namespace' under metadata for clarity",
				}}
			}

			return nil
		},
	}
}

func ruleLabelsPresent() Rule {
	return Rule{
		ID:          "labels-present",
		Name:        "Labels Present",
		Description: "Resources should have labels for organization",
		Severity:    SeverityInfo,
		Check: func(config map[string]interface{}, file string) []Issue {
			metadata, ok := config["metadata"].(map[string]interface{})
			if !ok {
				return nil
			}

			labels, ok := metadata["labels"].(map[string]interface{})
			if !ok || len(labels) == 0 {
				return []Issue{{
					Severity:   SeverityInfo,
					Rule:       "labels-present",
					Message:    "Resource has no labels",
					Suggestion: "Add labels like 'app', 'environment', 'team' for organization",
				}}
			}

			return nil
		},
	}
}

func ruleSecurityContext() Rule {
	return Rule{
		ID:          "security-context",
		Name:        "Security Context",
		Description: "Deployments should define security context",
		Severity:    SeverityWarning,
		Check: func(config map[string]interface{}, file string) []Issue {
			kind, _ := config["kind"].(string)
			if kind != "Deployment" && kind != "Pod" {
				return nil
			}

			spec, ok := config["spec"].(map[string]interface{})
			if !ok {
				return nil
			}

			if _, ok := spec["securityContext"]; !ok {
				return []Issue{{
					Severity:   SeverityWarning,
					Rule:       "security-context",
					Message:    "No security context defined",
					Suggestion: "Add 'securityContext' to define security settings",
				}}
			}

			return nil
		},
	}
}

func ruleHardcodedSecrets() Rule {
	secretPatterns := []string{
		"password",
		"secret",
		"api_key",
		"apikey",
		"token",
		"private_key",
	}

	return Rule{
		ID:          "hardcoded-secrets",
		Name:        "Hardcoded Secrets",
		Description: "Configuration should not contain hardcoded secrets",
		Severity:    SeverityError,
		Check: func(config map[string]interface{}, file string) []Issue {
			var issues []Issue

			var checkValue func(key string, value interface{})
			checkValue = func(key string, value interface{}) {
				keyLower := strings.ToLower(key)

				for _, pattern := range secretPatterns {
					if strings.Contains(keyLower, pattern) {
						if str, ok := value.(string); ok && len(str) > 0 {
							// Check if it looks like a reference
							if !strings.HasPrefix(str, "${") && !strings.HasPrefix(str, "$") {
								issues = append(issues, Issue{
									Severity:   SeverityError,
									Rule:       "hardcoded-secrets",
									Message:    fmt.Sprintf("Possible hardcoded secret in field '%s'", key),
									Suggestion: "Use environment variables or secret references instead",
								})
							}
						}
						break
					}
				}

				// Recurse into nested maps
				if m, ok := value.(map[string]interface{}); ok {
					for k, v := range m {
						checkValue(k, v)
					}
				}
			}

			for k, v := range config {
				checkValue(k, v)
			}

			return issues
		},
	}
}
