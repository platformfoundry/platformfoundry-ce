package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/platformfoundry/pf-ce/internal/scaling"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var scalingEngine *scaling.Engine
var scalingRecorder *scaling.InMemoryRecorder

func init() {
	// Initialize recorder
	scalingRecorder = scaling.NewInMemoryRecorder(1000)
}

var scalingCmd = &cobra.Command{
	Use:     "scaling",
	Aliases: []string{"scale", "autoscale"},
	Short:   "Manage predictive scaling policies",
	Long:    `Manage predictive auto-scaling policies with ML-based predictions, cost-aware scaling, and traffic pattern learning.`,
}

var scalingApplyCmd = &cobra.Command{
	Use:   "apply -f <file>",
	Short: "Apply a scaling policy from file",
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath, _ := cmd.Flags().GetString("file")
		if filePath == "" {
			return fmt.Errorf("file path is required (-f)")
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		var policy scaling.ScalingPolicy
		if strings.HasSuffix(filePath, ".json") {
			err = json.Unmarshal(data, &policy)
		} else {
			err = yaml.Unmarshal(data, &policy)
		}
		if err != nil {
			return fmt.Errorf("failed to parse policy: %w", err)
		}

		// Set defaults
		if policy.APIVersion == "" {
			policy.APIVersion = "platformfoundry.io/v1"
		}
		if policy.Kind == "" {
			policy.Kind = "ScalingPolicy"
		}

		// Initialize engine if needed
		if scalingEngine == nil {
			scalingEngine = scaling.NewEngine(nil, nil, nil, nil)
			scalingEngine.WithEventRecorder(scalingRecorder)
		}

		if err := scalingEngine.RegisterPolicy(&policy); err != nil {
			return fmt.Errorf("failed to register policy: %w", err)
		}

		fmt.Printf("Scaling policy '%s' applied successfully\n", policy.Metadata.Name)
		return nil
	},
}

var scalingListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all scaling policies",
	RunE: func(cmd *cobra.Command, args []string) error {
		if scalingEngine == nil {
			fmt.Println("No scaling policies registered")
			return nil
		}

		policies := scalingEngine.ListPolicies()
		if len(policies) == 0 {
			fmt.Println("No scaling policies found")
			return nil
		}

		outputFormat, _ := cmd.Flags().GetString("output")

		if outputFormat == "json" {
			data, _ := json.MarshalIndent(policies, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("%-30s %-15s %-12s %-10s %-10s %s\n",
			"NAME", "STRATEGY", "CURRENT", "DESIRED", "MIN/MAX", "LAST SCALE")

		for _, p := range policies {
			current := 0
			desired := 0
			lastScale := "Never"

			if p.Status != nil {
				current = p.Status.CurrentReplicas
				desired = p.Status.DesiredReplicas
				if p.Status.LastScaleTime != nil {
					lastScale = p.Status.LastScaleTime.Format(time.RFC3339)
				}
			}

			minMax := fmt.Sprintf("%d/%d", p.Spec.Constraints.MinReplicas, p.Spec.Constraints.MaxReplicas)

			fmt.Printf("%-30s %-15s %-12d %-10d %-10s %s\n",
				p.Metadata.Name,
				p.Spec.Strategy,
				current,
				desired,
				minMax,
				lastScale,
			)
		}

		return nil
	},
}

var scalingGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Get details of a scaling policy",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		if scalingEngine == nil {
			return fmt.Errorf("no scaling engine initialized")
		}

		policy, ok := scalingEngine.GetPolicy(name)
		if !ok {
			return fmt.Errorf("policy '%s' not found", name)
		}

		outputFormat, _ := cmd.Flags().GetString("output")

		if outputFormat == "json" {
			data, _ := json.MarshalIndent(policy, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if outputFormat == "yaml" {
			data, _ := yaml.Marshal(policy)
			fmt.Println(string(data))
			return nil
		}

		// Human-readable output
		fmt.Printf("Name:     %s\n", policy.Metadata.Name)
		fmt.Printf("Strategy: %s\n", policy.Spec.Strategy)
		fmt.Printf("Target:   %s/%s\n", policy.Spec.Target.Kind, policy.Spec.Target.Name)
		fmt.Printf("Min/Max:  %d/%d\n", policy.Spec.Constraints.MinReplicas, policy.Spec.Constraints.MaxReplicas)

		if policy.Status != nil {
			fmt.Printf("\nStatus:\n")
			fmt.Printf("  Current Replicas: %d\n", policy.Status.CurrentReplicas)
			fmt.Printf("  Desired Replicas: %d\n", policy.Status.DesiredReplicas)

			if policy.Status.LastScaleTime != nil {
				fmt.Printf("  Last Scale Time:  %s\n", policy.Status.LastScaleTime.Format(time.RFC3339))
			}

			if policy.Status.LastPrediction != nil {
				fmt.Printf("\nPrediction:\n")
				fmt.Printf("  Predicted Load:   %.2f\n", policy.Status.LastPrediction.PredictedLoad)
				fmt.Printf("  Confidence:       %.1f%%\n", policy.Status.LastPrediction.ConfidenceLevel*100)
				fmt.Printf("  Horizon:          %s\n", policy.Status.LastPrediction.Horizon)
			}

			if policy.Status.CostEstimate != nil {
				fmt.Printf("\nCost Estimate:\n")
				fmt.Printf("  Hourly:  $%.2f\n", policy.Status.CostEstimate.CurrentHourlyCost)
				fmt.Printf("  Daily:   $%.2f\n", policy.Status.CostEstimate.DailyEstimate)
				fmt.Printf("  Monthly: $%.2f\n", policy.Status.CostEstimate.MonthlyEstimate)
			}

			if len(policy.Status.Conditions) > 0 {
				fmt.Printf("\nConditions:\n")
				for _, c := range policy.Status.Conditions {
					fmt.Printf("  %s: %s (%s)\n", c.Type, c.Status, c.Reason)
				}
			}
		}

		return nil
	},
}

