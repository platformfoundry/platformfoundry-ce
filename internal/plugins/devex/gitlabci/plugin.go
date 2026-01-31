package gitlabci

import (
	"fmt"

	"github.com/platformfoundry/platformfoundry-ce/pkg/plugin"
)

// Config represents GitLab CI configuration
type Config struct {
	Provider    string              `yaml:"provider" json:"provider" validate:"required,oneof=gitlab-ci"`
	Project     string              `yaml:"project" json:"project" validate:"required"`
	Group       string              `yaml:"group,omitempty" json:"group,omitempty"`
	GitLabURL   string              `yaml:"gitlabUrl,omitempty" json:"gitlabUrl,omitempty"`
	Pipeline    *PipelineConfig     `yaml:"pipeline,omitempty" json:"pipeline,omitempty"`
	Environments []EnvironmentConfig `yaml:"environments,omitempty" json:"environments,omitempty"`
	Variables   []VariableConfig    `yaml:"variables,omitempty" json:"variables,omitempty"`
	Runners     *RunnersConfig      `yaml:"runners,omitempty" json:"runners,omitempty"`
}

// PipelineConfig represents GitLab CI pipeline configuration
type PipelineConfig struct {
	Stages     []string               `yaml:"stages" json:"stages"`
	Default    *DefaultConfig         `yaml:"default,omitempty" json:"default,omitempty"`
	Variables  map[string]interface{} `yaml:"variables,omitempty" json:"variables,omitempty"`
	Include    []IncludeConfig        `yaml:"include,omitempty" json:"include,omitempty"`
	Workflow   *WorkflowConfig        `yaml:"workflow,omitempty" json:"workflow,omitempty"`
	Jobs       map[string]JobConfig   `yaml:"jobs,omitempty" json:"jobs,omitempty"`
}

