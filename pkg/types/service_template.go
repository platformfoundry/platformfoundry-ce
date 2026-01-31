package types

import (
	"fmt"
	"regexp"
)

// ServiceTemplate represents a template for creating services
type ServiceTemplate struct {
	APIVersion string              `yaml:"apiVersion" json:"apiVersion"`
	Kind       string              `yaml:"kind" json:"kind"`
	Metadata   Metadata            `yaml:"metadata" json:"metadata"`
	Spec       ServiceTemplateSpec `yaml:"spec" json:"spec"`
}

// ServiceTemplateSpec defines the service template specification
type ServiceTemplateSpec struct {
	DisplayName string                 `yaml:"displayName" json:"displayName"`
	Description string                 `yaml:"description,omitempty" json:"description,omitempty"`
	Category    TemplateCategory       `yaml:"category" json:"category"`
	Tags        []string               `yaml:"tags,omitempty" json:"tags,omitempty"`
	Icon        string                 `yaml:"icon,omitempty" json:"icon,omitempty"`
	Parameters  []TemplateParameter    `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	Template    string                 `yaml:"template" json:"template"` // Go template for service.yaml
	Files       []TemplateFile         `yaml:"files,omitempty" json:"files,omitempty"`
	Hooks       *TemplateHooks         `yaml:"hooks,omitempty" json:"hooks,omitempty"`
}

// TemplateCategory represents the category of a template
type TemplateCategory string

const (
	TemplateCategoryBackend   TemplateCategory = "backend"
	TemplateCategoryFrontend  TemplateCategory = "frontend"
	TemplateCategoryDatabase  TemplateCategory = "database"
	TemplateCategoryCache     TemplateCategory = "cache"
	TemplateCategoryQueue     TemplateCategory = "queue"
	TemplateCategoryMonitoring TemplateCategory = "monitoring"
	TemplateCategoryCI        TemplateCategory = "ci"
)

// TemplateParameter defines a parameter for the template
type TemplateParameter struct {
	Name        string              `yaml:"name" json:"name"`
	Type        ParameterType       `yaml:"type" json:"type"`
	Description string              `yaml:"description,omitempty" json:"description,omitempty"`
	Required    bool                `yaml:"required,omitempty" json:"required,omitempty"`
	Default     interface{}         `yaml:"default,omitempty" json:"default,omitempty"`
	Enum        []string            `yaml:"enum,omitempty" json:"enum,omitempty"`
	Pattern     string              `yaml:"pattern,omitempty" json:"pattern,omitempty"`
	MinLength   int                 `yaml:"minLength,omitempty" json:"minLength,omitempty"`
	MaxLength   int                 `yaml:"maxLength,omitempty" json:"maxLength,omitempty"`
	Min         *float64            `yaml:"min,omitempty" json:"min,omitempty"`
	Max         *float64            `yaml:"max,omitempty" json:"max,omitempty"`
}

// ParameterType represents the type of a template parameter
type ParameterType string

const (
	ParameterTypeString  ParameterType = "string"
	ParameterTypeNumber  ParameterType = "number"
	ParameterTypeBoolean ParameterType = "boolean"
	ParameterTypeArray   ParameterType = "array"
	ParameterTypeObject  ParameterType = "object"
)

// TemplateFile defines a file to be generated from the template
type TemplateFile struct {
	Path     string `yaml:"path" json:"path"`
	Content  string `yaml:"content" json:"content"` // Go template for file content
	Encoding string `yaml:"encoding,omitempty" json:"encoding,omitempty"` // base64, plain (default)
	Mode     string `yaml:"mode,omitempty" json:"mode,omitempty"` // file permissions (e.g., "0755")
}

// TemplateHooks defines lifecycle hooks for template instantiation
type TemplateHooks struct {
	PreGenerate  []string `yaml:"preGenerate,omitempty" json:"preGenerate,omitempty"`   // Commands to run before generation
	PostGenerate []string `yaml:"postGenerate,omitempty" json:"postGenerate,omitempty"` // Commands to run after generation
}

var (
	// templateNameRegex validates template names
	templateNameRegex = regexp.MustCompile(`^[a-z][a-z0-9-]*[a-z0-9]$`)
	// parameterNameRegex validates parameter names (camelCase or snake_case)
	parameterNameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
)

// Validate validates the service template with security checks
func (st *ServiceTemplate) Validate() error {
	if st.APIVersion == "" {
		return ErrMissingAPIVersion
	}
	if st.Kind != "ServiceTemplate" {
		return ErrInvalidKind
	}
	if st.Metadata.Name == "" {
		return ErrMissingName
	}

	// Security: Validate template name format
	if len(st.Metadata.Name) < 2 || len(st.Metadata.Name) > 63 {
		return fmt.Errorf("template name must be between 2 and 63 characters")
	}
	if !templateNameRegex.MatchString(st.Metadata.Name) {
		return fmt.Errorf("template name must be lowercase alphanumeric with hyphens")
	}

	// Validate display name
	if st.Spec.DisplayName == "" {
		return fmt.Errorf("display name is required")
	}
	if len(st.Spec.DisplayName) > 200 {
		return fmt.Errorf("display name must be 200 characters or less")
	}

	// Validate description
	if len(st.Spec.Description) > 2000 {
		return fmt.Errorf("description must be 2000 characters or less")
	}

	// Validate category
	if st.Spec.Category == "" {
		return fmt.Errorf("category is required")
	}
	if !IsValidTemplateCategory(st.Spec.Category) {
		return fmt.Errorf("invalid template category: %s", st.Spec.Category)
	}

	// Security: Limit number of tags
	if len(st.Spec.Tags) > 20 {
		return fmt.Errorf("too many tags (max 20)")
	}
	for i, tag := range st.Spec.Tags {
		if len(tag) > 50 {
			return fmt.Errorf("tag %d must be 50 characters or less", i)
		}
	}

	// Validate template content
	if st.Spec.Template == "" {
		return fmt.Errorf("template content is required")
	}
	// Security: Limit template size
	if len(st.Spec.Template) > 1024*1024 { // 1MB
		return fmt.Errorf("template content too large (max 1MB)")
	}

	// Security: Limit number of parameters
	if len(st.Spec.Parameters) > 50 {
		return fmt.Errorf("too many parameters (max 50)")
	}

	// Validate parameters
	for i, param := range st.Spec.Parameters {
		if param.Name == "" {
			return fmt.Errorf("parameter %d: name is required", i)
		}
		if len(param.Name) > 100 {
			return fmt.Errorf("parameter %d: name must be 100 characters or less", i)
		}
		if !parameterNameRegex.MatchString(param.Name) {
			return fmt.Errorf("parameter %d: name must be alphanumeric with underscores", i)
		}

		if param.Type == "" {
			return fmt.Errorf("parameter %d: type is required", i)
		}
		if !IsValidParameterType(param.Type) {
			return fmt.Errorf("parameter %d: invalid type %s", i, param.Type)
		}

		// Validate enum
		if len(param.Enum) > 100 {
			return fmt.Errorf("parameter %d: too many enum values (max 100)", i)
		}

		// Validate pattern if provided
		if param.Pattern != "" {
			if _, err := regexp.Compile(param.Pattern); err != nil {
				return fmt.Errorf("parameter %d: invalid regex pattern: %v", i, err)
			}
			// Security: Limit pattern complexity
			if len(param.Pattern) > 500 {
				return fmt.Errorf("parameter %d: pattern too complex (max 500 chars)", i)
			}
		}

		// Validate length constraints
		if param.MinLength < 0 || param.MinLength > 10000 {
			return fmt.Errorf("parameter %d: minLength must be between 0 and 10000", i)
		}
		if param.MaxLength < 0 || param.MaxLength > 10000 {
			return fmt.Errorf("parameter %d: maxLength must be between 0 and 10000", i)
		}
		if param.MinLength > 0 && param.MaxLength > 0 && param.MinLength > param.MaxLength {
			return fmt.Errorf("parameter %d: minLength cannot be greater than maxLength", i)
		}

		// Validate numeric constraints
		if param.Min != nil && param.Max != nil && *param.Min > *param.Max {
			return fmt.Errorf("parameter %d: min cannot be greater than max", i)
		}
	}

	// Security: Limit number of files
	if len(st.Spec.Files) > 200 {
		return fmt.Errorf("too many files (max 200)")
	}

	// Validate files
	for i, file := range st.Spec.Files {
		if file.Path == "" {
			return fmt.Errorf("file %d: path is required", i)
		}
		if len(file.Path) > 1024 {
			return fmt.Errorf("file %d: path must be 1024 characters or less", i)
		}
		if file.Content == "" {
			return fmt.Errorf("file %d: content is required", i)
		}
		// Security: Limit file content size
		if len(file.Content) > 10*1024*1024 { // 10MB
			return fmt.Errorf("file %d: content too large (max 10MB)", i)
		}

		// Validate encoding
		if file.Encoding != "" && file.Encoding != "base64" && file.Encoding != "plain" {
			return fmt.Errorf("file %d: invalid encoding (must be 'base64' or 'plain')", i)
		}

		// Validate mode if provided
		if file.Mode != "" {
			if !regexp.MustCompile(`^0[0-7]{3}$`).MatchString(file.Mode) {
				return fmt.Errorf("file %d: invalid mode format (must be octal like '0644')", i)
			}
		}
	}

	// Validate hooks
	if st.Spec.Hooks != nil {
		// Security: Limit number of hooks
		if len(st.Spec.Hooks.PreGenerate) > 20 {
			return fmt.Errorf("too many pre-generate hooks (max 20)")
		}
		if len(st.Spec.Hooks.PostGenerate) > 20 {
			return fmt.Errorf("too many post-generate hooks (max 20)")
		}

		// Validate hook commands
		for i, cmd := range st.Spec.Hooks.PreGenerate {
			if len(cmd) > 2048 {
				return fmt.Errorf("pre-generate hook %d: command too long (max 2048 chars)", i)
			}
		}
		for i, cmd := range st.Spec.Hooks.PostGenerate {
			if len(cmd) > 2048 {
				return fmt.Errorf("post-generate hook %d: command too long (max 2048 chars)", i)
			}
		}
	}

	return nil
}

// IsValidTemplateCategory checks if a template category is valid
func IsValidTemplateCategory(category TemplateCategory) bool {
	return category == TemplateCategoryBackend ||
		category == TemplateCategoryFrontend ||
		category == TemplateCategoryDatabase ||
		category == TemplateCategoryCache ||
		category == TemplateCategoryQueue ||
		category == TemplateCategoryMonitoring ||
		category == TemplateCategoryCI
}

// IsValidParameterType checks if a parameter type is valid
func IsValidParameterType(paramType ParameterType) bool {
	return paramType == ParameterTypeString ||
		paramType == ParameterTypeNumber ||
		paramType == ParameterTypeBoolean ||
		paramType == ParameterTypeArray ||
		paramType == ParameterTypeObject
}
