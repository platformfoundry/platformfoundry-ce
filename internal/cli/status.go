package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show platform status overview",
	Long: `Display an overview of the current platform state including:
- Active resources and their health
- Recent deployments
- Pending approvals
- Active environments
- System health

Examples:
  pf status              # Show full status overview
  pf status --resources  # Show only resource status
  pf status --watch      # Continuously watch status`,
	RunE: runStatus,
}

var (
	statusResources bool
	statusWatch     bool
	statusNamespace string
)

func init() {
	statusCmd.Flags().BoolVar(&statusResources, "resources", false, "Show only resource status")
	statusCmd.Flags().BoolVarP(&statusWatch, "watch", "w", false, "Watch status continuously")
	statusCmd.Flags().StringVarP(&statusNamespace, "namespace", "n", "", "Filter by namespace")
}

func runStatus(cmd *cobra.Command, args []string) error {
	if statusWatch {
		return watchStatus(cmd)
	}

	return showStatus(cmd)
}

func showStatus(cmd *cobra.Command) error {
	fmt.Println("Platform Foundry Status")
	fmt.Println("=======================")
	fmt.Println()

	// Show system health
	showSystemHealth()

	if !statusResources {
		// Show recent activity
		showRecentActivity()

		// Show environment summary
		showEnvironmentSummary()

		// Show pending items
		showPendingItems()
	}

	// Show resource summary
	showResourceSummary()

	return nil
}

func showSystemHealth() {
	fmt.Println("System Health")
	fmt.Println("-------------")

	health := getSystemHealthStatus()

	for _, item := range health {
		icon := getHealthIcon(item.status)
		fmt.Printf("  %s %s: %s\n", icon, item.name, item.message)
	}
	fmt.Println()
}

type healthItem struct {
	name    string
	status  string
	message string
}

func getSystemHealthStatus() []healthItem {
	// In production, these would query actual systems
	return []healthItem{
		{name: "API Server", status: "healthy", message: "Running"},
		{name: "Plugin Registry", status: "healthy", message: "3 plugins loaded"},
		{name: "State Store", status: "healthy", message: "Connected"},
		{name: "Cluster Connection", status: "unknown", message: "Not configured"},
	}
}

func getHealthIcon(status string) string {
	switch status {
	case "healthy":
		return "[OK]"
	case "degraded":
		return "[WARN]"
	case "unhealthy":
		return "[ERR]"
	default:
		return "[--]"
	}
}

func showRecentActivity() {
	fmt.Println("Recent Activity (last 24h)")
	fmt.Println("--------------------------")

	activities := getRecentActivity()

	if len(activities) == 0 {
		fmt.Println("  No recent activity")
	} else {
		for _, act := range activities {
			fmt.Printf("  %s  %s - %s\n", formatStatusTimeAgo(act.timestamp), act.action, act.resource)
		}
	}
	fmt.Println()
}

type activity struct {
	timestamp time.Time
	action    string
	resource  string
}

func getRecentActivity() []activity {
	// In production, this would query activity logs
	return []activity{}
}

func formatStatusTimeAgo(t time.Time) string {
	diff := time.Since(t)

	if diff < time.Minute {
		return "just now"
	}
	if diff < time.Hour {
		mins := int(diff.Minutes())
		return fmt.Sprintf("%dm ago", mins)
	}
	if diff < 24*time.Hour {
		hours := int(diff.Hours())
		return fmt.Sprintf("%dh ago", hours)
	}
	days := int(diff.Hours() / 24)
	return fmt.Sprintf("%dd ago", days)
}

func showEnvironmentSummary() {
	fmt.Println("Environments")
	fmt.Println("------------")

	envs := getEnvironmentStatus()

	if len(envs) == 0 {
		fmt.Println("  No ephemeral environments active")
	} else {
		fmt.Printf("  %-20s %-12s %-15s %s\n", "NAME", "STATUS", "TTL", "OWNER")
		fmt.Printf("  %-20s %-12s %-15s %s\n", "----", "------", "---", "-----")
		for _, env := range envs {
			fmt.Printf("  %-20s %-12s %-15s %s\n", env.name, env.status, env.ttl, env.owner)
		}
	}
	fmt.Println()
}

type envStatus struct {
	name   string
	status string
	ttl    string
	owner  string
}

func getEnvironmentStatus() []envStatus {
	// In production, this would query environment manager
	return []envStatus{}
}

func showPendingItems() {
	fmt.Println("Pending Items")
	fmt.Println("-------------")

	pending := getPendingItems()

	if pending.approvals == 0 && pending.deployments == 0 && pending.alerts == 0 {
		fmt.Println("  No pending items")
	} else {
		if pending.approvals > 0 {
			fmt.Printf("  [!] %d pending approvals\n", pending.approvals)
		}
		if pending.deployments > 0 {
			fmt.Printf("  [>] %d deployments in progress\n", pending.deployments)
		}
		if pending.alerts > 0 {
			fmt.Printf("  [!] %d active alerts\n", pending.alerts)
		}
	}
	fmt.Println()
}

type pendingItems struct {
	approvals   int
	deployments int
	alerts      int
}

func getPendingItems() pendingItems {
	// In production, this would query various systems
	return pendingItems{}
}

func showResourceSummary() {
	fmt.Println("Resource Summary")
	fmt.Println("----------------")

	resources := getResourceSummary()

	if len(resources) == 0 {
		fmt.Println("  No resources managed")
		fmt.Println()
		fmt.Println("  Get started:")
		fmt.Println("    pf apply -f platform.yaml   # Deploy resources")
		fmt.Println("    pf doctor                   # Check system health")
	} else {
		fmt.Printf("  %-20s %-8s %-8s %-8s\n", "TYPE", "TOTAL", "HEALTHY", "ISSUES")
		fmt.Printf("  %-20s %-8s %-8s %-8s\n", "----", "-----", "-------", "------")
		for _, res := range resources {
			fmt.Printf("  %-20s %-8d %-8d %-8d\n", res.resourceType, res.total, res.healthy, res.issues)
		}
	}
	fmt.Println()
}

type resourceSummary struct {
	resourceType string
	total        int
	healthy      int
	issues       int
}

func getResourceSummary() []resourceSummary {
	// In production, this would query resource state
	return []resourceSummary{}
}

func watchStatus(cmd *cobra.Command) error {
	fmt.Println("Watching platform status (Ctrl+C to stop)...")
	fmt.Println()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Initial display
	clearScreen()
	if err := showStatus(cmd); err != nil {
		return err
	}

	for {
		select {
		case <-cmd.Context().Done():
			return nil
		case <-ticker.C:
			clearScreen()
			if err := showStatus(cmd); err != nil {
				return err
			}
			fmt.Printf("\nLast updated: %s\n", time.Now().Format("15:04:05"))
		}
	}
}

func clearScreen() {
	// ANSI escape to clear screen and move cursor to top
	fmt.Print("\033[H\033[2J")
}

// FormatStatus returns a formatted status string for embedding in other outputs
func FormatStatus() string {
	var sb strings.Builder

	sb.WriteString("Platform Foundry Status\n")
	sb.WriteString("=======================\n\n")

	health := getSystemHealthStatus()
	sb.WriteString("System Health:\n")
	for _, item := range health {
		icon := getHealthIcon(item.status)
		sb.WriteString(fmt.Sprintf("  %s %s: %s\n", icon, item.name, item.message))
	}

	return sb.String()
}
