// Package drift provides drift detection between desired and actual state.
package drift

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"
)

// DriftType represents the type of drift detected
type DriftType string

const (
	DriftAdded    DriftType = "added"    // Resource exists but not in desired state
	DriftDeleted  DriftType = "deleted"  // Resource in desired state but doesn't exist
	DriftModified DriftType = "modified" // Resource differs from desired state
	DriftUnknown  DriftType = "unknown"  // Cannot determine drift
)

// DriftSeverity indicates how critical the drift is
type DriftSeverity string

const (
	SeverityCritical DriftSeverity = "critical"
	SeverityHigh     DriftSeverity = "high"
	SeverityMedium   DriftSeverity = "medium"
	SeverityLow      DriftSeverity = "low"
	SeverityInfo     DriftSeverity = "info"
)

// Drift represents a detected drift
type Drift struct {
	ID           string                 `json:"id"`
	ResourceID   string                 `json:"resource_id"`
	ResourceType string                 `json:"resource_type"`
	ResourceName string                 `json:"resource_name"`
	Type         DriftType              `json:"type"`
	Severity     DriftSeverity          `json:"severity"`
	Description  string                 `json:"description"`
	DesiredValue interface{}            `json:"desired_value,omitempty"`
	ActualValue  interface{}            `json:"actual_value,omitempty"`
	Path         string                 `json:"path,omitempty"` // JSON path to drifted field
	DetectedAt   time.Time              `json:"detected_at"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Report contains all detected drifts
type Report struct {
	ID           string        `json:"id"`
	Drifts       []Drift       `json:"drifts"`
	Summary      Summary       `json:"summary"`
	StartedAt    time.Time     `json:"started_at"`
	CompletedAt  time.Time     `json:"completed_at"`
	Duration     time.Duration `json:"duration"`
	ResourcesChecked int       `json:"resources_checked"`
}

// Summary provides drift statistics
type Summary struct {
	Total      int `json:"total"`
	Critical   int `json:"critical"`
	High       int `json:"high"`
	Medium     int `json:"medium"`
	Low        int `json:"low"`
	Info       int `json:"info"`
	ByType     map[DriftType]int `json:"by_type"`
}

// Resource represents a resource to check for drift
type Resource struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Name     string                 `json:"name"`
	Desired  map[string]interface{} `json:"desired"`
	Actual   map[string]interface{} `json:"actual,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// StateProvider provides actual state for resources
type StateProvider interface {
	GetActualState(ctx context.Context, resourceType, resourceID string) (map[string]interface{}, error)
}

// DetectorConfig configures the drift detector
type DetectorConfig struct {
	IgnorePaths     []string      // Paths to ignore during comparison
	SeverityMapping map[string]DriftSeverity // Path -> severity mapping
	Concurrency     int           // Number of concurrent checks
}

// Detector detects drift between desired and actual state
type Detector struct {
	config        DetectorConfig
	stateProvider StateProvider
}

// NewDetector creates a new drift detector
func NewDetector(config DetectorConfig, provider StateProvider) *Detector {
	if config.Concurrency <= 0 {
		config.Concurrency = 5
	}

	return &Detector{
		config:        config,
		stateProvider: provider,
	}
}

// Detect checks for drift in the given resources
func (d *Detector) Detect(ctx context.Context, resources []Resource) (*Report, error) {
	report := &Report{
		ID:        generateReportID(),
		StartedAt: time.Now(),
		Drifts:    make([]Drift, 0),
		Summary: Summary{
			ByType: make(map[DriftType]int),
		},
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, d.config.Concurrency)

	for _, resource := range resources {
		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore

		go func(res Resource) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			drifts, err := d.checkResource(ctx, res)
			if err != nil {
				// Log error but continue
				mu.Lock()
				report.Drifts = append(report.Drifts, Drift{
					ID:           generateDriftID(),
					ResourceID:   res.ID,
					ResourceType: res.Type,
					ResourceName: res.Name,
					Type:         DriftUnknown,
					Severity:     SeverityInfo,
					Description:  fmt.Sprintf("Failed to check drift: %v", err),
					DetectedAt:   time.Now(),
				})
				mu.Unlock()
				return
			}

			if len(drifts) > 0 {
				mu.Lock()
				report.Drifts = append(report.Drifts, drifts...)
				mu.Unlock()
			}
		}(resource)
	}

	wg.Wait()

	report.CompletedAt = time.Now()
	report.Duration = report.CompletedAt.Sub(report.StartedAt)
	report.ResourcesChecked = len(resources)

	// Calculate summary
	for _, drift := range report.Drifts {
		report.Summary.Total++
		report.Summary.ByType[drift.Type]++

		switch drift.Severity {
		case SeverityCritical:
			report.Summary.Critical++
		case SeverityHigh:
			report.Summary.High++
		case SeverityMedium:
			report.Summary.Medium++
		case SeverityLow:
			report.Summary.Low++
		case SeverityInfo:
			report.Summary.Info++
		}
	}

	return report, nil
}

