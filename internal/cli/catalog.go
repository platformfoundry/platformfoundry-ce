package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/platformfoundry/pf-ce/internal/catalog"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var catalogCmd = &cobra.Command{
	Use:   "catalog",
	Short: "Self-service resource catalog commands",
	Long:  `Browse and provision resources from the self-service catalog.`,
}

var catalogListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available resources in catalog",
	Long:  `List all resource definitions available for provisioning.`,
	RunE:  runCatalogList,
}

var catalogShowCmd = &cobra.Command{
	Use:   "show [name]",
	Short: "Show resource definition details",
	Long:  `Show details of a specific resource definition.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runCatalogShow,
}

var catalogRequestCmd = &cobra.Command{
	Use:   "request [definition]",
	Short: "Request a new resource",
	Long:  `Submit a request to provision a new resource.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runCatalogRequest,
}

var catalogRequestsCmd = &cobra.Command{
	Use:   "requests",
	Short: "List resource requests",
	Long:  `List all resource requests.`,
	RunE:  runCatalogRequests,
}

var catalogApproveCmd = &cobra.Command{
	Use:   "approve [request-id]",
	Short: "Approve a pending request",
	Long:  `Approve a resource request that is pending approval.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runCatalogApprove,
}

var catalogRejectCmd = &cobra.Command{
	Use:   "reject [request-id]",
	Short: "Reject a pending request",
	Long:  `Reject a resource request that is pending approval.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runCatalogReject,
}

var catalogQuotaCmd = &cobra.Command{
	Use:   "quota",
	Short: "View or set quotas",
	Long:  `View or set resource quotas for teams.`,
	RunE:  runCatalogQuota,
}

// Flags
var (
	catalogFormat      string
	catalogCategory    string
	catalogTeam        string
	catalogApp         string
	catalogEnv         string
	catalogName        string
	catalogInputs      []string
	catalogRejectReason string
)

func init() {
	// List flags
	catalogListCmd.Flags().StringVar(&catalogFormat, "format", "table", "Output format (table, json, yaml)")
	catalogListCmd.Flags().StringVar(&catalogCategory, "category", "", "Filter by category")

	// Show flags
	catalogShowCmd.Flags().StringVar(&catalogFormat, "format", "yaml", "Output format (yaml, json)")

	// Request flags
	catalogRequestCmd.Flags().StringVar(&catalogTeam, "team", "", "Team name")
	catalogRequestCmd.Flags().StringVar(&catalogApp, "app", "", "Application name")
	catalogRequestCmd.Flags().StringVar(&catalogEnv, "env", "", "Environment name")
	catalogRequestCmd.Flags().StringVar(&catalogName, "name", "", "Resource name")
	catalogRequestCmd.Flags().StringArrayVar(&catalogInputs, "input", nil, "Input values (key=value)")
	catalogRequestCmd.MarkFlagRequired("team")
	catalogRequestCmd.MarkFlagRequired("app")
	catalogRequestCmd.MarkFlagRequired("env")
	catalogRequestCmd.MarkFlagRequired("name")

	// Requests flags
	catalogRequestsCmd.Flags().StringVar(&catalogFormat, "format", "table", "Output format (table, json)")
	catalogRequestsCmd.Flags().StringVar(&catalogTeam, "team", "", "Filter by team")
	catalogRequestsCmd.Flags().StringVar(&catalogApp, "app", "", "Filter by application")

	// Reject flags
	catalogRejectCmd.Flags().StringVar(&catalogRejectReason, "reason", "", "Rejection reason")

	// Quota flags
	catalogQuotaCmd.Flags().StringVar(&catalogTeam, "team", "", "Team name")
	catalogQuotaCmd.Flags().StringVar(&catalogFormat, "format", "table", "Output format (table, json)")

	catalogCmd.AddCommand(catalogListCmd)
	catalogCmd.AddCommand(catalogShowCmd)
	catalogCmd.AddCommand(catalogRequestCmd)
	catalogCmd.AddCommand(catalogRequestsCmd)
	catalogCmd.AddCommand(catalogApproveCmd)
	catalogCmd.AddCommand(catalogRejectCmd)
	catalogCmd.AddCommand(catalogQuotaCmd)
}

func runCatalogList(cmd *cobra.Command, args []string) error {
	cat := catalog.NewCatalog(nil)

	var defs []*catalog.ResourceDefinition
	if catalogCategory != "" {
		defs = cat.ListDefinitionsByCategory(catalogCategory)
	} else {
		defs = cat.ListDefinitions()
	}

	if catalogFormat == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(defs)
	}

	if catalogFormat == "yaml" {
		enc := yaml.NewEncoder(os.Stdout)
		return enc.Encode(defs)
	}

	// Table format
	fmt.Printf("%-25s %-12s %-15s %s\n", "NAME", "TYPE", "CATEGORY", "DESCRIPTION")
	fmt.Println(strings.Repeat("-", 80))

	for _, def := range defs {
		desc := def.Metadata.Description
		if len(desc) > 30 {
			desc = desc[:27] + "..."
		}
		fmt.Printf("%-25s %-12s %-15s %s\n",
			def.Metadata.Name, def.Spec.Type, def.Metadata.Category, desc)
	}

	return nil
}