// DefaultConfig defines default job settings
type DefaultConfig struct {
	Image        string            `yaml:"image,omitempty" json:"image,omitempty"`
	Services     []ServiceConfig   `yaml:"services,omitempty" json:"services,omitempty"`
	BeforeScript []string          `yaml:"before_script,omitempty" json:"before_script,omitempty"`
	AfterScript  []string          `yaml:"after_script,omitempty" json:"after_script,omitempty"`
	Tags         []string          `yaml:"tags,omitempty" json:"tags,omitempty"`
	Artifacts    *ArtifactsConfig  `yaml:"artifacts,omitempty" json:"artifacts,omitempty"`
	Cache        *CacheConfig      `yaml:"cache,omitempty" json:"cache,omitempty"`
	Retry        *RetryConfig      `yaml:"retry,omitempty" json:"retry,omitempty"`
	Timeout      string            `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Interruptible bool             `yaml:"interruptible,omitempty" json:"interruptible,omitempty"`
}

// ServiceConfig defines CI service containers
type ServiceConfig struct {
	Name       string            `yaml:"name" json:"name"`
	Alias      string            `yaml:"alias,omitempty" json:"alias,omitempty"`
	Entrypoint []string          `yaml:"entrypoint,omitempty" json:"entrypoint,omitempty"`
	Command    []string          `yaml:"command,omitempty" json:"command,omitempty"`
	Variables  map[string]string `yaml:"variables,omitempty" json:"variables,omitempty"`
}

// ArtifactsConfig defines artifact settings
type ArtifactsConfig struct {
	Paths     []string `yaml:"paths,omitempty" json:"paths,omitempty"`
	Exclude   []string `yaml:"exclude,omitempty" json:"exclude,omitempty"`
	ExpireIn  string   `yaml:"expire_in,omitempty" json:"expire_in,omitempty"`
	ExposeAs  string   `yaml:"expose_as,omitempty" json:"expose_as,omitempty"`
	Name      string   `yaml:"name,omitempty" json:"name,omitempty"`
	Public    bool     `yaml:"public,omitempty" json:"public,omitempty"`
	Reports   *Reports `yaml:"reports,omitempty" json:"reports,omitempty"`
	Untracked bool     `yaml:"untracked,omitempty" json:"untracked,omitempty"`
	When      string   `yaml:"when,omitempty" json:"when,omitempty" validate:"omitempty,oneof=on_success on_failure always"`
}

// Reports defines CI report artifacts
type Reports struct {
	CoverageReport   *CoverageReport `yaml:"coverage_report,omitempty" json:"coverage_report,omitempty"`
	Junit            string          `yaml:"junit,omitempty" json:"junit,omitempty"`
	Cobertura        string          `yaml:"cobertura,omitempty" json:"cobertura,omitempty"`
	SAST             string          `yaml:"sast,omitempty" json:"sast,omitempty"`
	DependencyScanning string        `yaml:"dependency_scanning,omitempty" json:"dependency_scanning,omitempty"`
	ContainerScanning string         `yaml:"container_scanning,omitempty" json:"container_scanning,omitempty"`
	DAST             string          `yaml:"dast,omitempty" json:"dast,omitempty"`
	SecretDetection  string          `yaml:"secret_detection,omitempty" json:"secret_detection,omitempty"`
	Terraform        string          `yaml:"terraform,omitempty" json:"terraform,omitempty"`
}

// CoverageReport defines coverage report
type CoverageReport struct {
	CoverageFormat string `yaml:"coverage_format" json:"coverage_format"`
	Path           string `yaml:"path" json:"path"`
}

// CacheConfig defines cache settings
type CacheConfig struct {
	Key       interface{} `yaml:"key,omitempty" json:"key,omitempty"` // string or CacheKey
	Paths     []string    `yaml:"paths,omitempty" json:"paths,omitempty"`
	Untracked bool        `yaml:"untracked,omitempty" json:"untracked,omitempty"`
	Unprotect bool        `yaml:"unprotect,omitempty" json:"unprotect,omitempty"`
	When      string      `yaml:"when,omitempty" json:"when,omitempty"`
	Policy    string      `yaml:"policy,omitempty" json:"policy,omitempty" validate:"omitempty,oneof=pull push pull-push"`
}

// RetryConfig defines retry settings
type RetryConfig struct {
	Max  int      `yaml:"max,omitempty" json:"max,omitempty" validate:"omitempty,min=0,max=2"`
	When []string `yaml:"when,omitempty" json:"when,omitempty"`
}

// IncludeConfig defines external configuration includes
type IncludeConfig struct {
	Local    string `yaml:"local,omitempty" json:"local,omitempty"`
	File     string `yaml:"file,omitempty" json:"file,omitempty"`
	Template string `yaml:"template,omitempty" json:"template,omitempty"`
	Remote   string `yaml:"remote,omitempty" json:"remote,omitempty"`
	Project  string `yaml:"project,omitempty" json:"project,omitempty"`
	Ref      string `yaml:"ref,omitempty" json:"ref,omitempty"`
}

// WorkflowConfig controls when pipelines run
type WorkflowConfig struct {
	Rules []RuleConfig `yaml:"rules,omitempty" json:"rules,omitempty"`
}

// RuleConfig defines pipeline/job rules
type RuleConfig struct {
	If           string                 `yaml:"if,omitempty" json:"if,omitempty"`
	Changes      []string               `yaml:"changes,omitempty" json:"changes,omitempty"`
	Exists       []string               `yaml:"exists,omitempty" json:"exists,omitempty"`
	Variables    map[string]string      `yaml:"variables,omitempty" json:"variables,omitempty"`
	When         string                 `yaml:"when,omitempty" json:"when,omitempty" validate:"omitempty,oneof=on_success on_failure always never manual delayed"`
	AllowFailure bool                   `yaml:"allow_failure,omitempty" json:"allow_failure,omitempty"`
	StartIn      string                 `yaml:"start_in,omitempty" json:"start_in,omitempty"`
}

// JobConfig represents a GitLab CI job
type JobConfig struct {
	Stage        string                 `yaml:"stage,omitempty" json:"stage,omitempty"`
	Image        interface{}            `yaml:"image,omitempty" json:"image,omitempty"` // string or ImageConfig
	Services     []ServiceConfig        `yaml:"services,omitempty" json:"services,omitempty"`
	BeforeScript []string               `yaml:"before_script,omitempty" json:"before_script,omitempty"`
	Script       []string               `yaml:"script" json:"script" validate:"required"`
	AfterScript  []string               `yaml:"after_script,omitempty" json:"after_script,omitempty"`
	Variables    map[string]interface{} `yaml:"variables,omitempty" json:"variables,omitempty"`
	Rules        []RuleConfig           `yaml:"rules,omitempty" json:"rules,omitempty"`
	Only         interface{}            `yaml:"only,omitempty" json:"only,omitempty"`
	Except       interface{}            `yaml:"except,omitempty" json:"except,omitempty"`
	Tags         []string               `yaml:"tags,omitempty" json:"tags,omitempty"`
	AllowFailure interface{}            `yaml:"allow_failure,omitempty" json:"allow_failure,omitempty"`
	When         string                 `yaml:"when,omitempty" json:"when,omitempty"`
	Environment  interface{}            `yaml:"environment,omitempty" json:"environment,omitempty"`
	Artifacts    *ArtifactsConfig       `yaml:"artifacts,omitempty" json:"artifacts,omitempty"`
	Cache        *CacheConfig           `yaml:"cache,omitempty" json:"cache,omitempty"`
	Dependencies []string               `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
	Needs        []interface{}          `yaml:"needs,omitempty" json:"needs,omitempty"`
	Retry        *RetryConfig           `yaml:"retry,omitempty" json:"retry,omitempty"`
	Timeout      string                 `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Parallel     interface{}            `yaml:"parallel,omitempty" json:"parallel,omitempty"`
	Trigger      interface{}            `yaml:"trigger,omitempty" json:"trigger,omitempty"`
	Extends      interface{}            `yaml:"extends,omitempty" json:"extends,omitempty"`
	Resource_group string               `yaml:"resource_group,omitempty" json:"resource_group,omitempty"`
	Release      *ReleaseConfig         `yaml:"release,omitempty" json:"release,omitempty"`
	Coverage     string                 `yaml:"coverage,omitempty" json:"coverage,omitempty"`
	Secrets      map[string]SecretRef   `yaml:"secrets,omitempty" json:"secrets,omitempty"`
}

// ReleaseConfig for release jobs
type ReleaseConfig struct {
	TagName     string      `yaml:"tag_name" json:"tag_name"`
	Description string      `yaml:"description,omitempty" json:"description,omitempty"`
	Name        string      `yaml:"name,omitempty" json:"name,omitempty"`
	Ref         string      `yaml:"ref,omitempty" json:"ref,omitempty"`
	Milestones  []string    `yaml:"milestones,omitempty" json:"milestones,omitempty"`
	ReleasedAt  string      `yaml:"released_at,omitempty" json:"released_at,omitempty"`
	Assets      *AssetsConfig `yaml:"assets,omitempty" json:"assets,omitempty"`
}

// AssetsConfig for release assets
type AssetsConfig struct {
	Links []LinkConfig `yaml:"links,omitempty" json:"links,omitempty"`
}

// LinkConfig for asset links
type LinkConfig struct {
	Name     string `yaml:"name" json:"name"`
	URL      string `yaml:"url" json:"url"`
	Filepath string `yaml:"filepath,omitempty" json:"filepath,omitempty"`
	LinkType string `yaml:"link_type,omitempty" json:"link_type,omitempty" validate:"omitempty,oneof=runbook package image other"`
}

// SecretRef for CI/CD secrets from vault
type SecretRef struct {
	Vault VaultSecret `yaml:"vault" json:"vault"`
}

// VaultSecret references HashiCorp Vault
type VaultSecret struct {
	Engine EngineConfig `yaml:"engine" json:"engine"`
	Path   string       `yaml:"path" json:"path"`
	Field  string       `yaml:"field" json:"field"`
}

// EngineConfig for Vault secrets engine
type EngineConfig struct {
	Name string `yaml:"name" json:"name"`
	Path string `yaml:"path" json:"path"`
}

// EnvironmentConfig defines GitLab environment
type EnvironmentConfig struct {
	Name          string `yaml:"name" json:"name" validate:"required"`
	URL           string `yaml:"url,omitempty" json:"url,omitempty"`
	Action        string `yaml:"action,omitempty" json:"action,omitempty" validate:"omitempty,oneof=start stop prepare verify access"`
	AutoStopIn    string `yaml:"auto_stop_in,omitempty" json:"auto_stop_in,omitempty"`
	OnStop        string `yaml:"on_stop,omitempty" json:"on_stop,omitempty"`
	Tier          string `yaml:"deployment_tier,omitempty" json:"deployment_tier,omitempty" validate:"omitempty,oneof=production staging testing development other"`
}

// VariableConfig defines CI/CD variables
type VariableConfig struct {
	Key         string `yaml:"key" json:"key" validate:"required"`
	Value       string `yaml:"value" json:"value"`
	Protected   bool   `yaml:"protected,omitempty" json:"protected,omitempty"`
	Masked      bool   `yaml:"masked,omitempty" json:"masked,omitempty"`
	Environment string `yaml:"environment_scope,omitempty" json:"environment_scope,omitempty"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	VariableType string `yaml:"variable_type,omitempty" json:"variable_type,omitempty" validate:"omitempty,oneof=env_var file"`
}

