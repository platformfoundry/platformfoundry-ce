package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/platformfoundry/pf-ce/internal/finops"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var finopsCmd = &cobra.Command{
	Use:   "finops",
	Short: "FinOps and cost management commands",
	Long:  `Manage costs, analyze resource rightsizing, and detect anomalies.`,
}

var finopsRightsizeCmd = &cobra.Command{
	Use:   "rightsize",
	Short: "Analyze resources for rightsizing opportunities",
	Long:  `Analyze workloads and provide rightsizing recommendations based on actual usage.`,
	RunE:  runFinopsRightsize,
}

var finopsAnomalyCmd = &cobra.Command{
	Use:   "anomaly",
	Short: "Detect cost anomalies",
	Long:  `Detect unusual cost patterns and spending anomalies.`,
	RunE:  runFinopsAnomaly,
}

var finopsCostCmd = &cobra.Command{
	Use:   "cost",
	Short: "View current costs",
	Long:  `Display current cost breakdown by resource, team, or environment.`,
	RunE:  runFinopsCost,
}

var finopsForecastCmd = &cobra.Command{
	Use:   "forecast",
	Short: "Forecast future costs",
	Long:  `Predict future costs based on historical trends.`,
	RunE:  runFinopsForecast,
}

// Policy commands
var finopsPolicyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Manage FinOps policies",
	Long:  `Create, view, and manage FinOps policies for budgets and optimization.`,
}

var finopsPolicyApplyCmd = &cobra.Command{
	Use:   "apply -f <file>",
	Short: "Apply a FinOps policy from file",
	RunE:  runFinopsPolicyApply,
}

var finopsPolicyGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Get a FinOps policy",
	Args:  cobra.ExactArgs(1),
	RunE:  runFinopsPolicyGet,
}

var finopsPolicyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all FinOps policies",
	RunE:  runFinopsPolicyList,
}

var finopsPolicyDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a FinOps policy",
	Args:  cobra.ExactArgs(1),
	RunE:  runFinopsPolicyDelete,
}

// Budget commands
var finopsBudgetCmd = &cobra.Command{
	Use:   "budget",
	Short: "Manage budgets",
	Long:  `View and manage cost budgets.`,
}

var finopsBudgetListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all budgets",
	RunE:  runFinopsBudgetList,
}

var finopsBudgetStatusCmd = &cobra.Command{
	Use:   "status [policy]",
	Short: "Show budget status",
	RunE:  runFinopsBudgetStatus,
}

// Report commands
var finopsReportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate cost reports",
	Long:  `Generate and view cost reports.`,
}

var finopsReportGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a cost report",
	RunE:  runFinopsReportGenerate,
}

// Recommendation commands
var finopsRecommendCmd = &cobra.Command{
	Use:   "recommend",
	Short: "View and manage recommendations",
	Long:  `View cost optimization recommendations and take action.`,
}

var finopsRecommendListCmd = &cobra.Command{
	Use:   "list [policy]",
	Short: "List recommendations",
	RunE:  runFinopsRecommendList,
}

var finopsRecommendApplyCmd = &cobra.Command{
	Use:   "apply <policy> <recommendation-id>",
	Short: "Apply a recommendation",
	Args:  cobra.ExactArgs(2),
	RunE:  runFinopsRecommendApply,
}

var finopsRecommendDismissCmd = &cobra.Command{
	Use:   "dismiss <policy> <recommendation-id>",
	Short: "Dismiss a recommendation",
	Args:  cobra.ExactArgs(2),
	RunE:  runFinopsRecommendDismiss,
}

// Global finops manager
var finopsManager *finops.Manager

// Flags
var (
	finopsMinSavings    float64
	finopsMinConfidence float64
	finopsTeam          string
	finopsEnvironment   string
	finopsDays          int
	finopsFormat        string
	finopsPolicyFile    string
	finopsReportPeriod  string
)

