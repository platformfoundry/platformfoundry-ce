// Package crd provides Custom Resource Definition support for Platform Foundry.
package crd

import (
	"fmt"
	"regexp"
	"strings"
)

// Validator validates custom resources against their CRD schemas
type Validator struct {
	registry *Registry
}

// NewValidator creates a new validator
func NewValidator(registry *Registry) *Validator {
	if registry == nil {
		registry = DefaultRegistry
	}
	return &Validator{registry: registry}
}

// Validate validates a custom resource against its CRD schema
func (v *Validator) Validate(cr *CustomResource) *ValidationResult {
	result := &ValidationResult{
		Valid: true,
	}

	// Parse API version
	group, version := parseAPIVersion(cr.APIVersion)

	// Find CRD
	gvk := GroupVersionKind{
		Group:   group,
		Version: version,
		Kind:    cr.Kind,
	}

	crd, err := v.registry.GetByGVK(gvk)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Path:    "",
			Message: fmt.Sprintf("Unknown resource type: %s", gvk),
		})
		return result
	}

	// Validate metadata
	v.validateMetadata(cr, crd, result)

	// Validate spec against schema
	if crd.Spec.Schema != nil && cr.Spec != nil {
		v.validateAgainstSchema("spec", cr.Spec, crd.Spec.Schema, result)
	}

	return result
}

// ValidateSchema validates a value against a JSON schema
func (v *Validator) ValidateSchema(path string, value interface{}, schema *JSONSchemaProps) *ValidationResult {
	result := &ValidationResult{Valid: true}
	v.validateAgainstSchema(path, value, schema, result)
	return result
}

func (v *Validator) validateMetadata(cr *CustomResource, crd *CustomResourceDefinition, result *ValidationResult) {
	if cr.Metadata.Name == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Path:    "metadata.name",
			Message: "name is required",
		})
	}

	// Validate namespace for namespaced resources
	if crd.Spec.Scope == NamespacedScope && cr.Metadata.Namespace == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Path:    "metadata.namespace",
			Message: "namespace is required for namespaced resources",
		})
	}

	// Warn about namespace for cluster resources
	if crd.Spec.Scope == ClusterScope && cr.Metadata.Namespace != "" {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Path:    "metadata.namespace",
			Message: "namespace is ignored for cluster-scoped resources",
		})
	}
}

func (v *Validator) validateAgainstSchema(path string, value interface{}, schema *JSONSchemaProps, result *ValidationResult) {
	if schema == nil {
		return
	}

	// Handle null values
	if value == nil {
		if schema.Nullable {
			return
		}
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Path:    path,
			Message: "value cannot be null",
		})
		return
	}

	// Validate type
	if !v.validateType(path, value, schema, result) {
		return
	}

	// Type-specific validation
	switch schema.Type {
	case "string":
		v.validateString(path, value.(string), schema, result)
	case "integer", "number":
		v.validateNumber(path, value, schema, result)
	case "array":
		v.validateArray(path, value, schema, result)
	case "object":
		v.validateObject(path, value, schema, result)
	}

	// Enum validation
	if len(schema.Enum) > 0 {
		v.validateEnum(path, value, schema.Enum, result)
	}
}

func (v *Validator) validateType(path string, value interface{}, schema *JSONSchemaProps, result *ValidationResult) bool {
	if schema.Type == "" {
		return true
	}

	valid := false
	switch schema.Type {
	case "string":
		_, valid = value.(string)
	case "integer":
		switch value.(type) {
		case int, int32, int64, float64:
			valid = true
		}
	case "number":
		switch value.(type) {
		case int, int32, int64, float32, float64:
			valid = true
		}
	case "boolean":
		_, valid = value.(bool)
	case "array":
		_, valid = value.([]interface{})
	case "object":
		_, valid = value.(map[string]interface{})
	}

	if !valid {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Path:    path,
			Message: fmt.Sprintf("expected type %s, got %T", schema.Type, value),
		})
	}

	return valid
}

func (v *Validator) validateString(path string, value string, schema *JSONSchemaProps, result *ValidationResult) {
	// Length validation
	if schema.MinLength != nil && int64(len(value)) < *schema.MinLength {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Path:    path,
			Message: fmt.Sprintf("string length %d is less than minimum %d", len(value), *schema.MinLength),
		})
	}

	if schema.MaxLength != nil && int64(len(value)) > *schema.MaxLength {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Path:    path,
			Message: fmt.Sprintf("string length %d is greater than maximum %d", len(value), *schema.MaxLength),
		})
	}

	// Pattern validation
	if schema.Pattern != "" {
		matched, err := regexp.MatchString(schema.Pattern, value)
		if err != nil || !matched {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Path:    path,
				Message: fmt.Sprintf("string does not match pattern %s", schema.Pattern),
			})
		}
	}

	// Format validation
	if schema.Format != "" {
		v.validateFormat(path, value, schema.Format, result)
	}
}

