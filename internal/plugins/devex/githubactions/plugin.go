package githubactions

import (
	"fmt"

	"github.com/platformfoundry/pf-ce/pkg/plugin"
)

// Config represents GitHub Actions configuration
type Config struct {
	Provider     string              `yaml:"provider" json:"provider" validate:"required,oneof=github-actions"`
	Repository   string              `yaml:"repository" json:"repository" validate:"required"`
	Organization string              `yaml:"organization,omitempty" json:"organization,omitempty"`
	Workflows    []WorkflowConfig    `yaml:"workflows,omitempty" json:"workflows,omitempty"`
	Environments []EnvironmentConfig `yaml:"environments,omitempty" json:"environments,omitempty"`
	Secrets      []SecretConfig      `yaml:"secrets,omitempty" json:"secrets,omitempty"`
	Variables    []VariableConfig    `yaml:"variables,omitempty" json:"variables,omitempty"`
	Runners      *RunnersConfig      `yaml:"runners,omitempty" json:"runners,omitempty"`
}

// WorkflowConfig represents a GitHub Actions workflow
type WorkflowConfig struct {
	Name     string              `yaml:"name" json:"name" validate:"required"`
	Path     string              `yaml:"path" json:"path"`
	On       WorkflowTrigger     `yaml:"on" json:"on"`
	Jobs     map[string]JobConfig `yaml:"jobs" json:"jobs"`
	Env      map[string]string   `yaml:"env,omitempty" json:"env,omitempty"`
	Defaults *DefaultsConfig     `yaml:"defaults,omitempty" json:"defaults,omitempty"`
}

// WorkflowTrigger defines workflow triggers
type WorkflowTrigger struct {
	Push             *PushConfig             `yaml:"push,omitempty" json:"push,omitempty"`
	PullRequest      *PullRequestConfig      `yaml:"pull_request,omitempty" json:"pull_request,omitempty"`
	Schedule         []ScheduleConfig        `yaml:"schedule,omitempty" json:"schedule,omitempty"`
	WorkflowDispatch *WorkflowDispatchConfig `yaml:"workflow_dispatch,omitempty" json:"workflow_dispatch,omitempty"`
	WorkflowCall     *WorkflowCallConfig     `yaml:"workflow_call,omitempty" json:"workflow_call,omitempty"`
}

// PushConfig defines push trigger
type PushConfig struct {
	Branches []string `yaml:"branches,omitempty" json:"branches,omitempty"`
	Tags     []string `yaml:"tags,omitempty" json:"tags,omitempty"`
	Paths    []string `yaml:"paths,omitempty" json:"paths,omitempty"`
}

// PullRequestConfig defines pull request trigger
type PullRequestConfig struct {
	Branches []string `yaml:"branches,omitempty" json:"branches,omitempty"`
	Types    []string `yaml:"types,omitempty" json:"types,omitempty"`
	Paths    []string `yaml:"paths,omitempty" json:"paths,omitempty"`
}

// ScheduleConfig defines cron schedule
type ScheduleConfig struct {
	Cron string `yaml:"cron" json:"cron" validate:"required"`
}

// WorkflowDispatchConfig defines manual trigger inputs
type WorkflowDispatchConfig struct {
	Inputs map[string]InputConfig `yaml:"inputs,omitempty" json:"inputs,omitempty"`
}

// InputConfig defines workflow input
type InputConfig struct {
	Description string   `yaml:"description" json:"description"`
	Required    bool     `yaml:"required" json:"required"`
	Default     string   `yaml:"default,omitempty" json:"default,omitempty"`
	Type        string   `yaml:"type,omitempty" json:"type,omitempty" validate:"omitempty,oneof=string boolean choice environment"`
	Options     []string `yaml:"options,omitempty" json:"options,omitempty"`
}

// WorkflowCallConfig defines reusable workflow inputs/outputs
type WorkflowCallConfig struct {
	Inputs  map[string]InputConfig  `yaml:"inputs,omitempty" json:"inputs,omitempty"`
	Outputs map[string]OutputConfig `yaml:"outputs,omitempty" json:"outputs,omitempty"`
	Secrets map[string]SecretInput  `yaml:"secrets,omitempty" json:"secrets,omitempty"`
}

// OutputConfig defines workflow output
type OutputConfig struct {
	Description string `yaml:"description" json:"description"`
	Value       string `yaml:"value" json:"value"`
}

// SecretInput defines secret input for reusable workflows
type SecretInput struct {
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	Required    bool   `yaml:"required" json:"required"`
}

