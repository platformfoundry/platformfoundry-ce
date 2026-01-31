package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/platformfoundry/platformfoundry-ce/internal/audit"
	"github.com/spf13/cobra"
)

var (
	auditLogFile   string
	auditStartTime string
	auditEndTime   string
	auditTypes     []string
	auditSeverity  []string
	auditUsername  string
	auditStatus    string
	auditResource  string
	auditLimit     int
	auditFollow    bool
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Audit log management",
	Long:  `View and search audit logs for platform operations.`,
}

var auditViewCmd = &cobra.Command{
	Use:   "view",
	Short: "View audit logs",
	Long:  `View audit logs with optional filters.`,
	Example: `  pf audit view --limit 50
  pf audit view --username admin --severity error
  pf audit view --start "2024-01-01" --end "2024-01-31"
  pf audit view --type auth.login --status failure
  pf audit view --resource platform/prod-cluster`,
	RunE: runAuditView,
}

var auditSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search audit logs",
	Long:  `Search audit logs for specific events or patterns.`,
	Example: `  pf audit search "login failed"
  pf audit search "cluster" --type resource.create
  pf audit search "error" --severity error`,
	Args: cobra.ExactArgs(1),
	RunE: runAuditSearch,
}

var auditStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show audit statistics",
	Long:  `Display statistics about audit events.`,
	Example: `  pf audit stats
  pf audit stats --start "2024-01-01"
  pf audit stats --username admin`,
	RunE: runAuditStats,
}

func init() {
	// View command flags
	auditViewCmd.Flags().StringVar(&auditLogFile, "log-file", "", "Path to audit log file")
	auditViewCmd.Flags().StringVar(&auditStartTime, "start", "", "Start time (RFC3339 format or YYYY-MM-DD)")
	auditViewCmd.Flags().StringVar(&auditEndTime, "end", "", "End time (RFC3339 format or YYYY-MM-DD)")
	auditViewCmd.Flags().StringSliceVar(&auditTypes, "type", []string{}, "Filter by event types")
	auditViewCmd.Flags().StringSliceVar(&auditSeverity, "severity", []string{}, "Filter by severity (info, warning, error, critical)")
	auditViewCmd.Flags().StringVar(&auditUsername, "username", "", "Filter by username")
	auditViewCmd.Flags().StringVar(&auditStatus, "status", "", "Filter by status (success, failure, partial)")
	auditViewCmd.Flags().StringVar(&auditResource, "resource", "", "Filter by resource ID")
	auditViewCmd.Flags().IntVar(&auditLimit, "limit", 100, "Maximum number of events to display")
	auditViewCmd.Flags().BoolVar(&auditFollow, "follow", false, "Follow log file for new events")

	// Search command flags
	auditSearchCmd.Flags().StringVar(&auditLogFile, "log-file", "", "Path to audit log file")
	auditSearchCmd.Flags().StringSliceVar(&auditTypes, "type", []string{}, "Filter by event types")
	auditSearchCmd.Flags().StringSliceVar(&auditSeverity, "severity", []string{}, "Filter by severity")
	auditSearchCmd.Flags().IntVar(&auditLimit, "limit", 100, "Maximum number of events to display")

	// Stats command flags
	auditStatsCmd.Flags().StringVar(&auditLogFile, "log-file", "", "Path to audit log file")
	auditStatsCmd.Flags().StringVar(&auditStartTime, "start", "", "Start time (RFC3339 format or YYYY-MM-DD)")
	auditStatsCmd.Flags().StringVar(&auditEndTime, "end", "", "End time (RFC3339 format or YYYY-MM-DD)")
	auditStatsCmd.Flags().StringVar(&auditUsername, "username", "", "Filter by username")

	// Add subcommands
	auditCmd.AddCommand(auditViewCmd)
	auditCmd.AddCommand(auditSearchCmd)
	auditCmd.AddCommand(auditStatsCmd)
}

func runAuditView(cmd *cobra.Command, args []string) error {
	logger, err := createAuditLogger()
	if err != nil {
		return err
	}
	defer logger.Close()

	filters := buildAuditFilters()

	events, err := logger.Query(filters, auditLimit)
	if err != nil {
		return fmt.Errorf("failed to query audit logs: %w", err)
	}

	if len(events) == 0 {
		fmt.Println("No audit events found matching the criteria")
		return nil
	}

	fmt.Printf("Audit Events (%d):\n", len(events))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for _, event := range events {
		printAuditEvent(&event, false)
		fmt.Println()
	}

	return nil
}

func runAuditSearch(cmd *cobra.Command, args []string) error {
	query := args[0]

	logger, err := createAuditLogger()
	if err != nil {
		return err
	}
	defer logger.Close()

	filters := buildAuditFilters()

	events, err := logger.Query(filters, auditLimit)
	if err != nil {
		return fmt.Errorf("failed to query audit logs: %w", err)
	}

	// Filter events by query string
	matchedEvents := make([]audit.Event, 0)
	queryLower := strings.ToLower(query)

	for _, event := range events {
		if matchesQuery(&event, queryLower) {
			matchedEvents = append(matchedEvents, event)
		}
	}

	if len(matchedEvents) == 0 {
		fmt.Printf("No audit events found matching query: %s\n", query)
		return nil
	}

	fmt.Printf("Search Results for '%s' (%d events):\n", query, len(matchedEvents))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for _, event := range matchedEvents {
		printAuditEvent(&event, true)
		fmt.Println()
	}

	return nil
}

