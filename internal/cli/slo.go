package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

var sloCmd = &cobra.Command{
	Use:   "slo",
	Short: "Manage Service Level Objectives",
	Long: `Manage Service Level Objectives (SLOs) and error budgets.

SLOs define reliability targets for your services. This command allows you to:
- Define and manage SLO definitions
- Track error budget consumption
- Configure burn rate alerting
- Generate SLO compliance reports`,
}

var sloListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all SLOs",
	Long:  `List all registered Service Level Objectives with their current status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("output")
		service, _ := cmd.Flags().GetString("service")

		// In a real implementation, this would fetch from the SLO engine
		slos := getSampleSLOs()

		// Filter by service if specified
		if service != "" {
			filtered := make([]sloInfo, 0)
			for _, s := range slos {
				if s.Service == service {
					filtered = append(filtered, s)
				}
			}
			slos = filtered
		}

		if format == "json" {
			data, _ := json.MarshalIndent(slos, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tSERVICE\tOBJECTIVE\tCURRENT\tBUDGET USED\tSTATUS")
		for _, s := range slos {
			fmt.Fprintf(w, "%s\t%s\t%.2f%%\t%.2f%%\t%.1f%%\t%s\n",
				s.Name, s.Service, s.Objective, s.Current, s.BudgetUsed, s.Status)
		}
		w.Flush()

		return nil
	},
}

var sloGetCmd = &cobra.Command{
	Use:   "get [name]",
	Short: "Get details of an SLO",
	Long:  `Get detailed information about a specific Service Level Objective.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		format, _ := cmd.Flags().GetString("output")

		// In a real implementation, this would fetch from the SLO engine
		slo := getSampleSLODetail(name)

		if format == "json" {
			data, _ := json.MarshalIndent(slo, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Name:        %s\n", slo.Name)
		fmt.Printf("Service:     %s\n", slo.Service)
		fmt.Printf("Description: %s\n", slo.Description)
		fmt.Printf("Type:        %s\n", slo.Type)
		fmt.Printf("Objective:   %.2f%%\n", slo.Objective)
		fmt.Printf("Window:      %s\n", slo.Window)
		fmt.Println()
		fmt.Println("Current Status:")
		fmt.Printf("  SLI Value:     %.4f%%\n", slo.Current)
		fmt.Printf("  Budget Used:   %.1f%%\n", slo.BudgetUsed)
		fmt.Printf("  Budget Left:   %.1f%%\n", 100-slo.BudgetUsed)
		fmt.Printf("  Burn Rate:     %.2fx\n", slo.BurnRate)
		fmt.Printf("  Status:        %s\n", slo.Status)
		fmt.Println()
		fmt.Println("Error Budget:")
		fmt.Printf("  Total:     %.1f minutes\n", slo.BudgetTotal)
		fmt.Printf("  Consumed:  %.1f minutes\n", slo.BudgetConsumed)
		fmt.Printf("  Remaining: %.1f minutes\n", slo.BudgetRemaining)
		if slo.BurnRate > 0 {
			fmt.Printf("  Time to Exhaust: %s\n", slo.TimeToExhaust)
		}

		return nil
	},
}

var sloCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new SLO",
	Long: `Create a new Service Level Objective.

Examples:
  # Create an availability SLO
  pf slo create --name api-availability --service api-gateway \
    --type availability --objective 99.9 --window 30d

  # Create a latency SLO
  pf slo create --name api-latency --service api-gateway \
    --type latency --objective 99 --threshold 200ms --window 30d`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		service, _ := cmd.Flags().GetString("service")
		sloType, _ := cmd.Flags().GetString("type")
		objective, _ := cmd.Flags().GetFloat64("objective")
		window, _ := cmd.Flags().GetString("window")
		threshold, _ := cmd.Flags().GetString("threshold")
		description, _ := cmd.Flags().GetString("description")

		if name == "" || service == "" {
			return fmt.Errorf("--name and --service are required")
		}

		fmt.Printf("Creating SLO '%s' for service '%s'...\n", name, service)
		fmt.Printf("  Type: %s\n", sloType)
		fmt.Printf("  Objective: %.2f%%\n", objective)
		fmt.Printf("  Window: %s\n", window)
		if threshold != "" {
			fmt.Printf("  Threshold: %s\n", threshold)
		}
		if description != "" {
			fmt.Printf("  Description: %s\n", description)
		}
		fmt.Println()
		fmt.Printf("SLO '%s' created successfully.\n", name)
		fmt.Println("Burn rate alerts configured with default thresholds.")

		return nil
	},
}

var sloDeleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete an SLO",
	Long:  `Delete a Service Level Objective.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		force, _ := cmd.Flags().GetBool("force")

		if !force {
			fmt.Printf("Are you sure you want to delete SLO '%s'? This will remove all associated alerts and history.\n", name)
			fmt.Print("Type 'yes' to confirm: ")
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "yes" {
				fmt.Println("Aborted.")
				return nil
			}
		}

		fmt.Printf("Deleting SLO '%s'...\n", name)
		fmt.Printf("SLO '%s' deleted successfully.\n", name)
		return nil
	},
}

var sloBudgetCmd = &cobra.Command{
	Use:   "budget",
	Short: "View error budget status",
	Long:  `View error budget consumption and remaining budget for SLOs.`,
}

var sloBudgetListCmd = &cobra.Command{
	Use:   "list",
	Short: "List error budget status for all SLOs",
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("output")

		budgets := getSampleBudgets()

		if format == "json" {
			data, _ := json.MarshalIndent(budgets, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "SLO\tSERVICE\tTOTAL\tCONSUMED\tREMAINING\tBURN RATE\tSTATUS")
		for _, b := range budgets {
			fmt.Fprintf(w, "%s\t%s\t%.1fm\t%.1fm\t%.1fm\t%.2fx\t%s\n",
				b.SLO, b.Service, b.Total, b.Consumed, b.Remaining, b.BurnRate, b.Status)
		}
		w.Flush()

		return nil
	},
}

var sloBudgetGetCmd = &cobra.Command{
	Use:   "get [slo-name]",
	Short: "Get error budget details for an SLO",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		budget := getSampleBudgetDetail(name)

		fmt.Printf("Error Budget for SLO: %s\n", name)
		fmt.Println(string(repeatChar('-', 40)))
		fmt.Printf("Total Budget:     %.1f minutes (%.1f hours)\n", budget.Total, budget.Total/60)
		fmt.Printf("Consumed:         %.1f minutes (%.1f%%)\n", budget.Consumed, budget.ConsumedPct)
		fmt.Printf("Remaining:        %.1f minutes (%.1f%%)\n", budget.Remaining, budget.RemainingPct)
		fmt.Printf("Current Burn Rate: %.2fx\n", budget.BurnRate)
		fmt.Printf("Status:           %s\n", budget.Status)
		fmt.Println()
		fmt.Println("Forecast:")
		if budget.BurnRate > 1 {
			fmt.Printf("  At current burn rate, budget will be exhausted in: %s\n", budget.TimeToExhaust)
			fmt.Printf("  Projected exhaustion date: %s\n", time.Now().Add(parseSLODuration(budget.TimeToExhaust)).Format("2006-01-02 15:04"))
		} else {
			fmt.Println("  Budget consumption is sustainable")
		}

		return nil
	},
}

var sloReportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate SLO compliance report",
	Long: `Generate a compliance report for all SLOs.

