package errors

import (
	"fmt"
	"strings"
)

// ErrorType represents the type of error
type ErrorType string

const (
	ErrorTypeValidation    ErrorType = "Validation"
	ErrorTypeNotFound      ErrorType = "NotFound"
	ErrorTypeAlreadyExists ErrorType = "AlreadyExists"
	ErrorTypePermission    ErrorType = "Permission"
	ErrorTypeNetwork       ErrorType = "Network"
	ErrorTypeConfiguration ErrorType = "Configuration"
	ErrorTypePlugin        ErrorType = "Plugin"
	ErrorTypeState         ErrorType = "State"
	ErrorTypeAuth          ErrorType = "Authentication"
)

// PlatformError represents a detailed platform error
type PlatformError struct {
	Type          ErrorType
	Message       string
	File          string
	Line          int
	Field         string
	Got           string
	Expected      string
	Suggestion    string
	Documentation string
	Cause         error
}

// Error implements the error interface
func (e *PlatformError) Error() string {
	var sb strings.Builder

	// Error type and message
	sb.WriteString(fmt.Sprintf("❌ %s Error: %s\n", e.Type, e.Message))

	// File and line information
	if e.File != "" {
		sb.WriteString(fmt.Sprintf("\n📄 File: %s", e.File))
		if e.Line > 0 {
			sb.WriteString(fmt.Sprintf(":%d", e.Line))
		}
		sb.WriteString("\n")
	}

	// Field information
	if e.Field != "" {
		sb.WriteString(fmt.Sprintf("🔍 Field: %s\n", e.Field))
	}

	// What we got vs what we expected
	if e.Got != "" {
		sb.WriteString(fmt.Sprintf("\n  You provided:\n    %s\n", indent(e.Got, "    ")))
	}

	if e.Expected != "" {
		sb.WriteString(fmt.Sprintf("\n  Expected:\n    %s\n", indent(e.Expected, "    ")))
	}

	// Suggestion
	if e.Suggestion != "" {
		sb.WriteString(fmt.Sprintf("\n💡 Suggestion: %s\n", e.Suggestion))
	}

	// Documentation link
	if e.Documentation != "" {
		sb.WriteString(fmt.Sprintf("📚 Documentation: %s\n", e.Documentation))
	}

	// Underlying cause
	if e.Cause != nil {
		sb.WriteString(fmt.Sprintf("\n🐛 Underlying error: %v\n", e.Cause))
	}

	return sb.String()
}

// Unwrap returns the underlying error
func (e *PlatformError) Unwrap() error {
	return e.Cause
}

// Helper function to indent multi-line strings
func indent(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if i > 0 {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}

// NewValidationError creates a validation error
func NewValidationError(message string, opts ...ErrorOption) *PlatformError {
	err := &PlatformError{
		Type:    ErrorTypeValidation,
		Message: message,
	}
	for _, opt := range opts {
		opt(err)
	}
	return err
}

// NewNotFoundError creates a not found error
func NewNotFoundError(resource string, name string) *PlatformError {
	return &PlatformError{
		Type:       ErrorTypeNotFound,
		Message:    fmt.Sprintf("%s '%s' not found", resource, name),
		Suggestion: fmt.Sprintf("Check if the %s exists using 'pf get %ss' or verify the name is correct", resource, strings.ToLower(resource)),
	}
}

// NewAlreadyExistsError creates an already exists error
func NewAlreadyExistsError(resource string, name string) *PlatformError {
	return &PlatformError{
		Type:       ErrorTypeAlreadyExists,
		Message:    fmt.Sprintf("%s '%s' already exists", resource, name),
		Suggestion: fmt.Sprintf("Use a different name or delete the existing %s first with 'pf delete %s %s'", resource, strings.ToLower(resource), name),
	}
}

// NewPermissionError creates a permission error
func NewPermissionError(operation string, resource string) *PlatformError {
	return &PlatformError{
		Type:       ErrorTypePermission,
		Message:    fmt.Sprintf("Permission denied: cannot %s %s", operation, resource),
		Suggestion: "Check your authentication with 'pf auth whoami' or contact your administrator",
	}
}

// NewNetworkError creates a network error
func NewNetworkError(operation string, err error) *PlatformError {
	return &PlatformError{
		Type:       ErrorTypeNetwork,
		Message:    fmt.Sprintf("Network error during %s", operation),
		Cause:      err,
		Suggestion: "Check your network connection and try again. If the problem persists, check if the service is accessible.",
	}
}

// NewConfigurationError creates a configuration error
func NewConfigurationError(message string, opts ...ErrorOption) *PlatformError {
	err := &PlatformError{
		Type:    ErrorTypeConfiguration,
		Message: message,
	}
	for _, opt := range opts {
		opt(err)
	}
	return err
}

// NewPluginError creates a plugin error
func NewPluginError(pluginName string, operation string, err error) *PlatformError {
	return &PlatformError{
		Type:          ErrorTypePlugin,
		Message:       fmt.Sprintf("Plugin '%s' failed during %s", pluginName, operation),
		Cause:         err,
		Suggestion:    fmt.Sprintf("Check plugin configuration and logs. Try reinstalling the plugin with 'pf plugin install %s'", pluginName),
		Documentation: "https://docs.platformfoundry.io/plugins/" + pluginName,
	}
}

// NewStateError creates a state error
func NewStateError(operation string, err error) *PlatformError {
	return &PlatformError{
		Type:       ErrorTypeState,
		Message:    fmt.Sprintf("State error during %s", operation),
		Cause:      err,
		Suggestion: "The state backend may be corrupted or inaccessible. Try 'pf troubleshoot' to diagnose the issue.",
	}
}

// NewAuthError creates an authentication error
func NewAuthError(message string) *PlatformError {
	return &PlatformError{
		Type:          ErrorTypeAuth,
		Message:       message,
		Suggestion:    "Run 'pf auth login' to authenticate or check your credentials",
		Documentation: "https://docs.platformfoundry.io/authentication",
	}
}

// ErrorOption is a functional option for configuring errors
type ErrorOption func(*PlatformError)

// WithFile sets the file information
func WithFile(file string, line int) ErrorOption {
	return func(e *PlatformError) {
		e.File = file
		e.Line = line
	}
}

// WithField sets the field information
func WithField(field string) ErrorOption {
	return func(e *PlatformError) {
		e.Field = field
	}
}

// WithGot sets what was provided
func WithGot(got string) ErrorOption {
	return func(e *PlatformError) {
		e.Got = got
	}
}

// WithExpected sets what was expected
func WithExpected(expected string) ErrorOption {
	return func(e *PlatformError) {
		e.Expected = expected
	}
}

// WithSuggestion sets a suggestion
func WithSuggestion(suggestion string) ErrorOption {
	return func(e *PlatformError) {
		e.Suggestion = suggestion
	}
}

// WithDocumentation sets a documentation link
func WithDocumentation(doc string) ErrorOption {
	return func(e *PlatformError) {
		e.Documentation = doc
	}
}

// WithCause sets the underlying cause
func WithCause(cause error) ErrorOption {
	return func(e *PlatformError) {
		e.Cause = cause
	}
}