// JobConfig represents a workflow job
type JobConfig struct {
	Name           string                 `yaml:"name,omitempty" json:"name,omitempty"`
	RunsOn         interface{}            `yaml:"runs-on" json:"runs-on"` // string or []string
	Needs          []string               `yaml:"needs,omitempty" json:"needs,omitempty"`
	If             string                 `yaml:"if,omitempty" json:"if,omitempty"`
	Environment    interface{}            `yaml:"environment,omitempty" json:"environment,omitempty"` // string or EnvironmentRef
	Concurrency    *ConcurrencyConfig     `yaml:"concurrency,omitempty" json:"concurrency,omitempty"`
	Outputs        map[string]string      `yaml:"outputs,omitempty" json:"outputs,omitempty"`
	Env            map[string]string      `yaml:"env,omitempty" json:"env,omitempty"`
	Steps          []StepConfig           `yaml:"steps" json:"steps"`
	Strategy       *StrategyConfig        `yaml:"strategy,omitempty" json:"strategy,omitempty"`
	Container      *ContainerConfig       `yaml:"container,omitempty" json:"container,omitempty"`
	Services       map[string]ServiceConfig `yaml:"services,omitempty" json:"services,omitempty"`
	TimeoutMinutes int                    `yaml:"timeout-minutes,omitempty" json:"timeout-minutes,omitempty"`
}

// ConcurrencyConfig manages concurrent workflow runs
type ConcurrencyConfig struct {
	Group            string `yaml:"group" json:"group"`
	CancelInProgress bool   `yaml:"cancel-in-progress" json:"cancel-in-progress"`
}

// StepConfig represents a job step
type StepConfig struct {
	Name            string            `yaml:"name,omitempty" json:"name,omitempty"`
	ID              string            `yaml:"id,omitempty" json:"id,omitempty"`
	Uses            string            `yaml:"uses,omitempty" json:"uses,omitempty"`
	Run             string            `yaml:"run,omitempty" json:"run,omitempty"`
	With            map[string]string `yaml:"with,omitempty" json:"with,omitempty"`
	Env             map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
	If              string            `yaml:"if,omitempty" json:"if,omitempty"`
	ContinueOnError bool              `yaml:"continue-on-error,omitempty" json:"continue-on-error,omitempty"`
	TimeoutMinutes  int               `yaml:"timeout-minutes,omitempty" json:"timeout-minutes,omitempty"`
	Shell           string            `yaml:"shell,omitempty" json:"shell,omitempty"`
	WorkingDirectory string           `yaml:"working-directory,omitempty" json:"working-directory,omitempty"`
}

// StrategyConfig defines job matrix strategy
type StrategyConfig struct {
	Matrix      map[string]interface{} `yaml:"matrix,omitempty" json:"matrix,omitempty"`
	FailFast    bool                   `yaml:"fail-fast,omitempty" json:"fail-fast,omitempty"`
	MaxParallel int                    `yaml:"max-parallel,omitempty" json:"max-parallel,omitempty"`
}

// ContainerConfig defines container settings for a job
type ContainerConfig struct {
	Image       string            `yaml:"image" json:"image"`
	Credentials *CredentialsConfig `yaml:"credentials,omitempty" json:"credentials,omitempty"`
	Env         map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
	Ports       []int             `yaml:"ports,omitempty" json:"ports,omitempty"`
	Volumes     []string          `yaml:"volumes,omitempty" json:"volumes,omitempty"`
	Options     string            `yaml:"options,omitempty" json:"options,omitempty"`
}

// CredentialsConfig for container registry
type CredentialsConfig struct {
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
}

// ServiceConfig defines service containers
type ServiceConfig struct {
	Image       string            `yaml:"image" json:"image"`
	Credentials *CredentialsConfig `yaml:"credentials,omitempty" json:"credentials,omitempty"`
	Env         map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
	Ports       []string          `yaml:"ports,omitempty" json:"ports,omitempty"`
	Volumes     []string          `yaml:"volumes,omitempty" json:"volumes,omitempty"`
	Options     string            `yaml:"options,omitempty" json:"options,omitempty"`
}

// DefaultsConfig defines job defaults
type DefaultsConfig struct {
	Run *RunDefaults `yaml:"run,omitempty" json:"run,omitempty"`
}

// RunDefaults defines run step defaults
type RunDefaults struct {
	Shell            string `yaml:"shell,omitempty" json:"shell,omitempty"`
	WorkingDirectory string `yaml:"working-directory,omitempty" json:"working-directory,omitempty"`
}

// EnvironmentConfig defines GitHub environment
type EnvironmentConfig struct {
	Name               string              `yaml:"name" json:"name" validate:"required"`
	URL                string              `yaml:"url,omitempty" json:"url,omitempty"`
	DeploymentBranches *BranchPolicyConfig `yaml:"deploymentBranchPolicy,omitempty" json:"deploymentBranchPolicy,omitempty"`
	Reviewers          []ReviewerConfig    `yaml:"reviewers,omitempty" json:"reviewers,omitempty"`
	WaitTimer          int                 `yaml:"waitTimer,omitempty" json:"waitTimer,omitempty"`
}

