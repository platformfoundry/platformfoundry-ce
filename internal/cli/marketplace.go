package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/platformfoundry/pf-ce/internal/marketplace"
	"github.com/spf13/cobra"
)

var marketplaceClient *marketplace.Client

func init() {
	marketplaceClient = marketplace.NewClient(marketplace.ClientConfig{})
}

var marketplaceCmd = &cobra.Command{
	Use:     "marketplace",
	Aliases: []string{"market", "mp"},
	Short:   "Plugin marketplace",
	Long:    `Browse, search, and manage plugins from the PlatformFoundry marketplace.`,
}

var marketplaceSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for plugins",
	RunE:  runMarketplaceSearch,
}

var marketplaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available plugins",
	RunE:  runMarketplaceList,
}

var marketplaceShowCmd = &cobra.Command{
	Use:   "show <plugin>",
	Short: "Show plugin details",
	Args:  cobra.ExactArgs(1),
	RunE:  runMarketplaceShow,
}

var marketplaceInstallCmd = &cobra.Command{
	Use:   "install <plugin> [version]",
	Short: "Install a plugin",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runMarketplaceInstall,
}

var marketplaceUninstallCmd = &cobra.Command{
	Use:   "uninstall <plugin>",
	Short: "Uninstall a plugin",
	Args:  cobra.ExactArgs(1),
	RunE:  runMarketplaceUninstall,
}

var marketplaceUpdateCmd = &cobra.Command{
	Use:   "update [plugin]",
	Short: "Update plugin(s)",
	RunE:  runMarketplaceUpdate,
}

var marketplaceInstalledCmd = &cobra.Command{
	Use:   "installed",
	Short: "List installed plugins",
	RunE:  runMarketplaceInstalled,
}

var marketplaceCategoriesCmd = &cobra.Command{
	Use:   "categories",
	Short: "List plugin categories",
	RunE:  runMarketplaceCategories,
}

var (
	mpCategory   string
	mpVerified   bool
	mpSortBy     string
	mpLimit      int
)

func init() {
	marketplaceCmd.AddCommand(marketplaceSearchCmd)
	marketplaceCmd.AddCommand(marketplaceListCmd)
	marketplaceCmd.AddCommand(marketplaceShowCmd)
	marketplaceCmd.AddCommand(marketplaceInstallCmd)
	marketplaceCmd.AddCommand(marketplaceUninstallCmd)
	marketplaceCmd.AddCommand(marketplaceUpdateCmd)
	marketplaceCmd.AddCommand(marketplaceInstalledCmd)
	marketplaceCmd.AddCommand(marketplaceCategoriesCmd)

	marketplaceSearchCmd.Flags().StringVarP(&mpCategory, "category", "c", "", "Filter by category")
	marketplaceSearchCmd.Flags().BoolVar(&mpVerified, "verified", false, "Only show verified plugins")
	marketplaceSearchCmd.Flags().StringVar(&mpSortBy, "sort", "downloads", "Sort by: downloads, rating, updated, name")
	marketplaceSearchCmd.Flags().IntVarP(&mpLimit, "limit", "l", 20, "Number of results")

	marketplaceListCmd.Flags().StringVarP(&mpCategory, "category", "c", "", "Filter by category")
	marketplaceListCmd.Flags().BoolVar(&mpVerified, "verified", false, "Only show verified plugins")
	marketplaceListCmd.Flags().StringVar(&mpSortBy, "sort", "downloads", "Sort by: downloads, rating, updated, name")
	marketplaceListCmd.Flags().IntVarP(&mpLimit, "limit", "l", 20, "Number of results")
}

