package terraform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/platformfoundry/platformfoundry-ce/pkg/plugin"
)

// Plugin wraps Terraform CLI as a plugin (Adapter Pattern)
type Plugin struct {
	name     string
	version  string
	tfBinary string

	workDir string
	mu      sync.RWMutex

	// Track state
	lastPlanFile string
	outputs      map[string]string
}

// Config represents the configuration schema for Terraform
type Config struct {
	WorkDir     string                 `yaml:"workDir" json:"workDir"`
	BackendType string                 `yaml:"backendType,omitempty" json:"backendType,omitempty"`
	Backend     map[string]interface{} `yaml:"backend,omitempty" json:"backend,omitempty"`
	Variables   map[string]interface{} `yaml:"variables,omitempty" json:"variables,omitempty"`
	VarFiles    []string               `yaml:"varFiles,omitempty" json:"varFiles,omitempty"`
	Targets     []string               `yaml:"targets,omitempty" json:"targets,omitempty"`
	Parallelism int                    `yaml:"parallelism,omitempty" json:"parallelism,omitempty"`
	AutoApprove bool                   `yaml:"autoApprove" json:"autoApprove"`
}

// New creates a new Terraform plugin
func New() *Plugin {
	return &Plugin{
		name:     "terraform",
		version:  "1.0.0",
		tfBinary: "terraform",
		outputs:  make(map[string]string),
	}
}

func (p *Plugin) Name() string    { return p.name }
func (p *Plugin) Type() string    { return "Infrastructure" }
func (p *Plugin) Version() string { return p.version }

func (p *Plugin) ConfigType() interface{} {
	return Config{}
}

// SetBinary allows overriding the terraform binary path
func (p *Plugin) SetBinary(path string) {
	p.tfBinary = path
}

