package cli

import (
	"context"
	"fmt"

	"github.com/platformfoundry/pf-ce/internal/cost"
	"github.com/spf13/cobra"
)

var (
	costProvider        string
	costRegion          string
	costPricingFile     string
	costEstimatesDir    string
	costBudgetsDir      string
	costBudgetAmount    float64
	costBudgetPeriod    string
	costBudgetThreshold float64
	costBudgetEmail     string
)

var costCmd = &cobra.Command{
	Use:   "cost",
	Short: "Cost estimation and FinOps",
	Long:  `Estimate costs and manage budgets for infrastructure resources.`,
}

var costEstimateCmd = &cobra.Command{
	Use:   "estimate",
	Short: "Estimate infrastructure costs",
	Long:  `Estimate costs for platform resources (placeholder - requires platform spec).`,
	Example: `  pf cost estimate --provider aws --region us-east-1
  pf cost estimate --provider gcp --region us-central1`,
	RunE: runCostEstimate,
}

var costListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List cost estimates",
	Long:    `List all cost estimates.`,
	Example: `  pf cost list`,
	RunE:    runCostList,
}

var costShowCmd = &cobra.Command{
	Use:     "show <estimate-id>",
	Short:   "Show cost estimate details",
	Long:    `Show detailed cost estimate.`,
	Example: `  pf cost show platform-20240115-120000.json`,
	Args:    cobra.ExactArgs(1),
	RunE:    runCostShow,
}

var budgetCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a budget",
	Long:  `Create a cost budget with alerts.`,
	Example: `  pf budget create monthly-prod --amount 10000 --period monthly --threshold 80
  pf budget create daily-dev --amount 100 --period daily`,
	Args: cobra.ExactArgs(1),
	RunE: runBudgetCreate,
}

var budgetListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List budgets",
	Long:    `List all cost budgets.`,
	Example: `  pf budget list`,
	RunE:    runBudgetList,
}

var budgetCmd = &cobra.Command{
	Use:   "budget",
	Short: "Budget management",
	Long:  `Manage cost budgets and alerts.`,
}

func init() {
	// Estimate command flags
	costEstimateCmd.Flags().StringVar(&costProvider, "provider", "aws", "Cloud provider (aws, gcp, azure)")
	costEstimateCmd.Flags().StringVar(&costRegion, "region", "us-east-1", "Cloud region")
	costEstimateCmd.Flags().StringVar(&costPricingFile, "pricing-file", "/etc/platformfoundry/pricing/prices.json", "Pricing database file")
	costEstimateCmd.Flags().StringVar(&costEstimatesDir, "estimates-dir", "/var/lib/platformfoundry/cost/estimates", "Estimates directory")

	// Add completion for provider and region flags
	costEstimateCmd.RegisterFlagCompletionFunc("provider", cloudProviderCompletion)
	costEstimateCmd.RegisterFlagCompletionFunc("region", cloudRegionCompletion)

	// List command flags
	costListCmd.Flags().StringVar(&costEstimatesDir, "estimates-dir", "/var/lib/platformfoundry/cost/estimates", "Estimates directory")

	// Show command flags
	costShowCmd.Flags().StringVar(&costEstimatesDir, "estimates-dir", "/var/lib/platformfoundry/cost/estimates", "Estimates directory")

	// Budget create flags
	budgetCreateCmd.Flags().Float64Var(&costBudgetAmount, "amount", 0, "Budget amount (required)")
	budgetCreateCmd.Flags().StringVar(&costBudgetPeriod, "period", "monthly", "Budget period (hourly, daily, monthly, yearly)")
	budgetCreateCmd.Flags().Float64Var(&costBudgetThreshold, "threshold", 80, "Alert threshold percentage")
	budgetCreateCmd.Flags().StringVar(&costBudgetEmail, "email", "", "Notification email")
	budgetCreateCmd.Flags().StringVar(&costBudgetsDir, "budgets-dir", "/var/lib/platformfoundry/cost/budgets", "Budgets directory")
	budgetCreateCmd.MarkFlagRequired("amount")

	// Budget list flags
	budgetListCmd.Flags().StringVar(&costBudgetsDir, "budgets-dir", "/var/lib/platformfoundry/cost/budgets", "Budgets directory")

	// Add subcommands
	costCmd.AddCommand(costEstimateCmd)
	costCmd.AddCommand(costListCmd)
	costCmd.AddCommand(costShowCmd)

	budgetCmd.AddCommand(budgetCreateCmd)
	budgetCmd.AddCommand(budgetListCmd)

	costCmd.AddCommand(budgetCmd)
}

