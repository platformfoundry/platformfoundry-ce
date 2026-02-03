package service

import (
	"encoding/json"
	"fmt"

	"github.com/platformfoundry/pf-ce/internal/state"
	"github.com/platformfoundry/pf-ce/pkg/types"
)

// TemplateManager handles CRUD operations for service templates
type TemplateManager struct {
	backend state.Backend
	engine  *TemplateEngine
}

// NewTemplateManager creates a new template manager
func NewTemplateManager(backend state.Backend) *TemplateManager {
	return &TemplateManager{
		backend: backend,
		engine:  NewTemplateEngine(),
	}
}

// Create creates a new service template
func (tm *TemplateManager) Create(template *types.ServiceTemplate) error {
	// Validate template
	if err := template.Validate(); err != nil {
		return fmt.Errorf("invalid template: %w", err)
	}

	// Check if template already exists
	existing, err := tm.Get(template.Metadata.Name, template.Metadata.Organization)
	if err == nil && existing != nil {
		return fmt.Errorf("template %s already exists in organization %s",
			template.Metadata.Name, template.Metadata.Organization)
	}

	// Convert template to state resource
	resource, err := tm.templateToResource(template)
	if err != nil {
		return fmt.Errorf("failed to convert template to resource: %w", err)
	}

	// Save to backend
	if err := tm.backend.Save(resource); err != nil {
		return fmt.Errorf("failed to save template: %w", err)
	}

	return nil
}

// Update updates an existing service template
func (tm *TemplateManager) Update(template *types.ServiceTemplate) error {
	// Validate template
	if err := template.Validate(); err != nil {
		return fmt.Errorf("invalid template: %w", err)
	}

	// Check if template exists
	existing, err := tm.Get(template.Metadata.Name, template.Metadata.Organization)
	if err != nil || existing == nil {
		return fmt.Errorf("template %s not found in organization %s",
			template.Metadata.Name, template.Metadata.Organization)
	}

	// Convert template to state resource
	resource, err := tm.templateToResource(template)
	if err != nil {
		return fmt.Errorf("failed to convert template to resource: %w", err)
	}

	// Save to backend
	if err := tm.backend.Save(resource); err != nil {
		return fmt.Errorf("failed to update template: %w", err)
	}

	return nil
}

// Get retrieves a template by name and organization
func (tm *TemplateManager) Get(name, organization string) (*types.ServiceTemplate, error) {
	resourceName := tm.buildResourceName(name, organization)
	resource, err := tm.backend.Get(resourceName)
	if err != nil {
		return nil, fmt.Errorf("template %s not found: %w", name, err)
	}

	// Convert resource to template
	template, err := tm.resourceToTemplate(resource)
	if err != nil {
		return nil, fmt.Errorf("failed to convert resource to template: %w", err)
	}

	return template, nil
}

// List returns all templates, optionally filtered by organization
func (tm *TemplateManager) List(organization string) ([]*types.ServiceTemplate, error) {
	resources, err := tm.backend.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}

	templates := make([]*types.ServiceTemplate, 0)
	for _, resource := range resources {
		// Filter by kind
		if resource.Kind != "ServiceTemplate" {
			continue
		}

		// Filter by organization if specified
		if organization != "" {
			orgValue, ok := resource.Spec["organization"]
			if !ok || orgValue != organization {
				continue
			}
		}

		template, err := tm.resourceToTemplate(resource)
		if err != nil {
			// Log error but continue
			continue
		}

		templates = append(templates, template)
	}

	return templates, nil
}

// Delete deletes a template
func (tm *TemplateManager) Delete(name, organization string) error {
	// Check if template exists
	existing, err := tm.Get(name, organization)
	if err != nil || existing == nil {
		return fmt.Errorf("template %s not found in organization %s", name, organization)
	}

	// Delete from backend
	resourceName := tm.buildResourceName(name, organization)
	if err := tm.backend.Delete(resourceName); err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	return nil
}

