package policy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
)

// Engine represents the policy engine interface
type Engine interface {
	// Evaluate evaluates a policy against input data
	Evaluate(ctx context.Context, policy string, input interface{}) (*Result, error)

	// LoadPolicy loads a policy from a file or string
	LoadPolicy(ctx context.Context, name, content string) error

	// DeletePolicy deletes a loaded policy
	DeletePolicy(ctx context.Context, name string) error

	// ListPolicies lists all loaded policies
	ListPolicies(ctx context.Context) ([]string, error)

	// Close closes the policy engine
	Close() error
}

// Result represents the result of a policy evaluation
type Result struct {
	Allowed bool                   `json:"allowed"`
	Reasons []string               `json:"reasons,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// Config represents policy engine configuration
type Config struct {
	// Type of policy engine: "opa", "local"
	Type string `yaml:"type" json:"type"`

	// OPA configuration
	OPAEndpoint string `yaml:"opaEndpoint,omitempty" json:"opaEndpoint,omitempty"`
	OPATimeout  int    `yaml:"opaTimeout,omitempty" json:"opaTimeout,omitempty"` // seconds

	// Local configuration
	PolicyDir string `yaml:"policyDir,omitempty" json:"policyDir,omitempty"`
}

// DefaultConfig returns default policy engine configuration
func DefaultConfig() *Config {
	return &Config{
		Type:        "local",
		OPAEndpoint: "http://localhost:8181",
		OPATimeout:  5,
		PolicyDir:   "/etc/platformfoundry/policies",
	}
}

// NewEngine creates a new policy engine based on configuration
func NewEngine(config *Config) (Engine, error) {
	if config == nil {
		config = DefaultConfig()
	}

	switch config.Type {
	case "opa":
		return NewOPAEngine(config)
	case "local":
		return NewLocalEngine(config)
	default:
		return nil, fmt.Errorf("unsupported policy engine type: %s", config.Type)
	}
}

// OPAEngine implements Engine using Open Policy Agent
type OPAEngine struct {
	endpoint   string
	httpClient *http.Client
}

// NewOPAEngine creates a new OPA policy engine
func NewOPAEngine(config *Config) (*OPAEngine, error) {
	if config.OPAEndpoint == "" {
		return nil, fmt.Errorf("OPA endpoint is required")
	}

	timeout := time.Duration(config.OPATimeout) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	return &OPAEngine{
		endpoint: config.OPAEndpoint,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// Evaluate evaluates a policy against input data using OPA
func (e *OPAEngine) Evaluate(ctx context.Context, policy string, input interface{}) (*Result, error) {
	// Prepare request body
	body := map[string]interface{}{
		"input": input,
	}

	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}

	// Build URL
	url := fmt.Sprintf("%s/v1/data/%s", e.endpoint, policy)

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to OPA: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OPA returned error: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var opaResp struct {
		Result interface{} `json:"result"`
	}

	if err := json.Unmarshal(respBody, &opaResp); err != nil {
		return nil, fmt.Errorf("failed to parse OPA response: %w", err)
	}

	// Convert OPA result to our Result format
	result := &Result{
		Data: make(map[string]interface{}),
	}

	// Check if result is a boolean (simple allow/deny)
	if allowed, ok := opaResp.Result.(bool); ok {
		result.Allowed = allowed
	} else if resultMap, ok := opaResp.Result.(map[string]interface{}); ok {
		// Check for "allow" field
		if allowed, ok := resultMap["allow"].(bool); ok {
			result.Allowed = allowed
		}

		// Check for "reasons" field
		if reasons, ok := resultMap["reasons"].([]interface{}); ok {
			result.Reasons = make([]string, len(reasons))
			for i, r := range reasons {
				if str, ok := r.(string); ok {
					result.Reasons[i] = str
				}
			}
		}

		result.Data = resultMap
	}

	return result, nil
}

// LoadPolicy loads a policy into OPA
func (e *OPAEngine) LoadPolicy(ctx context.Context, name, content string) error {
	url := fmt.Sprintf("%s/v1/policies/%s", e.endpoint, name)

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBufferString(content))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "text/plain")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to OPA: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("OPA returned error: HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeletePolicy deletes a policy from OPA
func (e *OPAEngine) DeletePolicy(ctx context.Context, name string) error {
	url := fmt.Sprintf("%s/v1/policies/%s", e.endpoint, name)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to OPA: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("OPA returned error: HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ListPolicies lists all loaded policies in OPA
func (e *OPAEngine) ListPolicies(ctx context.Context) ([]string, error) {
	url := fmt.Sprintf("%s/v1/policies", e.endpoint)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to OPA: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OPA returned error: HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var opaResp struct {
		Result []struct {
			ID string `json:"id"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&opaResp); err != nil {
		return nil, fmt.Errorf("failed to parse OPA response: %w", err)
	}

	policies := make([]string, len(opaResp.Result))
	for i, p := range opaResp.Result {
		policies[i] = p.ID
	}

	return policies, nil
}

// Close closes the OPA engine
func (e *OPAEngine) Close() error {
	e.httpClient.CloseIdleConnections()
	return nil
}

