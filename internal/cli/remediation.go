package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/platformfoundry/platformfoundry-ce/internal/remediation"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(remediationCmd)
	remediationCmd.AddCommand(remediationStatusCmd)
	remediationCmd.AddCommand(remediationRunCmd)
	remediationCmd.AddCommand(remediationHistoryCmd)
	remediationCmd.AddCommand(remediationRulesCmd)
	remediationCmd.AddCommand(remediationEnableCmd)
	remediationCmd.AddCommand(remediationDisableCmd)

	remediationRunCmd.Flags().Bool("dry-run", false, "Show what would be done without executing")
	remediationHistoryCmd.Flags().IntP("limit", "l", 10, "Number of results to show")
	remediationHistoryCmd.Flags().StringP("output", "o", "table", "Output format: table, json")
}

var remediationCmd = &cobra.Command{
	Use:     "remediation",
	Aliases: []string{"rem", "remediate"},
	Short:   "Manage auto-remediation",
	Long: `Manage automatic remediation of drift and policy violations.

Auto-remediation can automatically fix issues based on configurable rules,
or alert operators when manual intervention is required.`,
}

var remediationStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show remediation engine status",
	RunE:  runRemediationStatus,
}

var remediationRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run remediation check",
	Long:  `Run a remediation check to detect and optionally fix issues.`,
	RunE:  runRemediationRun,
}

var remediationHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Show remediation history",
	RunE:  runRemediationHistory,
}

var remediationRulesCmd = &cobra.Command{
	Use:   "rules",
	Short: "List remediation rules",
	RunE:  runRemediationRules,
}

var remediationEnableCmd = &cobra.Command{
	Use:   "enable [rule-name]",
	Short: "Enable a remediation rule",
	Args:  cobra.ExactArgs(1),
	RunE:  runRemediationEnable,
}

var remediationDisableCmd = &cobra.Command{
	Use:   "disable [rule-name]",
	Short: "Disable a remediation rule",
	Args:  cobra.ExactArgs(1),
	RunE:  runRemediationDisable,
}

func runRemediationStatus(cmd *cobra.Command, args []string) error {
	engine := remediation.NewEngine(nil)
	status := engine.GetStatus()

	fmt.Println("\nRemediation Engine Status")
	fmt.Println(strings.Repeat("=", 40))
	fmt.Printf("Enabled:       %v\n", status["enabled"])
	fmt.Printf("Rules Count:   %v\n", status["rules_count"])
	fmt.Printf("History Count: %v\n", status["history_count"])

	return nil
}

func runRemediationRun(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	engine := remediation.NewEngine(nil)
	ctx := context.Background()

	// Simulate some issues for demonstration
	issues := []remediation.Issue{
		{
			ID:          "drift-001",
			Type:        remediation.TriggerDrift,
			Severity:    remediation.SeverityLow,
			Resource:    "argocd",
			Environment: "staging",
			Description: "ArgoCD replicas drifted from 2 to 1",
		},
		{
			ID:          "drift-002",
			Type:        remediation.TriggerDrift,
			Severity:    remediation.SeverityMedium,
			Resource:    "prometheus",
			Environment: "production",
			Description: "Prometheus retention changed from 30d to 7d",
		},
	}

	fmt.Println("\nRemediation Check")
	fmt.Println(strings.Repeat("=", 60))

	if dryRun {
		fmt.Println("DRY RUN - No actions will be executed")
	}

	for _, issue := range issues {
		fmt.Printf("\nProcessing: %s\n", issue.ID)
		fmt.Printf("  Type:        %s\n", issue.Type)
		fmt.Printf("  Severity:    %s\n", issue.Severity)
		fmt.Printf("  Resource:    %s\n", issue.Resource)
		fmt.Printf("  Environment: %s\n", issue.Environment)
		fmt.Printf("  Description: %s\n", issue.Description)

		if !dryRun {
			result, err := engine.ProcessIssue(ctx, issue)
			if err != nil {
				fmt.Printf("  Result:      FAILED - %s\n", err)
			} else {
				fmt.Printf("  Action:      %s\n", result.Action)
				fmt.Printf("  Rule:        %s\n", result.Rule)
				fmt.Printf("  Result:      %s\n", result.Message)
			}
		} else {
			// Find matching rule for dry run
			fmt.Printf("  Would match: [simulated rule]\n")
			fmt.Printf("  Would execute: alert_only\n")
		}
	}

	fmt.Printf("\nProcessed %d issues\n", len(issues))

	return nil
}

func runRemediationHistory(cmd *cobra.Command, args []string) error {
	limit, _ := cmd.Flags().GetInt("limit")
	output, _ := cmd.Flags().GetString("output")

	engine := remediation.NewEngine(nil)
	history := engine.GetHistory()

	// Limit results
	if len(history) > limit {
		history = history[len(history)-limit:]
	}

	if output == "json" {
		data, _ := json.MarshalIndent(history, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	if len(history) == 0 {
		fmt.Println("\nNo remediation history found")
		return nil
	}

	fmt.Println("\nRemediation History")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("%-20s %-15s %-15s %-10s %-15s\n", "ISSUE", "RESOURCE", "ACTION", "SUCCESS", "EXECUTED")
	fmt.Println(strings.Repeat("-", 80))

	for _, h := range history {
		success := "Yes"
		if !h.Success {
			success = "No"
		}
		fmt.Printf("%-20s %-15s %-15s %-10s %-15s\n",
			truncateStr(h.Issue.ID, 20),
			truncateStr(h.Issue.Resource, 15),
			h.Action,
			success,
			h.ExecutedAt.Format("15:04:05"),
		)
	}

	return nil
}

func runRemediationRules(cmd *cobra.Command, args []string) error {
	config := remediation.DefaultConfig()

	fmt.Println("\nRemediation Rules")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("%-25s %-12s %-15s %-15s %-10s\n", "NAME", "TRIGGER", "SEVERITY", "ACTION", "ENABLED")
	fmt.Println(strings.Repeat("-", 80))

	for _, rule := range config.Rules {
		severities := "all"
		if len(rule.Trigger.Severity) > 0 {
			sevs := make([]string, len(rule.Trigger.Severity))
			for i, s := range rule.Trigger.Severity {
				sevs[i] = string(s)
			}
			severities = strings.Join(sevs, ",")
		}

		enabled := "Yes"
		if !rule.Enabled {
			enabled = "No"
		}

		fmt.Printf("%-25s %-12s %-15s %-15s %-10s\n",
			truncateStr(rule.Name, 25),
			rule.Trigger.Type,
			truncateStr(severities, 15),
			rule.Action.Type,
			enabled,
		)
	}

	return nil
}

func runRemediationEnable(cmd *cobra.Command, args []string) error {
	ruleName := args[0]
	engine := remediation.NewEngine(nil)

	if engine.EnableRule(ruleName) {
		fmt.Printf("Rule '%s' enabled\n", ruleName)
	} else {
		fmt.Printf("Rule '%s' not found\n", ruleName)
	}

	return nil
}

func runRemediationDisable(cmd *cobra.Command, args []string) error {
	ruleName := args[0]
	engine := remediation.NewEngine(nil)

	if engine.DisableRule(ruleName) {
		fmt.Printf("Rule '%s' disabled\n", ruleName)
	} else {
		fmt.Printf("Rule '%s' not found\n", ruleName)
	}

	return nil
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