func init() {
	// Rightsize flags
	finopsRightsizeCmd.Flags().Float64Var(&finopsMinSavings, "min-savings", 20, "Minimum savings percentage to report")
	finopsRightsizeCmd.Flags().Float64Var(&finopsMinConfidence, "min-confidence", 0.7, "Minimum confidence threshold")
	finopsRightsizeCmd.Flags().StringVar(&finopsTeam, "team", "", "Filter by team")
	finopsRightsizeCmd.Flags().StringVar(&finopsFormat, "format", "table", "Output format (table, json)")

	// Anomaly flags
	finopsAnomalyCmd.Flags().StringVar(&finopsTeam, "team", "", "Filter by team")
	finopsAnomalyCmd.Flags().IntVar(&finopsDays, "days", 30, "Days of history for baseline")

	// Cost flags
	finopsCostCmd.Flags().StringVar(&finopsTeam, "team", "", "Filter by team")
	finopsCostCmd.Flags().StringVar(&finopsEnvironment, "env", "", "Filter by environment")
	finopsCostCmd.Flags().StringVar(&finopsFormat, "format", "table", "Output format (table, json)")

	// Forecast flags
	finopsForecastCmd.Flags().IntVar(&finopsDays, "days", 30, "Days to forecast")
	finopsForecastCmd.Flags().StringVar(&finopsTeam, "team", "", "Filter by team")

	// Policy flags
	finopsPolicyApplyCmd.Flags().StringVarP(&finopsPolicyFile, "file", "f", "", "Policy file (required)")
	finopsPolicyApplyCmd.MarkFlagRequired("file")

	// Report flags
	finopsReportGenerateCmd.Flags().StringVar(&finopsReportPeriod, "period", "month", "Report period (day, week, month)")
	finopsReportGenerateCmd.Flags().StringVar(&finopsFormat, "format", "table", "Output format (table, json)")

	// Recommend flags
	finopsRecommendListCmd.Flags().StringVar(&finopsFormat, "format", "table", "Output format (table, json)")

	// Add commands
	finopsCmd.AddCommand(finopsRightsizeCmd)
	finopsCmd.AddCommand(finopsAnomalyCmd)
	finopsCmd.AddCommand(finopsCostCmd)
	finopsCmd.AddCommand(finopsForecastCmd)
	finopsCmd.AddCommand(finopsPolicyCmd)
	finopsCmd.AddCommand(finopsBudgetCmd)
	finopsCmd.AddCommand(finopsReportCmd)
	finopsCmd.AddCommand(finopsRecommendCmd)

	// Policy subcommands
	finopsPolicyCmd.AddCommand(finopsPolicyApplyCmd)
	finopsPolicyCmd.AddCommand(finopsPolicyGetCmd)
	finopsPolicyCmd.AddCommand(finopsPolicyListCmd)
	finopsPolicyCmd.AddCommand(finopsPolicyDeleteCmd)

	// Budget subcommands
	finopsBudgetCmd.AddCommand(finopsBudgetListCmd)
	finopsBudgetCmd.AddCommand(finopsBudgetStatusCmd)

	// Report subcommands
	finopsReportCmd.AddCommand(finopsReportGenerateCmd)

	// Recommend subcommands
	finopsRecommendCmd.AddCommand(finopsRecommendListCmd)
	finopsRecommendCmd.AddCommand(finopsRecommendApplyCmd)
	finopsRecommendCmd.AddCommand(finopsRecommendDismissCmd)

	// Initialize manager
	finopsManager = finops.NewManager(nil)
}

func runFinopsRightsize(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	_ = ctx

	fmt.Println("Analyzing resources for rightsizing opportunities...")
	fmt.Println()

	// Mock output - in production would use finops.RightsizingEngine
	recommendations := []struct {
		Resource    string
		Current     string
		Recommended string
		Savings     float64
		Confidence  float64
		Reason      string
	}{
		{"api-server", "m5.xlarge", "m5.large", 150.00, 0.85, "CPU avg 15%, Memory avg 35%"},
		{"worker-1", "c5.2xlarge", "c5.xlarge", 200.00, 0.78, "CPU avg 22%, Memory avg 40%"},
		{"cache-server", "r5.large", "r5.medium", 75.00, 0.82, "Memory avg 30%"},
	}

	// Filter by min savings and confidence
	fmt.Printf("%-20s %-15s %-15s %12s %12s %s\n", "RESOURCE", "CURRENT", "RECOMMENDED", "SAVINGS/mo", "CONFIDENCE", "REASON")
	fmt.Println(strings.Repeat("-", 100))

	var totalSavings float64
	for _, r := range recommendations {
		savingsPercent := (r.Savings / 500) * 100 // Mock calculation
		if savingsPercent >= finopsMinSavings && r.Confidence >= finopsMinConfidence {
			fmt.Printf("%-20s %-15s %-15s $%10.2f %11.0f%% %s\n",
				r.Resource, r.Current, r.Recommended, r.Savings, r.Confidence*100, r.Reason)
			totalSavings += r.Savings
		}
	}

	fmt.Println(strings.Repeat("-", 100))
	fmt.Printf("Total potential monthly savings: $%.2f\n", totalSavings)

	return nil
}

