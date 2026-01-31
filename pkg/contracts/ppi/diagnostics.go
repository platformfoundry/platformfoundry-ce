// Package ppi defines the Platform Provider Interface (PPI).
package ppi

import (
	"fmt"
	"strings"
)

// Diagnostics collects warnings and errors during operations
type Diagnostics struct {
	errors   []Diagnostic
	warnings []Diagnostic
}

// Diagnostic represents a single diagnostic message
type Diagnostic struct {
	Severity DiagnosticSeverity
	Summary  string
	Detail   string
	Path     []string
}

// DiagnosticSeverity indicates the severity of a diagnostic
type DiagnosticSeverity string

const (
	DiagnosticError   DiagnosticSeverity = "error"
	DiagnosticWarning DiagnosticSeverity = "warning"
)

// HasErrors returns true if there are any error diagnostics
func (d *Diagnostics) HasErrors() bool {
	return len(d.errors) > 0
}

// HasWarnings returns true if there are any warning diagnostics
func (d *Diagnostics) HasWarnings() bool {
	return len(d.warnings) > 0
}

// Errors returns all error diagnostics
func (d *Diagnostics) Errors() []Diagnostic {
	return d.errors
}

// Warnings returns all warning diagnostics
func (d *Diagnostics) Warnings() []Diagnostic {
	return d.warnings
}

// All returns all diagnostics
func (d *Diagnostics) All() []Diagnostic {
	all := make([]Diagnostic, 0, len(d.errors)+len(d.warnings))
	all = append(all, d.errors...)
	all = append(all, d.warnings...)
	return all
}

// AddError adds an error diagnostic
func (d *Diagnostics) AddError(summary string, detail string, args ...interface{}) {
	if len(args) > 0 {
		detail = fmt.Sprintf(detail, args...)
	}
	d.errors = append(d.errors, Diagnostic{
		Severity: DiagnosticError,
		Summary:  summary,
		Detail:   detail,
	})
}

// AddErrorAtPath adds an error diagnostic with a path
func (d *Diagnostics) AddErrorAtPath(path []string, summary string, detail string, args ...interface{}) {
	if len(args) > 0 {
		detail = fmt.Sprintf(detail, args...)
	}
	d.errors = append(d.errors, Diagnostic{
		Severity: DiagnosticError,
		Summary:  summary,
		Detail:   detail,
		Path:     path,
	})
}

// AddWarning adds a warning diagnostic
func (d *Diagnostics) AddWarning(summary string, detail string, args ...interface{}) {
	if len(args) > 0 {
		detail = fmt.Sprintf(detail, args...)
	}
	d.warnings = append(d.warnings, Diagnostic{
		Severity: DiagnosticWarning,
		Summary:  summary,
		Detail:   detail,
	})
}

// AddWarningAtPath adds a warning diagnostic with a path
func (d *Diagnostics) AddWarningAtPath(path []string, summary string, detail string, args ...interface{}) {
	if len(args) > 0 {
		detail = fmt.Sprintf(detail, args...)
	}
	d.warnings = append(d.warnings, Diagnostic{
		Severity: DiagnosticWarning,
		Summary:  summary,
		Detail:   detail,
		Path:     path,
	})
}

// Append adds all diagnostics from another Diagnostics
func (d *Diagnostics) Append(other *Diagnostics) {
	if other == nil {
		return
	}
	d.errors = append(d.errors, other.errors...)
	d.warnings = append(d.warnings, other.warnings...)
}

// Error returns a combined error message from all errors
func (d *Diagnostics) Error() string {
	if !d.HasErrors() {
		return ""
	}
	var msgs []string
	for _, diag := range d.errors {
		if len(diag.Path) > 0 {
			msgs = append(msgs, fmt.Sprintf("[%s] %s: %s", strings.Join(diag.Path, "."), diag.Summary, diag.Detail))
		} else {
			msgs = append(msgs, fmt.Sprintf("%s: %s", diag.Summary, diag.Detail))
		}
	}
	return strings.Join(msgs, "; ")
}

// String returns a string representation of all diagnostics
func (d *Diagnostics) String() string {
	var parts []string
	for _, diag := range d.All() {
		prefix := "ERROR"
		if diag.Severity == DiagnosticWarning {
			prefix = "WARNING"
		}
		if len(diag.Path) > 0 {
			parts = append(parts, fmt.Sprintf("[%s] %s: %s - %s", prefix, strings.Join(diag.Path, "."), diag.Summary, diag.Detail))
		} else {
			parts = append(parts, fmt.Sprintf("[%s] %s: %s", prefix, diag.Summary, diag.Detail))
		}
	}
	return strings.Join(parts, "\n")
}