func (p *Plugin) Validate(spec map[string]interface{}) error {
	cfg, err := p.parseConfig(spec)
	if err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Check working directory exists
	if _, err := os.Stat(cfg.WorkDir); os.IsNotExist(err) {
		return fmt.Errorf("terraform working directory does not exist: %s", cfg.WorkDir)
	}

	// Run terraform validate
	ctx := context.Background()
	if err := p.runInit(ctx, cfg); err != nil {
		return fmt.Errorf("terraform init failed: %w", err)
	}

	cmd := exec.CommandContext(ctx, p.tfBinary, "validate", "-json")
	cmd.Dir = cfg.WorkDir

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("terraform validate failed: %w", err)
	}

	var result struct {
		Valid        bool   `json:"valid"`
		ErrorCount   int    `json:"error_count"`
		WarningCount int    `json:"warning_count"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return fmt.Errorf("failed to parse validate output: %w", err)
	}

	if !result.Valid {
		return fmt.Errorf("terraform configuration is invalid (%d errors)", result.ErrorCount)
	}

	return nil
}

func (p *Plugin) Plan(spec map[string]interface{}) (*plugin.Plan, error) {
	cfg, err := p.parseConfig(spec)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	p.workDir = cfg.WorkDir
	p.mu.Unlock()

	ctx := context.Background()

	// Run terraform init
	if err := p.runInit(ctx, cfg); err != nil {
		return nil, fmt.Errorf("terraform init failed: %w", err)
	}

	// Build plan command
	planFile := filepath.Join(cfg.WorkDir, "tfplan")
	args := []string{"plan", "-out=" + planFile, "-json", "-no-color"}

	for k, v := range cfg.Variables {
		args = append(args, fmt.Sprintf("-var=%s=%v", k, v))
	}
	for _, f := range cfg.VarFiles {
		args = append(args, fmt.Sprintf("-var-file=%s", f))
	}
	for _, t := range cfg.Targets {
		args = append(args, fmt.Sprintf("-target=%s", t))
	}

	cmd := exec.CommandContext(ctx, p.tfBinary, args...)
	cmd.Dir = cfg.WorkDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	// Terraform plan exits with code 2 when there are changes, which is not an error
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() != 2 {
				return nil, fmt.Errorf("terraform plan failed: %s", stderr.String())
			}
		} else {
			return nil, fmt.Errorf("terraform plan failed: %w", err)
		}
	}

	p.mu.Lock()
	p.lastPlanFile = planFile
	p.mu.Unlock()

	// Parse plan output
	return p.parsePlanOutput(stdout.Bytes())
}

func (p *Plugin) Apply(spec map[string]interface{}) (*plugin.Result, error) {
	cfg, err := p.parseConfig(spec)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	planFile := p.lastPlanFile
	p.mu.Unlock()

	ctx := context.Background()

	// If no plan file, run init and plan first
	if planFile == "" {
		if err := p.runInit(ctx, cfg); err != nil {
			return nil, fmt.Errorf("terraform init failed: %w", err)
		}

		planFile = filepath.Join(cfg.WorkDir, "tfplan")
		planArgs := []string{"plan", "-out=" + planFile, "-no-color"}
		for k, v := range cfg.Variables {
			planArgs = append(planArgs, fmt.Sprintf("-var=%s=%v", k, v))
		}

		planCmd := exec.CommandContext(ctx, p.tfBinary, planArgs...)
		planCmd.Dir = cfg.WorkDir
		if output, err := planCmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("terraform plan failed: %s", string(output))
		}
	}

	// Build apply command
	args := []string{"apply", "-json", "-no-color"}
	if cfg.AutoApprove {
		args = append(args, "-auto-approve")
	}
	if cfg.Parallelism > 0 {
		args = append(args, fmt.Sprintf("-parallelism=%d", cfg.Parallelism))
	}
	args = append(args, planFile)

	cmd := exec.CommandContext(ctx, p.tfBinary, args...)
	cmd.Dir = cfg.WorkDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return &plugin.Result{
			Status:  "failed",
			Message: fmt.Sprintf("terraform apply failed: %s", stderr.String()),
		}, err
	}

	// Get outputs
	outputs, err := p.getOutputs(ctx, cfg.WorkDir)
	if err != nil {
		// Non-fatal - continue with empty outputs
		outputs = make(map[string]string)
	}

	p.mu.Lock()
	p.outputs = outputs
	p.lastPlanFile = "" // Clear plan file after apply
	p.mu.Unlock()

	return &plugin.Result{
		Status:    "success",
		Message:   "Terraform apply completed successfully",
		Resources: p.parseAppliedResources(stdout.Bytes()),
		Outputs:   outputs,
	}, nil
}

func (p *Plugin) Delete(name string) error {
	p.mu.RLock()
	workDir := p.workDir
	p.mu.RUnlock()

	if workDir == "" {
		return fmt.Errorf("working directory not set")
	}

	ctx := context.Background()
	args := []string{"destroy", "-auto-approve", "-json", "-no-color"}

	cmd := exec.CommandContext(ctx, p.tfBinary, args...)
	cmd.Dir = workDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("terraform destroy failed: %s", string(output))
	}

	return nil
}

func (p *Plugin) Status(name string) (*plugin.Status, error) {
	p.mu.RLock()
	workDir := p.workDir
	p.mu.RUnlock()

	if workDir == "" {
		return &plugin.Status{
			State:   "unknown",
			Ready:   false,
			Message: "Working directory not set",
		}, nil
	}

	ctx := context.Background()

	// Run terraform show
	cmd := exec.CommandContext(ctx, p.tfBinary, "show", "-json")
	cmd.Dir = workDir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("terraform show failed: %w", err)
	}

	var state struct {
		Values struct {
			RootModule struct {
				Resources []struct {
					Address string `json:"address"`
					Type    string `json:"type"`
					Name    string `json:"name"`
				} `json:"resources"`
			} `json:"root_module"`
		} `json:"values"`
	}

	if err := json.Unmarshal(output, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state: %w", err)
	}

	resourceCount := len(state.Values.RootModule.Resources)

	return &plugin.Status{
		State:   "applied",
		Ready:   resourceCount > 0,
		Message: fmt.Sprintf("Terraform state contains %d resources", resourceCount),
		Details: map[string]string{
			"resources": fmt.Sprintf("%d", resourceCount),
			"workDir":   workDir,
		},
	}, nil
}

func (p *Plugin) runInit(ctx context.Context, cfg *Config) error {
	args := []string{"init", "-input=false", "-no-color"}

	// Add backend config if specified
	if cfg.BackendType != "" {
		for k, v := range cfg.Backend {
			args = append(args, fmt.Sprintf("-backend-config=%s=%v", k, v))
		}
	}

	cmd := exec.CommandContext(ctx, p.tfBinary, args...)
	cmd.Dir = cfg.WorkDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("init failed: %s", string(output))
	}

	return nil
}

func (p *Plugin) getOutputs(ctx context.Context, workDir string) (map[string]string, error) {
	cmd := exec.CommandContext(ctx, p.tfBinary, "output", "-json")
	cmd.Dir = workDir

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var tfOutputs map[string]struct {
		Value interface{} `json:"value"`
		Type  interface{} `json:"type"`
	}

	if err := json.Unmarshal(output, &tfOutputs); err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for k, v := range tfOutputs {
		switch val := v.Value.(type) {
		case string:
			result[k] = val
		case float64:
			result[k] = fmt.Sprintf("%v", val)
		case bool:
			result[k] = fmt.Sprintf("%v", val)
		default:
			// For complex types, serialize to JSON
			data, _ := json.Marshal(val)
			result[k] = string(data)
		}
	}

	return result, nil
}

func (p *Plugin) parsePlanOutput(output []byte) (*plugin.Plan, error) {
	plan := &plugin.Plan{
		Actions: make([]string, 0),
		Changes: make(map[string]string),
	}

	// Parse JSON lines output
	lines := bytes.Split(output, []byte("\n"))
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		var msg struct {
			Type    string `json:"type"`
			Change  *struct {
				Resource struct {
					Addr string `json:"addr"`
				} `json:"resource"`
				Action string `json:"action"`
			} `json:"change"`
			Message string `json:"message"`
		}

		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}

		if msg.Type == "planned_change" && msg.Change != nil {
			action := fmt.Sprintf("%s %s", msg.Change.Action, msg.Change.Resource.Addr)
			plan.Actions = append(plan.Actions, action)
			plan.Changes[msg.Change.Resource.Addr] = msg.Change.Action
		}
	}

	return plan, nil
}

func (p *Plugin) parseAppliedResources(output []byte) []string {
	resources := make([]string, 0)

	lines := bytes.Split(output, []byte("\n"))
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		var msg struct {
			Type string `json:"type"`
			Hook *struct {
				Resource struct {
					Addr string `json:"addr"`
				} `json:"resource"`
				Action string `json:"action"`
			} `json:"hook"`
		}

		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}

		if msg.Type == "apply_complete" && msg.Hook != nil {
			resources = append(resources, msg.Hook.Resource.Addr)
		}
	}

	return resources
}

func (p *Plugin) parseConfig(spec map[string]interface{}) (*Config, error) {
	cfg := &Config{
		WorkDir:     ".",
		Variables:   make(map[string]interface{}),
		VarFiles:    make([]string, 0),
		Targets:     make([]string, 0),
		AutoApprove: true, // Default to auto-approve for automation
	}

	if wd, ok := spec["workDir"].(string); ok {
		cfg.WorkDir = wd
	}
	if bt, ok := spec["backendType"].(string); ok {
		cfg.BackendType = bt
	}
	if b, ok := spec["backend"].(map[string]interface{}); ok {
		cfg.Backend = b
	}
	if v, ok := spec["variables"].(map[string]interface{}); ok {
		cfg.Variables = v
	}
	if vf, ok := spec["varFiles"].([]interface{}); ok {
		for _, f := range vf {
			if fs, ok := f.(string); ok {
				cfg.VarFiles = append(cfg.VarFiles, fs)
			}
		}
	}
	if t, ok := spec["targets"].([]interface{}); ok {
		for _, target := range t {
			if ts, ok := target.(string); ok {
				cfg.Targets = append(cfg.Targets, ts)
			}
		}
	}
	if p, ok := spec["parallelism"].(int); ok {
		cfg.Parallelism = p
	}
	if p, ok := spec["parallelism"].(float64); ok {
		cfg.Parallelism = int(p)
	}
	if aa, ok := spec["autoApprove"].(bool); ok {
		cfg.AutoApprove = aa
	}

	// Expand home directory in workDir
	if strings.HasPrefix(cfg.WorkDir, "~") {
		home, _ := os.UserHomeDir()
		cfg.WorkDir = filepath.Join(home, cfg.WorkDir[1:])
	}

	return cfg, nil
}
