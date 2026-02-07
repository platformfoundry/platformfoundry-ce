package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/platformfoundry/pf-ce/internal/promise"
	"github.com/platformfoundry/pf-ce/pkg/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var promiseManager *promise.Manager

func init() {
	rootCmd.AddCommand(promiseCmd)

	// Subcommands
	promiseCmd.AddCommand(promiseListCmd)
	promiseCmd.AddCommand(promiseShowCmd)
	promiseCmd.AddCommand(promiseRequestCmd)
	promiseCmd.AddCommand(promiseInstancesCmd)
	promiseCmd.AddCommand(promiseStatusCmd)
	promiseCmd.AddCommand(promiseApproveCmd)
	promiseCmd.AddCommand(promiseRejectCmd)
	promiseCmd.AddCommand(promiseDeleteCmd)
	promiseCmd.AddCommand(promiseStatsCmd)

	// Flags for list
	promiseListCmd.Flags().StringP("category", "c", "", "Filter by category")
	promiseListCmd.Flags().StringP("output", "o", "table", "Output format: table, json, yaml")

	// Flags for show
	promiseShowCmd.Flags().StringP("output", "o", "yaml", "Output format: yaml, json")

	// Flags for request
	promiseRequestCmd.Flags().StringP("file", "f", "", "Request specification file")
	promiseRequestCmd.Flags().StringP("promise", "p", "", "Promise name (for interactive mode)")
	promiseRequestCmd.Flags().StringP("name", "n", "", "Instance name")
	promiseRequestCmd.Flags().StringP("team", "t", "", "Team name")
	promiseRequestCmd.Flags().StringP("environment", "e", "", "Target environment")
	promiseRequestCmd.Flags().StringToStringP("input", "i", nil, "Input values (key=value)")

	// Flags for instances
	promiseInstancesCmd.Flags().StringP("promise", "p", "", "Filter by promise")
	promiseInstancesCmd.Flags().StringP("team", "t", "", "Filter by team")
	promiseInstancesCmd.Flags().StringP("environment", "e", "", "Filter by environment")
	promiseInstancesCmd.Flags().StringP("output", "o", "table", "Output format: table, json, yaml")

	// Flags for status
	promiseStatusCmd.Flags().StringP("output", "o", "table", "Output format: table, json, yaml")

	// Flags for approve/reject
	promiseApproveCmd.Flags().String("reason", "", "Reason for approval")
	promiseRejectCmd.Flags().String("reason", "", "Reason for rejection")
	promiseRejectCmd.MarkFlagRequired("reason")

	// Initialize manager with builtin promises
	promiseManager = promise.NewManager()
	promiseManager.LoadBuiltinPromises()
}

var promiseCmd = &cobra.Command{
	Use:   "promise",
	Short: "Manage platform promises (self-service infrastructure)",
	Long: `Promises define platform capabilities that developers can self-service.
They provide a contract for what infrastructure can be requested and how it
will be provisioned, abstracting away the underlying complexity.

Promises enable:
- Self-service infrastructure provisioning
- Standardized configurations with guardrails
- Approval workflows for production resources
- Automatic policy enforcement`,
	Example: `  # List available promises
  pf promise list

  # Show promise details
  pf promise show postgresql-database

  # Request a new database (interactive)
  pf promise request -p postgresql-database

  # Request a database from file
  pf promise request -f database-request.yaml

  # List your instances
  pf promise instances

  # Check instance status
  pf promise status my-database`,
}

var promiseListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available promises",
	Long:  `List all available promises that can be requested.`,
	Example: `  # List all promises
  pf promise list

  # List database promises
  pf promise list -c database

  # Output as JSON
  pf promise list -o json`,
	RunE: runPromiseList,
}

var promiseShowCmd = &cobra.Command{
	Use:   "show [name]",
	Short: "Show promise details",
	Long:  `Show detailed information about a specific promise.`,
	Example: `  # Show promise details
  pf promise show postgresql-database`,
	Args: cobra.ExactArgs(1),
	RunE: runPromiseShow,
}