func runAuditStats(cmd *cobra.Command, args []string) error {
	logger, err := createAuditLogger()
	if err != nil {
		return err
	}
	defer logger.Close()

	stats, err := logger.GetStats()
	if err != nil {
		return fmt.Errorf("failed to get audit stats: %w", err)
	}

	totalEvents := stats["total_events"].(int)
	if totalEvents == 0 {
		fmt.Println("No audit events found")
		return nil
	}

	fmt.Println("Audit Statistics:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Total Events: %d\n\n", totalEvents)

	fmt.Println("By Type:")
	byType := stats["by_type"].(map[audit.EventType]int)
	for t, count := range byType {
		fmt.Printf("  %-30s %6d\n", t, count)
	}

	fmt.Println("\nBy Resource:")
	byResource := stats["by_resource"].(map[audit.ResourceType]int)
	for r, count := range byResource {
		fmt.Printf("  %-30s %6d\n", r, count)
	}

	fmt.Println("\nBy Status:")
	byStatus := stats["by_status"].(map[string]int)
	for s, count := range byStatus {
		fmt.Printf("  %-10s %6d\n", s, count)
	}

	fmt.Println("\nBy User:")
	byUser := stats["by_user"].(map[string]int)
	for u, count := range byUser {
		if u == "" {
			u = "(system)"
		}
		fmt.Printf("  %-30s %6d\n", u, count)
	}

	return nil
}

// Helper functions

func createAuditLogger() (*audit.Logger, error) {
	config := audit.Config{
		DestType:      "file",
		Destination:   auditLogFile,
		MaxBufferSize: 100,
		Retention:     90 * 24 * time.Hour,
	}

	if config.Destination == "" {
		config.Destination = "/var/log/platformfoundry/audit.log"
	}

	return audit.NewLogger(config)
}

func buildAuditFilters() map[string]interface{} {
	filters := make(map[string]interface{})

	// Parse time range
	if auditStartTime != "" {
		start, err := parseTime(auditStartTime)
		if err == nil {
			filters["after"] = start
		}
	}

	if auditEndTime != "" {
		end, err := parseTime(auditEndTime)
		if err == nil {
			filters["before"] = end
		}
	}

	// Event types
	if len(auditTypes) > 0 {
		filters["event_type"] = audit.EventType(auditTypes[0])
	}

	// Username
	if auditUsername != "" {
		filters["user"] = auditUsername
	}

	// Status
	if auditStatus != "" {
		filters["status"] = auditStatus
	}

	// Resource
	if auditResource != "" {
		filters["resource_name"] = auditResource
	}

	return filters
}

func parseTime(timeStr string) (time.Time, error) {
	// Try RFC3339 format first
	t, err := time.Parse(time.RFC3339, timeStr)
	if err == nil {
		return t, nil
	}

	// Try YYYY-MM-DD format
	t, err = time.Parse("2006-01-02", timeStr)
	if err == nil {
		return t, nil
	}

	// Try YYYY-MM-DD HH:MM:SS format
	t, err = time.Parse("2006-01-02 15:04:05", timeStr)
	if err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("unsupported time format (use RFC3339 or YYYY-MM-DD)")
}

func printAuditEvent(event *audit.Event, compact bool) {
	statusSymbol := getStatusSymbol(event.Status)

	if compact {
		fmt.Printf("[%s] %s | %s | %s | %s",
			event.Timestamp.Format("2006-01-02 15:04:05"),
			statusSymbol,
			event.EventType,
			event.User,
			event.Action,
		)
		if event.ResourceName != "" {
			fmt.Printf(" | %s", event.ResourceName)
		}
	} else {
		fmt.Printf("%s [%s] %s\n", event.Timestamp.Format("2006-01-02 15:04:05"), event.ID, statusSymbol)
		fmt.Printf("  Type: %s\n", event.EventType)
		fmt.Printf("  Action: %s\n", event.Action)
		if event.User != "" {
			fmt.Printf("  User: %s\n", event.User)
		}
		if event.ResourceName != "" {
			fmt.Printf("  Resource: %s (%s)\n", event.ResourceName, event.ResourceType)
		}
		if event.Message != "" {
			fmt.Printf("  Message: %s\n", event.Message)
		}
		if event.IPAddress != "" {
			fmt.Printf("  IP Address: %s\n", event.IPAddress)
		}
		if event.UserAgent != "" {
			fmt.Printf("  User Agent: %s\n", event.UserAgent)
		}
	}
}

func getStatusSymbol(status string) string {
	switch status {
	case "success":
		return "✓"
	case "failed":
		return "✗"
	default:
		return "•"
	}
}

func matchesQuery(event *audit.Event, query string) bool {
	// Check all searchable fields
	searchable := []string{
		string(event.EventType),
		event.Action,
		event.Message,
		event.User,
		event.ResourceName,
		string(event.ResourceType),
	}

	for _, field := range searchable {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}

	return false
}
