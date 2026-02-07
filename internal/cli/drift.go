package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/platformfoundry/pf-ce/internal/drift"
	"github.com/spf13/cobra"
)

var driftCmd = &cobra.Command{
	Use:   "drift",
	Short: "Drift detection commands",
	Long:  "Commands for detecting and reporting drift between desired and actual state.",
}

var driftDetectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Detect drift in managed resources",
	Long: `Detect drift between the desired state (as defined in your platform configurations)
and the actual state of deployed resources.

Examples:
  pf drift detect                    # Detect drift in all resources
  pf drift detect --type Deployment  # Detect drift only in Deployments
  pf drift detect --severity high    # Show only high severity drift`,
	RunE: runDriftDetect,
}

var (
	driftResourceType string
	driftMinSeverity  string
	driftOutputFormat string
)

func init() {
	driftCmd.AddCommand(driftDetectCmd)

	driftDetectCmd.Flags().StringVar(&driftResourceType, "type", "", "Filter by resource type")
	driftDetectCmd.Flags().StringVar(&driftMinSeverity, "severity", "", "Minimum severity to report (critical, high, medium, low)")
	driftDetectCmd.Flags().StringVarP(&driftOutputFormat, "output", "o", "text", "Output format (text, json)")
}

func runDriftDetect(cmd *cobra.Command, args []string) error {
	fmt.Println("Drift Detection")
	fmt.Println("===============")
	fmt.Println()

	// Create detector with default config
	config := drift.DetectorConfig{
		Concurrency: 5,
		IgnorePaths: []string{
			"metadata.resourceVersion",
			"metadata.uid",
			"metadata.generation",
			"metadata.creationTimestamp",
			"status*",
		},
	}

	detector := drift.NewDetector(config, nil)

	// In a real implementation, we would:
	// 1. Load desired state from platform configurations
	// 2. Query actual state from the cluster/cloud
	// For now, show a demo with sample resources

	fmt.Println("Scanning for drift...")
	fmt.Println()

	// Demo: Create sample resources to check
	resources := getSampleResources()

	if len(resources) == 0 {
		fmt.Println("No resources found to check for drift.")
		fmt.Println()
		fmt.Println("To use drift detection:")
		fmt.Println("  1. Define your desired state in platform configurations")
		fmt.Println("  2. Deploy resources using 'pf apply'")
		fmt.Println("  3. Run 'pf drift detect' to find differences")
		return nil
	}

	report, err := detector.Detect(cmd.Context(), resources)
	if err != nil {
		return fmt.Errorf("drift detection failed: %w", err)
	}

	// Filter by severity if specified
	if driftMinSeverity != "" {
		report = filterBySeverity(report, driftMinSeverity)
	}

	// Filter by type if specified
	if driftResourceType != "" {
		report = filterByType(report, driftResourceType)
	}

	// Output results
	if driftOutputFormat == "json" {
		return outputDriftJSON(report)
	}

	fmt.Print(drift.FormatReport(report))

	if report.Summary.Total > 0 {
		fmt.Println()
		fmt.Println("Run 'pf apply' to reconcile drift or 'pf drift detect --output json' for detailed output.")
		os.Exit(1) // Non-zero exit for CI/CD integration
	}

	return nil
}

func getSampleResources() []drift.Resource {
	// In production, this would load from:
	// - Platform configuration files
	// - State storage
	// - Kubernetes API
	// - Cloud provider APIs

	// Return empty for now - actual implementation would query real state
	return []drift.Resource{}
}

func filterBySeverity(report *drift.Report, minSeverity string) *drift.Report {
	severityOrder := map[drift.DriftSeverity]int{
		drift.SeverityCritical: 4,
		drift.SeverityHigh:     3,
		drift.SeverityMedium:   2,
		drift.SeverityLow:      1,
		drift.SeverityInfo:     0,
	}

	minLevel := severityOrder[drift.DriftSeverity(minSeverity)]

	filtered := &drift.Report{
		ID:               report.ID,
		StartedAt:        report.StartedAt,
		CompletedAt:      report.CompletedAt,
		Duration:         report.Duration,
		ResourcesChecked: report.ResourcesChecked,
		Drifts:           make([]drift.Drift, 0),
		Summary:          drift.Summary{ByType: make(map[drift.DriftType]int)},
	}

	for _, d := range report.Drifts {
		if severityOrder[d.Severity] >= minLevel {
			filtered.Drifts = append(filtered.Drifts, d)
			filtered.Summary.Total++
			filtered.Summary.ByType[d.Type]++

			switch d.Severity {
			case drift.SeverityCritical:
				filtered.Summary.Critical++
			case drift.SeverityHigh:
				filtered.Summary.High++
			case drift.SeverityMedium:
				filtered.Summary.Medium++
			case drift.SeverityLow:
				filtered.Summary.Low++
			case drift.SeverityInfo:
				filtered.Summary.Info++
			}
		}
	}

	return filtered
}

func filterByType(report *drift.Report, resourceType string) *drift.Report {
	filtered := &drift.Report{
		ID:               report.ID,
		StartedAt:        report.StartedAt,
		CompletedAt:      report.CompletedAt,
		Duration:         report.Duration,
		ResourcesChecked: report.ResourcesChecked,
		Drifts:           make([]drift.Drift, 0),
		Summary:          drift.Summary{ByType: make(map[drift.DriftType]int)},
	}

	for _, d := range report.Drifts {
		if d.ResourceType == resourceType {
			filtered.Drifts = append(filtered.Drifts, d)
			filtered.Summary.Total++
			filtered.Summary.ByType[d.Type]++

			switch d.Severity {
			case drift.SeverityCritical:
				filtered.Summary.Critical++
			case drift.SeverityHigh:
				filtered.Summary.High++
			case drift.SeverityMedium:
				filtered.Summary.Medium++
			case drift.SeverityLow:
				filtered.Summary.Low++
			case drift.SeverityInfo:
				filtered.Summary.Info++
			}
		}
	}

	return filtered
}

func outputDriftJSON(report *drift.Report) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}
	fmt.Println(string(data))
	return nil
}