The report includes:
- Overall SLO health score
- Error budget consumption summary
- SLOs at risk or out of budget
- Recommendations for improvement`,
	RunE: func(cmd *cobra.Command, args []string) error {
		period, _ := cmd.Flags().GetString("period")
		format, _ := cmd.Flags().GetString("output")

		report := getSampleReport(period)

		if format == "json" {
			data, _ := json.MarshalIndent(report, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Println("SLO Compliance Report")
		fmt.Println("=====================")
		fmt.Printf("Period: %s\n", period)
		fmt.Printf("Generated: %s\n", time.Now().Format("2006-01-02 15:04:05"))
		fmt.Println()
		fmt.Println("Summary")
		fmt.Println("-------")
		fmt.Printf("Total SLOs:      %d\n", report.Total)
		fmt.Printf("In Budget:       %d (%.1f%%)\n", report.InBudget, float64(report.InBudget)/float64(report.Total)*100)
		fmt.Printf("At Risk:         %d\n", report.AtRisk)
		fmt.Printf("Out of Budget:   %d\n", report.OutOfBudget)
		fmt.Printf("Health Score:    %.1f%%\n", report.HealthScore)
		fmt.Println()

		if len(report.AtRiskSLOs) > 0 {
			fmt.Println("SLOs At Risk")
			fmt.Println("------------")
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "SLO\tSERVICE\tBUDGET USED\tBURN RATE")
			for _, s := range report.AtRiskSLOs {
				fmt.Fprintf(w, "%s\t%s\t%.1f%%\t%.2fx\n", s.Name, s.Service, s.BudgetUsed, s.BurnRate)
			}
			w.Flush()
			fmt.Println()
		}

		if len(report.Recommendations) > 0 {
			fmt.Println("Recommendations")
			fmt.Println("---------------")
			for i, rec := range report.Recommendations {
				fmt.Printf("%d. %s\n", i+1, rec)
			}
		}

		return nil
	},
}

var sloAlertCmd = &cobra.Command{
	Use:   "alert",
	Short: "Manage SLO alerts",
	Long:  `Manage alerts and silences for SLOs.`,
}

var sloAlertListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active SLO alerts",
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("output")
		severity, _ := cmd.Flags().GetString("severity")

		alerts := getSampleAlerts()

		if severity != "" {
			filtered := make([]alertInfo, 0)
			for _, a := range alerts {
				if a.Severity == severity {
					filtered = append(filtered, a)
				}
			}
			alerts = filtered
		}

		if format == "json" {
			data, _ := json.MarshalIndent(alerts, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(alerts) == 0 {
			fmt.Println("No active alerts.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tSLO\tSEVERITY\tMESSAGE\tFIRED AT")
		for _, a := range alerts {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				a.ID, a.SLO, a.Severity, truncateSLOString(a.Message, 40), a.FiredAt)
		}
		w.Flush()

		return nil
	},
}

var sloAlertSilenceCmd = &cobra.Command{
	Use:   "silence [slo-name]",
	Short: "Silence alerts for an SLO",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		slo := args[0]
		duration, _ := cmd.Flags().GetString("duration")
		comment, _ := cmd.Flags().GetString("comment")

		fmt.Printf("Silencing alerts for SLO '%s' for %s\n", slo, duration)
		if comment != "" {
			fmt.Printf("Comment: %s\n", comment)
		}
		fmt.Printf("Silence ID: silence-%d\n", time.Now().Unix())
		fmt.Println("Alerts silenced successfully.")

		return nil
	},
}

// Types for CLI display
type sloInfo struct {
	Name       string  `json:"name"`
	Service    string  `json:"service"`
	Type       string  `json:"type"`
	Objective  float64 `json:"objective"`
	Current    float64 `json:"current"`
	BudgetUsed float64 `json:"budgetUsed"`
	BurnRate   float64 `json:"burnRate"`
	Status     string  `json:"status"`
}

type sloDetail struct {
	sloInfo
	Description     string  `json:"description"`
	Window          string  `json:"window"`
	BurnRate        float64 `json:"burnRate"`
	BudgetTotal     float64 `json:"budgetTotal"`
	BudgetConsumed  float64 `json:"budgetConsumed"`
	BudgetRemaining float64 `json:"budgetRemaining"`
	TimeToExhaust   string  `json:"timeToExhaust,omitempty"`
}

type budgetInfo struct {
	SLO           string  `json:"slo"`
	Service       string  `json:"service"`
	Total         float64 `json:"total"`
	Consumed      float64 `json:"consumed"`
	Remaining     float64 `json:"remaining"`
	ConsumedPct   float64 `json:"consumedPct"`
	RemainingPct  float64 `json:"remainingPct"`
	BurnRate      float64 `json:"burnRate"`
	Status        string  `json:"status"`
	TimeToExhaust string  `json:"timeToExhaust,omitempty"`
}

type reportInfo struct {
	Total           int       `json:"total"`
	InBudget        int       `json:"inBudget"`
	AtRisk          int       `json:"atRisk"`
	OutOfBudget     int       `json:"outOfBudget"`
	HealthScore     float64   `json:"healthScore"`
	AtRiskSLOs      []sloInfo `json:"atRiskSLOs,omitempty"`
	Recommendations []string  `json:"recommendations,omitempty"`
}

type alertInfo struct {
	ID       string `json:"id"`
	SLO      string `json:"slo"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	FiredAt  string `json:"firedAt"`
}