var promiseRequestCmd = &cobra.Command{
	Use:   "request",
	Short: "Request a promise instance",
	Long:  `Request a new instance of a promise. This will provision the infrastructure according to the promise contract.`,
	Example: `  # Request interactively
  pf promise request -p postgresql-database -n my-db -t orders

  # Request from file
  pf promise request -f database-request.yaml

  # Request with inline inputs
  pf promise request -p postgresql-database -n my-db -t orders -i size=medium -i version=15`,
	RunE: runPromiseRequest,
}

var promiseInstancesCmd = &cobra.Command{
	Use:   "instances",
	Short: "List promise instances",
	Long:  `List all promise instances that have been provisioned.`,
	Example: `  # List all instances
  pf promise instances

  # List instances for a team
  pf promise instances -t orders

  # List database instances
  pf promise instances -p postgresql-database`,
	RunE: runPromiseInstances,
}

var promiseStatusCmd = &cobra.Command{
	Use:   "status [name]",
	Short: "Check instance status",
	Long:  `Check the status of a promise instance.`,
	Example: `  # Check instance status
  pf promise status my-database`,
	Args: cobra.ExactArgs(1),
	RunE: runPromiseStatus,
}

var promiseApproveCmd = &cobra.Command{
	Use:   "approve [name]",
	Short: "Approve a pending request",
	Long:  `Approve a promise request that is awaiting approval.`,
	Example: `  # Approve a request
  pf promise approve my-database --reason "Approved for production use"`,
	Args: cobra.ExactArgs(1),
	RunE: runPromiseApprove,
}

var promiseRejectCmd = &cobra.Command{
	Use:   "reject [name]",
	Short: "Reject a pending request",
	Long:  `Reject a promise request that is awaiting approval.`,
	Example: `  # Reject a request
  pf promise reject my-database --reason "Does not meet security requirements"`,
	Args: cobra.ExactArgs(1),
	RunE: runPromiseReject,
}

var promiseDeleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete a promise instance",
	Long:  `Delete a promise instance and its associated resources.`,
	Example: `  # Delete an instance
  pf promise delete my-database`,
	Args: cobra.ExactArgs(1),
	RunE: runPromiseDelete,
}

var promiseStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show promise statistics",
	Long:  `Show statistics about promises and instances.`,
	RunE:  runPromiseStats,
}

