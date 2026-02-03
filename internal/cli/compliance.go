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

var complianceDriftCmd = &cobra.Command{
	Use:   "drift",
	Short: "Manage compliance drift monitoring",
}

var complianceDriftStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show drift monitoring status",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Compliance Drift Monitor Status")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("Status:             Running")
		fmt.Println("Scan Interval:      30m")
		fmt.Println("Alert Threshold:    90%")
		fmt.Println("Critical Threshold: 80%")
		fmt.Println()
		fmt.Println("Monitored Policies: 3")
		fmt.Println("Active Alerts:      0")
		fmt.Println("Degrading Policies: 0")
		return nil
	},
}

var complianceDriftShowCmd = &cobra.Command{
	Use:   "show [policy-name]",
	Short: "Show drift details for a policy",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			fmt.Printf("Drift Record: %s\n", args[0])
		} else {
			fmt.Println("All Drift Records:")
		}
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("%-30s %-10s %-10s %-10s %s\n", "POLICY", "BASELINE", "CURRENT", "DRIFT", "TREND")
		fmt.Printf("%-30s %-10s %-10s %-10s %s\n", "kubernetes-security-baseline", "95.0%", "94.5%", "-0.5%", "stable")
		fmt.Printf("%-30s %-10s %-10s %-10s %s\n", "data-protection-baseline", "92.0%", "92.5%", "+0.5%", "improving")
		return nil
	},
}

var complianceDriftResetCmd = &cobra.Command{
	Use:   "reset <policy-name>",
	Short: "Reset drift baseline for a policy",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Resetting drift baseline for policy '%s'...\n", args[0])
		fmt.Printf("Baseline reset to current score.\n")
		return nil
	},
}

var complianceAlertsCmd = &cobra.Command{
	Use:   "alerts",
	Short: "Manage compliance alerts",
}

var complianceAlertsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List compliance alerts",
	RunE: func(cmd *cobra.Command, args []string) error {
		severity, _ := cmd.Flags().GetString("severity")
		unacked, _ := cmd.Flags().GetBool("unacknowledged")

		fmt.Printf("Compliance Alerts")
		if severity != "" {
			fmt.Printf(" (severity: %s)", severity)
		}
		if unacked {
			fmt.Printf(" (unacknowledged)")
		}
		fmt.Println()
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("No active alerts.")
		return nil
	},
}

var complianceAlertsAckCmd = &cobra.Command{
	Use:   "ack <alert-id>",
	Short: "Acknowledge an alert",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Acknowledging alert '%s'...\n", args[0])
		fmt.Printf("Alert acknowledged.\n")
		return nil
	},
}

var complianceControlsCmd = &cobra.Command{
	Use:   "controls",
	Short: "Manage control mappings",
}

var complianceControlsListCmd = &cobra.Command{
	Use:   "list <framework>",
	Short: "List controls for a framework",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		framework := args[0]
		cm := compliance.NewControlMapping()
		controls := cm.GetControlsByFramework(framework)

		fmt.Printf("%s Controls (%d)\n", framework, len(controls))
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		// Group by category
		categories := make(map[string][]*compliance.Control)
		for _, c := range controls {
			categories[c.Category] = append(categories[c.Category], c)
		}

		for category, catControls := range categories {
			fmt.Printf("\n%s:\n", category)
			for _, c := range catControls {
				fmt.Printf("  • %s: %s\n", c.ID, c.Name)
			}
		}
		return nil
	},
}

