package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/platformfoundry/pf-ce/internal/tenancy"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Note: JIT (just-in-time) access commands are in jit.go

var tenantManager *tenancy.Manager

func init() {
	tenantManager = tenancy.NewManager()
}

var tenantCmd = &cobra.Command{
	Use:     "tenant",
	Aliases: []string{"tenants"},
	Short:   "Manage tenants and multi-tenancy",
	Long:    `Manage tenants, quotas, roles, and access control for multi-tenant environments.`,
}

var tenantCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new tenant",
	Long: `Create a new tenant with specified configuration.

Examples:
  # Create tenant from file
  pf tenant create -f tenant.yaml

  # Create tenant with inline options
  pf tenant create --name acme-corp --isolation namespace --owners user@example.com`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath, _ := cmd.Flags().GetString("file")
		name, _ := cmd.Flags().GetString("name")
		isolation, _ := cmd.Flags().GetString("isolation")
		owners, _ := cmd.Flags().GetStringSlice("owners")

		var tenant *tenancy.Tenant

		if filePath != "" {
			data, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}

			tenant = &tenancy.Tenant{}
			if strings.HasSuffix(filePath, ".json") {
				err = json.Unmarshal(data, tenant)
			} else {
				err = yaml.Unmarshal(data, tenant)
			}
			if err != nil {
				return fmt.Errorf("failed to parse tenant: %w", err)
			}
		} else {
			if name == "" {
				return fmt.Errorf("--name is required")
			}

			tenant = &tenancy.Tenant{
				Metadata: tenancy.TenantMetadata{
					Name: name,
				},
				Spec: tenancy.TenantSpec{
					Isolation: tenancy.IsolationLevel(isolation),
					Owners:    owners,
				},
			}
		}

		ctx := context.Background()
		if err := tenantManager.CreateTenant(ctx, tenant); err != nil {
			return err
		}

		fmt.Printf("Tenant '%s' created successfully\n", tenant.Metadata.Name)
		return nil
	},
}

var tenantListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tenants",
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("output")

		tenants := tenantManager.ListTenants()

		if format == "json" {
			data, _ := json.MarshalIndent(tenants, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(tenants) == 0 {
			fmt.Println("No tenants found")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tSTATUS\tISOLATION\tOWNERS\tCREATED")
		for _, t := range tenants {
			owners := strings.Join(t.Spec.Owners, ", ")
			if len(owners) > 30 {
				owners = owners[:27] + "..."
			}
			status := "Active"
			if t.Status != nil {
				status = string(t.Status.Phase)
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				t.Metadata.Name,
				status,
				t.Spec.Isolation,
				owners,
				t.Metadata.CreatedAt.Format("2006-01-02"),
			)
		}
		w.Flush()

		return nil
	},
}

var tenantGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Get tenant details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		format, _ := cmd.Flags().GetString("output")

		tenant, err := tenantManager.GetTenant(name)
		if err != nil {
			return err
		}

		if format == "json" {
			data, _ := json.MarshalIndent(tenant, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if format == "yaml" {
			data, _ := yaml.Marshal(tenant)
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Name:        %s\n", tenant.Metadata.Name)
		fmt.Printf("ID:          %s\n", tenant.Metadata.ID)
		if tenant.Metadata.DisplayName != "" {
			fmt.Printf("Display:     %s\n", tenant.Metadata.DisplayName)
		}
		fmt.Printf("Isolation:   %s\n", tenant.Spec.Isolation)
		fmt.Printf("Created:     %s\n", tenant.Metadata.CreatedAt.Format(time.RFC3339))
		fmt.Printf("Updated:     %s\n", tenant.Metadata.UpdatedAt.Format(time.RFC3339))

		if len(tenant.Spec.Owners) > 0 {
			fmt.Printf("\nOwners:\n")
			for _, o := range tenant.Spec.Owners {
				fmt.Printf("  - %s\n", o)
			}
		}

		if tenant.Spec.Quotas != nil {
			fmt.Printf("\nQuotas:\n")
			if tenant.Spec.Quotas.CPU != "" {
				fmt.Printf("  CPU:          %s\n", tenant.Spec.Quotas.CPU)
			}
			if tenant.Spec.Quotas.Memory != "" {
				fmt.Printf("  Memory:       %s\n", tenant.Spec.Quotas.Memory)
			}
			if tenant.Spec.Quotas.Storage != "" {
				fmt.Printf("  Storage:      %s\n", tenant.Spec.Quotas.Storage)
			}
			if tenant.Spec.Quotas.Pods > 0 {
				fmt.Printf("  Pods:         %d\n", tenant.Spec.Quotas.Pods)
			}
			if tenant.Spec.Quotas.Environments > 0 {
				fmt.Printf("  Environments: %d\n", tenant.Spec.Quotas.Environments)
			}
		}

		if tenant.Status != nil {
			fmt.Printf("\nStatus:\n")
			fmt.Printf("  Phase: %s\n", tenant.Status.Phase)

			if tenant.Status.ResourceUsage != nil {
				fmt.Printf("\nResource Usage:\n")
				fmt.Printf("  CPU:     %s (%.1f%%)\n", tenant.Status.ResourceUsage.CPU, tenant.Status.ResourceUsage.CPUPercent)
				fmt.Printf("  Memory:  %s (%.1f%%)\n", tenant.Status.ResourceUsage.Memory, tenant.Status.ResourceUsage.MemoryPercent)
				fmt.Printf("  Storage: %s (%.1f%%)\n", tenant.Status.ResourceUsage.Storage, tenant.Status.ResourceUsage.StoragePercent)
				fmt.Printf("  Pods:    %d (%.1f%%)\n", tenant.Status.ResourceUsage.Pods, tenant.Status.ResourceUsage.PodsPercent)
			}
		}

		return nil
	},
}

var tenantDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a tenant",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		force, _ := cmd.Flags().GetBool("force")

		if !force {
			fmt.Printf("Are you sure you want to delete tenant '%s'? This cannot be undone.\n", name)
			fmt.Print("Type 'yes' to confirm: ")
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "yes" {
				fmt.Println("Aborted.")
				return nil
			}
		}

		ctx := context.Background()
		if err := tenantManager.DeleteTenant(ctx, name); err != nil {
			return err
		}

		fmt.Printf("Tenant '%s' deleted\n", name)
		return nil
	},
}