func (d *Detector) checkResource(ctx context.Context, resource Resource) ([]Drift, error) {
	// Get actual state if not provided
	actual := resource.Actual
	if actual == nil && d.stateProvider != nil {
		var err error
		actual, err = d.stateProvider.GetActualState(ctx, resource.Type, resource.ID)
		if err != nil {
			return nil, err
		}
	}

	// Resource doesn't exist
	if actual == nil {
		return []Drift{{
			ID:           generateDriftID(),
			ResourceID:   resource.ID,
			ResourceType: resource.Type,
			ResourceName: resource.Name,
			Type:         DriftDeleted,
			Severity:     SeverityHigh,
			Description:  "Resource exists in desired state but not found in actual state",
			DesiredValue: resource.Desired,
			DetectedAt:   time.Now(),
		}}, nil
	}

	// Compare desired vs actual
	return d.compare(resource, resource.Desired, actual, ""), nil
}

func (d *Detector) compare(resource Resource, desired, actual map[string]interface{}, path string) []Drift {
	var drifts []Drift

	// Check all desired fields
	for key, desiredVal := range desired {
		currentPath := joinPath(path, key)

		// Skip ignored paths
		if d.isIgnored(currentPath) {
			continue
		}

		actualVal, exists := actual[key]

		if !exists {
			drifts = append(drifts, Drift{
				ID:           generateDriftID(),
				ResourceID:   resource.ID,
				ResourceType: resource.Type,
				ResourceName: resource.Name,
				Type:         DriftModified,
				Severity:     d.getSeverity(currentPath),
				Description:  fmt.Sprintf("Field '%s' missing in actual state", currentPath),
				Path:         currentPath,
				DesiredValue: desiredVal,
				ActualValue:  nil,
				DetectedAt:   time.Now(),
			})
			continue
		}

		// Recursive comparison for nested maps
		desiredMap, desiredIsMap := desiredVal.(map[string]interface{})
		actualMap, actualIsMap := actualVal.(map[string]interface{})

		if desiredIsMap && actualIsMap {
			nested := d.compare(resource, desiredMap, actualMap, currentPath)
			drifts = append(drifts, nested...)
			continue
		}

		// Direct comparison
		if !reflect.DeepEqual(desiredVal, actualVal) {
			drifts = append(drifts, Drift{
				ID:           generateDriftID(),
				ResourceID:   resource.ID,
				ResourceType: resource.Type,
				ResourceName: resource.Name,
				Type:         DriftModified,
				Severity:     d.getSeverity(currentPath),
				Description:  fmt.Sprintf("Field '%s' differs from desired state", currentPath),
				Path:         currentPath,
				DesiredValue: desiredVal,
				ActualValue:  actualVal,
				DetectedAt:   time.Now(),
			})
		}
	}

	// Check for fields in actual but not in desired (added)
	for key, actualVal := range actual {
		currentPath := joinPath(path, key)

		if d.isIgnored(currentPath) {
			continue
		}

		if _, exists := desired[key]; !exists {
			drifts = append(drifts, Drift{
				ID:           generateDriftID(),
				ResourceID:   resource.ID,
				ResourceType: resource.Type,
				ResourceName: resource.Name,
				Type:         DriftAdded,
				Severity:     SeverityLow,
				Description:  fmt.Sprintf("Field '%s' exists in actual but not in desired state", currentPath),
				Path:         currentPath,
				DesiredValue: nil,
				ActualValue:  actualVal,
				DetectedAt:   time.Now(),
			})
		}
	}

	return drifts
}

