package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/platformfoundry/pf-ce/internal/diff"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(diffCmd)
	diffCmd.Flags().StringP("resource", "r", "", "Filter by resource name")
	diffCmd.Flags().StringP("output", "o", "table", "Output format: table, json, yaml")
	diffCmd.Flags().Bool("ignore-status", true, "Ignore status fields in comparison")
}

var diffCmd = &cobra.Command{
	Use:   "diff <source> <target>",
	Short: "Compare platform states across environments",
	Long: `Compare platform configurations or states between two environments.

This command helps identify differences between environments such as:
- Configuration drift between staging and production
- Missing resources in one environment
- Different resource specifications

Sources can be:
- Environment names (loads from state)
- File paths (YAML/JSON configuration files)`,
	Example: `  # Compare staging and production environments
  pf diff staging production

  # Compare two configuration files
  pf diff platform-staging.yaml platform-prod.yaml

  # Filter by specific resource
  pf diff staging production -r argocd

  # Output as JSON
  pf diff staging production -o json`,
	Args: cobra.ExactArgs(2),
	RunE: runDiff,
}

func runDiff(cmd *cobra.Command, args []string) error {
	source := args[0]
	target := args[1]
	resourceFilter, _ := cmd.Flags().GetString("resource")
	output, _ := cmd.Flags().GetString("output")
	ignoreStatus, _ := cmd.Flags().GetBool("ignore-status")

	// Load source and target states
	sourceState, err := loadPlatformState(source)
	if err != nil {
		return fmt.Errorf("failed to load source '%s': %w", source, err)
	}

	targetState, err := loadPlatformState(target)
	if err != nil {
		return fmt.Errorf("failed to load target '%s': %w", target, err)
	}

	// Create differ with options
	differ := diff.NewDiffer()
	if ignoreStatus {
		differ.WithIgnoredPaths([]string{
			"metadata.creationTimestamp",
			"metadata.uid",
			"metadata.resourceVersion",
			"status",
		})
	}

	// Perform comparison
	result := differ.Compare(sourceState, targetState)

	// Filter by resource if specified
	if resourceFilter != "" {
		result = filterDiffByResource(result, resourceFilter)
	}

	// Output results
	switch output {
	case "json":
		return outputDiffJSON(result)
	case "yaml":
		return outputDiffYAML(result)
	default:
		return outputDiffTable(result)
	}
}

func loadPlatformState(source string) (*diff.PlatformState, error) {
	// Check if source is a file
	if _, err := os.Stat(source); err == nil {
		return loadStateFromFile(source)
	}

	// Otherwise, treat as environment name and create sample state
	// In production, this would load from state backend
	return loadStateFromEnvironment(source)
}

