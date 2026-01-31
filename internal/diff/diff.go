// Package diff provides platform comparison functionality across environments.
package diff

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// DiffType represents the type of difference
type DiffType string

const (
	DiffTypeAdded    DiffType = "added"
	DiffTypeRemoved  DiffType = "removed"
	DiffTypeModified DiffType = "modified"
)

// Severity represents the severity of a difference
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityWarning  Severity = "warning"
	SeverityInfo     Severity = "info"
)

// ResourceDiff represents a difference in a single resource
type ResourceDiff struct {
	Resource    string      `json:"resource"`
	Type        DiffType    `json:"type"`
	Path        string      `json:"path,omitempty"`
	SourceValue interface{} `json:"source_value,omitempty"`
	TargetValue interface{} `json:"target_value,omitempty"`
	Severity    Severity    `json:"severity"`
}

// PlatformDiff represents the complete diff between two platform states
type PlatformDiff struct {
	Source       string         `json:"source"`
	Target       string         `json:"target"`
	Differences  []ResourceDiff `json:"differences"`
	OnlyInSource []string       `json:"only_in_source"`
	OnlyInTarget []string       `json:"only_in_target"`
	Summary      DiffSummary    `json:"summary"`
}

// DiffSummary provides summary statistics
type DiffSummary struct {
	TotalDifferences int `json:"total_differences"`
	Added            int `json:"added"`
	Removed          int `json:"removed"`
	Modified         int `json:"modified"`
	Critical         int `json:"critical"`
	Warning          int `json:"warning"`
	Info             int `json:"info"`
}

// PlatformState represents the state of a platform
type PlatformState struct {
	Name      string                            `json:"name"`
	Resources map[string]map[string]interface{} `json:"resources"`
}

// Differ performs platform state comparisons
type Differ struct {
	criticalPaths []string // Paths that are critical if changed
	ignoredPaths  []string // Paths to ignore in comparison
}

// NewDiffer creates a new Differ
func NewDiffer() *Differ {
	return &Differ{
		criticalPaths: []string{
			"replicas",
			"resources.limits",
			"resources.requests",
			"image",
			"version",
			"secrets",
			"tls",
			"authentication",
		},
		ignoredPaths: []string{
			"metadata.creationTimestamp",
			"metadata.uid",
			"metadata.resourceVersion",
			"status",
		},
	}
}

// WithCriticalPaths sets paths that should be marked critical if changed
func (d *Differ) WithCriticalPaths(paths []string) *Differ {
	d.criticalPaths = paths
	return d
}

// WithIgnoredPaths sets paths to ignore during comparison
func (d *Differ) WithIgnoredPaths(paths []string) *Differ {
	d.ignoredPaths = paths
	return d
}

// Compare compares two platform states and returns the differences
func (d *Differ) Compare(source, target *PlatformState) *PlatformDiff {
	diff := &PlatformDiff{
		Source:       source.Name,
		Target:       target.Name,
		Differences:  make([]ResourceDiff, 0),
		OnlyInSource: make([]string, 0),
		OnlyInTarget: make([]string, 0),
	}

	// Find resources only in source
	for name := range source.Resources {
		if _, exists := target.Resources[name]; !exists {
			diff.OnlyInSource = append(diff.OnlyInSource, name)
			diff.Differences = append(diff.Differences, ResourceDiff{
				Resource:    name,
				Type:        DiffTypeRemoved,
				SourceValue: source.Resources[name],
				Severity:    SeverityWarning,
			})
		}
	}

	// Find resources only in target
	for name := range target.Resources {
		if _, exists := source.Resources[name]; !exists {
			diff.OnlyInTarget = append(diff.OnlyInTarget, name)
			diff.Differences = append(diff.Differences, ResourceDiff{
				Resource:    name,
				Type:        DiffTypeAdded,
				TargetValue: target.Resources[name],
				Severity:    SeverityInfo,
			})
		}
	}

	// Compare resources that exist in both
	for name, sourceRes := range source.Resources {
		if targetRes, exists := target.Resources[name]; exists {
			resourceDiffs := d.compareResource(name, sourceRes, targetRes, "")
			diff.Differences = append(diff.Differences, resourceDiffs...)
		}
	}

	// Calculate summary
	diff.Summary = d.calculateSummary(diff)

	// Sort differences by severity
	sort.Slice(diff.Differences, func(i, j int) bool {
		return severityOrder(diff.Differences[i].Severity) < severityOrder(diff.Differences[j].Severity)
	})

	sort.Strings(diff.OnlyInSource)
	sort.Strings(diff.OnlyInTarget)

	return diff
}

