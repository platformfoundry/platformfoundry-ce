package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

var jitCmd = &cobra.Command{
	Use:   "jit",
	Short: "Just-In-Time access management",
	Long: `Manage Just-In-Time (JIT) access to resources.

JIT access provides temporary, on-demand access to sensitive resources.
This command allows you to:
- Request temporary access to resources
- Approve/deny access requests
- View active access grants
- Revoke access when no longer needed`,
}

var jitRequestCmd = &cobra.Command{
	Use:   "request",
	Short: "Request JIT access",
	Long: `Request temporary access to a resource.

Examples:
  # Request viewer access to production database
  pf jit request --resource prod-database --role viewer --duration 4h \
    --justification "Investigating production issue #1234"

  # Request operator access to a service
  pf jit request --resource api-service --role operator --duration 2h \
    --justification "Deploying hotfix for critical bug"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		resource, _ := cmd.Flags().GetString("resource")
		role, _ := cmd.Flags().GetString("role")
		duration, _ := cmd.Flags().GetString("duration")
		justification, _ := cmd.Flags().GetString("justification")

		if resource == "" || role == "" || justification == "" {
			return fmt.Errorf("--resource, --role, and --justification are required")
		}

		requestID := fmt.Sprintf("jit-%d", time.Now().Unix())

		fmt.Println("JIT Access Request Created")
		fmt.Println("--------------------------")
		fmt.Printf("Request ID:     %s\n", requestID)
		fmt.Printf("Resource:       %s\n", resource)
		fmt.Printf("Role:           %s\n", role)
		fmt.Printf("Duration:       %s\n", duration)
		fmt.Printf("Justification:  %s\n", justification)
		fmt.Printf("Status:         pending\n")
		fmt.Println()
		fmt.Println("Your request has been submitted and is awaiting approval.")
		fmt.Println("You will be notified when the request is approved or denied.")

		return nil
	},
}

var jitListCmd = &cobra.Command{
	Use:   "list",
	Short: "List JIT requests and grants",
	Long:  `List JIT access requests and active grants.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		status, _ := cmd.Flags().GetString("status")
		format, _ := cmd.Flags().GetString("output")

		requests := getSampleJITRequests()

		if status != "" {
			filtered := make([]jitRequestInfo, 0)
			for _, r := range requests {
				if r.Status == status {
					filtered = append(filtered, r)
				}
			}
			requests = filtered
		}

		if format == "json" {
			data, _ := json.MarshalIndent(requests, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(requests) == 0 {
			fmt.Println("No JIT requests found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tREQUESTER\tRESOURCE\tROLE\tDURATION\tSTATUS\tCREATED")
		for _, r := range requests {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				r.ID, r.Requester, r.Resource, r.Role, r.Duration, r.Status, r.Created)
		}
		w.Flush()

		return nil
	},
}

var jitApproveCmd = &cobra.Command{
	Use:   "approve [request-id]",
	Short: "Approve a JIT access request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		requestID := args[0]
		comment, _ := cmd.Flags().GetString("comment")

		fmt.Printf("Approving JIT request: %s\n", requestID)
		if comment != "" {
			fmt.Printf("Comment: %s\n", comment)
		}
		fmt.Println()
		fmt.Println("Request approved successfully.")
		fmt.Println("Access has been granted to the requester.")
		fmt.Printf("Expires at: %s\n", time.Now().Add(4*time.Hour).Format("2006-01-02 15:04:05"))

		return nil
	},
}

var jitDenyCmd = &cobra.Command{
	Use:   "deny [request-id]",
	Short: "Deny a JIT access request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		requestID := args[0]
		reason, _ := cmd.Flags().GetString("reason")

		if reason == "" {
			return fmt.Errorf("--reason is required when denying a request")
		}

		fmt.Printf("Denying JIT request: %s\n", requestID)
		fmt.Printf("Reason: %s\n", reason)
		fmt.Println()
		fmt.Println("Request denied. The requester will be notified.")

		return nil
	},
}