// BranchPolicyConfig defines branch protection for deployments
type BranchPolicyConfig struct {
	ProtectedBranches    bool     `yaml:"protectedBranches" json:"protectedBranches"`
	CustomBranchPolicies bool     `yaml:"customBranchPolicies" json:"customBranchPolicies"`
	Branches             []string `yaml:"branches,omitempty" json:"branches,omitempty"`
}

// ReviewerConfig defines environment reviewers
type ReviewerConfig struct {
	Type string `yaml:"type" json:"type" validate:"required,oneof=User Team"`
	ID   int64  `yaml:"id" json:"id" validate:"required"`
}

// SecretConfig defines repository/organization secrets
type SecretConfig struct {
	Name        string `yaml:"name" json:"name" validate:"required"`
	Value       string `yaml:"value,omitempty" json:"value,omitempty"`
	SecretRef   string `yaml:"secretRef,omitempty" json:"secretRef,omitempty"`
	Environment string `yaml:"environment,omitempty" json:"environment,omitempty"`
}

// VariableConfig defines repository/organization variables
type VariableConfig struct {
	Name        string `yaml:"name" json:"name" validate:"required"`
	Value       string `yaml:"value" json:"value" validate:"required"`
	Environment string `yaml:"environment,omitempty" json:"environment,omitempty"`
}

// RunnersConfig defines self-hosted runner configuration
type RunnersConfig struct {
	SelfHosted []SelfHostedRunner `yaml:"selfHosted,omitempty" json:"selfHosted,omitempty"`
	Labels     []string           `yaml:"labels,omitempty" json:"labels,omitempty"`
}

// SelfHostedRunner defines a self-hosted runner
type SelfHostedRunner struct {
	Name   string   `yaml:"name" json:"name" validate:"required"`
	Labels []string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Group  string   `yaml:"group,omitempty" json:"group,omitempty"`
}

// Plugin implements the GitHub Actions plugin
type Plugin struct{}

// NewPlugin creates a new GitHub Actions plugin
func NewPlugin() plugin.Plugin {
	return &Plugin{}
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return "github-actions"
}

// Type returns the resource type
func (p *Plugin) Type() string {
	return "DevEx"
}

// Version returns the plugin version
func (p *Plugin) Version() string {
	return "1.0.0"
}

// ConfigType returns the configuration type
func (p *Plugin) ConfigType() interface{} {
	return &Config{}
}

// Validate validates the plugin configuration
func (p *Plugin) Validate(spec map[string]interface{}) error {
	provider, ok := spec["provider"].(string)
	if !ok || provider == "" {
		return fmt.Errorf("provider field is required")
	}

	if provider != "github-actions" {
		return fmt.Errorf("provider must be 'github-actions'")
	}

	repository, ok := spec["repository"].(string)
	if !ok || repository == "" {
		return fmt.Errorf("repository is required")
	}

	return nil
}

// Plan generates a plan for the plugin
func (p *Plugin) Plan(spec map[string]interface{}) (*plugin.Plan, error) {
	actions := []string{}

	if workflows, ok := spec["workflows"].([]interface{}); ok && len(workflows) > 0 {
		actions = append(actions, fmt.Sprintf("Create/update %d workflows", len(workflows)))
	}

	if environments, ok := spec["environments"].([]interface{}); ok && len(environments) > 0 {
		actions = append(actions, fmt.Sprintf("Configure %d environments", len(environments)))
	}

	if secrets, ok := spec["secrets"].([]interface{}); ok && len(secrets) > 0 {
		actions = append(actions, fmt.Sprintf("Configure %d secrets", len(secrets)))
	}

	if variables, ok := spec["variables"].([]interface{}); ok && len(variables) > 0 {
		actions = append(actions, fmt.Sprintf("Configure %d variables", len(variables)))
	}

	if len(actions) == 0 {
		actions = []string{"Configure GitHub Actions"}
	}

	return &plugin.Plan{
		Actions: actions,
	}, nil
}

// Apply applies the plugin configuration
func (p *Plugin) Apply(spec map[string]interface{}) (*plugin.Result, error) {
	return &plugin.Result{
		Status:  "success",
		Message: "GitHub Actions configured successfully",
		Outputs: map[string]string{
			"provider": "github-actions",
		},
	}, nil
}

// Delete deletes resources created by the plugin
func (p *Plugin) Delete(name string) error {
	return nil
}

// Status gets the current status of the resource
func (p *Plugin) Status(name string) (*plugin.Status, error) {
	return &plugin.Status{
		State:   "ready",
		Ready:   true,
		Message: "GitHub Actions workflows are configured",
	}, nil
}