// compareResource recursively compares two resource maps
func (d *Differ) compareResource(name string, source, target map[string]interface{}, path string) []ResourceDiff {
	diffs := make([]ResourceDiff, 0)

	// Get all keys from both maps
	keys := make(map[string]bool)
	for k := range source {
		keys[k] = true
	}
	for k := range target {
		keys[k] = true
	}

	for key := range keys {
		currentPath := key
		if path != "" {
			currentPath = path + "." + key
		}

		// Skip ignored paths
		if d.shouldIgnore(currentPath) {
			continue
		}

		sourceVal, sourceExists := source[key]
		targetVal, targetExists := target[key]

		if !sourceExists {
			// Key only in target
			diffs = append(diffs, ResourceDiff{
				Resource:    name,
				Type:        DiffTypeAdded,
				Path:        currentPath,
				TargetValue: targetVal,
				Severity:    d.getSeverity(currentPath),
			})
		} else if !targetExists {
			// Key only in source
			diffs = append(diffs, ResourceDiff{
				Resource:    name,
				Type:        DiffTypeRemoved,
				Path:        currentPath,
				SourceValue: sourceVal,
				Severity:    d.getSeverity(currentPath),
			})
		} else if !d.equalValues(sourceVal, targetVal) {
			// Values differ
			// If both are maps, recurse
			sourceMap, sourceIsMap := sourceVal.(map[string]interface{})
			targetMap, targetIsMap := targetVal.(map[string]interface{})

			if sourceIsMap && targetIsMap {
				subDiffs := d.compareResource(name, sourceMap, targetMap, currentPath)
				diffs = append(diffs, subDiffs...)
			} else {
				diffs = append(diffs, ResourceDiff{
					Resource:    name,
					Type:        DiffTypeModified,
					Path:        currentPath,
					SourceValue: sourceVal,
					TargetValue: targetVal,
					Severity:    d.getSeverity(currentPath),
				})
			}
		}
	}

	return diffs
}

// equalValues checks if two values are equal
func (d *Differ) equalValues(a, b interface{}) bool {
	// Handle nil cases
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Use JSON marshaling for deep comparison
	aJSON, err1 := json.Marshal(a)
	bJSON, err2 := json.Marshal(b)
	if err1 != nil || err2 != nil {
		return reflect.DeepEqual(a, b)
	}
	return string(aJSON) == string(bJSON)
}

// shouldIgnore checks if a path should be ignored
func (d *Differ) shouldIgnore(path string) bool {
	for _, ignored := range d.ignoredPaths {
		if strings.HasPrefix(path, ignored) || path == ignored {
			return true
		}
	}
	return false
}

// getSeverity determines the severity of a change based on the path
func (d *Differ) getSeverity(path string) Severity {
	for _, critical := range d.criticalPaths {
		if strings.Contains(path, critical) {
			return SeverityWarning
		}
	}
	return SeverityInfo
}

// calculateSummary calculates summary statistics for the diff
func (d *Differ) calculateSummary(diff *PlatformDiff) DiffSummary {
	summary := DiffSummary{
		TotalDifferences: len(diff.Differences),
	}

	for _, d := range diff.Differences {
		switch d.Type {
		case DiffTypeAdded:
			summary.Added++
		case DiffTypeRemoved:
			summary.Removed++
		case DiffTypeModified:
			summary.Modified++
		}

		switch d.Severity {
		case SeverityCritical:
			summary.Critical++
		case SeverityWarning:
			summary.Warning++
		case SeverityInfo:
			summary.Info++
		}
	}

	return summary
}

// severityOrder returns the sort order for severity
func severityOrder(s Severity) int {
	switch s {
	case SeverityCritical:
		return 0
	case SeverityWarning:
		return 1
	case SeverityInfo:
		return 2
	default:
		return 3
	}
}