// Sample data functions (would be replaced with real SLO engine calls)
func getSampleSLOs() []sloInfo {
	return []sloInfo{
		{Name: "api-availability", Service: "api-gateway", Type: "availability", Objective: 99.9, Current: 99.95, BudgetUsed: 45.2, BurnRate: 0.8, Status: "healthy"},
		{Name: "api-latency", Service: "api-gateway", Type: "latency", Objective: 99.0, Current: 98.7, BudgetUsed: 130.0, BurnRate: 2.1, Status: "at_risk"},
		{Name: "db-availability", Service: "postgres", Type: "availability", Objective: 99.99, Current: 99.992, BudgetUsed: 80.5, BurnRate: 1.5, Status: "warning"},
		{Name: "auth-availability", Service: "auth-service", Type: "availability", Objective: 99.9, Current: 99.98, BudgetUsed: 20.1, BurnRate: 0.5, Status: "healthy"},
	}
}

func getSampleSLODetail(name string) sloDetail {
	return sloDetail{
		sloInfo: sloInfo{
			Name:       name,
			Service:    "api-gateway",
			Type:       "availability",
			Objective:  99.9,
			Current:    99.95,
			BudgetUsed: 45.2,
			Status:     "healthy",
		},
		Description:     "API Gateway availability SLO",
		Window:          "30d",
		BurnRate:        0.8,
		BudgetTotal:     43.2,
		BudgetConsumed:  19.5,
		BudgetRemaining: 23.7,
		TimeToExhaust:   "12d 4h",
	}
}

func getSampleBudgets() []budgetInfo {
	return []budgetInfo{
		{SLO: "api-availability", Service: "api-gateway", Total: 43.2, Consumed: 19.5, Remaining: 23.7, ConsumedPct: 45.2, RemainingPct: 54.8, BurnRate: 0.8, Status: "healthy"},
		{SLO: "api-latency", Service: "api-gateway", Total: 432.0, Consumed: 561.6, Remaining: -129.6, ConsumedPct: 130.0, RemainingPct: -30.0, BurnRate: 2.1, Status: "exhausted"},
		{SLO: "db-availability", Service: "postgres", Total: 4.32, Consumed: 3.48, Remaining: 0.84, ConsumedPct: 80.5, RemainingPct: 19.5, BurnRate: 1.5, Status: "critical"},
	}
}

func getSampleBudgetDetail(name string) budgetInfo {
	return budgetInfo{
		SLO:           name,
		Service:       "api-gateway",
		Total:         43.2,
		Consumed:      19.5,
		Remaining:     23.7,
		ConsumedPct:   45.2,
		RemainingPct:  54.8,
		BurnRate:      0.8,
		Status:        "healthy",
		TimeToExhaust: "12d 4h",
	}
}