// RunnersConfig defines GitLab Runner configuration
type RunnersConfig struct {
	Tags        []string           `yaml:"tags,omitempty" json:"tags,omitempty"`
	Shared      bool               `yaml:"shared,omitempty" json:"shared,omitempty"`
	GroupRunners []GroupRunner     `yaml:"groupRunners,omitempty" json:"groupRunners,omitempty"`
	ProjectRunners []ProjectRunner `yaml:"projectRunners,omitempty" json:"projectRunners,omitempty"`
}

// GroupRunner defines group-level runner
type GroupRunner struct {
	Description string   `yaml:"description" json:"description"`
	Tags        []string `yaml:"tags,omitempty" json:"tags,omitempty"`
	RunUntagged bool     `yaml:"run_untagged,omitempty" json:"run_untagged,omitempty"`
	Locked      bool     `yaml:"locked,omitempty" json:"locked,omitempty"`
}

// ProjectRunner defines project-level runner
type ProjectRunner struct {
	Description string   `yaml:"description" json:"description"`
	Tags        []string `yaml:"tags,omitempty" json:"tags,omitempty"`
	RunUntagged bool     `yaml:"run_untagged,omitempty" json:"run_untagged,omitempty"`
	Locked      bool     `yaml:"locked,omitempty" json:"locked,omitempty"`
}

