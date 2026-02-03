package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/platformfoundry/pf-ce/internal/policy"
	"github.com/spf13/cobra"
)

var (
	policyType     string
	policyEndpoint string
	policyDir      string
	policyInput    string
)

var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Policy management",
	Long:  `Manage policies for governance and authorization using Open Policy Agent.`,
}

var policyLoadCmd = &cobra.Command{
	Use:   "load <name> <file>",
	Short: "Load a policy",
	Long:  `Load a policy from a Rego file.`,
	Example: `  pf policy load rbac rbac.rego
  pf policy load authz /path/to/authz.rego`,
	Args: cobra.ExactArgs(2),
	RunE: runPolicyLoad,
}

var policyDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a policy",
	Long:  `Delete a loaded policy.`,
	Example: `  pf policy delete rbac
  pf policy delete authz`,
	Args: cobra.ExactArgs(1),
	RunE: runPolicyDelete,
}

var policyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all policies",
	Long:  `List all loaded policies.`,
	Example: `  pf policy list
  pf policy list --type opa`,
	RunE: runPolicyList,
}

var policyEvalCmd = &cobra.Command{
	Use:   "eval <policy> <input>",
	Short: "Evaluate a policy",
	Long:  `Evaluate a policy against input data.`,
	Example: `  pf policy eval rbac '{"user":"alice","action":"read","resource":"document"}'
  pf policy eval authz --input input.json`,
	Args: cobra.MinimumNArgs(1),
	RunE: runPolicyEval,
}

func init() {
	// Common flags
	policyLoadCmd.Flags().StringVar(&policyType, "type", "local", "Policy engine type (local, opa)")
	policyLoadCmd.Flags().StringVar(&policyEndpoint, "endpoint", "http://localhost:8181", "OPA endpoint")
	policyLoadCmd.Flags().StringVar(&policyDir, "dir", "/etc/platformfoundry/policies", "Policy directory (for local)")

	policyDeleteCmd.Flags().StringVar(&policyType, "type", "local", "Policy engine type (local, opa)")
	policyDeleteCmd.Flags().StringVar(&policyEndpoint, "endpoint", "http://localhost:8181", "OPA endpoint")
	policyDeleteCmd.Flags().StringVar(&policyDir, "dir", "/etc/platformfoundry/policies", "Policy directory (for local)")

	policyListCmd.Flags().StringVar(&policyType, "type", "local", "Policy engine type (local, opa)")
	policyListCmd.Flags().StringVar(&policyEndpoint, "endpoint", "http://localhost:8181", "OPA endpoint")
	policyListCmd.Flags().StringVar(&policyDir, "dir", "/etc/platformfoundry/policies", "Policy directory (for local)")

	policyEvalCmd.Flags().StringVar(&policyType, "type", "local", "Policy engine type (local, opa)")
	policyEvalCmd.Flags().StringVar(&policyEndpoint, "endpoint", "http://localhost:8181", "OPA endpoint")
	policyEvalCmd.Flags().StringVar(&policyDir, "dir", "/etc/platformfoundry/policies", "Policy directory (for local)")
	policyEvalCmd.Flags().StringVar(&policyInput, "input", "", "Input JSON file")

	// Add subcommands
	policyCmd.AddCommand(policyLoadCmd)
	policyCmd.AddCommand(policyDeleteCmd)
	policyCmd.AddCommand(policyListCmd)
	policyCmd.AddCommand(policyEvalCmd)
}

func runPolicyLoad(cmd *cobra.Command, args []string) error {
	name := args[0]
	file := args[1]

	engine, err := createPolicyEngine()
	if err != nil {
		return err
	}
	defer engine.Close()

	// Read policy file
	content, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read policy file: %w", err)
	}

	ctx := context.Background()
	if err := engine.LoadPolicy(ctx, name, string(content)); err != nil {
		return fmt.Errorf("failed to load policy: %w", err)
	}

	fmt.Printf("✓ Policy loaded: %s\n", name)
	fmt.Printf("  Source: %s\n", file)

	return nil
}

func runPolicyDelete(cmd *cobra.Command, args []string) error {
	name := args[0]

	engine, err := createPolicyEngine()
	if err != nil {
		return err
	}
	defer engine.Close()

	ctx := context.Background()
	if err := engine.DeletePolicy(ctx, name); err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	fmt.Printf("✓ Policy deleted: %s\n", name)

	return nil
}

func runPolicyList(cmd *cobra.Command, args []string) error {
	engine, err := createPolicyEngine()
	if err != nil {
		return err
	}
	defer engine.Close()

	ctx := context.Background()
	policies, err := engine.ListPolicies(ctx)
	if err != nil {
		return fmt.Errorf("failed to list policies: %w", err)
	}

	if len(policies) == 0 {
		fmt.Println("No policies found")
		return nil
	}

	fmt.Printf("Policies (%d):\n", len(policies))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for i, p := range policies {
		fmt.Printf("%d. %s\n", i+1, p)
	}

	return nil
}

func runPolicyEval(cmd *cobra.Command, args []string) error {
	policyName := args[0]

	engine, err := createPolicyEngine()
	if err != nil {
		return err
	}
	defer engine.Close()

	// Parse input
	var input interface{}

	if policyInput != "" {
		// Load from file
		data, err := os.ReadFile(policyInput)
		if err != nil {
			return fmt.Errorf("failed to read input file: %w", err)
		}

		if err := json.Unmarshal(data, &input); err != nil {
			return fmt.Errorf("failed to parse input JSON: %w", err)
		}
	} else if len(args) > 1 {
		// Parse from argument
		if err := json.Unmarshal([]byte(args[1]), &input); err != nil {
			return fmt.Errorf("failed to parse input JSON: %w", err)
		}
	} else {
		// Empty input
		input = map[string]interface{}{}
	}

	ctx := context.Background()
	result, err := engine.Evaluate(ctx, policyName, input)
	if err != nil {
		return fmt.Errorf("failed to evaluate policy: %w", err)
	}

	fmt.Println("Policy Evaluation Result:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Policy: %s\n", policyName)

	if result.Allowed {
		fmt.Println("Decision: ✓ ALLOWED")
	} else {
		fmt.Println("Decision: ✗ DENIED")
	}

	if len(result.Reasons) > 0 {
		fmt.Println("\nReasons:")
		for _, reason := range result.Reasons {
			fmt.Printf("  - %s\n", reason)
		}
	}

	if len(result.Data) > 0 {
		fmt.Println("\nData:")
		data, err := json.MarshalIndent(result.Data, "  ", "  ")
		if err == nil {
			fmt.Printf("  %s\n", string(data))
		}
	}

	return nil
}

// Helper functions

func createPolicyEngine() (policy.Engine, error) {
	config := &policy.Config{
		Type:        policyType,
		OPAEndpoint: policyEndpoint,
		OPATimeout:  5,
		PolicyDir:   policyDir,
	}

	return policy.NewEngine(config)
}