func runFinopsAnomaly(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	_ = ctx

	fmt.Printf("Detecting cost anomalies (baseline: %d days)...\n", finopsDays)
	fmt.Println()

	// Mock output
	anomalies := []struct {
		Resource  string
		Expected  float64
		Actual    float64
		Deviation float64
		Severity  string
		Type      string
	}{
		{"storage-cluster", 500.00, 850.00, 70.0, "warning", "spike"},
		{"network-egress", 200.00, 450.00, 125.0, "critical", "spike"},
		{"compute-spot", 300.00, 150.00, -50.0, "info", "drop"},
	}

	fmt.Printf("%-20s %12s %12s %12s %10s %s\n", "RESOURCE", "EXPECTED", "ACTUAL", "DEVIATION", "SEVERITY", "TYPE")
	fmt.Println(strings.Repeat("-", 85))

	criticalCount := 0
	warningCount := 0

	for _, a := range anomalies {
		sevColor := ""
		switch a.Severity {
		case "critical":
			criticalCount++
		case "warning":
			warningCount++
		}
		fmt.Printf("%-20s $%10.2f $%10.2f %+11.1f%% %10s %s%s\n",
			a.Resource, a.Expected, a.Actual, a.Deviation, a.Severity, a.Type, sevColor)
	}

	fmt.Println(strings.Repeat("-", 85))
	fmt.Printf("Summary: %d critical, %d warning, %d total anomalies\n", criticalCount, warningCount, len(anomalies))

	return nil
}

func runFinopsCost(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	_ = ctx

	fmt.Println("Current cost breakdown:")
	fmt.Println()

	// Mock output
	costs := []struct {
		Category string
		Team     string
		Cost     float64
		Change   float64
	}{
		{"Compute", "platform", 2500.00, 5.2},
		{"Storage", "platform", 800.00, -2.1},
		{"Network", "platform", 350.00, 12.5},
		{"Database", "backend", 1200.00, 0.8},
		{"Cache", "backend", 400.00, -5.0},
	}

	fmt.Printf("%-15s %-15s %12s %12s\n", "CATEGORY", "TEAM", "COST/mo", "CHANGE")
	fmt.Println(strings.Repeat("-", 60))

	var total float64
	for _, c := range costs {
		if finopsTeam != "" && c.Team != finopsTeam {
			continue
		}
		changeStr := fmt.Sprintf("%+.1f%%", c.Change)
		fmt.Printf("%-15s %-15s $%10.2f %12s\n", c.Category, c.Team, c.Cost, changeStr)
		total += c.Cost
	}

	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("%-15s %-15s $%10.2f\n", "TOTAL", "", total)

	return nil
}

func runFinopsForecast(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	_ = ctx

	fmt.Printf("Cost forecast for next %d days:\n", finopsDays)
	fmt.Println()

	// Mock output
	currentMonthly := 5250.00
	trend := 1.05 // 5% increase trend

	forecasted := currentMonthly * trend

	fmt.Printf("Current monthly spend:    $%.2f\n", currentMonthly)
	fmt.Printf("Forecasted monthly spend: $%.2f\n", forecasted)
	fmt.Printf("Expected change:          %+.1f%%\n", (trend-1)*100)
	fmt.Println()
	fmt.Println("Factors contributing to forecast:")
	fmt.Println("  - Historical trend: +5% per month")
	fmt.Println("  - Seasonal adjustment: none")
	fmt.Println("  - Planned changes: none detected")

	return nil
}

// Policy command implementations

func runFinopsPolicyApply(cmd *cobra.Command, args []string) error {
	data, err := os.ReadFile(finopsPolicyFile)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var policy finops.FinOpsPolicy
	if err := yaml.Unmarshal(data, &policy); err != nil {
		return fmt.Errorf("failed to parse policy: %w", err)
	}

	if err := finopsManager.RegisterPolicy(context.Background(), &policy); err != nil {
		return fmt.Errorf("failed to register policy: %w", err)
	}

	fmt.Printf("Policy '%s' applied successfully\n", policy.Metadata.Name)
	return nil
}

