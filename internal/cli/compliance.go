package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/platformfoundry/pf-ce/internal/compliance"
	"github.com/spf13/cobra"
)

var (
	complianceFramework string
	complianceChecksDir string
	complianceReportsDir string
	complianceShowAll   bool
)

var complianceCmd = &cobra.Command{
	Use:   "compliance",
	Short: "Compliance management",
	Long:  `Manage compliance checks and reports for regulatory frameworks.`,
}

var complianceCheckCmd = &cobra.Command{
	Use:   "check <framework>",
	Short: "Run compliance checks",
	Long:  `Run compliance checks for a specific framework.`,
	Example: `  pf compliance check SOC2
  pf compliance check HIPAA
  pf compliance check PCI-DSS`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: complianceFrameworkCompletion,
	RunE:              runComplianceCheck,
}

var complianceReportCmd = &cobra.Command{
	Use:   "report [report-id]",
	Short: "View compliance report",
	Long:  `View a compliance report by ID or the latest report.`,
	Example: `  pf compliance report
  pf compliance report SOC2-20240115-120000.json`,
	RunE: runComplianceReport,
}

var complianceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List compliance reports",
	Long:  `List all compliance reports.`,
	Example: `  pf compliance list
  pf compliance list --framework SOC2`,
	RunE: runComplianceList,
}

var complianceFrameworksCmd = &cobra.Command{
	Use:   "frameworks",
	Short: "List supported frameworks",
	Long:  `List all supported compliance frameworks.`,
	Example: `  pf compliance frameworks`,
	RunE: runComplianceFrameworks,
}

var complianceChecksCmd = &cobra.Command{
	Use:   "checks <framework>",
	Short: "List framework checks",
	Long:  `List all checks for a specific compliance framework.`,
	Example: `  pf compliance checks SOC2
  pf compliance checks HIPAA`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: complianceFrameworkCompletion,
	RunE:              runComplianceChecks,
}

func init() {
	// Check command flags
	complianceCheckCmd.Flags().StringVar(&complianceChecksDir, "checks-dir", "/etc/platformfoundry/compliance/checks", "Checks directory")
	complianceCheckCmd.Flags().StringVar(&complianceReportsDir, "reports-dir", "/var/lib/platformfoundry/compliance/reports", "Reports directory")

	// Report command flags
	complianceReportCmd.Flags().StringVar(&complianceReportsDir, "reports-dir", "/var/lib/platformfoundry/compliance/reports", "Reports directory")
	complianceReportCmd.Flags().BoolVar(&complianceShowAll, "all", false, "Show all check details")

	// List command flags
	complianceListCmd.Flags().StringVar(&complianceFramework, "framework", "", "Filter by framework")
	complianceListCmd.Flags().StringVar(&complianceReportsDir, "reports-dir", "/var/lib/platformfoundry/compliance/reports", "Reports directory")

	// Frameworks command flags
	complianceFrameworksCmd.Flags().StringVar(&complianceChecksDir, "checks-dir", "/etc/platformfoundry/compliance/checks", "Checks directory")

	// Checks command flags
	complianceChecksCmd.Flags().StringVar(&complianceChecksDir, "checks-dir", "/etc/platformfoundry/compliance/checks", "Checks directory")

	// Add subcommands
	complianceCmd.AddCommand(complianceCheckCmd)
	complianceCmd.AddCommand(complianceReportCmd)
	complianceCmd.AddCommand(complianceListCmd)
	complianceCmd.AddCommand(complianceFrameworksCmd)
	complianceCmd.AddCommand(complianceChecksCmd)
}