func runCostEstimate(cmd *cobra.Command, args []string) error {
	config := &cost.Config{
		PricingFile:  costPricingFile,
		EstimatesDir: costEstimatesDir,
		BudgetsDir:   costBudgetsDir,
	}

	estimator, err := cost.NewEstimator(config)
	if err != nil {
		return fmt.Errorf("failed to create cost estimator: %w", err)
	}

	// Example resources (placeholder - in real use would come from platform spec)
	resources := []*cost.Resource{
		{
			Name:     "api-server",
			Type:     cost.ResourceTypeCompute,
			Provider: cost.Provider(costProvider),
			Region:   costRegion,
			Spec: map[string]interface{}{
				"cpus":      float64(4),
				"memory_gb": float64(16),
			},
		},
		{
			Name:     "database",
			Type:     cost.ResourceTypeDatabase,
			Provider: cost.Provider(costProvider),
			Region:   costRegion,
			Spec: map[string]interface{}{
				"cpus":       float64(2),
				"memory_gb":  float64(8),
				"storage_gb": float64(100),
			},
		},
		{
			Name:     "storage",
			Type:     cost.ResourceTypeStorage,
			Provider: cost.Provider(costProvider),
			Region:   costRegion,
			Spec: map[string]interface{}{
				"size_gb": float64(1000),
			},
		},
		{
			Name:     "k8s-cluster",
			Type:     cost.ResourceTypeKubernetes,
			Provider: cost.Provider(costProvider),
			Region:   costRegion,
			Spec:     map[string]interface{}{},
		},
		{
			Name:     "load-balancer",
			Type:     cost.ResourceTypeLoadBalancer,
			Provider: cost.Provider(costProvider),
			Region:   costRegion,
			Spec:     map[string]interface{}{},
		},
	}

	ctx := context.Background()
	estimate, err := estimator.EstimatePlatform(ctx, "example-platform", cost.Provider(costProvider), resources)
	if err != nil {
		return fmt.Errorf("failed to estimate costs: %w", err)
	}

	// Display estimate
	fmt.Println("Cost Estimate:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Platform: %s\n", estimate.PlatformName)
	fmt.Printf("Provider: %s\n", estimate.Provider)
	fmt.Printf("Region: %s\n\n", costRegion)

	fmt.Println("Resource Costs:")
	for _, resource := range estimate.Resources {
		fmt.Printf("\n  %s (%s)\n", resource.Name, resource.Type)
		fmt.Printf("    Hourly:  $%.2f\n", resource.HourlyCost)
		fmt.Printf("    Monthly: $%.2f\n", resource.MonthlyCost)
		fmt.Printf("    Yearly:  $%.2f\n", resource.YearlyCost)
	}

	fmt.Println("\nTotal Costs:")
	fmt.Printf("  Hourly:  $%.2f %s\n", estimate.TotalHourly, estimate.Currency)
	fmt.Printf("  Monthly: $%.2f %s\n", estimate.TotalMonthly, estimate.Currency)
	fmt.Printf("  Yearly:  $%.2f %s\n\n", estimate.TotalYearly, estimate.Currency)

	if len(estimate.Recommendations) > 0 {
		fmt.Println("Cost Optimization Recommendations:")
		for i, rec := range estimate.Recommendations {
			fmt.Printf("  %d. %s\n", i+1, rec)
		}
	}

	return nil
}

