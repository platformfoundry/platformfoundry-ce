// Package scaffold provides configuration generation from templates
// for rapid platform setup and component scaffolding.
package scaffold

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// ScaffoldType defines what to scaffold
type ScaffoldType string

const (
	ScaffoldPlatform       ScaffoldType = "platform"
	ScaffoldInfrastructure ScaffoldType = "infrastructure"
	ScaffoldOrchestrator   ScaffoldType = "orchestrator"
	ScaffoldObservability  ScaffoldType = "observability"
	ScaffoldDevEx          ScaffoldType = "devex"
	ScaffoldSecurity       ScaffoldType = "security"
	ScaffoldFull           ScaffoldType = "full"
)

// ScaffoldConfig configures scaffold generation
type ScaffoldConfig struct {
	Type          ScaffoldType
	Name          string
	OutputDir     string
	CloudProvider string
	MockMode      bool
	IncludeTests  bool
	Environment   string
	Overwrite     bool
	DryRun        bool
}

// Generator creates scaffold configurations
type Generator struct {
	templates map[string]*template.Template
}

// GenerateResult contains generation results
type GenerateResult struct {
	Files     []GeneratedFile
	Warnings  []string
	NextSteps []string
}

// GeneratedFile represents a generated file
type GeneratedFile struct {
	Path    string
	Content string
	Created bool
	Skipped bool
	Reason  string
}

// NewGenerator creates a new scaffold generator
func NewGenerator() *Generator {
	g := &Generator{
		templates: make(map[string]*template.Template),
	}
	g.loadTemplates()
	return g
}

// Generate creates scaffold files
func (g *Generator) Generate(config ScaffoldConfig) (*GenerateResult, error) {
	switch config.Type {
	case ScaffoldFull:
		return g.generateFullPlatform(config)
	case ScaffoldPlatform:
		return g.generatePlatform(config)
	case ScaffoldInfrastructure:
		return g.generateInfrastructure(config)
	case ScaffoldOrchestrator:
		return g.generateOrchestrator(config)
	case ScaffoldObservability:
		return g.generateObservability(config)
	case ScaffoldDevEx:
		return g.generateDevEx(config)
	case ScaffoldSecurity:
		return g.generateSecurity(config)
	default:
		return nil, fmt.Errorf("unknown scaffold type: %s", config.Type)
	}
}