var tenantSuspendCmd = &cobra.Command{
	Use:   "suspend <name>",
	Short: "Suspend a tenant",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		reason, _ := cmd.Flags().GetString("reason")

		ctx := context.Background()
		if err := tenantManager.SuspendTenant(ctx, name, reason); err != nil {
			return err
		}

		fmt.Printf("Tenant '%s' suspended\n", name)
		return nil
	},
}

var tenantActivateCmd = &cobra.Command{
	Use:   "activate <name>",
	Short: "Activate a suspended tenant",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		ctx := context.Background()
		if err := tenantManager.ActivateTenant(ctx, name); err != nil {
			return err
		}

		fmt.Printf("Tenant '%s' activated\n", name)
		return nil
	},
}

// Role commands
var roleCmd = &cobra.Command{
	Use:   "role",
	Short: "Manage roles",
}

var roleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List roles",
	RunE: func(cmd *cobra.Command, args []string) error {
		tenantName, _ := cmd.Flags().GetString("tenant")
		if tenantName == "" {
			tenantName = "_global"
		}

		roles := tenantManager.ListRoles(tenantName)

		if len(roles) == 0 {
			fmt.Println("No roles found")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tTENANT\tDESCRIPTION\tPERMISSIONS")
		for _, r := range roles {
			tenant := r.Metadata.Tenant
			if tenant == "" || tenant == "_global" {
				tenant = "(global)"
			}
			perms := len(r.Spec.Permissions)
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\n",
				r.Metadata.Name,
				tenant,
				truncate(r.Metadata.Description, 40),
				perms,
			)
		}
		w.Flush()

		return nil
	},
}

var roleGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Get role details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		tenantName, _ := cmd.Flags().GetString("tenant")
		if tenantName == "" {
			tenantName = "_global"
		}

		role, err := tenantManager.GetRole(tenantName, name)
		if err != nil {
			return err
		}

		fmt.Printf("Name:        %s\n", role.Metadata.Name)
		if role.Metadata.Description != "" {
			fmt.Printf("Description: %s\n", role.Metadata.Description)
		}
		if role.Metadata.Tenant != "" && role.Metadata.Tenant != "_global" {
			fmt.Printf("Tenant:      %s\n", role.Metadata.Tenant)
		}

		fmt.Printf("\nPermissions:\n")
		for i, perm := range role.Spec.Permissions {
			fmt.Printf("  %d. Resources: %s\n", i+1, strings.Join(perm.Resources, ", "))
			fmt.Printf("     Verbs:     %s\n", strings.Join(perm.Verbs, ", "))
			if len(perm.Environments) > 0 {
				fmt.Printf("     Envs:      %s\n", strings.Join(perm.Environments, ", "))
			}
		}

		if len(role.Spec.Constraints) > 0 {
			fmt.Printf("\nConstraints:\n")
			for _, c := range role.Spec.Constraints {
				fmt.Printf("  - %s: %s\n", c.Type, c.Value)
			}
		}

		if len(role.Spec.InheritFrom) > 0 {
			fmt.Printf("\nInherits From: %s\n", strings.Join(role.Spec.InheritFrom, ", "))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(tenantCmd)

	// Create command
	tenantCreateCmd.Flags().StringP("file", "f", "", "Path to tenant configuration file")
	tenantCreateCmd.Flags().String("name", "", "Tenant name")
	tenantCreateCmd.Flags().String("isolation", "namespace", "Isolation level (namespace, cluster, vpc)")
	tenantCreateCmd.Flags().StringSlice("owners", nil, "Tenant owners")
	tenantCmd.AddCommand(tenantCreateCmd)

	// List command
	tenantListCmd.Flags().StringP("output", "o", "", "Output format (json)")
	tenantCmd.AddCommand(tenantListCmd)

	// Get command
	tenantGetCmd.Flags().StringP("output", "o", "", "Output format (json, yaml)")
	tenantCmd.AddCommand(tenantGetCmd)

	// Delete command
	tenantDeleteCmd.Flags().BoolP("force", "f", false, "Force delete without confirmation")
	tenantCmd.AddCommand(tenantDeleteCmd)

	// Suspend command
	tenantSuspendCmd.Flags().String("reason", "", "Suspension reason")
	tenantCmd.AddCommand(tenantSuspendCmd)

	// Activate command
	tenantCmd.AddCommand(tenantActivateCmd)

	// Role commands
	tenantCmd.AddCommand(roleCmd)
	roleListCmd.Flags().String("tenant", "", "Filter by tenant")
	roleCmd.AddCommand(roleListCmd)
	roleGetCmd.Flags().String("tenant", "", "Tenant name")
	roleCmd.AddCommand(roleGetCmd)
}