func loadStateFromFile(path string) (*diff.PlatformState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var content map[string]interface{}
	if err := json.Unmarshal(data, &content); err != nil {
		// Try YAML parsing - for now just return error
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	state := &diff.PlatformState{
		Name:      path,
		Resources: make(map[string]map[string]interface{}),
	}

	for k, v := range content {
		if m, ok := v.(map[string]interface{}); ok {
			state.Resources[k] = m
		}
	}

	return state, nil
}

func loadStateFromEnvironment(env string) (*diff.PlatformState, error) {
	// In production, this would load from state backend
	// For now, return a sample state for demonstration
	state := &diff.PlatformState{
		Name:      env,
		Resources: make(map[string]map[string]interface{}),
	}

	// Create sample resources based on environment
	switch env {
	case "staging":
		state.Resources["argocd"] = map[string]interface{}{
			"replicas":  float64(1),
			"version":   "v2.9.0",
			"namespace": "argocd",
			"sync":      "automated",
		}
		state.Resources["prometheus"] = map[string]interface{}{
			"retention":  "7d",
			"replicas":   float64(1),
			"scrapeInterval": "30s",
		}
		state.Resources["web-api"] = map[string]interface{}{
			"replicas": float64(2),
			"image":    "web-api:v1.2.0",
			"resources": map[string]interface{}{
				"cpu":    "500m",
				"memory": "512Mi",
			},
		}
	case "production":
		state.Resources["argocd"] = map[string]interface{}{
			"replicas":  float64(3),
			"version":   "v2.9.0",
			"namespace": "argocd",
			"sync":      "manual",
		}
		state.Resources["prometheus"] = map[string]interface{}{
			"retention":  "30d",
			"replicas":   float64(2),
			"scrapeInterval": "15s",
		}
		state.Resources["web-api"] = map[string]interface{}{
			"replicas": float64(5),
			"image":    "web-api:v1.2.0",
			"resources": map[string]interface{}{
				"cpu":    "2000m",
				"memory": "2Gi",
			},
		}
		state.Resources["disaster-recovery"] = map[string]interface{}{
			"enabled":  true,
			"region":   "us-west-2",
			"replicas": float64(2),
		}
	default:
		state.Resources["platform"] = map[string]interface{}{
			"name":    env,
			"version": "1.0.0",
		}
	}

	return state, nil
}

func filterDiffByResource(result *diff.PlatformDiff, filter string) *diff.PlatformDiff {
	filtered := &diff.PlatformDiff{
		Source:       result.Source,
		Target:       result.Target,
		Differences:  make([]diff.ResourceDiff, 0),
		OnlyInSource: make([]string, 0),
		OnlyInTarget: make([]string, 0),
	}

	for _, d := range result.Differences {
		if strings.Contains(strings.ToLower(d.Resource), strings.ToLower(filter)) {
			filtered.Differences = append(filtered.Differences, d)
		}
	}

	for _, r := range result.OnlyInSource {
		if strings.Contains(strings.ToLower(r), strings.ToLower(filter)) {
			filtered.OnlyInSource = append(filtered.OnlyInSource, r)
		}
	}

	for _, r := range result.OnlyInTarget {
		if strings.Contains(strings.ToLower(r), strings.ToLower(filter)) {
			filtered.OnlyInTarget = append(filtered.OnlyInTarget, r)
		}
	}

	// Recalculate summary
	filtered.Summary = diff.DiffSummary{
		TotalDifferences: len(filtered.Differences),
	}
	for _, d := range filtered.Differences {
		switch d.Type {
		case diff.DiffTypeAdded:
			filtered.Summary.Added++
		case diff.DiffTypeRemoved:
			filtered.Summary.Removed++
		case diff.DiffTypeModified:
			filtered.Summary.Modified++
		}
		switch d.Severity {
		case diff.SeverityCritical:
			filtered.Summary.Critical++
		case diff.SeverityWarning:
			filtered.Summary.Warning++
		case diff.SeverityInfo:
			filtered.Summary.Info++
		}
	}

	return filtered
}

func outputDiffJSON(result *diff.PlatformDiff) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func outputDiffYAML(result *diff.PlatformDiff) error {
	// Simple YAML-like output
	fmt.Printf("source: %s\n", result.Source)
	fmt.Printf("target: %s\n", result.Target)
	fmt.Printf("summary:\n")
	fmt.Printf("  total: %d\n", result.Summary.TotalDifferences)
	fmt.Printf("  added: %d\n", result.Summary.Added)
	fmt.Printf("  removed: %d\n", result.Summary.Removed)
	fmt.Printf("  modified: %d\n", result.Summary.Modified)
	fmt.Println("differences:")
	for _, d := range result.Differences {
		fmt.Printf("  - resource: %s\n", d.Resource)
		fmt.Printf("    type: %s\n", d.Type)
		if d.Path != "" {
			fmt.Printf("    path: %s\n", d.Path)
		}
		if d.SourceValue != nil {
			fmt.Printf("    source_value: %v\n", d.SourceValue)
		}
		if d.TargetValue != nil {
			fmt.Printf("    target_value: %v\n", d.TargetValue)
		}
	}
	return nil
}

func outputDiffTable(result *diff.PlatformDiff) error {
	fmt.Print(diff.FormatDiff(result))
	return nil
}