// generateFullPlatform generates a complete platform scaffold
func (g *Generator) generateFullPlatform(config ScaffoldConfig) (*GenerateResult, error) {
	result := &GenerateResult{
		Files:     make([]GeneratedFile, 0),
		NextSteps: make([]string, 0),
	}

	// Create output directory structure
	dirs := []string{
		filepath.Join(config.OutputDir, "infrastructure"),
		filepath.Join(config.OutputDir, "orchestrator"),
		filepath.Join(config.OutputDir, "observability"),
		filepath.Join(config.OutputDir, "devex"),
		filepath.Join(config.OutputDir, "environments"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil && !config.DryRun {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Generate platform file
	platformResult, err := g.generatePlatform(config)
	if err != nil {
		return nil, err
	}
	result.Files = append(result.Files, platformResult.Files...)

	// Generate infrastructure
	infraConfig := config
	infraConfig.Type = ScaffoldInfrastructure
	infraResult, err := g.generateInfrastructure(infraConfig)
	if err != nil {
		return nil, err
	}
	result.Files = append(result.Files, infraResult.Files...)

	// Generate orchestrator
	orchConfig := config
	orchConfig.Type = ScaffoldOrchestrator
	orchResult, err := g.generateOrchestrator(orchConfig)
	if err != nil {
		return nil, err
	}
	result.Files = append(result.Files, orchResult.Files...)

	// Generate observability
	obsConfig := config
	obsConfig.Type = ScaffoldObservability
	obsResult, err := g.generateObservability(obsConfig)
	if err != nil {
		return nil, err
	}
	result.Files = append(result.Files, obsResult.Files...)

	// Generate environments
	envResult, err := g.generateEnvironments(config)
	if err != nil {
		return nil, err
	}
	result.Files = append(result.Files, envResult.Files...)

	// Add next steps
	if config.MockMode {
		result.NextSteps = []string{
			fmt.Sprintf("Review generated files in %s", config.OutputDir),
			fmt.Sprintf("Run: pf apply -f %s/platform.yaml --mock", config.OutputDir),
			"Test your platform locally",
			"When ready, remove --mock flag to deploy for real",
		}
	} else {
		result.NextSteps = []string{
			fmt.Sprintf("Review and customize generated files in %s", config.OutputDir),
			"Set required environment variables",
			fmt.Sprintf("Run: pf plan -f %s/platform.yaml", config.OutputDir),
			fmt.Sprintf("Run: pf apply -f %s/platform.yaml", config.OutputDir),
		}
	}

	return result, nil
}

// generatePlatform generates the main platform file
func (g *Generator) generatePlatform(config ScaffoldConfig) (*GenerateResult, error) {
	data := g.buildTemplateData(config)

	content, err := g.executeTemplate("platform", data)
	if err != nil {
		return nil, err
	}

	file := GeneratedFile{
		Path:    filepath.Join(config.OutputDir, "platform.yaml"),
		Content: content,
	}

	if !config.DryRun {
		if err := g.writeFile(file.Path, content, config.Overwrite); err != nil {
			return nil, err
		}
		file.Created = true
	}

	return &GenerateResult{
		Files: []GeneratedFile{file},
	}, nil
}

// generateInfrastructure generates infrastructure files
func (g *Generator) generateInfrastructure(config ScaffoldConfig) (*GenerateResult, error) {
	data := g.buildTemplateData(config)

	content, err := g.executeTemplate("infrastructure", data)
	if err != nil {
		return nil, err
	}

	file := GeneratedFile{
		Path:    filepath.Join(config.OutputDir, "infrastructure", "infrastructure.yaml"),
		Content: content,
	}

	if !config.DryRun {
		if err := g.writeFile(file.Path, content, config.Overwrite); err != nil {
			return nil, err
		}
		file.Created = true
	}

	return &GenerateResult{
		Files: []GeneratedFile{file},
	}, nil
}

// generateOrchestrator generates orchestrator files
func (g *Generator) generateOrchestrator(config ScaffoldConfig) (*GenerateResult, error) {
	data := g.buildTemplateData(config)

	content, err := g.executeTemplate("orchestrator", data)
	if err != nil {
		return nil, err
	}

	file := GeneratedFile{
		Path:    filepath.Join(config.OutputDir, "orchestrator", "orchestrator.yaml"),
		Content: content,
	}

	if !config.DryRun {
		if err := g.writeFile(file.Path, content, config.Overwrite); err != nil {
			return nil, err
		}
		file.Created = true
	}

	return &GenerateResult{
		Files: []GeneratedFile{file},
	}, nil
}

// generateObservability generates observability files
func (g *Generator) generateObservability(config ScaffoldConfig) (*GenerateResult, error) {
	data := g.buildTemplateData(config)

	content, err := g.executeTemplate("observability", data)
	if err != nil {
		return nil, err
	}

	file := GeneratedFile{
		Path:    filepath.Join(config.OutputDir, "observability", "observability.yaml"),
		Content: content,
	}

	if !config.DryRun {
		if err := g.writeFile(file.Path, content, config.Overwrite); err != nil {
			return nil, err
		}
		file.Created = true
	}

	return &GenerateResult{
		Files: []GeneratedFile{file},
	}, nil
}

// generateDevEx generates devex files
func (g *Generator) generateDevEx(config ScaffoldConfig) (*GenerateResult, error) {
	data := g.buildTemplateData(config)

	content, err := g.executeTemplate("devex", data)
	if err != nil {
		return nil, err
	}

	file := GeneratedFile{
		Path:    filepath.Join(config.OutputDir, "devex", "devex.yaml"),
		Content: content,
	}

	if !config.DryRun {
		if err := g.writeFile(file.Path, content, config.Overwrite); err != nil {
			return nil, err
		}
		file.Created = true
	}

	return &GenerateResult{
		Files: []GeneratedFile{file},
	}, nil
}

// generateSecurity generates security files
func (g *Generator) generateSecurity(config ScaffoldConfig) (*GenerateResult, error) {
	data := g.buildTemplateData(config)

	content, err := g.executeTemplate("security", data)
	if err != nil {
		return nil, err
	}

	file := GeneratedFile{
		Path:    filepath.Join(config.OutputDir, "security", "security.yaml"),
		Content: content,
	}

	if !config.DryRun {
		if err := g.writeFile(file.Path, content, config.Overwrite); err != nil {
			return nil, err
		}
		file.Created = true
	}

	return &GenerateResult{
		Files: []GeneratedFile{file},
	}, nil
}

// buildTemplateData creates the data map for templates
func (g *Generator) buildTemplateData(config ScaffoldConfig) map[string]interface{} {
	return map[string]interface{}{
		"Name":          config.Name,
		"CloudProvider": config.CloudProvider,
		"Environment":   config.Environment,
		"MockMode":      config.MockMode,
		"IsProd":        config.Environment == "prod" || config.Environment == "production",
	}
}

// executeTemplate executes a template with data
func (g *Generator) executeTemplate(name string, data map[string]interface{}) (string, error) {
	tmpl, ok := g.templates[name]
	if !ok {
		return "", fmt.Errorf("template %s not found", name)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}

	return buf.String(), nil
}

// writeFile writes content to a file
func (g *Generator) writeFile(path, content string, overwrite bool) error {
	// Create parent directory
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Check if file exists
	if _, err := os.Stat(path); err == nil && !overwrite {
		return fmt.Errorf("file %s already exists, use --overwrite to replace", path)
	}

	// Write file
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}

	return nil
}

// loadTemplates loads all scaffold templates
func (g *Generator) loadTemplates() {
	funcMap := template.FuncMap{
		"indent":  indent,
		"toYaml":  toYaml,
		"quote":   quote,
		"lower":   strings.ToLower,
		"upper":   strings.ToUpper,
		"default": defaultValue,
	}

	g.templates["platform"] = template.Must(template.New("platform").Funcs(funcMap).Parse(platformTemplate))
	g.templates["infrastructure"] = template.Must(template.New("infrastructure").Funcs(funcMap).Parse(infrastructureTemplate))
	g.templates["orchestrator"] = template.Must(template.New("orchestrator").Funcs(funcMap).Parse(orchestratorTemplate))
	g.templates["observability"] = template.Must(template.New("observability").Funcs(funcMap).Parse(observabilityTemplate))
	g.templates["devex"] = template.Must(template.New("devex").Funcs(funcMap).Parse(devexTemplate))
	g.templates["security"] = template.Must(template.New("security").Funcs(funcMap).Parse(securityTemplate))
}

// Template helper functions
func indent(spaces int, s string) string {
	pad := strings.Repeat(" ", spaces)
	return pad + strings.ReplaceAll(s, "\n", "\n"+pad)
}

func toYaml(v interface{}) string {
	b, _ := yaml.Marshal(v)
	return string(b)
}

func quote(s string) string {
	return fmt.Sprintf("%q", s)
}

func defaultValue(def, val interface{}) interface{} {
	if val == nil || val == "" {
		return def
	}
	return val
}

// Template definitions
const platformTemplate = `apiVersion: platformfoundry.io/v1
kind: Platform
metadata:
  name: {{ .Name }}
  labels:
    environment: {{ .Environment }}
    provider: {{ .CloudProvider }}
spec:
{{- if .MockMode }}
  # MOCK MODE - Using mock providers for fast iteration
  mockEnabled: true
{{- end }}

  components:
    infrastructure:
      ref: {{ .Name }}-infrastructure
    orchestrator:
      ref: {{ .Name }}-orchestrator
    observability:
      ref: {{ .Name }}-observability
    devex:
      ref: {{ .Name }}-devex
`

const infrastructureTemplate = `apiVersion: platformfoundry.io/v1
kind: Infrastructure
metadata:
  name: {{ .Name }}-infrastructure
spec:
{{- if .MockMode }}
  provider: mock
{{- else }}
  provider: {{ .CloudProvider }}
{{- end }}

{{- if eq .CloudProvider "aws" }}
  region: us-west-2

  vpc:
    name: {{ .Name }}-vpc
    cidr: "10.0.0.0/16"
    subnets:
      - name: private-1
        cidr: "10.0.1.0/24"
        zone: us-west-2a
        public: false
      - name: private-2
        cidr: "10.0.2.0/24"
        zone: us-west-2b
        public: false
      - name: public-1
        cidr: "10.0.101.0/24"
        zone: us-west-2a
        public: true

  cluster:
    name: {{ .Name }}-eks
    version: "1.28"
    nodeGroups:
      - name: default
        instanceType: t3.medium
        minSize: 2
        maxSize: 10
        desiredSize: 3
{{- end }}

{{- if eq .CloudProvider "gcp" }}
  region: us-central1
  project: {{ .Name }}-project

  network:
    name: {{ .Name }}-network
    subnets:
      - name: primary
        cidr: "10.0.0.0/20"
        region: us-central1

  cluster:
    name: {{ .Name }}-gke
    version: "1.28"
    nodePool:
      name: default
      machineType: e2-medium
      minNodes: 2
      maxNodes: 10
{{- end }}

{{- if eq .CloudProvider "azure" }}
  region: eastus
  resourceGroup: {{ .Name }}-rg

  network:
    name: {{ .Name }}-vnet
    addressSpace: "10.0.0.0/16"
    subnets:
      - name: default
        addressPrefix: "10.0.1.0/24"

  cluster:
    name: {{ .Name }}-aks
    version: "1.28"
    nodePool:
      name: default
      vmSize: Standard_D2s_v3
      minCount: 2
      maxCount: 10
{{- end }}
`

const orchestratorTemplate = `apiVersion: platformfoundry.io/v1
kind: Orchestrator
metadata:
  name: {{ .Name }}-orchestrator
spec:
{{- if .MockMode }}
  provider: mock
{{- else }}
  provider: argocd
{{- end }}

  clusterRef: {{ .Name }}-infrastructure

  argocd:
    version: "2.9.0"
    namespace: argocd
    ha: {{ if .IsProd }}true{{ else }}false{{ end }}

    repositories:
      - name: main-apps
        url: https://github.com/{{ .Name }}/gitops.git

    applications:
      - name: platform-apps
        source:
          repoURL: https://github.com/{{ .Name }}/gitops.git
          path: apps
          targetRevision: HEAD
        destination:
          namespace: default
        syncPolicy:
          automated:
            prune: true
            selfHeal: true
`

const observabilityTemplate = `apiVersion: platformfoundry.io/v1
kind: Observability
metadata:
  name: {{ .Name }}-observability
spec:
{{- if .MockMode }}
  provider: mock
{{- else }}
  provider: prometheus-stack
{{- end }}

  clusterRef: {{ .Name }}-infrastructure

  prometheus:
    version: "2.47.0"
    retention: {{ if .IsProd }}30d{{ else }}7d{{ end }}
    storage:
      size: {{ if .IsProd }}100Gi{{ else }}20Gi{{ end }}

  grafana:
    version: "10.1.0"
    dashboards:
      - kubernetes
      - node-exporter
      - argocd

  alertmanager:
    enabled: true
    receivers:
      - name: slack
        type: slack
        config:
          channel: "#alerts-{{ .Environment }}"
`

const devexTemplate = `apiVersion: platformfoundry.io/v1
kind: DevEx
metadata:
  name: {{ .Name }}-devex
spec:
{{- if .MockMode }}
  provider: mock
{{- else }}
  provider: backstage
{{- end }}

  clusterRef: {{ .Name }}-infrastructure

  backstage:
    version: "1.20.0"

    catalog:
      locations:
        - type: url
          target: https://github.com/{{ .Name }}/software-catalog/blob/main/catalog-info.yaml

    integrations:
      github:
        - host: github.com
          apps:
            - appId: ${GITHUB_APP_ID}
              privateKey: ${GITHUB_APP_PRIVATE_KEY}

    auth:
      providers:
        github:
          clientId: ${GITHUB_CLIENT_ID}
          clientSecret: ${GITHUB_CLIENT_SECRET}
`

const securityTemplate = `apiVersion: platformfoundry.io/v1
kind: Security
metadata:
  name: {{ .Name }}-security
spec:
{{- if .MockMode }}
  provider: mock
{{- else }}
  provider: vault
{{- end }}

  clusterRef: {{ .Name }}-infrastructure

  vault:
    version: "1.15.0"
    ha: {{ if .IsProd }}true{{ else }}false{{ end }}
    audit:
      enabled: true

  externalSecrets:
    enabled: true
    provider: {{ .CloudProvider }}

  policies:
    enabled: true
    engine: opa-gatekeeper
    rules:
      - require-labels
      - restrict-privileged
      - require-probes
`
