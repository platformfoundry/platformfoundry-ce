// Package sdk provides the Plugin SDK for Platform Foundry.
package sdk

import (
	"github.com/platformfoundry/platformfoundry-ce/pkg/contracts/ppi"
)

// DiagnosticsBuilder helps construct diagnostics with a fluent API
type DiagnosticsBuilder struct {
	diags *ppi.Diagnostics
}

// NewDiagnosticsBuilder creates a new diagnostics builder
func NewDiagnosticsBuilder() *DiagnosticsBuilder {
	return &DiagnosticsBuilder{
		diags: &ppi.Diagnostics{},
	}
}

// Error adds an error diagnostic
func (b *DiagnosticsBuilder) Error(summary, detail string, args ...interface{}) *DiagnosticsBuilder {
	b.diags.AddError(summary, detail, args...)
	return b
}

// ErrorAtPath adds an error diagnostic with a path
func (b *DiagnosticsBuilder) ErrorAtPath(path []string, summary, detail string, args ...interface{}) *DiagnosticsBuilder {
	b.diags.AddErrorAtPath(path, summary, detail, args...)
	return b
}

// Warning adds a warning diagnostic
func (b *DiagnosticsBuilder) Warning(summary, detail string, args ...interface{}) *DiagnosticsBuilder {
	b.diags.AddWarning(summary, detail, args...)
	return b
}

// WarningAtPath adds a warning diagnostic with a path
func (b *DiagnosticsBuilder) WarningAtPath(path []string, summary, detail string, args ...interface{}) *DiagnosticsBuilder {
	b.diags.AddWarningAtPath(path, summary, detail, args...)
	return b
}

// Append appends diagnostics from another builder or Diagnostics
func (b *DiagnosticsBuilder) Append(other *ppi.Diagnostics) *DiagnosticsBuilder {
	b.diags.Append(other)
	return b
}

// Build returns the constructed diagnostics
func (b *DiagnosticsBuilder) Build() *ppi.Diagnostics {
	return b.diags
}

// HasErrors returns whether there are any errors
func (b *DiagnosticsBuilder) HasErrors() bool {
	return b.diags.HasErrors()
}

// Convenience functions for creating common diagnostics

// ConfigError creates a configuration error diagnostic
func ConfigError(attr string, detail string, args ...interface{}) *ppi.Diagnostics {
	return NewDiagnosticsBuilder().
		ErrorAtPath([]string{attr}, "Invalid configuration", detail, args...).
		Build()
}

// MissingRequiredError creates a missing required attribute error
func MissingRequiredError(attr string) *ppi.Diagnostics {
	return NewDiagnosticsBuilder().
		ErrorAtPath([]string{attr}, "Missing required attribute", "The attribute %q is required", attr).
		Build()
}

// InvalidValueError creates an invalid value error
func InvalidValueError(attr string, value interface{}, reason string) *ppi.Diagnostics {
	return NewDiagnosticsBuilder().
		ErrorAtPath([]string{attr}, "Invalid value", "Value %v is invalid: %s", value, reason).
		Build()
}

// ResourceNotFoundError creates a resource not found error
func ResourceNotFoundError(typeName, id string) *ppi.Diagnostics {
	return NewDiagnosticsBuilder().
		Error("Resource not found", "Resource %s with ID %q was not found", typeName, id).
		Build()
}

// APIError creates an API error diagnostic
func APIError(operation string, err error) *ppi.Diagnostics {
	return NewDiagnosticsBuilder().
		Error("API error", "Error during %s: %v", operation, err).
		Build()
}

// TimeoutError creates a timeout error diagnostic
func TimeoutError(operation string) *ppi.Diagnostics {
	return NewDiagnosticsBuilder().
		Error("Operation timed out", "The %s operation timed out", operation).
		Build()
}

// DeprecationWarning creates a deprecation warning
func DeprecationWarning(attr, alternative string) *ppi.Diagnostics {
	return NewDiagnosticsBuilder().
		WarningAtPath([]string{attr}, "Deprecated attribute", "The attribute %q is deprecated. Use %q instead.", attr, alternative).
		Build()
}

// ValidateDiagnostics validates values and returns diagnostics
type ValidateDiagnostics struct {
	diags *ppi.Diagnostics
}

// NewValidateDiagnostics creates a new validation diagnostics helper
func NewValidateDiagnostics() *ValidateDiagnostics {
	return &ValidateDiagnostics{
		diags: &ppi.Diagnostics{},
	}
}

// RequireString validates that a value is a non-empty string
func (v *ValidateDiagnostics) RequireString(path string, value interface{}) string {
	if value == nil {
		v.diags.AddErrorAtPath([]string{path}, "Missing required value", "A string value is required for %q", path)
		return ""
	}
	s, ok := value.(string)
	if !ok {
		v.diags.AddErrorAtPath([]string{path}, "Invalid type", "Expected string, got %T", value)
		return ""
	}
	if s == "" {
		v.diags.AddErrorAtPath([]string{path}, "Empty value", "A non-empty string is required for %q", path)
		return ""
	}
	return s
}

// RequireInt validates that a value is an integer
func (v *ValidateDiagnostics) RequireInt(path string, value interface{}) int {
	if value == nil {
		v.diags.AddErrorAtPath([]string{path}, "Missing required value", "An integer value is required for %q", path)
		return 0
	}
	switch n := value.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		v.diags.AddErrorAtPath([]string{path}, "Invalid type", "Expected integer, got %T", value)
		return 0
	}
}

// RequireBool validates that a value is a boolean
func (v *ValidateDiagnostics) RequireBool(path string, value interface{}) bool {
	if value == nil {
		v.diags.AddErrorAtPath([]string{path}, "Missing required value", "A boolean value is required for %q", path)
		return false
	}
	b, ok := value.(bool)
	if !ok {
		v.diags.AddErrorAtPath([]string{path}, "Invalid type", "Expected boolean, got %T", value)
		return false
	}
	return b
}

// Build returns the accumulated diagnostics
func (v *ValidateDiagnostics) Build() *ppi.Diagnostics {
	return v.diags
}

// HasErrors returns whether validation had errors
func (v *ValidateDiagnostics) HasErrors() bool {
	return v.diags.HasErrors()
}
