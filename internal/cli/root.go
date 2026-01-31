package cli

import (
	"fmt"
	"os"

	"github.com/platformfoundry/platformfoundry-ce/pkg/extensions"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "pf",
	Short: "Platform Foundry - Infrastructure as YAML with Plugins",
	Long: `Platform Foundry is a declarative platform provisioning tool.
Define your infrastructure in YAML and let plugins handle the provisioning.

Examples:
  pf apply -f platform.yaml      Apply resources from a file
  pf get clusters                List all clusters
  pf delete pipeline jenkins-ci  Delete a specific resource
  pf plugin list                 List installed plugins`,
	Version: "0.1.0",
}

// Execute runs the root command
func Execute() {
	// Register extension commands
	for _, cmd := range extensions.GetCommands() {
		rootCmd.AddCommand(cmd)
	}

	// Initialize extensions
	if err := extensions.InitializeAll(); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to initialize extensions:", err)
		os.Exit(1)
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// GetRootCmd returns the root command for extension registration
func GetRootCmd() *cobra.Command {
	return rootCmd
}

func init() {
	// Add commands
	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(describeCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(pluginCmd)
	rootCmd.AddCommand(jobsCmd)
	rootCmd.AddCommand(waitCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(planCmd)
	rootCmd.AddCommand(rollbackCmd)
	rootCmd.AddCommand(orgCmd)
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(tlsCmd)
	rootCmd.AddCommand(secretsCmd)
	rootCmd.AddCommand(auditCmd)
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(policyCmd)
	rootCmd.AddCommand(complianceCmd)
	rootCmd.AddCommand(costCmd)
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(troubleshootCmd)
	rootCmd.AddCommand(serviceCmd)
	rootCmd.AddCommand(templateCmd)
	rootCmd.AddCommand(NewScaffoldCommand())
	rootCmd.AddCommand(workflowCmd)
	rootCmd.AddCommand(envCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(driftCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(graphCmd)
	rootCmd.AddCommand(lintCmd)
}