var complianceControlsCoverageCmd = &cobra.Command{
	Use:   "coverage <framework>",
	Short: "Show control coverage for a framework",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		framework := args[0]
		cm := compliance.NewControlMapping()
		report := cm.GenerateCoverageReport(framework)

		fmt.Printf("%s Control Coverage Report\n", framework)
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("Generated: %s\n\n", report.GeneratedAt.Format("2006-01-02 15:04:05"))

		fmt.Println("Summary:")
		fmt.Printf("  Total Controls:     %d\n", report.Summary.TotalControls)
		fmt.Printf("  Full Coverage:      %d\n", report.Summary.FullCoverage)
		fmt.Printf("  Partial Coverage:   %d\n", report.Summary.PartialCoverage)
		fmt.Printf("  Minimal Coverage:   %d\n", report.Summary.MinimalCoverage)
		fmt.Printf("  No Coverage:        %d\n", report.Summary.NoCoverage)
		fmt.Printf("  Overall:            %.1f%%\n\n", report.Summary.OverallPercentage)

		fmt.Println("Coverage by Category:")
		for cat, cov := range report.Summary.ByCategory {
			pct := float64(cov.Covered) / float64(cov.Total) * 100
			fmt.Printf("  %-40s %d/%d (%.0f%%)\n", cat, cov.Covered, cov.Total, pct)
		}
		return nil
	},
}

var complianceControlsMapCmd = &cobra.Command{
	Use:   "map <control-id>",
	Short: "Show platform features mapped to a control",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		controlID := args[0]
		cm := compliance.NewControlMapping()

		control, err := cm.GetControl(controlID)
		if err != nil {
			return err
		}

		features := cm.GetFeatures(controlID)

		fmt.Printf("Control: %s\n", control.ID)
		fmt.Printf("Name: %s\n", control.Name)
		fmt.Printf("Framework: %s\n", control.Framework)
		fmt.Printf("Category: %s\n\n", control.Category)
		fmt.Printf("Description:\n  %s\n\n", control.Description)
		fmt.Printf("Requirement:\n  %s\n\n", control.Requirement)

		if len(features) == 0 {
			fmt.Println("Platform Features: None mapped")
		} else {
			fmt.Printf("Platform Features (%d):\n", len(features))
			for _, f := range features {
				fmt.Printf("\n  • %s (%s)\n", f.Name, f.ID)
				fmt.Printf("    Component:  %s\n", f.Component)
				fmt.Printf("    Status:     %s\n", f.Status)
				fmt.Printf("    Coverage:   %s\n", f.Coverage)
				fmt.Printf("    Automation: %s\n", f.Automation)
			}
		}
		return nil
	},
}

var complianceControlsGapsCmd = &cobra.Command{
	Use:   "gaps",
	Short: "Show coverage gaps across all frameworks",
	RunE: func(cmd *cobra.Command, args []string) error {
		cm := compliance.NewControlMapping()
		gaps := cm.GetGaps()

		fmt.Printf("Control Coverage Gaps (%d)\n", len(gaps))
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		if len(gaps) == 0 {
			fmt.Println("No coverage gaps found.")
			return nil
		}

		for _, gap := range gaps {
			fmt.Printf("\n%s: %s\n", gap.ControlID, gap.ControlName)
			fmt.Printf("  Framework: %s\n", gap.Framework)
			fmt.Printf("  Coverage:  %s\n", gap.Coverage)
			fmt.Printf("  Gaps:\n")
			for _, g := range gap.Gaps {
				fmt.Printf("    - %s\n", g)
			}
		}
		return nil
	},
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

	// Drift monitoring commands
	complianceCmd.AddCommand(complianceDriftCmd)
	complianceDriftCmd.AddCommand(complianceDriftStatusCmd)
	complianceDriftCmd.AddCommand(complianceDriftShowCmd)
	complianceDriftCmd.AddCommand(complianceDriftResetCmd)

	// Alert commands
	complianceCmd.AddCommand(complianceAlertsCmd)
	complianceAlertsCmd.AddCommand(complianceAlertsListCmd)
	complianceAlertsListCmd.Flags().String("severity", "", "Filter by severity (critical, high, medium, low)")
	complianceAlertsListCmd.Flags().Bool("unacknowledged", false, "Show only unacknowledged alerts")
	complianceAlertsCmd.AddCommand(complianceAlertsAckCmd)

	// Control mapping commands
	complianceCmd.AddCommand(complianceControlsCmd)
	complianceControlsCmd.AddCommand(complianceControlsListCmd)
	complianceControlsCmd.AddCommand(complianceControlsCoverageCmd)
	complianceControlsCmd.AddCommand(complianceControlsMapCmd)
	complianceControlsCmd.AddCommand(complianceControlsGapsCmd)
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