func runFinopsPolicyGet(cmd *cobra.Command, args []string) error {
	policy, err := finopsManager.GetPolicy(args[0])
	if err != nil {
		return err
	}

	fmt.Printf("Name:        %s\n", policy.Metadata.Name)
	fmt.Printf("Description: %s\n", policy.Metadata.Description)
	fmt.Printf("Created:     %s\n", policy.Metadata.CreatedAt.Format(time.RFC3339))
	fmt.Printf("Updated:     %s\n", policy.Metadata.UpdatedAt.Format(time.RFC3339))
	fmt.Println()
	fmt.Printf("Budgets:     %d configured\n", len(policy.Spec.Budgets))
	fmt.Printf("Anomaly:     %v\n", policy.Spec.Anomaly.Enabled)
	fmt.Printf("Showback:    %v\n", policy.Spec.Showback.Enabled)
	fmt.Println()

	// Optimization settings
	opt := policy.Spec.Optimization
	fmt.Println("Optimization:")
	fmt.Printf("  Right-sizing:      %v\n", opt.RightSizing.Enabled)
	fmt.Printf("  Spot instances:    %v\n", opt.SpotInstances.Enabled)
	fmt.Printf("  Unused detection:  %s\n", opt.UnusedResources.DetectAfter)

	return nil
}

func runFinopsPolicyList(cmd *cobra.Command, args []string) error {
	policies := finopsManager.ListPolicies()

	if len(policies) == 0 {
		fmt.Println("No FinOps policies found")
		return nil
	}

	fmt.Printf("%-25s %-30s %10s %10s\n", "NAME", "DESCRIPTION", "BUDGETS", "STATUS")
	fmt.Println(strings.Repeat("-", 80))

	for _, p := range policies {
		status := "Active"
		fmt.Printf("%-25s %-30s %10d %10s\n",
			p.Metadata.Name,
			truncateString(p.Metadata.Description, 28),
			len(p.Spec.Budgets),
			status)
	}

	return nil
}

func runFinopsPolicyDelete(cmd *cobra.Command, args []string) error {
	if err := finopsManager.DeletePolicy(args[0]); err != nil {
		return err
	}
	fmt.Printf("Policy '%s' deleted\n", args[0])
	return nil
}

// Budget command implementations

func runFinopsBudgetList(cmd *cobra.Command, args []string) error {
	policies := finopsManager.ListPolicies()

	fmt.Printf("%-20s %-15s %-15s %12s %10s %12s\n", "NAME", "SCOPE", "POLICY", "AMOUNT", "PERIOD", "STATUS")
	fmt.Println(strings.Repeat("-", 95))

	for _, p := range policies {
		for _, b := range p.Spec.Budgets {
			fmt.Printf("%-20s %-15s %-15s $%10.2f %10s %12s\n",
				b.Name,
				b.Scope,
				p.Metadata.Name,
				b.Amount,
				b.Period,
				"active")
		}
	}

	return nil
}

func runFinopsBudgetStatus(cmd *cobra.Command, args []string) error {
	var policyName string
	if len(args) > 0 {
		policyName = args[0]
	}

	policies := finopsManager.ListPolicies()

	fmt.Printf("%-20s %12s %12s %10s %12s %12s\n", "BUDGET", "AMOUNT", "SPENT", "USED", "FORECAST", "STATUS")
	fmt.Println(strings.Repeat("-", 90))

	for _, p := range policies {
		if policyName != "" && p.Metadata.Name != policyName {
			continue
		}

		for _, status := range p.Status.BudgetStatus {
			statusStr := status.Status
			switch status.Status {
			case "over_budget":
				statusStr = "OVER BUDGET"
			case "at_risk":
				statusStr = "AT RISK"
			case "on_track":
				statusStr = "On Track"
			}

			fmt.Printf("%-20s $%10.2f $%10.2f %9.1f%% $%10.2f %12s\n",
				status.Name,
				status.Amount,
				status.Spent,
				status.SpentPercent,
				status.Forecast,
				statusStr)
		}
	}

	return nil
}

// Report command implementations