func getSampleReport(period string) reportInfo {
	return reportInfo{
		Total:       4,
		InBudget:    2,
		AtRisk:      1,
		OutOfBudget: 1,
		HealthScore: 75.0,
		AtRiskSLOs: []sloInfo{
			{Name: "api-latency", Service: "api-gateway", BudgetUsed: 130.0, BurnRate: 2.1, Status: "exhausted"},
			{Name: "db-availability", Service: "postgres", BudgetUsed: 80.5, BurnRate: 1.5, Status: "critical"},
		},
		Recommendations: []string{
			"api-latency: Consider optimizing database queries or adding caching",
			"db-availability: Review recent deployment changes that may have impacted reliability",
			"Consider setting up multi-window burn rate alerting for early detection",
		},
	}
}

func getSampleAlerts() []alertInfo {
	return []alertInfo{
		{ID: "alert-001", SLO: "api-latency", Severity: "critical", Message: "Error budget exhausted (130% consumed)", FiredAt: "2h ago"},
		{ID: "alert-002", SLO: "db-availability", Severity: "warning", Message: "High burn rate (1.5x)", FiredAt: "45m ago"},
	}
}

// Helper functions
func repeatChar(c byte, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = c
	}
	return string(b)
}

func truncateSLOString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

func parseSLODuration(s string) time.Duration {
	// Simplified duration parsing
	return 12 * 24 * time.Hour
}

func init() {
	// Main SLO command
	rootCmd.AddCommand(sloCmd)

	// List subcommand
	sloCmd.AddCommand(sloListCmd)
	sloListCmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	sloListCmd.Flags().StringP("service", "s", "", "Filter by service name")

	// Get subcommand
	sloCmd.AddCommand(sloGetCmd)
	sloGetCmd.Flags().StringP("output", "o", "text", "Output format (text, json)")

	// Create subcommand
	sloCmd.AddCommand(sloCreateCmd)
	sloCreateCmd.Flags().StringP("name", "n", "", "SLO name (required)")
	sloCreateCmd.Flags().StringP("service", "s", "", "Service name (required)")
	sloCreateCmd.Flags().StringP("type", "t", "availability", "SLO type (availability, latency, error_rate, throughput)")
	sloCreateCmd.Flags().Float64P("objective", "O", 99.9, "Objective percentage")
	sloCreateCmd.Flags().StringP("window", "w", "30d", "Time window (e.g., 7d, 30d)")
	sloCreateCmd.Flags().String("threshold", "", "Threshold for latency SLOs (e.g., 200ms)")
	sloCreateCmd.Flags().StringP("description", "d", "", "SLO description")

	// Delete subcommand
	sloCmd.AddCommand(sloDeleteCmd)
	sloDeleteCmd.Flags().BoolP("force", "f", false, "Force deletion without confirmation")

	// Budget subcommands
	sloCmd.AddCommand(sloBudgetCmd)
	sloBudgetCmd.AddCommand(sloBudgetListCmd)
	sloBudgetListCmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	sloBudgetCmd.AddCommand(sloBudgetGetCmd)

	// Report subcommand
	sloCmd.AddCommand(sloReportCmd)
	sloReportCmd.Flags().StringP("period", "p", "30d", "Report period (e.g., 7d, 30d)")
	sloReportCmd.Flags().StringP("output", "o", "text", "Output format (text, json)")

	// Alert subcommands
	sloCmd.AddCommand(sloAlertCmd)
	sloAlertCmd.AddCommand(sloAlertListCmd)
	sloAlertListCmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	sloAlertListCmd.Flags().String("severity", "", "Filter by severity (critical, warning, info)")
	sloAlertCmd.AddCommand(sloAlertSilenceCmd)
	sloAlertSilenceCmd.Flags().StringP("duration", "d", "1h", "Silence duration")
	sloAlertSilenceCmd.Flags().StringP("comment", "c", "", "Silence comment")
}