func runCostList(cmd *cobra.Command, args []string) error {
	config := &cost.Config{
		EstimatesDir: costEstimatesDir,
	}

	estimator, err := cost.NewEstimator(config)
	if err != nil {
		return fmt.Errorf("failed to create cost estimator: %w", err)
	}

	estimates, err := estimator.ListEstimates()
	if err != nil {
		return fmt.Errorf("failed to list estimates: %w", err)
	}

	if len(estimates) == 0 {
		fmt.Println("No cost estimates found")
		return nil
	}

	fmt.Printf("Cost Estimates (%d):\n", len(estimates))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for i, est := range estimates {
		fmt.Printf("%d. %s (%s)\n", i+1, est.PlatformName, est.Provider)
		fmt.Printf("   Timestamp: %s\n", est.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("   Monthly Cost: $%.2f %s\n", est.TotalMonthly, est.Currency)
		fmt.Printf("   Resources: %d\n", len(est.Resources))

		if i < len(estimates)-1 {
			fmt.Println()
		}
	}

	return nil
}

func runCostShow(cmd *cobra.Command, args []string) error {
	estimateID := args[0]

	config := &cost.Config{
		EstimatesDir: costEstimatesDir,
	}

	estimator, err := cost.NewEstimator(config)
	if err != nil {
		return fmt.Errorf("failed to create cost estimator: %w", err)
	}

	estimate, err := estimator.LoadEstimate(estimateID)
	if err != nil {
		return fmt.Errorf("failed to load estimate: %w", err)
	}

	// Display detailed estimate
	fmt.Println("Cost Estimate Details:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Platform: %s\n", estimate.PlatformName)
	fmt.Printf("Provider: %s\n", estimate.Provider)
	fmt.Printf("Timestamp: %s\n\n", estimate.Timestamp.Format("2006-01-02 15:04:05"))

	fmt.Println("Resources:")
	for _, resource := range estimate.Resources {
		fmt.Printf("\n  %s (%s)\n", resource.Name, resource.Type)
		fmt.Printf("    Hourly:  $%.4f\n", resource.HourlyCost)
		fmt.Printf("    Monthly: $%.2f\n", resource.MonthlyCost)
		fmt.Printf("    Yearly:  $%.2f\n", resource.YearlyCost)
	}

	fmt.Println("\nTotal Costs:")
	fmt.Printf("  Hourly:  $%.4f %s\n", estimate.TotalHourly, estimate.Currency)
	fmt.Printf("  Monthly: $%.2f %s\n", estimate.TotalMonthly, estimate.Currency)
	fmt.Printf("  Yearly:  $%.2f %s\n\n", estimate.TotalYearly, estimate.Currency)

	if len(estimate.Recommendations) > 0 {
		fmt.Println("Recommendations:")
		for i, rec := range estimate.Recommendations {
			fmt.Printf("  %d. %s\n", i+1, rec)
		}
	}

	return nil
}

func runBudgetCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	config := &cost.Config{
		BudgetsDir: costBudgetsDir,
	}

	estimator, err := cost.NewEstimator(config)
	if err != nil {
		return fmt.Errorf("failed to create cost estimator: %w", err)
	}

	budget := &cost.Budget{
		Name:        name,
		Amount:      costBudgetAmount,
		Period:      costBudgetPeriod,
		Threshold:   costBudgetThreshold,
		Currency:    "USD",
		NotifyEmail: costBudgetEmail,
	}

	if err := estimator.CreateBudget(budget); err != nil {
		return fmt.Errorf("failed to create budget: %w", err)
	}

	fmt.Printf("✓ Budget created: %s\n", name)
	fmt.Printf("  Amount: $%.2f %s per %s\n", budget.Amount, budget.Currency, budget.Period)
	fmt.Printf("  Alert Threshold: %.0f%%\n", budget.Threshold)

	if budget.NotifyEmail != "" {
		fmt.Printf("  Notifications: %s\n", budget.NotifyEmail)
	}

	return nil
}

func runBudgetList(cmd *cobra.Command, args []string) error {
	config := &cost.Config{
		BudgetsDir: costBudgetsDir,
	}

	estimator, err := cost.NewEstimator(config)
	if err != nil {
		return fmt.Errorf("failed to create cost estimator: %w", err)
	}

	budgets, err := estimator.ListBudgets()
	if err != nil {
		return fmt.Errorf("failed to list budgets: %w", err)
	}

	if len(budgets) == 0 {
		fmt.Println("No budgets found")
		return nil
	}

	fmt.Printf("Cost Budgets (%d):\n", len(budgets))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for i, budget := range budgets {
		fmt.Printf("%d. %s\n", i+1, budget.Name)
		fmt.Printf("   Amount: $%.2f %s per %s\n", budget.Amount, budget.Currency, budget.Period)
		fmt.Printf("   Alert Threshold: %.0f%%\n", budget.Threshold)
		fmt.Printf("   Created: %s\n", budget.CreatedAt.Format("2006-01-02"))

		if i < len(budgets)-1 {
			fmt.Println()
		}
	}

	return nil
}
