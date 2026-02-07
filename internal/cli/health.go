package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/platformfoundry/pf-ce/internal/health"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(healthCmd)
	healthCmd.Flags().StringP("platform", "p", "", "Platform name to check")
	healthCmd.Flags().StringSliceP("files", "f", []string{}, "Configuration files to check")
	healthCmd.Flags().StringP("output", "o", "table", "Output format: table, json, yaml")
	healthCmd.Flags().BoolP("watch", "w", false, "Continuously watch health status")
	healthCmd.Flags().DurationP("interval", "i", 30*time.Second, "Watch interval")
	healthCmd.Flags().Float64("budget", 0, "Monthly cost budget for scoring")
}

var healthCmd = &cobra.Command{
	Use:   "health [platform]",
	Short: "Check platform health score",
	Long: `Check the overall health of a platform by aggregating:
- Configuration quality (linting)
- Infrastructure drift status
- Policy compliance
- Cost efficiency
- Security posture

The health score is a weighted aggregate of all categories, ranging from 0-100.`,
	Example: `  # Check health of current directory
  pf health

  # Check health of a specific platform
  pf health my-platform

  # Check health with specific files
  pf health -f platform.yaml -f infrastructure.yaml

  # Check health with cost budget
  pf health --budget 5000

  # Output as JSON
  pf health -o json

  # Continuously watch health
  pf health --watch --interval 1m`,
	RunE: runHealth,
}

func runHealth(cmd *cobra.Command, args []string) error {
	platform, _ := cmd.Flags().GetString("platform")
	files, _ := cmd.Flags().GetStringSlice("files")
	output, _ := cmd.Flags().GetString("output")
	watch, _ := cmd.Flags().GetBool("watch")
	interval, _ := cmd.Flags().GetDuration("interval")
	budget, _ := cmd.Flags().GetFloat64("budget")

	// Get platform name from args if not specified
	if platform == "" && len(args) > 0 {
		platform = args[0]
	}

	if platform == "" {
		platform = "default"
	}

	// Auto-discover config files if not specified
	if len(files) == 0 {
		files = discoverConfigFiles(".")
	}

	// Create health checker config
	config := health.DefaultConfig()
	if budget > 0 {
		config.CostBudget = budget
	}

	checker := health.NewChecker(config)

	ctx := context.Background()

	if watch {
		return watchHealth(ctx, checker, platform, files, output, interval)
	}

	return checkAndDisplayHealth(ctx, checker, platform, files, output)
}

func checkAndDisplayHealth(ctx context.Context, checker *health.Checker, platform string, files []string, output string) error {
	score, err := checker.Check(ctx, platform, files)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	switch output {
	case "json":
		return outputJSON(score)
	case "yaml":
		return outputYAML(score)
	default:
		return outputTable(score)
	}
}

func watchHealth(ctx context.Context, checker *health.Checker, platform string, files []string, output string, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial check
	if err := checkAndDisplayHealth(ctx, checker, platform, files, output); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			fmt.Println("\n" + strings.Repeat("=", 60))
			fmt.Printf("Health check at %s\n", time.Now().Format(time.RFC3339))
			fmt.Println(strings.Repeat("=", 60))

			if err := checkAndDisplayHealth(ctx, checker, platform, files, output); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		}
	}
}