func runCatalogShow(cmd *cobra.Command, args []string) error {
	cat := catalog.NewCatalog(nil)

	def, err := cat.GetDefinition(args[0])
	if err != nil {
		return err
	}

	if catalogFormat == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(def)
	}

	// YAML format (default)
	enc := yaml.NewEncoder(os.Stdout)
	return enc.Encode(def)
}

func runCatalogRequest(cmd *cobra.Command, args []string) error {
	cat := catalog.NewCatalog(nil)

	// Parse inputs
	inputs := make(map[string]interface{})
	for _, input := range catalogInputs {
		parts := strings.SplitN(input, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid input format: %s (expected key=value)", input)
		}
		inputs[parts[0]] = parts[1]
	}

	req := &catalog.ResourceRequest{
		DefinitionName: args[0],
		Name:           catalogName,
		Application:    catalogApp,
		Environment:    catalogEnv,
		Team:           catalogTeam,
		Inputs:         inputs,
		RequestedBy:    "cli-user", // In production, get from auth context
	}

	if err := cat.CreateRequest(context.Background(), req); err != nil {
		return err
	}

	fmt.Printf("Request created: %s\n", req.ID)
	fmt.Printf("Status: %s\n", req.Status)

	if req.EstimatedCost != nil {
		fmt.Printf("Estimated cost: $%.2f/month\n", req.EstimatedCost.MonthlyCost)
	}

	if req.Status == catalog.RequestPendingApproval {
		fmt.Println("\nThis request requires approval before provisioning.")
	}

	return nil
}

func runCatalogRequests(cmd *cobra.Command, args []string) error {
	cat := catalog.NewCatalog(nil)

	filters := catalog.RequestFilters{
		Team:        catalogTeam,
		Application: catalogApp,
	}

	requests := cat.ListRequests(filters)

	if catalogFormat == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(requests)
	}

	// Table format
	fmt.Printf("%-40s %-20s %-15s %-15s %s\n", "ID", "DEFINITION", "ENVIRONMENT", "STATUS", "CREATED")
	fmt.Println(strings.Repeat("-", 110))

	for _, req := range requests {
		created := req.CreatedAt.Format("2006-01-02 15:04")
		fmt.Printf("%-40s %-20s %-15s %-15s %s\n",
			catalogTruncate(req.ID, 40), req.DefinitionName, req.Environment, req.Status, created)
	}

	return nil
}

func runCatalogApprove(cmd *cobra.Command, args []string) error {
	cat := catalog.NewCatalog(nil)

	if err := cat.ApproveRequest(context.Background(), args[0], "cli-admin"); err != nil {
		return err
	}

	fmt.Printf("Request %s approved\n", args[0])
	return nil
}

func runCatalogReject(cmd *cobra.Command, args []string) error {
	cat := catalog.NewCatalog(nil)

	if err := cat.RejectRequest(context.Background(), args[0], "cli-admin", catalogRejectReason); err != nil {
		return err
	}

	fmt.Printf("Request %s rejected\n", args[0])
	return nil
}

func runCatalogQuota(cmd *cobra.Command, args []string) error {
	cat := catalog.NewCatalog(nil)

	if catalogTeam == "" {
		return fmt.Errorf("--team is required")
	}

	quota := cat.GetQuota(catalogTeam, "")

	if quota == nil {
		fmt.Printf("No quota defined for team: %s\n", catalogTeam)
		return nil
	}

	if catalogFormat == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(quota)
	}

	// Table format
	fmt.Printf("Quota for team: %s\n\n", catalogTeam)

	fmt.Printf("%-15s %10s %10s\n", "RESOURCE", "USED", "LIMIT")
	fmt.Println(strings.Repeat("-", 40))

	for resourceType, limit := range quota.Limits {
		used := quota.Used[resourceType]
		fmt.Printf("%-15s %10d %10d\n", resourceType, used, limit)
	}

	if quota.CostLimit > 0 {
		fmt.Println()
		fmt.Printf("Cost: $%.2f / $%.2f\n", quota.CostUsed, quota.CostLimit)
	}

	return nil
}

func catalogTruncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// GetCatalogCmd returns the catalog command for registration
func GetCatalogCmd() *cobra.Command {
	return catalogCmd
}