func runComplianceCheck(cmd *cobra.Command, args []string) error {
	framework := compliance.Framework(args[0])

	config := &compliance.Config{
		ChecksDir:  complianceChecksDir,
		ReportsDir: complianceReportsDir,
	}

	manager, err := compliance.NewManager(config)
	if err != nil {
		return fmt.Errorf("failed to create compliance manager: %w", err)
	}

	fmt.Printf("Running %s compliance checks...\n\n", framework)

	ctx := context.Background()
	report, err := manager.RunChecks(ctx, framework)
	if err != nil {
		return fmt.Errorf("failed to run compliance checks: %w", err)
	}

	// Display summary
	fmt.Println("Compliance Report:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Framework: %s\n", report.Framework)
	fmt.Printf("Report ID: %s\n", report.ID)
	fmt.Printf("Timestamp: %s\n\n", report.Timestamp.Format("2006-01-02 15:04:05"))

	fmt.Printf("Total Checks: %d\n", report.TotalChecks)
	fmt.Printf("  ✓ Passed:  %d\n", report.PassedChecks)
	fmt.Printf("  ✗ Failed:  %d\n", report.FailedChecks)
	fmt.Printf("  ⚠ Warning: %d\n", report.WarningChecks)
	fmt.Printf("  - Skipped: %d\n\n", report.SkippedChecks)

	fmt.Printf("Compliance: %.1f%%\n\n", report.Compliance)

	// Display failed checks
	if report.FailedChecks > 0 {
		fmt.Println("Failed Checks:")
		for _, check := range report.Checks {
			if check.Status == compliance.StatusFail {
				fmt.Printf("\n  ✗ %s\n", check.Title)
				fmt.Printf("    ID: %s\n", check.ID)
				fmt.Printf("    Severity: %s\n", check.Severity)
				fmt.Printf("    Message: %s\n", check.Message)
				if check.Remediation != "" {
					fmt.Printf("    Remediation: %s\n", check.Remediation)
				}
			}
		}
		fmt.Println()
	}

	// Display warning checks
	if report.WarningChecks > 0 {
		fmt.Println("Warning Checks:")
		for _, check := range report.Checks {
			if check.Status == compliance.StatusWarning {
				fmt.Printf("\n  ⚠ %s\n", check.Title)
				fmt.Printf("    ID: %s\n", check.ID)
				fmt.Printf("    Message: %s\n", check.Message)
			}
		}
		fmt.Println()
	}

	return nil
}

func runComplianceReport(cmd *cobra.Command, args []string) error {
	config := &compliance.Config{
		ReportsDir: complianceReportsDir,
	}

	manager, err := compliance.NewManager(config)
	if err != nil {
		return fmt.Errorf("failed to create compliance manager: %w", err)
	}

	var report *compliance.Report

	if len(args) > 0 {
		// Load specific report
		report, err = manager.LoadReport(args[0])
		if err != nil {
			return fmt.Errorf("failed to load report: %w", err)
		}
	} else {
		// Load latest report
		reports, err := manager.ListReports(nil)
		if err != nil {
			return fmt.Errorf("failed to list reports: %w", err)
		}

		if len(reports) == 0 {
			fmt.Println("No reports found")
			return nil
		}

		// Get latest report (sort by timestamp)
		latest := reports[0]
		for _, r := range reports {
			if r.Timestamp.After(latest.Timestamp) {
				latest = r
			}
		}
		report = latest
	}

	// Display report
	fmt.Println("Compliance Report:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Framework: %s\n", report.Framework)
	fmt.Printf("Report ID: %s\n", report.ID)
	fmt.Printf("Timestamp: %s\n\n", report.Timestamp.Format("2006-01-02 15:04:05"))

	fmt.Printf("Total Checks: %d\n", report.TotalChecks)
	fmt.Printf("  ✓ Passed:  %d\n", report.PassedChecks)
	fmt.Printf("  ✗ Failed:  %d\n", report.FailedChecks)
	fmt.Printf("  ⚠ Warning: %d\n", report.WarningChecks)
	fmt.Printf("  - Skipped: %d\n\n", report.SkippedChecks)

	fmt.Printf("Compliance: %.1f%%\n\n", report.Compliance)

	if complianceShowAll {
		fmt.Println("All Checks:")
		for _, check := range report.Checks {
			fmt.Printf("\n%s %s\n", getCheckSymbol(check.Status), check.Title)
			fmt.Printf("  ID: %s\n", check.ID)
			fmt.Printf("  Category: %s\n", check.Category)
			fmt.Printf("  Severity: %s\n", check.Severity)
			fmt.Printf("  Status: %s\n", check.Status)
			if check.Message != "" {
				fmt.Printf("  Message: %s\n", check.Message)
			}
			if check.Status != compliance.StatusPass && check.Remediation != "" {
				fmt.Printf("  Remediation: %s\n", check.Remediation)
			}
		}
	}

	return nil
}

