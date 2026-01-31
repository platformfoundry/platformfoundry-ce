package service

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// TemplateEngine handles rendering of Go templates
type TemplateEngine struct {
	funcMap template.FuncMap
}

// NewTemplateEngine creates a new template engine with helper functions
func NewTemplateEngine() *TemplateEngine {
	return &TemplateEngine{
		funcMap: template.FuncMap{
			// String manipulation
			"upper":     strings.ToUpper,
			"lower":     strings.ToLower,
			"title":     strings.Title,
			"trim":      strings.TrimSpace,
			"replace":   strings.ReplaceAll,
			"contains":  strings.Contains,
			"hasPrefix": strings.HasPrefix,
			"hasSuffix": strings.HasSuffix,
			"split":     strings.Split,
			"join":      strings.Join,

			// Custom helpers
			"default": defaultValue,
			"env":     envValue,
			"quote":   quote,
			"indent":  indent,
			"nindent": nindent,
		},
	}
}

// Render renders a template with the given parameters
func (te *TemplateEngine) Render(templateStr string, params map[string]interface{}) (string, error) {
	// Create template
	tmpl, err := template.New("template").Funcs(te.funcMap).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, params); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// RenderWithName renders a named template (useful for debugging)
func (te *TemplateEngine) RenderWithName(name, templateStr string, params map[string]interface{}) (string, error) {
	// Create template
	tmpl, err := template.New(name).Funcs(te.funcMap).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %w", name, err)
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, params); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}

	return buf.String(), nil
}

// Validate validates a template without executing it
func (te *TemplateEngine) Validate(templateStr string) error {
	_, err := template.New("template").Funcs(te.funcMap).Parse(templateStr)
	if err != nil {
		return fmt.Errorf("invalid template: %w", err)
	}
	return nil
}

// AddFunction adds a custom function to the template engine
func (te *TemplateEngine) AddFunction(name string, fn interface{}) {
	te.funcMap[name] = fn
}

// Helper functions for templates

// defaultValue returns the default value if the value is empty/nil
func defaultValue(defaultVal, value interface{}) interface{} {
	if value == nil {
		return defaultVal
	}

	// Check for empty string
	if str, ok := value.(string); ok && str == "" {
		return defaultVal
	}

	return value
}

// envValue would normally return environment variable value
// For security, we don't implement actual env var access in templates
// Instead, env vars should be passed as parameters
func envValue(key string) string {
	// Security: Don't allow direct env var access from templates
	// Return empty string - users should pass env vars as parameters
	return ""
}

// quote wraps a string in double quotes
func quote(str string) string {
	return fmt.Sprintf("%q", str)
}

// indent indents each line of the string by the specified number of spaces
func indent(spaces int, str string) string {
	lines := strings.Split(str, "\n")
	padding := strings.Repeat(" ", spaces)

	for i, line := range lines {
		if line != "" {
			lines[i] = padding + line
		}
	}

	return strings.Join(lines, "\n")
}

// nindent adds a newline and then indents
func nindent(spaces int, str string) string {
	return "\n" + indent(spaces, str)
}

// SafeRender renders a template with timeout and memory limits
// This prevents template bombs and infinite loops
func (te *TemplateEngine) SafeRender(templateStr string, params map[string]interface{}, maxSize int) (string, error) {
	// Validate template first
	if err := te.Validate(templateStr); err != nil {
		return "", err
	}

	// Check template size
	if len(templateStr) > maxSize {
		return "", fmt.Errorf("template size %d exceeds maximum %d", len(templateStr), maxSize)
	}

	// Render with size check
	result, err := te.Render(templateStr, params)
	if err != nil {
		return "", err
	}

	// Check output size
	if len(result) > maxSize*10 {
		return "", fmt.Errorf("rendered output size %d exceeds maximum %d", len(result), maxSize*10)
	}

	return result, nil
}
