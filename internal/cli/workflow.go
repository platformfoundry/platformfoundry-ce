package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/platformfoundry/platformfoundry-ce/internal/workflow"
	"github.com/spf13/cobra"
)

var workflowEngine *workflow.Engine

func init() {
	workflowEngine = workflow.NewEngine()
}

var workflowCmd = &cobra.Command{
	Use:   "workflow",
	Short: "Manage approval workflows",
	Long:  `Create, manage, and interact with approval workflows for deployments and other sensitive operations.`,
}

var workflowListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all workflows",
	RunE:  runWorkflowList,
}

var workflowStatusCmd = &cobra.Command{
	Use:   "status <execution-id>",
	Short: "Get status of a workflow execution",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkflowStatus,
}

var workflowApproveCmd = &cobra.Command{
	Use:   "approve <execution-id>",
	Short: "Approve a workflow execution",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkflowApprove,
}

var workflowRejectCmd = &cobra.Command{
	Use:   "reject <execution-id>",
	Short: "Reject a workflow execution",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkflowReject,
}

var workflowHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Show workflow execution history",
	RunE:  runWorkflowHistory,
}

var (
	workflowApproveComment string
	workflowRejectComment  string
	workflowUser           string
	workflowRole           string
	workflowStatusFilter   string
)

func init() {
	workflowCmd.AddCommand(workflowListCmd)
	workflowCmd.AddCommand(workflowStatusCmd)
	workflowCmd.AddCommand(workflowApproveCmd)
	workflowCmd.AddCommand(workflowRejectCmd)
	workflowCmd.AddCommand(workflowHistoryCmd)

	workflowApproveCmd.Flags().StringVar(&workflowApproveComment, "comment", "", "Approval comment")
	workflowApproveCmd.Flags().StringVar(&workflowUser, "user", "", "Approver username")
	workflowApproveCmd.Flags().StringVar(&workflowRole, "role", "", "Approver role")

	workflowRejectCmd.Flags().StringVar(&workflowRejectComment, "comment", "", "Rejection reason (required)")
	workflowRejectCmd.Flags().StringVar(&workflowUser, "user", "", "Rejector username")
	workflowRejectCmd.Flags().StringVar(&workflowRole, "role", "", "Rejector role")
	workflowRejectCmd.MarkFlagRequired("comment")

	workflowHistoryCmd.Flags().StringVar(&workflowStatusFilter, "status", "", "Filter by status")
}

func runWorkflowList(cmd *cobra.Command, args []string) error {
	workflows := workflowEngine.ListWorkflows()

	if len(workflows) == 0 {
		fmt.Println("No workflows configured.")
		return nil
	}

	fmt.Printf("%-30s %-15s %-20s %-10s\n", "NAME", "TRIGGER", "TARGET", "APPROVALS")
	fmt.Println(strings.Repeat("-", 80))

	for _, wf := range workflows {
		target := ""
		if wf.Trigger.Target.Environment != "" {
			target = wf.Trigger.Target.Environment
		}
		if wf.Trigger.Target.Service != "" {
			if target != "" {
				target += "/"
			}
			target += wf.Trigger.Target.Service
		}
		if target == "" {
			target = "*"
		}

		fmt.Printf("%-30s %-15s %-20s %-10d\n",
			wf.Name,
			wf.Trigger.Action,
			target,
			wf.Approvals.Required,
		)
	}

	return nil
}

func runWorkflowStatus(cmd *cobra.Command, args []string) error {
	executionID := args[0]

	exec, err := workflowEngine.GetExecution(executionID)
	if err != nil {
		return err
	}

	fmt.Printf("Workflow: %s (%s)\n", exec.WorkflowName, exec.ID)
	fmt.Printf("Status: %s\n", formatStatus(exec.Status))
	fmt.Printf("Requester: %s\n", exec.Requester)
	fmt.Printf("Requested: %s\n", exec.RequestedAt.Format(time.RFC3339))

	if exec.Target.Environment != "" {
		fmt.Printf("Environment: %s\n", exec.Target.Environment)
	}
	if exec.Target.Service != "" {
		fmt.Printf("Service: %s\n", exec.Target.Service)
	}

	// Conditions
	if len(exec.ConditionResults) > 0 {
		fmt.Println("\nConditions:")
		for _, cond := range exec.ConditionResults {
			icon := getConditionIcon(cond.Status)
			fmt.Printf("  %s %s: %s\n", icon, cond.Type, cond.Message)
		}
	}

	// Approvals
	if len(exec.Approvals) > 0 || exec.Status == workflow.WorkflowStatusAwaitApproval {
		fmt.Println("\nApprovals:")
		for _, approval := range exec.Approvals {
			icon := "?"
			if approval.Decision == "approved" {
				icon = "V"
			} else if approval.Decision == "rejected" {
				icon = "X"
			}
			comment := ""
			if approval.Comment != "" {
				comment = fmt.Sprintf(" - \"%s\"", approval.Comment)
			}
			fmt.Printf("  %s %s (%s)%s\n", icon, approval.User, approval.Role, comment)
		}
		if exec.Status == workflow.WorkflowStatusAwaitApproval {
			fmt.Println("  ... Waiting for more approvals")
		}
	}

	// Error
	if exec.Error != "" {
		fmt.Printf("\nError: %s\n", exec.Error)
	}

	// Timing
	if exec.StartedAt != nil {
		fmt.Printf("\nStarted: %s\n", exec.StartedAt.Format(time.RFC3339))
	}
	if exec.CompletedAt != nil {
		fmt.Printf("Completed: %s\n", exec.CompletedAt.Format(time.RFC3339))
		if exec.StartedAt != nil {
			duration := exec.CompletedAt.Sub(*exec.StartedAt)
			fmt.Printf("Duration: %s\n", duration.Round(time.Second))
		}
	}

	return nil
}

