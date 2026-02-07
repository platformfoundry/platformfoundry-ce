package parser

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/platformfoundry/pf-ce/pkg/types"
	"gopkg.in/yaml.v3"
)

// Parser handles parsing of Platform Foundry YAML files
// Implements US-6.1: Multi-Resource YAML Parser Update
type Parser struct{}

// ParsedResource represents any parsed resource type
type ParsedResource interface {
	Validate() error
	GetKind() string
	GetName() string
}

// New creates a new Parser
func New() *Parser {
	return &Parser{}
}

// ParseFile parses a YAML file and returns a list of resources
func (p *Parser) ParseFile(filename string) ([]types.Resource, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	return p.Parse(data)
}

// ParseString parses a YAML string and returns a list of resources
func (p *Parser) ParseString(yamlStr string) ([]types.Resource, error) {
	return p.Parse([]byte(yamlStr))
}

// Parse parses YAML data and returns a list of resources
// Supports both single and multi-document YAML (separated by ---)
func (p *Parser) Parse(data []byte) ([]types.Resource, error) {
	var resources []types.Resource

	decoder := yaml.NewDecoder(bytes.NewReader(data))

	for {
		var resource types.Resource
		err := decoder.Decode(&resource)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}

		// Skip empty documents
		if resource.Kind == "" {
			continue
		}

		// Validate required fields
		if err := p.validateResource(&resource); err != nil {
			return nil, err
		}

		resources = append(resources, resource)
	}

	// Return empty list if no resources (not an error for empty input)
	return resources, nil
}

// ParseTyped parses YAML data and returns typed resources
// This supports parsing of Platform, Infrastructure, Orchestrator, Observability, DevEx types
func (p *Parser) ParseTyped(data []byte) ([]ParsedResource, error) {
	var resources []ParsedResource

	decoder := yaml.NewDecoder(bytes.NewReader(data))

	for {
		// First decode to determine type
		var raw map[string]interface{}
		err := decoder.Decode(&raw)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}

		// Skip empty documents
		kind, ok := raw["kind"].(string)
		if !ok || kind == "" {
			continue
		}

		// Re-parse as specific type
		rawBytes, err := yaml.Marshal(raw)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal raw data: %w", err)
		}

		switch kind {
		case "Platform":
			var platform types.Platform
			if err := yaml.Unmarshal(rawBytes, &platform); err != nil {
				return nil, fmt.Errorf("failed to parse Platform: %w", err)
			}
			if err := platform.Validate(); err != nil {
				return nil, fmt.Errorf("invalid Platform %s: %w", platform.Metadata.Name, err)
			}
			resources = append(resources, &PlatformResource{Platform: &platform})

		case "Infrastructure":
			var infra types.Infrastructure
			if err := yaml.Unmarshal(rawBytes, &infra); err != nil {
				return nil, fmt.Errorf("failed to parse Infrastructure: %w", err)
			}
			if err := infra.Validate(); err != nil {
				return nil, fmt.Errorf("invalid Infrastructure %s: %w", infra.Metadata.Name, err)
			}
			resources = append(resources, &InfrastructureResource{Infrastructure: &infra})

		case "Orchestrator":
			var orch types.Orchestrator
			if err := yaml.Unmarshal(rawBytes, &orch); err != nil {
				return nil, fmt.Errorf("failed to parse Orchestrator: %w", err)
			}
			if err := orch.Validate(); err != nil {
				return nil, fmt.Errorf("invalid Orchestrator %s: %w", orch.Metadata.Name, err)
			}
			resources = append(resources, &OrchestratorResource{Orchestrator: &orch})

		case "Observability":
			var obs types.Observability
			if err := yaml.Unmarshal(rawBytes, &obs); err != nil {
				return nil, fmt.Errorf("failed to parse Observability: %w", err)
			}
			if err := obs.Validate(); err != nil {
				return nil, fmt.Errorf("invalid Observability %s: %w", obs.Metadata.Name, err)
			}
			resources = append(resources, &ObservabilityResource{Observability: &obs})

		case "DevEx":
			var devex types.DevEx
			if err := yaml.Unmarshal(rawBytes, &devex); err != nil {
				return nil, fmt.Errorf("failed to parse DevEx: %w", err)
			}
			if err := devex.Validate(); err != nil {
				return nil, fmt.Errorf("invalid DevEx %s: %w", devex.Metadata.Name, err)
			}
			resources = append(resources, &DevExResource{DevEx: &devex})

		case "Environment":
			var env types.Environment
			if err := yaml.Unmarshal(rawBytes, &env); err != nil {
				return nil, fmt.Errorf("failed to parse Environment: %w", err)
			}
			if err := env.Validate(); err != nil {
				return nil, fmt.Errorf("invalid Environment %s: %w", env.Metadata.Name, err)
			}
			resources = append(resources, &EnvironmentResource{Environment: &env})

		case "Service":
			var service types.Service
			if err := yaml.Unmarshal(rawBytes, &service); err != nil {
				return nil, fmt.Errorf("failed to parse Service: %w", err)
			}
			if err := service.Validate(); err != nil {
				return nil, fmt.Errorf("invalid Service %s: %w", service.Metadata.Name, err)
			}
			resources = append(resources, &ServiceResource{Service: &service})

		case "ServiceTemplate":
			var template types.ServiceTemplate
			if err := yaml.Unmarshal(rawBytes, &template); err != nil {
				return nil, fmt.Errorf("failed to parse ServiceTemplate: %w", err)
			}
			if err := template.Validate(); err != nil {
				return nil, fmt.Errorf("invalid ServiceTemplate %s: %w", template.Metadata.Name, err)
			}
			resources = append(resources, &ServiceTemplateResource{ServiceTemplate: &template})

		case "ServiceAction":
			var action types.ServiceAction
			if err := yaml.Unmarshal(rawBytes, &action); err != nil {
				return nil, fmt.Errorf("failed to parse ServiceAction: %w", err)
			}
			if err := action.Validate(); err != nil {
				return nil, fmt.Errorf("invalid ServiceAction %s: %w", action.Metadata.Name, err)
			}
			resources = append(resources, &ServiceActionResource{ServiceAction: &action})

		case "ServiceScorecard":
			var scorecard types.ServiceScorecard
			if err := yaml.Unmarshal(rawBytes, &scorecard); err != nil {
				return nil, fmt.Errorf("failed to parse ServiceScorecard: %w", err)
			}
			if err := scorecard.Validate(); err != nil {
				return nil, fmt.Errorf("invalid ServiceScorecard %s: %w", scorecard.Metadata.Name, err)
			}
			resources = append(resources, &ServiceScorecardResource{ServiceScorecard: &scorecard})

		default:
			return nil, fmt.Errorf("unknown resource kind: %s", kind)
		}
	}

	if len(resources) == 0 {
		return nil, fmt.Errorf("no valid resources found in YAML")
	}

	return resources, nil
}

