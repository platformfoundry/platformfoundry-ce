package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/platformfoundry/pf-ce/internal/cost"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(forecastCmd)

	forecastCmd.Flags().StringP("resource", "r", "", "Specific resource to forecast")
	forecastCmd.Flags().IntP("days", "d", 30, "Number of days to forecast")
	forecastCmd.Flags().Bool("recommendations", true, "Include cost optimization recommendations")
	forecastCmd.Flags().String("format", "table", "Output format (table, json)")
}

var forecastCmd = &cobra.Command{
	Use:   "forecast",
	Short: "Forecast infrastructure costs",
	Long: `Predict future infrastructure costs based on historical data and trends.

The forecast command analyzes past cost data and uses statistical models
to predict future costs. It also provides cost optimization recommendations.

Models used:
  - Exponential Smoothing (default)
  - Linear Regression
  - Moving Average`,
	Example: `  # Forecast total costs for the next 30 days
  pf forecast

  # Forecast a specific resource
  pf forecast -r compute/api-gateway

  # Forecast for 90 days
  pf forecast -d 90

  # Without recommendations
  pf forecast --recommendations=false`,
	RunE: runForecast,
}

func runForecast(cmd *cobra.Command, args []string) error {
	resource, _ := cmd.Flags().GetString("resource")
	days, _ := cmd.Flags().GetInt("days")
	showRecs, _ := cmd.Flags().GetBool("recommendations")

	// Create forecaster with mock data source
	// In production, this would use real cost data
	dataSource := cost.NewMockDataSource()
	forecaster := cost.NewForecaster(cost.ForecasterConfig{
		DataSource: dataSource,
	})

	ctx := context.Background()

	if resource != "" {
		// Forecast specific resource
		return forecastResource(ctx, forecaster, resource, days, showRecs)
	}

	// Forecast all resources
	return forecastAll(ctx, forecaster, days, showRecs)
}

func forecastResource(ctx context.Context, forecaster *cost.Forecaster, resource string, days int, showRecs bool) error {
	fmt.Printf("Forecasting costs for %s (%d days)...\n\n", resource, days)

	fc, err := forecaster.Predict(ctx, resource, days)
	if err != nil {
		return fmt.Errorf("forecast failed: %w", err)
	}

	printForecast(fc, showRecs)
	return nil
}

func forecastAll(ctx context.Context, forecaster *cost.Forecaster, days int, showRecs bool) error {
	fmt.Printf("Forecasting total costs (%d days)...\n\n", days)

	fc, err := forecaster.GetTotalForecast(ctx, days)
	if err != nil {
		return fmt.Errorf("forecast failed: %w", err)
	}

	printForecast(fc, showRecs)

	// Show breakdown
	if len(fc.BreakdownBy) > 0 {
		fmt.Println("\nCost Breakdown by Resource:")
		fmt.Println(strings.Repeat("-", 50))
		for resource, predicted := range fc.BreakdownBy {
			fmt.Printf("  %-30s $%.2f/day\n", resource, predicted)
		}
	}

	return nil
}

func printForecast(fc *cost.CostForecast, showRecs bool) {
	// Header
	fmt.Println("Cost Forecast")
	fmt.Println(strings.Repeat("=", 60))

	// Summary
	trendIcon := getTrendIcon(fc.Trend)
	fmt.Printf("\nResource: %s\n", fc.Resource)
	fmt.Printf("Period:   %s\n", fc.Period)
	fmt.Printf("\nCurrent Cost:   $%.2f/day\n", fc.CurrentCost)
	fmt.Printf("Predicted Cost: $%.2f/day\n", fc.PredictedCost)
	fmt.Printf("Change:         %s $%.2f (%.1f%%)\n", trendIcon, fc.CostChange, fc.CostChangePercent)
	fmt.Printf("Trend:          %s\n", fc.Trend)
	fmt.Printf("Confidence:     %.0f%%\n", fc.Confidence*100)

	// Forecast points (show first/last few)
	if len(fc.Forecasts) > 0 {
		fmt.Println("\nForecast Timeline:")
		fmt.Println(strings.Repeat("-", 60))
		fmt.Printf("%-12s %-12s %-12s %-12s %-10s\n", "Date", "Predicted", "Low", "High", "Confidence")

		// Show first 3 and last 3 if more than 6
		showPoints := fc.Forecasts
		if len(fc.Forecasts) > 6 {
			showPoints = append(fc.Forecasts[:3], fc.Forecasts[len(fc.Forecasts)-3:]...)
		}

		prevIdx := -1
		for _, fp := range showPoints {
			idx := 0
			for i, orig := range fc.Forecasts {
				if orig.Timestamp == fp.Timestamp {
					idx = i
					break
				}
			}

			if prevIdx != -1 && idx > prevIdx+1 {
				fmt.Println("  ...")
			}
			prevIdx = idx

			fmt.Printf("%-12s $%-11.2f $%-11.2f $%-11.2f %.0f%%\n",
				fp.Timestamp.Format("Jan 02"),
				fp.Predicted,
				fp.LowerBound,
				fp.UpperBound,
				fp.Confidence*100,
			)
		}
	}

	// Recommendations
	if showRecs && len(fc.Recommendations) > 0 {
		fmt.Println("\nCost Optimization Recommendations:")
		fmt.Println(strings.Repeat("-", 60))

		for i, rec := range fc.Recommendations {
			impactIcon := getImpactIcon(rec.Impact)
			fmt.Printf("\n%d. %s [%s %s]\n", i+1, rec.Description, impactIcon, rec.Impact)
			fmt.Printf("   Type: %s\n", rec.Type)
			fmt.Printf("   Potential Saving: $%.2f/day (%.0f%% confidence)\n",
				rec.PotentialSaving, rec.Confidence*100)
			fmt.Printf("   Action: %s\n", rec.Action)
		}

		// Calculate total potential savings
		var totalSavings float64
		for _, rec := range fc.Recommendations {
			totalSavings += rec.PotentialSaving
		}
		fmt.Printf("\nTotal Potential Savings: $%.2f/day ($%.2f/month)\n",
			totalSavings, totalSavings*30)
	}
}

func getTrendIcon(trend string) string {
	switch trend {
	case "increasing":
		return "↑"
	case "decreasing":
		return "↓"
	default:
		return "→"
	}
}

func getImpactIcon(impact string) string {
	switch impact {
	case "high":
		return "🔴"
	case "medium":
		return "🟡"
	case "low":
		return "🟢"
	default:
		return "•"
	}
}