func runMarketplaceSearch(cmd *cobra.Command, args []string) error {
	query := marketplace.SearchQuery{
		SortBy: mpSortBy,
		Limit:  mpLimit,
	}

	if len(args) > 0 {
		query.Query = args[0]
	}

	if mpCategory != "" {
		query.Categories = []string{mpCategory}
	}

	if mpVerified {
		query.Verified = &mpVerified
	}

	ctx := context.Background()
	result, err := marketplaceClient.Search(ctx, query)
	if err != nil {
		return err
	}

	if len(result.Plugins) == 0 {
		fmt.Println("No plugins found")
		return nil
	}

	fmt.Printf("Found %d plugins", result.Total)
	if query.Query != "" {
		fmt.Printf(" for \"%s\"", query.Query)
	}
	fmt.Println()
	fmt.Println(strings.Repeat("-", 80))

	printPluginList(result.Plugins)

	return nil
}

func runMarketplaceList(cmd *cobra.Command, args []string) error {
	query := marketplace.SearchQuery{
		SortBy: mpSortBy,
		Limit:  mpLimit,
	}

	if mpCategory != "" {
		query.Categories = []string{mpCategory}
	}

	if mpVerified {
		query.Verified = &mpVerified
	}

	ctx := context.Background()
	result, err := marketplaceClient.Search(ctx, query)
	if err != nil {
		return err
	}

	if len(result.Plugins) == 0 {
		fmt.Println("No plugins available")
		return nil
	}

	fmt.Printf("Available Plugins (%d total)\n", result.Total)
	fmt.Println(strings.Repeat("=", 80))

	printPluginList(result.Plugins)

	return nil
}

func runMarketplaceShow(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	plugin, err := marketplaceClient.GetPlugin(ctx, args[0])
	if err != nil {
		return err
	}

	fmt.Printf("%s\n", plugin.Name)
	fmt.Println(strings.Repeat("=", 60))

	fmt.Printf("\nVersion:     %s\n", plugin.Version)
	fmt.Printf("Description: %s\n", plugin.Description)
	fmt.Printf("Author:      %s", plugin.Author.Name)
	if plugin.Author.Verified {
		fmt.Print(" (Verified)")
	}
	fmt.Println()
	fmt.Printf("License:     %s\n", plugin.License)

	if len(plugin.Categories) > 0 {
		fmt.Printf("Categories:  %s\n", strings.Join(plugin.Categories, ", "))
	}

	if len(plugin.Keywords) > 0 {
		fmt.Printf("Keywords:    %s\n", strings.Join(plugin.Keywords, ", "))
	}

	fmt.Printf("\nDownloads:   %d\n", plugin.Downloads)
	fmt.Printf("Rating:      %.1f/5.0 (%d reviews)\n", plugin.Rating, plugin.RatingCount)
	fmt.Printf("Published:   %s\n", plugin.PublishedAt.Format("2006-01-02"))
	fmt.Printf("Updated:     %s\n", plugin.UpdatedAt.Format("2006-01-02"))

	if len(plugin.Capabilities) > 0 {
		fmt.Printf("\nCapabilities:\n")
		for _, cap := range plugin.Capabilities {
			fmt.Printf("  - %s\n", cap)
		}
	}

	// Check if installed
	installed, _ := marketplaceClient.ListInstalled(ctx)
	for _, p := range installed {
		if p.Name == plugin.Name {
			fmt.Printf("\nInstalled: v%s", p.Version)
			if p.HasUpdate {
				fmt.Printf(" (update available: v%s)", p.LatestVersion)
			}
			fmt.Println()
			break
		}
	}

	fmt.Printf("\nInstall:     pf marketplace install %s\n", plugin.Name)

	return nil
}

func runMarketplaceInstall(cmd *cobra.Command, args []string) error {
	name := args[0]
	version := ""
	if len(args) > 1 {
		version = args[1]
	}

	ctx := context.Background()

	// Check if plugin exists
	plugin, err := marketplaceClient.GetPlugin(ctx, name)
	if err != nil {
		return err
	}

	if version == "" {
		version = plugin.Version
	}

	fmt.Printf("Installing %s@%s...\n", name, version)

	if err := marketplaceClient.Install(ctx, name, version); err != nil {
		return err
	}

	fmt.Printf("Successfully installed %s@%s\n", name, version)
	return nil
}