var scalingDeleteCmd = &cobra.Command{
	Use:     "delete <name>",
	Aliases: []string{"remove", "rm"},
	Short:   "Delete a scaling policy",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		if scalingEngine == nil {
			return fmt.Errorf("no scaling engine initialized")
		}

		scalingEngine.UnregisterPolicy(name)
		fmt.Printf("Scaling policy '%s' deleted\n", name)
		return nil
	},
}

var scalingHistoryCmd = &cobra.Command{
	Use:   "history [name]",
	Short: "Show scaling event history",
	RunE: func(cmd *cobra.Command, args []string) error {
		limit, _ := cmd.Flags().GetInt("limit")
		outputFormat, _ := cmd.Flags().GetString("output")

		var events []scaling.ScalingEvent

		if len(args) > 0 {
			events = scalingRecorder.GetHistory(args[0], limit)
		} else {
			events = scalingRecorder.GetAllHistory(limit)
		}

		if len(events) == 0 {
			fmt.Println("No scaling events found")
			return nil
		}

		if outputFormat == "json" {
			data, _ := json.MarshalIndent(events, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("%-25s %-20s %-10s %-8s %-8s %-10s %s\n",
			"TIMESTAMP", "POLICY", "DIRECTION", "FROM", "TO", "SUCCESS", "REASON")

		for _, e := range events {
			success := "Yes"
			if !e.Success {
				success = "No"
			}

			fmt.Printf("%-25s %-20s %-10s %-8d %-8d %-10s %s\n",
				e.Timestamp.Format("2006-01-02 15:04:05"),
				truncate(e.PolicyName, 20),
				e.Direction,
				e.FromReplicas,
				e.ToReplicas,
				success,
				truncate(e.Reason, 40),
			)
		}

		return nil
	},
}

var scalingPredictCmd = &cobra.Command{
	Use:   "predict <policy-name>",
	Short: "Show predictions for a scaling policy",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		horizon, _ := cmd.Flags().GetString("horizon")
		outputFormat, _ := cmd.Flags().GetString("output")

		if scalingEngine == nil {
			return fmt.Errorf("no scaling engine initialized")
		}

		policy, ok := scalingEngine.GetPolicy(name)
		if !ok {
			return fmt.Errorf("policy '%s' not found", name)
		}

		if policy.Status == nil || policy.Status.LastPrediction == nil {
			return fmt.Errorf("no prediction available for policy '%s'", name)
		}

		pred := policy.Status.LastPrediction

		if outputFormat == "json" {
			data, _ := json.MarshalIndent(pred, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Prediction for: %s\n", name)
		fmt.Printf("Horizon:        %s\n", horizon)
		fmt.Printf("Predicted Load: %.2f\n", pred.PredictedLoad)
		fmt.Printf("Confidence:     %.1f%%\n", pred.ConfidenceLevel*100)
		fmt.Printf("Recommended:    %d replicas\n", pred.RecommendedPods)

		if pred.ModelMetrics != nil {
			fmt.Printf("\nModel Metrics:\n")
			fmt.Printf("  MAE:          %.4f\n", pred.ModelMetrics.MAE)
			fmt.Printf("  RMSE:         %.4f\n", pred.ModelMetrics.RMSE)
			fmt.Printf("  MAPE:         %.2f%%\n", pred.ModelMetrics.MAPE)
			fmt.Printf("  Training Age: %s\n", pred.ModelMetrics.TrainingAge)
		}

		if len(pred.Forecast) > 0 {
			fmt.Printf("\nForecast:\n")
			fmt.Printf("%-25s %-12s %-12s %-12s\n", "TIME", "VALUE", "LOWER", "UPPER")
			for _, f := range pred.Forecast {
				if len(pred.Forecast) > 10 {
					// Show only first and last few for large forecasts
					break
				}
				fmt.Printf("%-25s %-12.2f %-12.2f %-12.2f\n",
					f.Timestamp.Format("2006-01-02 15:04"),
					f.Value,
					f.LowerBound,
					f.UpperBound,
				)
			}
		}

		return nil
	},
}

var scalingPatternCmd = &cobra.Command{
	Use:   "pattern <policy-name>",
	Short: "Show learned traffic patterns",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		outputFormat, _ := cmd.Flags().GetString("output")

		// For now, just show that patterns would be displayed
		fmt.Printf("Traffic patterns for: %s\n\n", name)

		if outputFormat == "json" {
			fmt.Println(`{"message": "Pattern data would be shown here"}`)
			return nil
		}

		fmt.Println("Day       | Peak Hours    | Trough Hours  | Avg Load")
		fmt.Println("----------|---------------|---------------|----------")
		days := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
		for _, day := range days {
			fmt.Printf("%-9s | 09:00-11:00   | 02:00-05:00   | --\n", day)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(scalingCmd)

	// Apply command
	scalingApplyCmd.Flags().StringP("file", "f", "", "Path to scaling policy file")
	scalingCmd.AddCommand(scalingApplyCmd)

	// List command
	scalingListCmd.Flags().StringP("output", "o", "", "Output format (json)")
	scalingCmd.AddCommand(scalingListCmd)

	// Get command
	scalingGetCmd.Flags().StringP("output", "o", "", "Output format (json, yaml)")
	scalingCmd.AddCommand(scalingGetCmd)

	// Delete command
	scalingCmd.AddCommand(scalingDeleteCmd)

	// History command
	scalingHistoryCmd.Flags().IntP("limit", "l", 20, "Maximum number of events to show")
	scalingHistoryCmd.Flags().StringP("output", "o", "", "Output format (json)")
	scalingCmd.AddCommand(scalingHistoryCmd)

	// Predict command
	scalingPredictCmd.Flags().String("horizon", "1h", "Prediction horizon")
	scalingPredictCmd.Flags().StringP("output", "o", "", "Output format (json)")
	scalingCmd.AddCommand(scalingPredictCmd)

	// Pattern command
	scalingPatternCmd.Flags().StringP("output", "o", "", "Output format (json)")
	scalingCmd.AddCommand(scalingPatternCmd)
}