func runPromiseList(cmd *cobra.Command, args []string) error {
	category, _ := cmd.Flags().GetString("category")
	output, _ := cmd.Flags().GetString("output")

	var promises []*types.Promise
	if category != "" {
		promises = promiseManager.ListPromisesByCategory(category)
	} else {
		promises = promiseManager.ListPromises()
	}

	// Sort by name
	sort.Slice(promises, func(i, j int) bool {
		return promises[i].Metadata.Name < promises[j].Metadata.Name
	})

	switch output {
	case "json":
		data, err := json.MarshalIndent(promises, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	case "yaml":
		data, err := yaml.Marshal(promises)
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	default:
		fmt.Println()
		fmt.Println("Available Promises:")
		fmt.Println(strings.Repeat("=", 80))
		fmt.Println()
		fmt.Printf("%-25s %-12s %-40s\n", "NAME", "CATEGORY", "DESCRIPTION")
		fmt.Println(strings.Repeat("-", 80))
		for _, p := range promises {
			desc := p.Metadata.Description
			if len(desc) > 40 {
				desc = desc[:37] + "..."
			}
			fmt.Printf("%-25s %-12s %-40s\n", p.Metadata.Name, p.Spec.Category, desc)
		}
		fmt.Println()
		fmt.Printf("Total: %d promises\n", len(promises))
		fmt.Println("\nUse 'pf promise show <name>' for details")
	}

	return nil
}

func runPromiseShow(cmd *cobra.Command, args []string) error {
	name := args[0]
	output, _ := cmd.Flags().GetString("output")

	p, err := promiseManager.GetPromise(name)
	if err != nil {
		return err
	}

	switch output {
	case "json":
		data, err := json.MarshalIndent(p, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	default:
		// Pretty print
		fmt.Println()
		fmt.Printf("Promise: %s\n", p.Metadata.Name)
		fmt.Println(strings.Repeat("=", 60))
		fmt.Println()
		fmt.Printf("Description: %s\n", p.Spec.Description)
		fmt.Printf("Category:    %s\n", p.Spec.Category)
		fmt.Printf("Provider:    %s\n", p.Spec.Provider)

		if p.Spec.Approval != nil && p.Spec.Approval.Required {
			envs := "all environments"
			if len(p.Spec.Approval.Environments) > 0 {
				envs = strings.Join(p.Spec.Approval.Environments, ", ")
			}
			fmt.Printf("Approval:    Required for %s\n", envs)
		}

		fmt.Println()
		fmt.Println("Inputs:")
		fmt.Println(strings.Repeat("-", 60))
		for _, input := range p.Spec.Inputs {
			required := ""
			if input.Required {
				required = " (required)"
			}
			defaultVal := ""
			if input.Default != nil {
				defaultVal = fmt.Sprintf(" [default: %v]", input.Default)
			}
			fmt.Printf("  %s (%s)%s%s\n", input.Name, input.Type, required, defaultVal)
			fmt.Printf("    %s\n", input.Description)
			if len(input.Enum) > 0 {
				fmt.Printf("    Options: %s\n", strings.Join(input.Enum, ", "))
			}
		}

		fmt.Println()
		fmt.Println("Outputs:")
		fmt.Println(strings.Repeat("-", 60))
		for _, output := range p.Spec.Outputs {
			fmt.Printf("  %s (%s)\n", output.Name, output.Type)
			fmt.Printf("    %s\n", output.Description)
		}

		fmt.Println()
		fmt.Println("Example Request:")
		fmt.Println(strings.Repeat("-", 60))
		fmt.Printf(`apiVersion: platformfoundry.io/v1
kind: PromiseRequest
metadata:
  name: my-%s
  team: my-team
spec:
  promise: %s
  inputs:
`, p.Metadata.Name, p.Metadata.Name)
		for _, input := range p.Spec.Inputs {
			if input.Default != nil {
				fmt.Printf("    %s: %v\n", input.Name, input.Default)
			} else if input.Required {
				fmt.Printf("    %s: <required>\n", input.Name)
			}
		}
	}

	return nil
}

func runPromiseRequest(cmd *cobra.Command, args []string) error {
	file, _ := cmd.Flags().GetString("file")
	promiseName, _ := cmd.Flags().GetString("promise")
	name, _ := cmd.Flags().GetString("name")
	team, _ := cmd.Flags().GetString("team")
	env, _ := cmd.Flags().GetString("environment")
	inputFlags, _ := cmd.Flags().GetStringToString("input")

	var req *types.PromiseRequest

	if file != "" {
		// Load from file
		data, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		req = &types.PromiseRequest{}
		if err := yaml.Unmarshal(data, req); err != nil {
			return fmt.Errorf("failed to parse YAML: %w", err)
		}
	} else {
		// Build from flags
		if promiseName == "" {
			return fmt.Errorf("either --file or --promise is required")
		}
		if name == "" {
			return fmt.Errorf("--name is required")
		}
		if team == "" {
			return fmt.Errorf("--team is required")
		}

		inputs := make(map[string]interface{})
		for k, v := range inputFlags {
			inputs[k] = v
		}

		req = &types.PromiseRequest{
			APIVersion: "platformfoundry.io/v1",
			Kind:       "PromiseRequest",
			Metadata: types.PromiseRequestMetadata{
				Name:        name,
				Team:        team,
				Environment: env,
			},
			Spec: types.PromiseRequestSpec{
				Promise: promiseName,
				Inputs:  inputs,
			},
		}
	}

	ctx := context.Background()
	result, err := promiseManager.Request(ctx, req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}

	fmt.Println()
	fmt.Printf("Promise Request: %s\n", result.Metadata.Name)
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Promise:     %s\n", result.Spec.Promise)
	fmt.Printf("Team:        %s\n", result.Metadata.Team)
	if result.Metadata.Environment != "" {
		fmt.Printf("Environment: %s\n", result.Metadata.Environment)
	}
	fmt.Printf("State:       %s\n", result.Status.State)
	fmt.Printf("Message:     %s\n", result.Status.Message)

	if result.Status.State == types.PromiseRequestStateAwaitingApproval {
		fmt.Println()
		fmt.Println("This request requires approval.")
		fmt.Println("An approver will review your request.")
	} else if result.Status.State == types.PromiseRequestStateReady {
		fmt.Println()
		fmt.Println("Outputs:")
		for k, v := range result.Status.Outputs {
			fmt.Printf("  %s: %v\n", k, v)
		}
	}

	return nil
}

func runPromiseInstances(cmd *cobra.Command, args []string) error {
	promiseName, _ := cmd.Flags().GetString("promise")
	team, _ := cmd.Flags().GetString("team")
	env, _ := cmd.Flags().GetString("environment")
	output, _ := cmd.Flags().GetString("output")

	filters := promise.InstanceFilters{
		Promise:     promiseName,
		Team:        team,
		Environment: env,
	}

	instances := promiseManager.ListInstances(filters)

	// Sort by name
	sort.Slice(instances, func(i, j int) bool {
		return instances[i].Name < instances[j].Name
	})

	switch output {
	case "json":
		data, err := json.MarshalIndent(instances, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	case "yaml":
		data, err := yaml.Marshal(instances)
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	default:
		fmt.Println()
		fmt.Println("+-------------------------+-----------------------+----------+-------------+----------+")
		fmt.Println("| NAME                    | PROMISE               | TEAM     | ENVIRONMENT | STATE    |")
		fmt.Println("+-------------------------+-----------------------+----------+-------------+----------+")
		for _, inst := range instances {
			fmt.Printf("| %-23s | %-21s | %-8s | %-11s | %-8s |\n",
				promiseTruncate(inst.Name, 23),
				promiseTruncate(inst.Promise, 21),
				promiseTruncate(inst.Team, 8),
				promiseTruncate(inst.Environment, 11),
				inst.State)
		}
		fmt.Println("+-------------------------+-----------------------+----------+-------------+----------+")
		fmt.Printf("\nTotal: %d instances\n", len(instances))
	}

	return nil
}

func runPromiseStatus(cmd *cobra.Command, args []string) error {
	name := args[0]
	output, _ := cmd.Flags().GetString("output")

	// Try to get instance first
	inst, err := promiseManager.GetInstance(name)
	if err != nil {
		// Fall back to request
		req, reqErr := promiseManager.GetRequest(name)
		if reqErr != nil {
			return fmt.Errorf("not found: %s", name)
		}

		switch output {
		case "json":
			data, err := json.MarshalIndent(req, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
		case "yaml":
			data, err := yaml.Marshal(req)
			if err != nil {
				return err
			}
			fmt.Println(string(data))
		default:
			printRequestStatus(req)
		}
		return nil
	}

	switch output {
	case "json":
		data, err := json.MarshalIndent(inst, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	case "yaml":
		data, err := yaml.Marshal(inst)
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	default:
		printInstanceStatus(inst)
	}

	return nil
}

func runPromiseApprove(cmd *cobra.Command, args []string) error {
	name := args[0]
	reason, _ := cmd.Flags().GetString("reason")

	ctx := context.Background()
	req, err := promiseManager.Approve(ctx, name, "admin", reason)
	if err != nil {
		return err
	}

	fmt.Printf("Request '%s' approved.\n", name)
	fmt.Printf("State: %s\n", req.Status.State)
	if req.Status.State == types.PromiseRequestStateReady {
		fmt.Println("\nOutputs:")
		for k, v := range req.Status.Outputs {
			fmt.Printf("  %s: %v\n", k, v)
		}
	}

	return nil
}

func runPromiseReject(cmd *cobra.Command, args []string) error {
	name := args[0]
	reason, _ := cmd.Flags().GetString("reason")

	ctx := context.Background()
	_, err := promiseManager.Reject(ctx, name, "admin", reason)
	if err != nil {
		return err
	}

	fmt.Printf("Request '%s' rejected.\n", name)
	fmt.Printf("Reason: %s\n", reason)

	return nil
}

func runPromiseDelete(cmd *cobra.Command, args []string) error {
	name := args[0]

	ctx := context.Background()
	if err := promiseManager.Delete(ctx, name); err != nil {
		return err
	}

	fmt.Printf("Instance '%s' deleted.\n", name)

	return nil
}

func runPromiseStats(cmd *cobra.Command, args []string) error {
	stats := promiseManager.GetStats()

	fmt.Println()
	fmt.Println("Promise Statistics")
	fmt.Println(strings.Repeat("=", 40))
	fmt.Printf("Total Promises:    %d\n", stats.TotalPromises)
	fmt.Printf("Total Instances:   %d\n", stats.TotalInstances)
	fmt.Printf("Total Requests:    %d\n", stats.TotalRequests)
	fmt.Printf("Pending Approval:  %d\n", stats.PendingApproval)

	if len(stats.ByCategory) > 0 {
		fmt.Println()
		fmt.Println("By Category:")
		for cat, count := range stats.ByCategory {
			fmt.Printf("  %-15s %d\n", cat, count)
		}
	}

	if len(stats.ByState) > 0 {
		fmt.Println()
		fmt.Println("By State:")
		for state, count := range stats.ByState {
			fmt.Printf("  %-20s %d\n", state, count)
		}
	}

	return nil
}

func printRequestStatus(req *types.PromiseRequest) {
	fmt.Println()
	fmt.Printf("Request: %s\n", req.Metadata.Name)
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Promise:     %s\n", req.Spec.Promise)
	fmt.Printf("Team:        %s\n", req.Metadata.Team)
	fmt.Printf("Environment: %s\n", req.Metadata.Environment)
	fmt.Printf("State:       %s\n", req.Status.State)
	fmt.Printf("Message:     %s\n", req.Status.Message)
	fmt.Printf("Created:     %s\n", req.Status.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated:     %s\n", req.Status.UpdatedAt.Format("2006-01-02 15:04:05"))

	if req.Status.ApprovalInfo != nil {
		fmt.Println()
		fmt.Println("Approval Info:")
		fmt.Printf("  Required: %v\n", req.Status.ApprovalInfo.Required)
		if req.Status.ApprovalInfo.ApprovedBy != "" {
			fmt.Printf("  Approved By: %s\n", req.Status.ApprovalInfo.ApprovedBy)
		}
		if req.Status.ApprovalInfo.RejectedBy != "" {
			fmt.Printf("  Rejected By: %s\n", req.Status.ApprovalInfo.RejectedBy)
			fmt.Printf("  Reason: %s\n", req.Status.ApprovalInfo.Reason)
		}
	}

	if len(req.Spec.Inputs) > 0 {
		fmt.Println()
		fmt.Println("Inputs:")
		for k, v := range req.Spec.Inputs {
			fmt.Printf("  %s: %v\n", k, v)
		}
	}

	if len(req.Status.Outputs) > 0 {
		fmt.Println()
		fmt.Println("Outputs:")
		for k, v := range req.Status.Outputs {
			fmt.Printf("  %s: %v\n", k, v)
		}
	}
}

func printInstanceStatus(inst *types.PromiseInstance) {
	fmt.Println()
	fmt.Printf("Instance: %s\n", inst.Name)
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Promise:     %s\n", inst.Promise)
	fmt.Printf("Team:        %s\n", inst.Team)
	fmt.Printf("Environment: %s\n", inst.Environment)
	fmt.Printf("State:       %s\n", inst.State)
	fmt.Printf("Created:     %s\n", inst.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated:     %s\n", inst.UpdatedAt.Format("2006-01-02 15:04:05"))

	if len(inst.Inputs) > 0 {
		fmt.Println()
		fmt.Println("Inputs:")
		for k, v := range inst.Inputs {
			fmt.Printf("  %s: %v\n", k, v)
		}
	}

	if len(inst.Outputs) > 0 {
		fmt.Println()
		fmt.Println("Outputs:")
		for k, v := range inst.Outputs {
			fmt.Printf("  %s: %v\n", k, v)
		}
	}
}

func promiseTruncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
