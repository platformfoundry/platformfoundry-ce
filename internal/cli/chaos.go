package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/platformfoundry/pf-ce/internal/chaos"
	"github.com/platformfoundry/pf-ce/pkg/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var chaosEngine *chaos.Engine

func init() {
	rootCmd.AddCommand(chaosCmd)

	chaosCmd.AddCommand(chaosRunCmd)
	chaosCmd.AddCommand(chaosListCmd)
	chaosCmd.AddCommand(chaosStatusCmd)
	chaosCmd.AddCommand(chaosStopCmd)
	chaosCmd.AddCommand(chaosReportCmd)

	chaosRunCmd.Flags().StringP("file", "f", "", "Path to experiment definition file")
	chaosRunCmd.Flags().Bool("dry-run", false, "Validate without executing")
	chaosRunCmd.Flags().StringP("environment", "e", "", "Target environment")

	chaosListCmd.Flags().String("status", "", "Filter by status (running, completed, failed)")

	chaosReportCmd.Flags().StringP("output", "o", "table", "Output format (table, json, yaml)")
}

var chaosCmd = &cobra.Command{
	Use:   "chaos",
	Short: "Chaos engineering experiments",
	Long: `Run chaos engineering experiments to test system resilience.

Chaos experiments help identify weaknesses in your infrastructure by
intentionally introducing failures and observing system behavior.

Supported experiment types:
  - Pod failures (kill, failure)
  - Network chaos (delay, loss, partition)
  - Resource stress (CPU, memory, IO)
  - Service failures (unavailability, HTTP errors)`,
	Example: `  # Run an experiment from a file
  pf chaos run -f experiment.yaml

  # List all experiments
  pf chaos list

  # Check experiment status
  pf chaos status my-experiment

  # Stop a running experiment
  pf chaos stop my-experiment`,
}

var chaosRunCmd = &cobra.Command{
	Use:   "run [experiment-name]",
	Short: "Run a chaos experiment",
	Long:  `Run a chaos engineering experiment by name or from a definition file.`,
	RunE:  runChaosExperiment,
}

var chaosListCmd = &cobra.Command{
	Use:   "list",
	Short: "List chaos experiments",
	RunE:  listChaosExperiments,
}

var chaosStatusCmd = &cobra.Command{
	Use:   "status [experiment-name]",
	Short: "Show experiment status",
	Args:  cobra.ExactArgs(1),
	RunE:  chaosStatus,
}

var chaosStopCmd = &cobra.Command{
	Use:   "stop [experiment-name]",
	Short: "Stop a running experiment",
	Args:  cobra.ExactArgs(1),
	RunE:  stopChaosExperiment,
}

var chaosReportCmd = &cobra.Command{
	Use:   "report [experiment-name]",
	Short: "Show experiment report",
	Args:  cobra.ExactArgs(1),
	RunE:  chaosReport,
}

func initChaosEngine() {
	if chaosEngine == nil {
		chaosEngine = chaos.NewEngine(chaos.EngineConfig{
			Executor:      &chaos.MockExecutor{},
			HealthChecker: &chaos.MockHealthChecker{HealthyByDefault: true},
		})

		// Register sample experiments
		registerSampleExperiments()
	}
}

func registerSampleExperiments() {
	// Pod failure experiment
	chaosEngine.RegisterExperiment(&types.ChaosExperiment{
		APIVersion: "platformfoundry.io/v1",
		Kind:       "ChaosExperiment",
		Metadata: types.ChaosMetadata{
			Name: "pod-failure-test",
			Labels: map[string]string{
				"category": "resilience",
			},
		},
		Spec: types.ChaosExperimentSpec{
			Target: types.ChaosTarget{
				Service:     "order-service",
				Environment: "staging",
			},
			Experiments: []types.ChaosAction{
				{
					Name:        "kill-pods",
					Type:        types.ChaosActionPodKill,
					Duration:    "30s",
					Probability: 0.5,
				},
			},
			Safety: types.ChaosSafetyRules{
				MaxImpact:           "30%",
				RollbackOnError:     true,
				HealthCheckInterval: "10s",
			},
		},
	})

	// Network latency experiment
	chaosEngine.RegisterExperiment(&types.ChaosExperiment{
		APIVersion: "platformfoundry.io/v1",
		Kind:       "ChaosExperiment",
		Metadata: types.ChaosMetadata{
			Name: "network-latency-test",
			Labels: map[string]string{
				"category": "network",
			},
		},
		Spec: types.ChaosExperimentSpec{
			Target: types.ChaosTarget{
				Service:     "api-gateway",
				Environment: "staging",
			},
			Experiments: []types.ChaosAction{
				{
					Name:     "inject-latency",
					Type:     types.ChaosActionNetworkDelay,
					Duration: "1m",
					Parameters: map[string]interface{}{
						"latency": "200ms",
						"jitter":  "50ms",
					},
				},
			},
			Safety: types.ChaosSafetyRules{
				MaxImpact:           "50%",
				RollbackOnError:     true,
				HealthCheckInterval: "15s",
			},
		},
	})

	// Resource stress experiment
	chaosEngine.RegisterExperiment(&types.ChaosExperiment{
		APIVersion: "platformfoundry.io/v1",
		Kind:       "ChaosExperiment",
		Metadata: types.ChaosMetadata{
			Name: "memory-stress-test",
			Labels: map[string]string{
				"category": "resource",
			},
		},
		Spec: types.ChaosExperimentSpec{
			Target: types.ChaosTarget{
				Service:     "payment-service",
				Environment: "staging",
			},
			Experiments: []types.ChaosAction{
				{
					Name:     "memory-stress",
					Type:     types.ChaosActionMemoryStress,
					Duration: "2m",
					Parameters: map[string]interface{}{
						"workers":    2,
						"percentage": 80,
					},
				},
			},
			Safety: types.ChaosSafetyRules{
				MaxImpact:           "20%",
				RollbackOnError:     true,
				HealthCheckInterval: "30s",
				StopOnFailure:       true,
			},
		},
	})
}