// ParseFileTyped parses a YAML file and returns typed resources
func (p *Parser) ParseFileTyped(filename string) ([]ParsedResource, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	return p.ParseTyped(data)
}

// validateResource validates that a resource has required fields
func (p *Parser) validateResource(resource *types.Resource) error {
	if resource.APIVersion == "" {
		return fmt.Errorf("resource missing apiVersion")
	}
	if resource.Kind == "" {
		return fmt.Errorf("resource missing kind")
	}
	if resource.Metadata.Name == "" {
		return fmt.Errorf("resource missing metadata.name")
	}
	if resource.Spec == nil {
		return fmt.Errorf("resource %s missing spec", resource.Metadata.Name)
	}

	return nil
}

// Wrapper types for typed resources

type PlatformResource struct {
	Platform *types.Platform
}

func (r *PlatformResource) Validate() error {
	return r.Platform.Validate()
}

func (r *PlatformResource) GetKind() string {
	return "Platform"
}

func (r *PlatformResource) GetName() string {
	return r.Platform.Metadata.Name
}

type InfrastructureResource struct {
	Infrastructure *types.Infrastructure
}

func (r *InfrastructureResource) Validate() error {
	return r.Infrastructure.Validate()
}

func (r *InfrastructureResource) GetKind() string {
	return "Infrastructure"
}

func (r *InfrastructureResource) GetName() string {
	return r.Infrastructure.Metadata.Name
}

type OrchestratorResource struct {
	Orchestrator *types.Orchestrator
}

func (r *OrchestratorResource) Validate() error {
	return r.Orchestrator.Validate()
}

func (r *OrchestratorResource) GetKind() string {
	return "Orchestrator"
}

func (r *OrchestratorResource) GetName() string {
	return r.Orchestrator.Metadata.Name
}

type ObservabilityResource struct {
	Observability *types.Observability
}

func (r *ObservabilityResource) Validate() error {
	return r.Observability.Validate()
}

func (r *ObservabilityResource) GetKind() string {
	return "Observability"
}

func (r *ObservabilityResource) GetName() string {
	return r.Observability.Metadata.Name
}

type DevExResource struct {
	DevEx *types.DevEx
}

func (r *DevExResource) Validate() error {
	return r.DevEx.Validate()
}

func (r *DevExResource) GetKind() string {
	return "DevEx"
}

func (r *DevExResource) GetName() string {
	return r.DevEx.Metadata.Name
}

type EnvironmentResource struct {
	Environment *types.Environment
}

func (r *EnvironmentResource) Validate() error {
	return r.Environment.Validate()
}

func (r *EnvironmentResource) GetKind() string {
	return "Environment"
}

func (r *EnvironmentResource) GetName() string {
	return r.Environment.Metadata.Name
}

type ServiceResource struct {
	Service *types.Service
}

func (r *ServiceResource) Validate() error {
	return r.Service.Validate()
}

func (r *ServiceResource) GetKind() string {
	return "Service"
}

func (r *ServiceResource) GetName() string {
	return r.Service.Metadata.Name
}

type ServiceTemplateResource struct {
	ServiceTemplate *types.ServiceTemplate
}

func (r *ServiceTemplateResource) Validate() error {
	return r.ServiceTemplate.Validate()
}

func (r *ServiceTemplateResource) GetKind() string {
	return "ServiceTemplate"
}

func (r *ServiceTemplateResource) GetName() string {
	return r.ServiceTemplate.Metadata.Name
}

type ServiceActionResource struct {
	ServiceAction *types.ServiceAction
}

func (r *ServiceActionResource) Validate() error {
	return r.ServiceAction.Validate()
}

func (r *ServiceActionResource) GetKind() string {
	return "ServiceAction"
}

func (r *ServiceActionResource) GetName() string {
	return r.ServiceAction.Metadata.Name
}

type ServiceScorecardResource struct {
	ServiceScorecard *types.ServiceScorecard
}

func (r *ServiceScorecardResource) Validate() error {
	return r.ServiceScorecard.Validate()
}

func (r *ServiceScorecardResource) GetKind() string {
	return "ServiceScorecard"
}

func (r *ServiceScorecardResource) GetName() string {
	return r.ServiceScorecard.Metadata.Name
}