// LocalEngine implements a simple local policy engine
type LocalEngine struct {
	policyDir string
	policies  map[string]string
}

// NewLocalEngine creates a new local policy engine
func NewLocalEngine(config *Config) (*LocalEngine, error) {
	policyDir := config.PolicyDir
	if policyDir == "" {
		policyDir = "/etc/platformfoundry/policies"
	}

	// Ensure directory exists
	if err := os.MkdirAll(policyDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create policy directory: %w", err)
	}

	engine := &LocalEngine{
		policyDir: policyDir,
		policies:  make(map[string]string),
	}

	// Load existing policies
	if err := engine.loadPoliciesFromDir(); err != nil {
		return nil, err
	}

	return engine, nil
}

// loadPoliciesFromDir loads all policies from the policy directory
func (e *LocalEngine) loadPoliciesFromDir() error {
	entries, err := os.ReadDir(e.policyDir)
	if err != nil {
		return fmt.Errorf("failed to read policy directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".rego" {
			continue
		}

		name := entry.Name()[:len(entry.Name())-5] // Remove .rego extension
		path := filepath.Join(e.policyDir, entry.Name())

		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		e.policies[name] = string(content)
	}

	return nil
}

// Evaluate evaluates a policy using the OPA Rego engine
func (e *LocalEngine) Evaluate(ctx context.Context, policy string, input interface{}) (*Result, error) {
	// Check if policy exists
	policyContent, exists := e.policies[policy]
	if !exists {
		return nil, fmt.Errorf("policy not found: %s", policy)
	}

	result := &Result{
		Allowed: false, // Default deny
		Data:    make(map[string]interface{}),
		Reasons: []string{},
	}

	// Create Rego query for "allow" rule with v0 compatibility for backward compat
	allowQuery, err := rego.New(
		rego.Query("data."+extractPackageName(policyContent)+".allow"),
		rego.Module(policy+".rego", policyContent),
		rego.Input(input),
		rego.SetRegoVersion(ast.RegoV0),
	).PrepareForEval(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare allow query: %w", err)
	}

	// Evaluate allow rule
	allowResults, err := allowQuery.Eval(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate allow rule: %w", err)
	}

	// Check if allowed
	if len(allowResults) > 0 && len(allowResults[0].Expressions) > 0 {
		if allowed, ok := allowResults[0].Expressions[0].Value.(bool); ok {
			result.Allowed = allowed
		}
	}

	// Create Rego query for "deny" rule to get reasons
	denyQuery, err := rego.New(
		rego.Query("data."+extractPackageName(policyContent)+".deny"),
		rego.Module(policy+".rego", policyContent),
		rego.Input(input),
		rego.SetRegoVersion(ast.RegoV0),
	).PrepareForEval(ctx)
	if err != nil {
		// Deny rule may not exist, which is fine
		return result, nil
	}

	// Evaluate deny rule
	denyResults, err := denyQuery.Eval(ctx)
	if err != nil {
		// Deny rule evaluation error is not critical
		return result, nil
	}

	// Collect deny reasons
	if len(denyResults) > 0 && len(denyResults[0].Expressions) > 0 {
		if denySet, ok := denyResults[0].Expressions[0].Value.([]interface{}); ok {
			for _, msg := range denySet {
				if str, ok := msg.(string); ok {
					result.Reasons = append(result.Reasons, str)
				}
			}
		}
	}

	result.Data["policy"] = policy
	result.Data["evaluated"] = true

	return result, nil
}

// extractPackageName extracts the package name from Rego policy content
func extractPackageName(content string) string {
	// Simple extraction: look for "package X" at the beginning
	lines := bytes.Split([]byte(content), []byte("\n"))
	for _, line := range lines {
		lineStr := string(bytes.TrimSpace(line))
		if len(lineStr) > 8 && lineStr[:8] == "package " {
			return lineStr[8:]
		}
	}
	return "policy"
}

// LoadPolicy loads a policy from a file path or raw content
func (e *LocalEngine) LoadPolicy(ctx context.Context, name, contentOrPath string) error {
	var content string

	// Check if contentOrPath is a file path (ends with .rego or exists as a file)
	if filepath.Ext(contentOrPath) == ".rego" || fileExists(contentOrPath) {
		data, err := os.ReadFile(contentOrPath)
		if err != nil {
			return fmt.Errorf("failed to read policy file: %w", err)
		}
		content = string(data)
	} else {
		content = contentOrPath
	}

	// Save to policy directory
	path := filepath.Join(e.policyDir, name+".rego")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write policy file: %w", err)
	}

	e.policies[name] = content
	return nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// DeletePolicy deletes a policy
func (e *LocalEngine) DeletePolicy(ctx context.Context, name string) error {
	path := filepath.Join(e.policyDir, name+".rego")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete policy file: %w", err)
	}

	delete(e.policies, name)
	return nil
}

// ListPolicies lists all loaded policies
func (e *LocalEngine) ListPolicies(ctx context.Context) ([]string, error) {
	policies := make([]string, 0, len(e.policies))
	for name := range e.policies {
		policies = append(policies, name)
	}
	return policies, nil
}

// Close closes the local engine
func (e *LocalEngine) Close() error {
	return nil
}