func runChaosExperiment(cmd *cobra.Command, args []string) error {
	initChaosEngine()

	filePath, _ := cmd.Flags().GetString("file")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	var experiment *types.ChaosExperiment

	if filePath != "" {
		// Load from file
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		experiment = &types.ChaosExperiment{}
		if err := yaml.Unmarshal(data, experiment); err != nil {
			return fmt.Errorf("failed to parse experiment: %w", err)
		}

		if err := chaosEngine.RegisterExperiment(experiment); err != nil {
			return fmt.Errorf("failed to register experiment: %w", err)
		}
	} else if len(args) > 0 {
		// Get by name
		var err error
		experiment, err = chaosEngine.GetExperiment(args[0])
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("either --file or experiment name is required")
	}

	if dryRun {
		fmt.Println("Dry run mode - experiment validation only")
		fmt.Println()
		printExperimentDetails(experiment)
		return nil
	}

	fmt.Printf("Running experiment: %s\n", experiment.Metadata.Name)
	fmt.Printf("Target: %s (%s)\n", experiment.Spec.Target.Service, experiment.Spec.Target.Environment)
	fmt.Printf("Actions: %d\n", len(experiment.Spec.Experiments))
	fmt.Println(strings.Repeat("-", 60))

	// Run experiment
	ctx := context.Background()
	startTime := time.Now()

	report, err := chaosEngine.RunExperiment(ctx, experiment.Metadata.Name)
	if err != nil {
		return fmt.Errorf("experiment failed: %w", err)
	}

	elapsed := time.Since(startTime)

	// Print results
	fmt.Println()
	printChaosReport(report, elapsed)

	return nil
}

func listChaosExperiments(cmd *cobra.Command, args []string) error {
	initChaosEngine()

	statusFilter, _ := cmd.Flags().GetString("status")

	experiments := chaosEngine.ListExperiments()
	if len(experiments) == 0 {
		fmt.Println("No chaos experiments registered")
		return nil
	}

	fmt.Println("Chaos Experiments")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("%-25s %-15s %-15s %-15s\n", "NAME", "TARGET", "ENVIRONMENT", "STATUS")
	fmt.Println(strings.Repeat("-", 70))

	for _, exp := range experiments {
		status := "Created"
		if exp.Status != nil {
			status = string(exp.Status.Phase)
		}

		if statusFilter != "" && strings.ToLower(status) != strings.ToLower(statusFilter) {
			continue
		}

		fmt.Printf("%-25s %-15s %-15s %-15s\n",
			truncateChaosString(exp.Metadata.Name, 25),
			truncateChaosString(exp.Spec.Target.Service, 15),
			truncateChaosString(exp.Spec.Target.Environment, 15),
			status,
		)
	}

	// Show active runs
	activeRuns := chaosEngine.GetActiveRuns()
	if len(activeRuns) > 0 {
		fmt.Println()
		fmt.Printf("Active Runs: %d\n", len(activeRuns))
		for name, run := range activeRuns {
			fmt.Printf("  • %s (started %s)\n", name, run.StartTime.Format(time.RFC3339))
		}
	}

	return nil
}

func chaosStatus(cmd *cobra.Command, args []string) error {
	initChaosEngine()

	exp, err := chaosEngine.GetExperiment(args[0])
	if err != nil {
		return err
	}

	printExperimentDetails(exp)
	return nil
}