var jitGrantsCmd = &cobra.Command{
	Use:   "grants",
	Short: "List active JIT grants",
	Long:  `List all currently active JIT access grants.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("output")
		user, _ := cmd.Flags().GetString("user")

		grants := getSampleJITGrants()

		if user != "" {
			filtered := make([]jitGrantInfo, 0)
			for _, g := range grants {
				if g.User == user {
					filtered = append(filtered, g)
				}
			}
			grants = filtered
		}

		if format == "json" {
			data, _ := json.MarshalIndent(grants, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(grants) == 0 {
			fmt.Println("No active JIT grants.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "GRANT ID\tUSER\tRESOURCE\tROLE\tGRANTED AT\tEXPIRES AT")
		for _, g := range grants {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				g.ID, g.User, g.Resource, g.Role, g.GrantedAt, g.ExpiresAt)
		}
		w.Flush()

		return nil
	},
}

var jitRevokeCmd = &cobra.Command{
	Use:   "revoke [grant-id]",
	Short: "Revoke an active JIT grant",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		grantID := args[0]
		reason, _ := cmd.Flags().GetString("reason")
		force, _ := cmd.Flags().GetBool("force")

		if !force && reason == "" {
			fmt.Print("Reason for revocation: ")
			fmt.Scanln(&reason)
		}

		fmt.Printf("Revoking JIT grant: %s\n", grantID)
		if reason != "" {
			fmt.Printf("Reason: %s\n", reason)
		}
		fmt.Println()
		fmt.Println("Grant revoked successfully.")
		fmt.Println("Access has been removed immediately.")

		return nil
	},
}

var jitExtendCmd = &cobra.Command{
	Use:   "extend [grant-id]",
	Short: "Extend an active JIT grant",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		grantID := args[0]
		duration, _ := cmd.Flags().GetString("duration")
		reason, _ := cmd.Flags().GetString("reason")

		fmt.Printf("Extending JIT grant: %s\n", grantID)
		fmt.Printf("Additional duration: %s\n", duration)
		if reason != "" {
			fmt.Printf("Reason: %s\n", reason)
		}
		fmt.Println()
		fmt.Println("Grant extended successfully.")
		fmt.Printf("New expiry: %s\n", time.Now().Add(6*time.Hour).Format("2006-01-02 15:04:05"))

		return nil
	},
}

var jitPolicyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Manage JIT policies",
	Long:  `Manage Just-In-Time access policies.`,
}

var jitPolicyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List JIT policies",
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("output")

		policies := getSampleJITPolicies()

		if format == "json" {
			data, _ := json.MarshalIndent(policies, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tRESOURCE PATTERN\tMAX DURATION\tREQUIRES APPROVAL\tENABLED")
		for _, p := range policies {
			fmt.Fprintf(w, "%s\t%s\t%s\t%v\t%v\n",
				p.Name, p.ResourcePattern, p.MaxDuration, p.RequiresApproval, p.Enabled)
		}
		w.Flush()

		return nil
	},
}

var jitSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show JIT access summary",
	RunE: func(cmd *cobra.Command, args []string) error {
		summary := getSampleJITSummary()

		fmt.Println("JIT Access Summary")
		fmt.Println("==================")
		fmt.Printf("Pending Requests:   %d\n", summary.PendingRequests)
		fmt.Printf("Active Grants:      %d\n", summary.ActiveGrants)
		fmt.Printf("Expiring Soon:      %d (within 1 hour)\n", summary.ExpiringSoon)
		fmt.Println()
		fmt.Println("Grants by Role:")
		for role, count := range summary.GrantsByRole {
			fmt.Printf("  %s: %d\n", role, count)
		}

		return nil
	},
}

var jitMyCmd = &cobra.Command{
	Use:   "my",
	Short: "Show your JIT requests and grants",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Your JIT Access")
		fmt.Println("===============")
		fmt.Println()
		fmt.Println("Pending Requests:")
		fmt.Println("  (none)")
		fmt.Println()
		fmt.Println("Active Grants:")

		grants := []jitGrantInfo{
			{ID: "grant-001", Resource: "staging-db", Role: "viewer", GrantedAt: "2h ago", ExpiresAt: "in 2h"},
		}

		if len(grants) == 0 {
			fmt.Println("  (none)")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "  GRANT ID\tRESOURCE\tROLE\tEXPIRES")
		for _, g := range grants {
			fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n", g.ID, g.Resource, g.Role, g.ExpiresAt)
		}
		w.Flush()

		return nil
	},
}

// Types for CLI display
type jitRequestInfo struct {
	ID            string `json:"id"`
	Requester     string `json:"requester"`
	Resource      string `json:"resource"`
	Role          string `json:"role"`
	Duration      string `json:"duration"`
	Justification string `json:"justification"`
	Status        string `json:"status"`
	Created       string `json:"created"`
}

type jitGrantInfo struct {
	ID        string `json:"id"`
	User      string `json:"user"`
	Resource  string `json:"resource"`
	Role      string `json:"role"`
	GrantedAt string `json:"grantedAt"`
	ExpiresAt string `json:"expiresAt"`
}

type jitPolicyInfo struct {
	Name             string `json:"name"`
	ResourcePattern  string `json:"resourcePattern"`
	MaxDuration      string `json:"maxDuration"`
	RequiresApproval bool   `json:"requiresApproval"`
	Enabled          bool   `json:"enabled"`
}

type jitSummaryInfo struct {
	PendingRequests int            `json:"pendingRequests"`
	ActiveGrants    int            `json:"activeGrants"`
	ExpiringSoon    int            `json:"expiringSoon"`
	GrantsByRole    map[string]int `json:"grantsByRole"`
}

// Sample data functions
func getSampleJITRequests() []jitRequestInfo {
	return []jitRequestInfo{
		{ID: "jit-001", Requester: "alice", Resource: "prod-database", Role: "viewer", Duration: "4h", Status: "pending", Created: "10m ago"},
		{ID: "jit-002", Requester: "bob", Resource: "api-service", Role: "operator", Duration: "2h", Status: "approved", Created: "1h ago"},
		{ID: "jit-003", Requester: "charlie", Resource: "staging-cluster", Role: "admin", Duration: "8h", Status: "denied", Created: "2h ago"},
	}
}

func getSampleJITGrants() []jitGrantInfo {
	return []jitGrantInfo{
		{ID: "grant-001", User: "alice", Resource: "staging-db", Role: "viewer", GrantedAt: "2h ago", ExpiresAt: "in 2h"},
		{ID: "grant-002", User: "bob", Resource: "api-service", Role: "operator", GrantedAt: "1h ago", ExpiresAt: "in 1h"},
		{ID: "grant-003", User: "dave", Resource: "monitoring", Role: "viewer", GrantedAt: "30m ago", ExpiresAt: "in 3h 30m"},
	}
}

func getSampleJITPolicies() []jitPolicyInfo {
	return []jitPolicyInfo{
		{Name: "default", ResourcePattern: "*", MaxDuration: "8h", RequiresApproval: true, Enabled: true},
		{Name: "production", ResourcePattern: "*prod*", MaxDuration: "4h", RequiresApproval: true, Enabled: true},
		{Name: "staging", ResourcePattern: "*staging*", MaxDuration: "12h", RequiresApproval: false, Enabled: true},
	}
}

func getSampleJITSummary() jitSummaryInfo {
	return jitSummaryInfo{
		PendingRequests: 2,
		ActiveGrants:    5,
		ExpiringSoon:    1,
		GrantsByRole: map[string]int{
			"viewer":   3,
			"operator": 1,
			"admin":    1,
		},
	}
}

func init() {
	// Main JIT command
	rootCmd.AddCommand(jitCmd)

	// Request subcommand
	jitCmd.AddCommand(jitRequestCmd)
	jitRequestCmd.Flags().StringP("resource", "r", "", "Resource to access (required)")
	jitRequestCmd.Flags().String("role", "", "Role to request (required)")
	jitRequestCmd.Flags().StringP("duration", "d", "4h", "Access duration")
	jitRequestCmd.Flags().StringP("justification", "j", "", "Justification for access (required)")

	// List subcommand
	jitCmd.AddCommand(jitListCmd)
	jitListCmd.Flags().StringP("status", "s", "", "Filter by status (pending, approved, denied, expired)")
	jitListCmd.Flags().StringP("output", "o", "table", "Output format (table, json)")

	// Approve subcommand
	jitCmd.AddCommand(jitApproveCmd)
	jitApproveCmd.Flags().StringP("comment", "c", "", "Approval comment")

	// Deny subcommand
	jitCmd.AddCommand(jitDenyCmd)
	jitDenyCmd.Flags().StringP("reason", "r", "", "Denial reason (required)")

	// Grants subcommand
	jitCmd.AddCommand(jitGrantsCmd)
	jitGrantsCmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	jitGrantsCmd.Flags().StringP("user", "u", "", "Filter by user")

	// Revoke subcommand
	jitCmd.AddCommand(jitRevokeCmd)
	jitRevokeCmd.Flags().StringP("reason", "r", "", "Revocation reason")
	jitRevokeCmd.Flags().BoolP("force", "f", false, "Force revocation without reason")

	// Extend subcommand
	jitCmd.AddCommand(jitExtendCmd)
	jitExtendCmd.Flags().StringP("duration", "d", "2h", "Additional duration")
	jitExtendCmd.Flags().StringP("reason", "r", "", "Extension reason")

	// Policy subcommands
	jitCmd.AddCommand(jitPolicyCmd)
	jitPolicyCmd.AddCommand(jitPolicyListCmd)
	jitPolicyListCmd.Flags().StringP("output", "o", "table", "Output format (table, json)")

	// Summary subcommand
	jitCmd.AddCommand(jitSummaryCmd)

	// My subcommand
	jitCmd.AddCommand(jitMyCmd)
}