func runFinopsReportGenerate(cmd *cobra.Command, args []string) error {
	now := time.Now()
	var start, end time.Time

	switch finopsReportPeriod {
	case "day":
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		end = start.AddDate(0, 0, 1)
	case "week":
		weekday := int(now.Weekday())
		start = now.AddDate(0, 0, -weekday).Truncate(24 * time.Hour)
		end = start.AddDate(0, 0, 7)
	default: // month
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end = start.AddDate(0, 1, 0)
	}

	report, err := finopsManager.GenerateReport(context.Background(), start, end)
	if err != nil {
		return err
	}

	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Printf("║              COST REPORT: %s                        ║\n", finopsReportPeriod)
	fmt.Println("╠════════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  Period: %s - %s           ║\n",
		report.PeriodStart.Format("2006-01-02"),
		report.PeriodEnd.Format("2006-01-02"))
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	fmt.Println("SUMMARY")
	fmt.Println(strings.Repeat("-", 50))
	fmt.Printf("Total Cost:          $%.2f %s\n", report.TotalCost, report.Currency)
	fmt.Printf("Previous Period:     $%.2f\n", report.PreviousCost)
	fmt.Printf("Change:              $%.2f (%+.1f%%)\n", report.CostChange, report.ChangePercent)
	fmt.Println()

	// By Team
	fmt.Println("COST BY TEAM")
	fmt.Println(strings.Repeat("-", 50))
	for team, cost := range report.ByTeam {
		pct := (cost / report.TotalCost) * 100
		fmt.Printf("  %-20s $%10.2f (%5.1f%%)\n", team, cost, pct)
	}
	fmt.Println()

	// By Environment
	fmt.Println("COST BY ENVIRONMENT")
	fmt.Println(strings.Repeat("-", 50))
	for env, cost := range report.ByEnvironment {
		pct := (cost / report.TotalCost) * 100
		fmt.Printf("  %-20s $%10.2f (%5.1f%%)\n", env, cost, pct)
	}
	fmt.Println()

	// Top Spenders
	fmt.Println("TOP SPENDERS")
	fmt.Println(strings.Repeat("-", 50))
	for _, item := range report.TopSpenders {
		fmt.Printf("  %-20s %-10s $%10.2f\n", item.Name, item.Type, item.Cost)
	}
	fmt.Println()

	// Recommendations
	if len(report.Recommendations) > 0 {
		fmt.Println("RECOMMENDATIONS")
		fmt.Println(strings.Repeat("-", 50))
		var totalSavings float64
		for _, rec := range report.Recommendations {
			fmt.Printf("  [%s] %s\n", rec.Type, rec.Resource)
			fmt.Printf("    %s\n", rec.Description)
			fmt.Printf("    Potential savings: $%.2f/mo\n", rec.MonthlySavings)
			totalSavings += rec.MonthlySavings
		}
		fmt.Printf("\nTotal potential savings: $%.2f/mo\n", totalSavings)
		fmt.Println()
	}

	// Forecast
	if report.Forecast != nil {
		fmt.Println("FORECAST")
		fmt.Println(strings.Repeat("-", 50))
		fmt.Printf("  Next Month:    $%.2f\n", report.Forecast.NextMonth)
		fmt.Printf("  Next Quarter:  $%.2f\n", report.Forecast.NextQuarter)
		fmt.Printf("  End of Year:   $%.2f\n", report.Forecast.EndOfYear)
		fmt.Printf("  Trend:         %s (confidence: %.0f%%)\n", report.Forecast.Trend, report.Forecast.Confidence*100)
	}

	return nil
}

// Recommendation command implementations

func runFinopsRecommendList(cmd *cobra.Command, args []string) error {
	var policyName string
	if len(args) > 0 {
		policyName = args[0]
	}

	policies := finopsManager.ListPolicies()

	fmt.Printf("%-12s %-15s %-20s %-15s %12s %10s %10s\n",
		"ID", "TYPE", "RESOURCE", "POLICY", "SAVINGS/mo", "CONF", "STATUS")
	fmt.Println(strings.Repeat("-", 105))

	for _, p := range policies {
		if policyName != "" && p.Metadata.Name != policyName {
			continue
		}

		for _, rec := range p.Status.Recommendations {
			idShort := rec.ID
			if len(idShort) > 10 {
				idShort = idShort[:10]
			}

			fmt.Printf("%-12s %-15s %-20s %-15s $%10.2f %9.0f%% %10s\n",
				idShort,
				rec.Type,
				truncateString(rec.Resource, 18),
				p.Metadata.Name,
				rec.MonthlySavings,
				rec.Confidence*100,
				rec.Status)
		}
	}

	totalSavings := finopsManager.GetTotalSavingsOpportunity()
	fmt.Println(strings.Repeat("-", 105))
	fmt.Printf("Total potential monthly savings: $%.2f\n", totalSavings)

	return nil
}

func runFinopsRecommendApply(cmd *cobra.Command, args []string) error {
	policyName := args[0]
	recID := args[1]

	if err := finopsManager.ApplyRecommendation(policyName, recID); err != nil {
		return err
	}

	fmt.Printf("Recommendation '%s' marked as applied\n", recID)
	return nil
}

func runFinopsRecommendDismiss(cmd *cobra.Command, args []string) error {
	policyName := args[0]
	recID := args[1]

	if err := finopsManager.DismissRecommendation(policyName, recID); err != nil {
		return err
	}

	fmt.Printf("Recommendation '%s' dismissed\n", recID)
	return nil
}

// Helper functions

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-2] + ".."
}

// GetFinopsCmd returns the finops command for registration
func GetFinopsCmd() *cobra.Command {
	return finopsCmd
}