// ListByCategory returns templates filtered by category
func (tm *TemplateManager) ListByCategory(category types.TemplateCategory, organization string) ([]*types.ServiceTemplate, error) {
	allTemplates, err := tm.List(organization)
	if err != nil {
		return nil, err
	}

	filtered := make([]*types.ServiceTemplate, 0)
	for _, template := range allTemplates {
		if template.Spec.Category == category {
			filtered = append(filtered, template)
		}
	}

	return filtered, nil
}

// SearchByTag returns templates that have the specified tag
func (tm *TemplateManager) SearchByTag(tag, organization string) ([]*types.ServiceTemplate, error) {
	allTemplates, err := tm.List(organization)
	if err != nil {
		return nil, err
	}

	matched := make([]*types.ServiceTemplate, 0)
	for _, template := range allTemplates {
		for _, t := range template.Spec.Tags {
			if t == tag {
				matched = append(matched, template)
				break
			}
		}
	}

	return matched, nil
}

// Instantiate creates a service from a template with the given parameters
func (tm *TemplateManager) Instantiate(templateName, organization string, params map[string]interface{}) (*types.Service, error) {
	// Get template
	template, err := tm.Get(templateName, organization)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// Validate parameters
	if err := tm.validateParameters(template, params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// Render template
	serviceYAML, err := tm.engine.Render(template.Spec.Template, params)
	if err != nil {
		return nil, fmt.Errorf("failed to render template: %w", err)
	}

	// Parse service YAML
	var service types.Service
	if err := json.Unmarshal([]byte(serviceYAML), &service); err != nil {
		return nil, fmt.Errorf("failed to parse service YAML: %w", err)
	}

	// Validate service
	if err := service.Validate(); err != nil {
		return nil, fmt.Errorf("invalid service: %w", err)
	}

	return &service, nil
}

// InstantiateFiles generates all files from a template with the given parameters
func (tm *TemplateManager) InstantiateFiles(templateName, organization string, params map[string]interface{}) (map[string]string, error) {
	// Get template
	template, err := tm.Get(templateName, organization)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// Validate parameters
	if err := tm.validateParameters(template, params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// Render all files
	files := make(map[string]string)
	for _, file := range template.Spec.Files {
		content, err := tm.engine.Render(file.Content, params)
		if err != nil {
			return nil, fmt.Errorf("failed to render file %s: %w", file.Path, err)
		}

		// Render path as template too (for dynamic paths)
		path, err := tm.engine.Render(file.Path, params)
		if err != nil {
			return nil, fmt.Errorf("failed to render path %s: %w", file.Path, err)
		}

		files[path] = content
	}

	return files, nil
}

// validateParameters validates that all required parameters are provided
func (tm *TemplateManager) validateParameters(template *types.ServiceTemplate, params map[string]interface{}) error {
	for _, param := range template.Spec.Parameters {
		value, exists := params[param.Name]

		// Check if required parameter is missing
		if param.Required && !exists {
			return fmt.Errorf("required parameter '%s' is missing", param.Name)
		}

		// If parameter doesn't exist and has default, use default
		if !exists && param.Default != nil {
			params[param.Name] = param.Default
			continue
		}

		// Skip validation if parameter doesn't exist
		if !exists {
			continue
		}

		// Validate parameter type
		if err := tm.validateParameterType(param, value); err != nil {
			return fmt.Errorf("parameter '%s': %w", param.Name, err)
		}

		// Validate enum if specified
		if len(param.Enum) > 0 {
			if err := tm.validateEnum(param, value); err != nil {
				return fmt.Errorf("parameter '%s': %w", param.Name, err)
			}
		}

		// Validate string constraints
		if param.Type == types.ParameterTypeString {
			if err := tm.validateStringConstraints(param, value); err != nil {
				return fmt.Errorf("parameter '%s': %w", param.Name, err)
			}
		}

		// Validate numeric constraints
		if param.Type == types.ParameterTypeNumber {
			if err := tm.validateNumericConstraints(param, value); err != nil {
				return fmt.Errorf("parameter '%s': %w", param.Name, err)
			}
		}
	}

	return nil
}

// validateParameterType validates the type of a parameter value
func (tm *TemplateManager) validateParameterType(param types.TemplateParameter, value interface{}) error {
	switch param.Type {
	case types.ParameterTypeString:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
	case types.ParameterTypeNumber:
		switch value.(type) {
		case int, int32, int64, float32, float64:
			// Valid numeric types
		default:
			return fmt.Errorf("expected number, got %T", value)
		}
	case types.ParameterTypeBoolean:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", value)
		}
	case types.ParameterTypeArray:
		if _, ok := value.([]interface{}); !ok {
			return fmt.Errorf("expected array, got %T", value)
		}
	case types.ParameterTypeObject:
		if _, ok := value.(map[string]interface{}); !ok {
			return fmt.Errorf("expected object, got %T", value)
		}
	}
	return nil
}

// validateEnum validates that a value is in the enum list
func (tm *TemplateManager) validateEnum(param types.TemplateParameter, value interface{}) error {
	strValue := fmt.Sprintf("%v", value)
	for _, enumValue := range param.Enum {
		if enumValue == strValue {
			return nil
		}
	}
	return fmt.Errorf("value '%v' is not in enum: %v", value, param.Enum)
}

// validateStringConstraints validates string length constraints
func (tm *TemplateManager) validateStringConstraints(param types.TemplateParameter, value interface{}) error {
	strValue, ok := value.(string)
	if !ok {
		return nil
	}

	if param.MinLength > 0 && len(strValue) < param.MinLength {
		return fmt.Errorf("string length %d is less than minimum %d", len(strValue), param.MinLength)
	}

	if param.MaxLength > 0 && len(strValue) > param.MaxLength {
		return fmt.Errorf("string length %d exceeds maximum %d", len(strValue), param.MaxLength)
	}

	return nil
}

// validateNumericConstraints validates numeric range constraints
func (tm *TemplateManager) validateNumericConstraints(param types.TemplateParameter, value interface{}) error {
	var numValue float64

	switch v := value.(type) {
	case int:
		numValue = float64(v)
	case int32:
		numValue = float64(v)
	case int64:
		numValue = float64(v)
	case float32:
		numValue = float64(v)
	case float64:
		numValue = v
	default:
		return nil
	}

	if param.Min != nil && numValue < *param.Min {
		return fmt.Errorf("value %v is less than minimum %v", numValue, *param.Min)
	}

	if param.Max != nil && numValue > *param.Max {
		return fmt.Errorf("value %v exceeds maximum %v", numValue, *param.Max)
	}

	return nil
}

// Helper functions

func (tm *TemplateManager) buildResourceName(name, organization string) string {
	if organization != "" {
		return fmt.Sprintf("%s/%s", organization, name)
	}
	return name
}

func (tm *TemplateManager) templateToResource(template *types.ServiceTemplate) (*state.Resource, error) {
	// Marshal spec
	specMap, err := structToMap(template.Spec)
	if err != nil {
		return nil, err
	}

	// Add metadata to spec for filtering
	specMap["organization"] = template.Metadata.Organization
	specMap["name"] = template.Metadata.Name

	return &state.Resource{
		Name:       tm.buildResourceName(template.Metadata.Name, template.Metadata.Organization),
		Kind:       template.Kind,
		APIVersion: template.APIVersion,
		Spec:       specMap,
	}, nil
}

func (tm *TemplateManager) resourceToTemplate(resource *state.Resource) (*types.ServiceTemplate, error) {
	// Marshal and unmarshal to convert maps to structs
	data, err := json.Marshal(map[string]interface{}{
		"apiVersion": resource.APIVersion,
		"kind":       resource.Kind,
		"metadata": map[string]interface{}{
			"name":         resource.Spec["name"],
			"organization": resource.Spec["organization"],
		},
		"spec": resource.Spec,
	})
	if err != nil {
		return nil, err
	}

	var template types.ServiceTemplate
	if err := json.Unmarshal(data, &template); err != nil {
		return nil, err
	}

	return &template, nil
}