func (d *Detector) isIgnored(path string) bool {
	for _, ignorePath := range d.config.IgnorePaths {
		if strings.HasPrefix(path, ignorePath) || matchWildcard(ignorePath, path) {
			return true
		}
	}
	return false
}

func (d *Detector) getSeverity(path string) DriftSeverity {
	if severity, ok := d.config.SeverityMapping[path]; ok {
		return severity
	}

	// Default severity based on common patterns
	if strings.Contains(path, "password") || strings.Contains(path, "secret") {
		return SeverityCritical
	}
	if strings.Contains(path, "replicas") || strings.Contains(path, "resources") {
		return SeverityHigh
	}
	if strings.Contains(path, "labels") || strings.Contains(path, "annotations") {
		return SeverityLow
	}

	return SeverityMedium
}

func joinPath(base, key string) string {
	if base == "" {
		return key
	}
	return base + "." + key
}

func matchWildcard(pattern, path string) bool {
	// Simple wildcard matching
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(path, prefix)
	}
	return pattern == path
}

func generateReportID() string {
	return "drift-report-" + time.Now().Format("20060102-150405")
}

func generateDriftID() string {
	data := fmt.Sprintf("%d", time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return "drift-" + hex.EncodeToString(hash[:8])
}

// FormatReport formats the drift report for console output
func FormatReport(report *Report) string {
	var sb strings.Builder

	sb.WriteString("Drift Detection Report\n")
	sb.WriteString("======================\n\n")

	sb.WriteString(fmt.Sprintf("Report ID: %s\n", report.ID))
	sb.WriteString(fmt.Sprintf("Resources Checked: %d\n", report.ResourcesChecked))
	sb.WriteString(fmt.Sprintf("Duration: %s\n\n", report.Duration.Round(time.Millisecond)))

	if len(report.Drifts) == 0 {
		sb.WriteString("No drift detected. All resources match desired state.\n")
		return sb.String()
	}

	sb.WriteString("Drifts Detected:\n")
	sb.WriteString("----------------\n")

	for _, drift := range report.Drifts {
		icon := getSeverityIcon(drift.Severity)
		sb.WriteString(fmt.Sprintf("\n%s [%s] %s/%s\n", icon, drift.Severity, drift.ResourceType, drift.ResourceName))
		sb.WriteString(fmt.Sprintf("   Type: %s\n", drift.Type))
		sb.WriteString(fmt.Sprintf("   Description: %s\n", drift.Description))
		if drift.Path != "" {
			sb.WriteString(fmt.Sprintf("   Path: %s\n", drift.Path))
		}
		if drift.DesiredValue != nil {
			sb.WriteString(fmt.Sprintf("   Desired: %v\n", formatValue(drift.DesiredValue)))
		}
		if drift.ActualValue != nil {
			sb.WriteString(fmt.Sprintf("   Actual: %v\n", formatValue(drift.ActualValue)))
		}
	}

	sb.WriteString("\nSummary:\n")
	sb.WriteString(fmt.Sprintf("  Total Drifts: %d\n", report.Summary.Total))
	if report.Summary.Critical > 0 {
		sb.WriteString(fmt.Sprintf("  Critical: %d\n", report.Summary.Critical))
	}
	if report.Summary.High > 0 {
		sb.WriteString(fmt.Sprintf("  High: %d\n", report.Summary.High))
	}
	if report.Summary.Medium > 0 {
		sb.WriteString(fmt.Sprintf("  Medium: %d\n", report.Summary.Medium))
	}
	if report.Summary.Low > 0 {
		sb.WriteString(fmt.Sprintf("  Low: %d\n", report.Summary.Low))
	}

	return sb.String()
}

func getSeverityIcon(severity DriftSeverity) string {
	switch severity {
	case SeverityCritical:
		return "[CRIT]"
	case SeverityHigh:
		return "[HIGH]"
	case SeverityMedium:
		return "[MED]"
	case SeverityLow:
		return "[LOW]"
	case SeverityInfo:
		return "[INFO]"
	default:
		return "[?]"
	}
}

func formatValue(v interface{}) string {
	if v == nil {
		return "<nil>"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	s := string(b)
	if len(s) > 50 {
		return s[:47] + "..."
	}
	return s
}