func runComplianceList(cmd *cobra.Command, args []string) error {
	config := &compliance.Config{
		ReportsDir: complianceReportsDir,
	}

	manager, err := compliance.NewManager(config)
	if err != nil {
		return fmt.Errorf("failed to create compliance manager: %w", err)
	}

	var frameworkFilter *compliance.Framework
	if complianceFramework != "" {
		f := compliance.Framework(complianceFramework)
		frameworkFilter = &f
	}

	reports, err := manager.ListReports(frameworkFilter)
	if err != nil {
		return fmt.Errorf("failed to list reports: %w", err)
	}

	if len(reports) == 0 {
		fmt.Println("No reports found")
		return nil
	}

	fmt.Printf("Compliance Reports (%d):\n", len(reports))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for i, report := range reports {
		fmt.Printf("%d. %s - %s\n", i+1, report.Framework, report.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("   Report ID: %s\n", report.ID)
		fmt.Printf("   Compliance: %.1f%% (%d/%d passed)\n", report.Compliance, report.PassedChecks, report.TotalChecks)

		if i < len(reports)-1 {
			fmt.Println()
		}
	}

	return nil
}

func runComplianceFrameworks(cmd *cobra.Command, args []string) error {
	config := &compliance.Config{
		ChecksDir: complianceChecksDir,
	}

	manager, err := compliance.NewManager(config)
	if err != nil {
		return fmt.Errorf("failed to create compliance manager: %w", err)
	}

	frameworks := manager.ListFrameworks()

	fmt.Printf("Supported Compliance Frameworks (%d):\n", len(frameworks))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	descriptions := map[compliance.Framework]string{
		compliance.FrameworkSOC2:    "SOC 2 - Service Organization Control 2",
		compliance.FrameworkHIPAA:   "HIPAA - Health Insurance Portability and Accountability Act",
		compliance.FrameworkPCIDSS:  "PCI-DSS - Payment Card Industry Data Security Standard",
		compliance.FrameworkGDPR:    "GDPR - General Data Protection Regulation",
		compliance.FrameworkISO27001: "ISO 27001 - Information Security Management",
		compliance.FrameworkNIST:    "NIST - National Institute of Standards and Technology",
	}

	for i, f := range frameworks {
		desc, ok := descriptions[f]
		if !ok {
			desc = string(f)
		}

		checks, _ := manager.GetChecks(f)
		fmt.Printf("%d. %s\n", i+1, f)
		fmt.Printf("   %s\n", desc)
		fmt.Printf("   Checks: %d\n", len(checks))

		if i < len(frameworks)-1 {
			fmt.Println()
		}
	}

	return nil
}

func runComplianceChecks(cmd *cobra.Command, args []string) error {
	framework := compliance.Framework(args[0])

	config := &compliance.Config{
		ChecksDir: complianceChecksDir,
	}

	manager, err := compliance.NewManager(config)
	if err != nil {
		return fmt.Errorf("failed to create compliance manager: %w", err)
	}

	checks, err := manager.GetChecks(framework)
	if err != nil {
		return fmt.Errorf("failed to get checks: %w", err)
	}

	fmt.Printf("%s Compliance Checks (%d):\n", framework, len(checks))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Group by category
	categories := make(map[string][]*compliance.Check)
	for _, check := range checks {
		categories[check.Category] = append(categories[check.Category], check)
	}

	for category, categoryChecks := range categories {
		fmt.Printf("\n%s:\n", category)
		for _, check := range categoryChecks {
			fmt.Printf("  • %s (ID: %s)\n", check.Title, check.ID)
			fmt.Printf("    %s\n", check.Description)
			fmt.Printf("    Severity: %s\n", strings.ToUpper(check.Severity))
		}
	}

	return nil
}

// Helper functions

func getCheckSymbol(status compliance.CheckStatus) string {
	switch status {
	case compliance.StatusPass:
		return "✓"
	case compliance.StatusFail:
		return "✗"
	case compliance.StatusWarning:
		return "⚠"
	case compliance.StatusSkipped:
		return "-"
	default:
		return "•"
	}
}