func runMarketplaceUninstall(cmd *cobra.Command, args []string) error {
	name := args[0]
	ctx := context.Background()

	fmt.Printf("Uninstalling %s...\n", name)

	if err := marketplaceClient.Uninstall(ctx, name); err != nil {
		return err
	}

	fmt.Printf("Successfully uninstalled %s\n", name)
	return nil
}

func runMarketplaceUpdate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if len(args) > 0 {
		// Update specific plugin
		name := args[0]
		fmt.Printf("Updating %s...\n", name)
		if err := marketplaceClient.Update(ctx, name); err != nil {
			return err
		}
		fmt.Printf("Successfully updated %s\n", name)
		return nil
	}

	// Check for updates
	updates, err := marketplaceClient.CheckUpdates(ctx)
	if err != nil {
		return err
	}

	if len(updates) == 0 {
		fmt.Println("All plugins are up to date")
		return nil
	}

	fmt.Printf("Found %d updates:\n", len(updates))
	for name, version := range updates {
		fmt.Printf("  %s -> %s\n", name, version)
	}

	// Update all
	for name := range updates {
		fmt.Printf("Updating %s...\n", name)
		if err := marketplaceClient.Update(ctx, name); err != nil {
			fmt.Printf("  Failed: %v\n", err)
		} else {
			fmt.Printf("  Done\n")
		}
	}

	return nil
}

func runMarketplaceInstalled(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	installed, err := marketplaceClient.ListInstalled(ctx)
	if err != nil {
		return err
	}

	if len(installed) == 0 {
		fmt.Println("No plugins installed")
		return nil
	}

	fmt.Printf("Installed Plugins (%d)\n", len(installed))
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("%-25s %-12s %-10s %-12s %s\n", "NAME", "VERSION", "ENABLED", "AUTO-UPDATE", "STATUS")
	fmt.Println(strings.Repeat("-", 70))

	for _, p := range installed {
		enabled := "Yes"
		if !p.Enabled {
			enabled = "No"
		}

		autoUpdate := "Yes"
		if !p.AutoUpdate {
			autoUpdate = "No"
		}

		status := "Up to date"
		if p.HasUpdate {
			status = fmt.Sprintf("Update: %s", p.LatestVersion)
		}

		fmt.Printf("%-25s %-12s %-10s %-12s %s\n",
			p.Name,
			p.Version,
			enabled,
			autoUpdate,
			status)
	}

	return nil
}

func runMarketplaceCategories(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	categories, err := marketplaceClient.GetCategories(ctx)
	if err != nil {
		return err
	}

	fmt.Println("Plugin Categories")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("%-20s %-35s %s\n", "CATEGORY", "DESCRIPTION", "PLUGINS")
	fmt.Println(strings.Repeat("-", 60))

	for _, cat := range categories {
		fmt.Printf("%-20s %-35s %d\n",
			cat.Name,
			truncateMarketplace(cat.Description, 35),
			cat.Count)
	}

	return nil
}

func printPluginList(plugins []marketplace.Plugin) {
	fmt.Printf("%-25s %-10s %-6s %-10s %s\n", "NAME", "VERSION", "RATING", "DOWNLOADS", "DESCRIPTION")
	fmt.Println(strings.Repeat("-", 80))

	for _, p := range plugins {
		verified := ""
		if p.Verified {
			verified = "*"
		}

		fmt.Printf("%-25s %-10s %-6.1f %-10d %s\n",
			p.Name+verified,
			p.Version,
			p.Rating,
			p.Downloads,
			truncateMarketplace(p.Description, 30))
	}

	fmt.Println()
	fmt.Println("* = Verified publisher")
}

func truncateMarketplace(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// GetMarketplaceCmd returns the marketplace command for registration
func GetMarketplaceCmd() *cobra.Command {
	return marketplaceCmd
}