func outputJSON(score *health.Score) error {
	data, err := json.MarshalIndent(score, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func outputYAML(score *health.Score) error {
	// Simple YAML-like output
	fmt.Printf("overall: %d\n", score.Overall)
	fmt.Printf("status: %s\n", score.Status)
	fmt.Printf("platform: %s\n", score.Platform)
	fmt.Printf("checked_at: %s\n", score.CheckedAt.Format(time.RFC3339))
	fmt.Println("categories:")
	for name, cat := range score.Categories {
		fmt.Printf("  %s:\n", name)
		fmt.Printf("    score: %d\n", cat.Score)
		fmt.Printf("    status: %s\n", cat.Status)
		fmt.Printf("    issues: %d\n", cat.IssueCount)
	}
	return nil
}

func outputTable(score *health.Score) error {
	// Header
	statusIcon := healthStatusIcon(score.Status)
	fmt.Printf("\nPlatform Health: %s\n", score.Platform)
	fmt.Println(strings.Repeat("=", 65))
	fmt.Printf("\nOverall Score: %d/100 %s\n\n", score.Overall, statusIcon)

	// Categories table
	fmt.Println("+----------------------+-------+----------+--------------------+")
	fmt.Println("| Category             | Score | Status   | Issues             |")
	fmt.Println("+----------------------+-------+----------+--------------------+")

	categoryOrder := []string{"configuration", "drift", "policy", "cost", "security"}
	for _, name := range categoryOrder {
		cat, ok := score.Categories[name]
		if !ok {
			continue
		}
		icon := healthStatusIcon(cat.Status)
		message := cat.Message
		if len(message) > 18 {
			message = message[:15] + "..."
		}
		fmt.Printf("| %-20s | %5d | %-2s       | %-18s |\n",
			healthCapitalizeFirst(cat.Name), cat.Score, icon, message)
	}
	fmt.Println("+----------------------+-------+----------+--------------------+")

	// Top issues
	if len(score.Issues) > 0 {
		fmt.Println("\nTop Issues:")
		maxIssues := 5
		if len(score.Issues) < maxIssues {
			maxIssues = len(score.Issues)
		}
		for i := 0; i < maxIssues; i++ {
			issue := score.Issues[i]
			severityIcon := healthSeverityIcon(issue.Severity)
			fmt.Printf("%d. [%s] %s: %s\n", i+1, severityIcon, strings.ToUpper(issue.Category), issue.Title)
			if issue.Description != "" && issue.Description != issue.Title {
				fmt.Printf("   %s\n", healthTruncate(issue.Description, 60))
			}
		}
		if len(score.Issues) > maxIssues {
			fmt.Printf("\n... and %d more issues\n", len(score.Issues)-maxIssues)
		}
	}

	// Recommendations
	if len(score.Recommendations) > 0 {
		fmt.Println("\nRecommendations:")
		for _, rec := range score.Recommendations {
			priorityIcon := healthPriorityIcon(rec.Priority)
			fmt.Printf("  %s %s\n", priorityIcon, rec.Title)
			if rec.Command != "" {
				fmt.Printf("     Run: %s\n", rec.Command)
			}
		}
	}

	fmt.Printf("\nLast checked: %s\n", score.CheckedAt.Format("2006-01-02 15:04:05"))

	return nil
}

func discoverConfigFiles(dir string) []string {
	files := []string{}
	patterns := []string{"*.yaml", "*.yml"}

	for _, pattern := range patterns {
		matches, _ := filepath.Glob(filepath.Join(dir, pattern))
		for _, match := range matches {
			// Skip hidden files and common non-config files
			base := filepath.Base(match)
			if strings.HasPrefix(base, ".") {
				continue
			}
			files = append(files, match)
		}
	}

	return files
}

func healthStatusIcon(status health.Status) string {
	switch status {
	case health.StatusHealthy:
		return "\u2705" // checkmark
	case health.StatusWarning:
		return "\u26a0\ufe0f" // warning
	case health.StatusCritical:
		return "\u274c" // x
	default:
		return "\u2753" // question
	}
}

func healthSeverityIcon(severity string) string {
	switch severity {
	case "critical":
		return "CRIT"
	case "high":
		return "HIGH"
	case "medium":
		return "MED "
	case "low":
		return "LOW "
	case "info":
		return "INFO"
	default:
		return "    "
	}
}

func healthPriorityIcon(priority string) string {
	switch priority {
	case "high":
		return "\u2757" // exclamation
	case "medium":
		return "\u2022" // bullet
	case "low":
		return "\u25cb" // circle
	default:
		return " "
	}
}

func healthCapitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func healthTruncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
