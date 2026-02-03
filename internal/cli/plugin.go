package cli

import (
	"fmt"

	"github.com/platformfoundry/pf-ce/internal/plugin"
	"github.com/platformfoundry/pf-ce/internal/plugins/clusterexisting"
	"github.com/spf13/cobra"
)

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage plugins",
	Long:  `Manage Platform Foundry plugins.`,
}

var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed plugins",
	Long:  `List all installed plugins and their versions.`,
	RunE:  runPluginList,
}

func init() {
	pluginCmd.AddCommand(pluginListCmd)
}

func runPluginList(cmd *cobra.Command, args []string) error {
	// Initialize plugin manager
	pm := plugin.NewManager()

	// Register built-in plugins
	pm.Register(clusterexisting.NewPlugin())

	plugins := pm.List()

	if len(plugins) == 0 {
		fmt.Println("No plugins installed")
		return nil
	}

	fmt.Printf("%-20s %-15s %-10s %s\n", "NAME", "TYPE", "VERSION", "SOURCE")
	fmt.Println("------------------------------------------------------------")

	for _, p := range plugins {
		fmt.Printf("%-20s %-15s %-10s %s\n", p.Name(), p.Type(), p.Version(), "built-in")
	}

	return nil
}