// FormatDiff formats the diff for display
func FormatDiff(diff *PlatformDiff) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("\nPlatform Diff: %s -> %s\n", diff.Source, diff.Target))
	sb.WriteString(strings.Repeat("=", 65) + "\n\n")

	sb.WriteString(fmt.Sprintf("Summary: %d differences, %d only in %s, %d only in %s\n\n",
		diff.Summary.TotalDifferences,
		len(diff.OnlyInSource), diff.Source,
		len(diff.OnlyInTarget), diff.Target))

	if len(diff.Differences) > 0 {
		// Group by resource
		byResource := make(map[string][]ResourceDiff)
		for _, d := range diff.Differences {
			if d.Path != "" {
				byResource[d.Resource] = append(byResource[d.Resource], d)
			}
		}

		sb.WriteString("+------------------------+----------------------+----------------------+----------+\n")
		sb.WriteString("| Resource               | " + diff.Source + strings.Repeat(" ", 20-len(diff.Source)) + " | " + diff.Target + strings.Repeat(" ", 20-len(diff.Target)) + " | Severity |\n")
		sb.WriteString("+------------------------+----------------------+----------------------+----------+\n")

		for _, d := range diff.Differences {
			if d.Path == "" {
				continue // Skip top-level adds/removes
			}
			resourcePath := d.Resource
			if d.Path != "" {
				resourcePath = d.Resource + "." + d.Path
			}
			if len(resourcePath) > 22 {
				resourcePath = resourcePath[:19] + "..."
			}

			sourceStr := formatValue(d.SourceValue)
			targetStr := formatValue(d.TargetValue)

			if len(sourceStr) > 20 {
				sourceStr = sourceStr[:17] + "..."
			}
			if len(targetStr) > 20 {
				targetStr = targetStr[:17] + "..."
			}

			severityStr := string(d.Severity)
			sb.WriteString(fmt.Sprintf("| %-22s | %-20s | %-20s | %-8s |\n",
				resourcePath, sourceStr, targetStr, severityStr))
		}
		sb.WriteString("+------------------------+----------------------+----------------------+----------+\n")
	}

	if len(diff.OnlyInSource) > 0 {
		sb.WriteString(fmt.Sprintf("\nOnly in %s:\n", diff.Source))
		for _, r := range diff.OnlyInSource {
			sb.WriteString(fmt.Sprintf("  - %s\n", r))
		}
	}

	if len(diff.OnlyInTarget) > 0 {
		sb.WriteString(fmt.Sprintf("\nOnly in %s:\n", diff.Target))
		for _, r := range diff.OnlyInTarget {
			sb.WriteString(fmt.Sprintf("  + %s\n", r))
		}
	}

	return sb.String()
}

// formatValue formats a value for display
func formatValue(v interface{}) string {
	if v == nil {
		return "(none)"
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return fmt.Sprintf("%.0f", val)
	case bool:
		return fmt.Sprintf("%t", val)
	case map[string]interface{}:
		return "(object)"
	case []interface{}:
		return fmt.Sprintf("[%d items]", len(val))
	default:
		return fmt.Sprintf("%v", val)
	}
}

// CompareFromJSON compares two JSON-encoded platform states
func CompareFromJSON(sourceJSON, targetJSON []byte, sourceName, targetName string) (*PlatformDiff, error) {
	var sourceData, targetData map[string]interface{}

	if err := json.Unmarshal(sourceJSON, &sourceData); err != nil {
		return nil, fmt.Errorf("failed to parse source: %w", err)
	}

	if err := json.Unmarshal(targetJSON, &targetData); err != nil {
		return nil, fmt.Errorf("failed to parse target: %w", err)
	}

	source := &PlatformState{
		Name:      sourceName,
		Resources: make(map[string]map[string]interface{}),
	}
	target := &PlatformState{
		Name:      targetName,
		Resources: make(map[string]map[string]interface{}),
	}

	// Convert top-level keys to resources
	for k, v := range sourceData {
		if m, ok := v.(map[string]interface{}); ok {
			source.Resources[k] = m
		}
	}
	for k, v := range targetData {
		if m, ok := v.(map[string]interface{}); ok {
			target.Resources[k] = m
		}
	}

	differ := NewDiffer()
	return differ.Compare(source, target), nil
}