func stopChaosExperiment(cmd *cobra.Command, args []string) error {
	initChaosEngine()

	if err := chaosEngine.StopExperiment(args[0]); err != nil {
		return err
	}

	fmt.Printf("Experiment %s stopped\n", args[0])
	return nil
}

func chaosReport(cmd *cobra.Command, args []string) error {
	initChaosEngine()

	exp, err := chaosEngine.GetExperiment(args[0])
	if err != nil {
		return err
	}

	if exp.Status == nil || len(exp.Status.History) == 0 {
		fmt.Println("No run history available for this experiment")
		return nil
	}

	// Get last run
	lastRun := exp.Status.History[len(exp.Status.History)-1]

	fmt.Println("Last Run Report")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Run ID:    %s\n", lastRun.RunID)
	fmt.Printf("Result:    %s\n", lastRun.Result)
	fmt.Printf("Started:   %s\n", lastRun.StartTime.Format(time.RFC3339))
	if lastRun.EndTime != nil {
		fmt.Printf("Ended:     %s\n", lastRun.EndTime.Format(time.RFC3339))
		fmt.Printf("Duration:  %s\n", lastRun.EndTime.Sub(lastRun.StartTime))
	}

	if len(lastRun.Actions) > 0 {
		fmt.Println("\nActions:")
		for _, action := range lastRun.Actions {
			icon := "✓"
			if action.Result != "success" {
				icon = "✗"
			}
			fmt.Printf("  %s %s (%s) - %s\n", icon, action.Name, action.Type, action.Result)
		}
	}

	return nil
}

func printExperimentDetails(exp *types.ChaosExperiment) {
	fmt.Println("Experiment Details")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Name:        %s\n", exp.Metadata.Name)
	fmt.Printf("Environment: %s\n", exp.Spec.Target.Environment)
	fmt.Printf("Service:     %s\n", exp.Spec.Target.Service)

	if exp.Status != nil {
		fmt.Printf("\nStatus:      %s\n", exp.Status.Phase)
		if exp.Status.LastRunTime != nil {
			fmt.Printf("Last Run:    %s\n", exp.Status.LastRunTime.Format(time.RFC3339))
		}
		fmt.Printf("Successful:  %d\n", exp.Status.SuccessfulRuns)
		fmt.Printf("Failed:      %d\n", exp.Status.FailedRuns)
	}

	fmt.Println("\nActions:")
	for _, action := range exp.Spec.Experiments {
		fmt.Printf("  • %s (%s) - %s\n", action.Name, action.Type, action.Duration)
	}

	fmt.Println("\nSafety Rules:")
	fmt.Printf("  Max Impact:       %s\n", exp.Spec.Safety.MaxImpact)
	fmt.Printf("  Rollback on Error: %v\n", exp.Spec.Safety.RollbackOnError)
	fmt.Printf("  Health Interval:   %s\n", exp.Spec.Safety.HealthCheckInterval)
}

func printChaosReport(report *types.ChaosReport, elapsed time.Duration) {
	// Result header
	resultIcon := "✓"
	if report.OverallResult == "failed" {
		resultIcon = "✗"
	} else if report.OverallResult == "partial" {
		resultIcon = "⚠"
	}

	fmt.Printf("%s Experiment Result: %s\n", resultIcon, strings.ToUpper(report.OverallResult))
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("Duration:    %s\n", elapsed.Round(time.Millisecond))
	fmt.Printf("Actions:     %d total, %d successful, %d failed\n",
		report.TotalActions, report.SuccessfulActions, report.FailedActions)

	// Findings
	if len(report.Findings) > 0 {
		fmt.Println("\nFindings:")
		for i, finding := range report.Findings {
			icon := getSeverityIcon(finding.Severity)
			fmt.Printf("%d. %s [%s] %s\n", i+1, icon, finding.Severity, finding.Description)
			fmt.Printf("   Component: %s\n", finding.Component)
			fmt.Printf("   Impact: %s\n", finding.Impact)
			if finding.Remediation != "" {
				fmt.Printf("   Remediation: %s\n", finding.Remediation)
			}
		}
	} else {
		fmt.Println("\nNo significant findings during this experiment.")
	}

	// Recommendations
	if len(report.Recommendations) > 0 {
		fmt.Println("\nRecommendations:")
		for i, rec := range report.Recommendations {
			fmt.Printf("  %d. %s\n", i+1, rec)
		}
	}
}

func getSeverityIcon(severity string) string {
	switch severity {
	case "critical":
		return "🔴"
	case "high":
		return "🟠"
	case "medium":
		return "🟡"
	case "low":
		return "🟢"
	default:
		return "•"
	}
}

// truncateString truncates a string to max length with ellipsis (chaos-specific)
func truncateChaosString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
