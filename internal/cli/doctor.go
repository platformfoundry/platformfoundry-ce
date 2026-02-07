package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/platformfoundry/pf-ce/internal/doctor"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system health and prerequisites",
	Long: `Verify that your system has all required tools and configurations
for Platform Foundry to work correctly.

This command checks for:
  - Required tools (Docker, kubectl, Helm)
  - Optional tools (Terraform, kind, Go)
  - System resources (disk space, memory)
  - Network connectivity

Examples:
  pf doctor              # Run all health checks
  pf doctor --json       # Output results as JSON
  pf doctor --verbose    # Show detailed information`,
	RunE: runDoctor,
}

var (
	doctorJSON    bool
	doctorVerbose bool
)

func init() {
	doctorCmd.Flags().BoolVar(&doctorJSON, "json", false, "Output results as JSON")
	doctorCmd.Flags().BoolVar(&doctorVerbose, "verbose", false, "Show detailed information")
}

func runDoctor(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Println("Running health checks...")
	fmt.Println()

	doc := doctor.New()
	report := doc.RunAll(ctx)

	if doctorJSON {
		output, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal report: %w", err)
		}
		fmt.Println(string(output))
		return nil
	}

	// Print formatted report
	fmt.Print(doctor.FormatReport(report))

	// Return error if critical checks failed
	if report.Summary.Errors > 0 {
		return fmt.Errorf("%d critical check(s) failed", report.Summary.Errors)
	}

	return nil
}