func (v *Validator) validateNumber(path string, value interface{}, schema *JSONSchemaProps, result *ValidationResult) {
	var num float64
	switch n := value.(type) {
	case int:
		num = float64(n)
	case int64:
		num = float64(n)
	case float64:
		num = n
	}

	// Minimum validation
	if schema.Minimum != nil {
		if schema.ExclusiveMinimum && num <= *schema.Minimum {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Path:    path,
				Message: fmt.Sprintf("value %v must be greater than %v", num, *schema.Minimum),
			})
		} else if !schema.ExclusiveMinimum && num < *schema.Minimum {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Path:    path,
				Message: fmt.Sprintf("value %v must be at least %v", num, *schema.Minimum),
			})
		}
	}

	// Maximum validation
	if schema.Maximum != nil {
		if schema.ExclusiveMaximum && num >= *schema.Maximum {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Path:    path,
				Message: fmt.Sprintf("value %v must be less than %v", num, *schema.Maximum),
			})
		} else if !schema.ExclusiveMaximum && num > *schema.Maximum {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Path:    path,
				Message: fmt.Sprintf("value %v must be at most %v", num, *schema.Maximum),
			})
		}
	}
}

func (v *Validator) validateArray(path string, value interface{}, schema *JSONSchemaProps, result *ValidationResult) {
	arr, ok := value.([]interface{})
	if !ok {
		return
	}

	// Length validation
	if schema.MinItems != nil && int64(len(arr)) < *schema.MinItems {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Path:    path,
			Message: fmt.Sprintf("array length %d is less than minimum %d", len(arr), *schema.MinItems),
		})
	}

	if schema.MaxItems != nil && int64(len(arr)) > *schema.MaxItems {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Path:    path,
			Message: fmt.Sprintf("array length %d is greater than maximum %d", len(arr), *schema.MaxItems),
		})
	}

	// Item validation
	if schema.Items != nil && schema.Items.Schema != nil {
		for i, item := range arr {
			itemPath := fmt.Sprintf("%s[%d]", path, i)
			v.validateAgainstSchema(itemPath, item, schema.Items.Schema, result)
		}
	}

	// Unique items validation
	if schema.UniqueItems {
		seen := make(map[interface{}]bool)
		for i, item := range arr {
			if seen[item] {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Path:    fmt.Sprintf("%s[%d]", path, i),
					Message: "duplicate value in array with uniqueItems constraint",
				})
			}
			seen[item] = true
		}
	}
}

func (v *Validator) validateObject(path string, value interface{}, schema *JSONSchemaProps, result *ValidationResult) {
	obj, ok := value.(map[string]interface{})
	if !ok {
		return
	}

	// Property count validation
	if schema.MinProperties != nil && int64(len(obj)) < *schema.MinProperties {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Path:    path,
			Message: fmt.Sprintf("object has %d properties, minimum is %d", len(obj), *schema.MinProperties),
		})
	}

	if schema.MaxProperties != nil && int64(len(obj)) > *schema.MaxProperties {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Path:    path,
			Message: fmt.Sprintf("object has %d properties, maximum is %d", len(obj), *schema.MaxProperties),
		})
	}

	// Required properties validation
	for _, required := range schema.Required {
		if _, ok := obj[required]; !ok {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Path:    path + "." + required,
				Message: "required property is missing",
			})
		}
	}

	// Property validation
	for prop, propSchema := range schema.Properties {
		if val, ok := obj[prop]; ok {
			propPath := path + "." + prop
			v.validateAgainstSchema(propPath, val, &propSchema, result)
		}
	}

	// Additional properties validation
	if schema.AdditionalProperties != nil && !schema.AdditionalProperties.Allows && schema.AdditionalProperties.Schema == nil {
		for prop := range obj {
			if _, ok := schema.Properties[prop]; !ok {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Path:    path + "." + prop,
					Message: "additional property not allowed",
				})
			}
		}
	}
}

func (v *Validator) validateEnum(path string, value interface{}, enum []interface{}, result *ValidationResult) {
	for _, allowed := range enum {
		if value == allowed {
			return
		}
	}

	result.Valid = false
	result.Errors = append(result.Errors, ValidationError{
		Path:    path,
		Message: fmt.Sprintf("value must be one of: %v", enum),
	})
}

func (v *Validator) validateFormat(path string, value string, format string, result *ValidationResult) {
	var pattern string
	switch format {
	case "email":
		pattern = `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	case "uri", "url":
		pattern = `^https?://`
	case "date":
		pattern = `^\d{4}-\d{2}-\d{2}$`
	case "date-time":
		pattern = `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`
	default:
		return
	}

	matched, err := regexp.MatchString(pattern, value)
	if err != nil || !matched {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Path:    path,
			Message: fmt.Sprintf("string does not match format %s", format),
		})
	}
}

func parseAPIVersion(apiVersion string) (group, version string) {
	parts := strings.Split(apiVersion, "/")
	if len(parts) == 1 {
		return "", parts[0]
	}
	return parts[0], parts[1]
}

// ValidationResult contains the result of validation
type ValidationResult struct {
	// Valid indicates if validation passed
	Valid bool

	// Errors contains validation errors
	Errors []ValidationError

	// Warnings contains validation warnings
	Warnings []ValidationWarning
}

// ValidationError represents a validation error
type ValidationError struct {
	// Path is the path to the invalid value
	Path string

	// Message describes the error
	Message string
}

func (e ValidationError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("%s: %s", e.Path, e.Message)
	}
	return e.Message
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	// Path is the path to the value
	Path string

	// Message describes the warning
	Message string
}

func (w ValidationWarning) String() string {
	if w.Path != "" {
		return fmt.Sprintf("%s: %s", w.Path, w.Message)
	}
	return w.Message
}