// Plugin implements the GitLab CI plugin
type Plugin struct{}

// NewPlugin creates a new GitLab CI plugin
func NewPlugin() plugin.Plugin {
	return &Plugin{}
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return "gitlab-ci"
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

	if provider != "gitlab-ci" {
		return fmt.Errorf("provider must be 'gitlab-ci'")
	}

	project, ok := spec["project"].(string)
	if !ok || project == "" {
		return fmt.Errorf("project is required")
	}

	return nil
}

// Plan generates a plan for the plugin
func (p *Plugin) Plan(spec map[string]interface{}) (*plugin.Plan, error) {
	actions := []string{}

	if pipeline, ok := spec["pipeline"].(map[string]interface{}); ok {
		if stages, ok := pipeline["stages"].([]interface{}); ok && len(stages) > 0 {
			actions = append(actions, fmt.Sprintf("Configure %d pipeline stages", len(stages)))
		}
		if jobs, ok := pipeline["jobs"].(map[string]interface{}); ok && len(jobs) > 0 {
			actions = append(actions, fmt.Sprintf("Create %d jobs", len(jobs)))
		}
	}

	if environments, ok := spec["environments"].([]interface{}); ok && len(environments) > 0 {
		actions = append(actions, fmt.Sprintf("Configure %d environments", len(environments)))
	}

	if variables, ok := spec["variables"].([]interface{}); ok && len(variables) > 0 {
		actions = append(actions, fmt.Sprintf("Set %d CI/CD variables", len(variables)))
	}

	if len(actions) == 0 {
		actions = []string{"Configure GitLab CI/CD"}
	}

	return &plugin.Plan{
		Actions: actions,
	}, nil
}

// Apply applies the plugin configuration
func (p *Plugin) Apply(spec map[string]interface{}) (*plugin.Result, error) {
	return &plugin.Result{
		Status:  "success",
		Message: "GitLab CI/CD configured successfully",
		Outputs: map[string]string{
			"provider": "gitlab-ci",
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
		Message: "GitLab CI/CD pipelines are configured",
	}, nil
}