func runWorkflowApprove(cmd *cobra.Command, args []string) error {
	executionID := args[0]

	user := workflowUser
	if user == "" {
		user = "cli-user" // Default user
	}

	role := workflowRole
	if role == "" {
		role = "approver" // Default role
	}

	if err := workflowEngine.Approve(executionID, user, role, workflowApproveComment); err != nil {
		return err
	}

	fmt.Printf("Approval recorded for workflow %s\n", executionID)

	// Show current status
	exec, err := workflowEngine.GetExecution(executionID)
	if err == nil {
		approvedCount := 0
		for _, a := range exec.Approvals {
			if a.Decision == "approved" {
				approvedCount++
			}
		}
		fmt.Printf("Approvals: %d recorded\n", approvedCount)
	}

	return nil
}

func runWorkflowReject(cmd *cobra.Command, args []string) error {
	executionID := args[0]

	user := workflowUser
	if user == "" {
		user = "cli-user"
	}

	role := workflowRole
	if role == "" {
		role = "approver"
	}

	if err := workflowEngine.Reject(executionID, user, role, workflowRejectComment); err != nil {
		return err
	}

	fmt.Printf("Workflow %s rejected\n", executionID)
	return nil
}

func runWorkflowHistory(cmd *cobra.Command, args []string) error {
	var status workflow.WorkflowStatus
	if workflowStatusFilter != "" {
		status = workflow.WorkflowStatus(workflowStatusFilter)
	}

	executions := workflowEngine.ListExecutions("", status)

	if len(executions) == 0 {
		fmt.Println("No workflow executions found.")
		return nil
	}

	fmt.Printf("%-12s %-25s %-15s %-20s %-20s\n", "ID", "WORKFLOW", "STATUS", "REQUESTER", "REQUESTED")
	fmt.Println(strings.Repeat("-", 100))

	for _, exec := range executions {
		shortID := exec.ID
		if len(shortID) > 12 {
			shortID = shortID[:12]
		}
		fmt.Printf("%-12s %-25s %-15s %-20s %-20s\n",
			shortID,
			exec.WorkflowName,
			exec.Status,
			exec.Requester,
			exec.RequestedAt.Format("2006-01-02 15:04"),
		)
	}

	return nil
}

// TriggerWorkflow starts a workflow execution for an action
func TriggerWorkflow(ctx context.Context, workflowName, requester, action, environment, service string) (*workflow.WorkflowExecution, error) {
	target := workflow.WorkflowTarget{
		Environment: environment,
		Service:     service,
	}

	return workflowEngine.StartExecution(ctx, workflowName, requester, target, action)
}

func formatStatus(status workflow.WorkflowStatus) string {
	switch status {
	case workflow.WorkflowStatusPending:
		return "Pending"
	case workflow.WorkflowStatusConditions:
		return "Checking Conditions"
	case workflow.WorkflowStatusAwaitApproval:
		return "Awaiting Approval"
	case workflow.WorkflowStatusApproved:
		return "Approved"
	case workflow.WorkflowStatusRejected:
		return "Rejected"
	case workflow.WorkflowStatusExecuting:
		return "Executing"
	case workflow.WorkflowStatusCompleted:
		return "Completed"
	case workflow.WorkflowStatusFailed:
		return "Failed"
	case workflow.WorkflowStatusRolledBack:
		return "Rolled Back"
	case workflow.WorkflowStatusTimedOut:
		return "Timed Out"
	case workflow.WorkflowStatusBlocked:
		return "Blocked"
	default:
		return string(status)
	}
}

func getConditionIcon(status workflow.ConditionStatus) string {
	switch status {
	case workflow.ConditionStatusPassed:
		return "V"
	case workflow.ConditionStatusFailed:
		return "X"
	case workflow.ConditionStatusPending:
		return "?"
	case workflow.ConditionStatusSkipped:
		return "-"
	default:
		return "?"
	}
}
