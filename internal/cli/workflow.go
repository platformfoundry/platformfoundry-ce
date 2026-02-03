package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/platformfoundry/pf-ce/internal/workflow"
	"github.com/spf13/cobra"
)

var workflowEngine *workflow.Engine
var dagRuntime *workflow.Runtime

func init() {
	workflowEngine = workflow.NewEngine()
	dagRuntime = workflow.NewRuntime(workflow.RuntimeConfig{})
}

var workflowCmd = &cobra.Command{
	Use:   "workflow",
	Short: "Manage approval and DAG workflows",
	Long:  `Create, manage, and interact with approval workflows and DAG-based automation workflows.`,
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

// DAG workflow commands
var workflowRunCmd = &cobra.Command{
	Use:   "run <workflow-name>",
	Short: "Run a DAG workflow",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkflowRun,
}

var workflowApplyCmd = &cobra.Command{
	Use:   "apply -f <file>",
	Short: "Create or update a DAG workflow from a YAML file",
	RunE:  runWorkflowApply,
}

var workflowGetCmd = &cobra.Command{
	Use:   "get <workflow-name>",
	Short: "Get a DAG workflow definition",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkflowGet,
}

var workflowLogsCmd = &cobra.Command{
	Use:   "logs <execution-id>",
	Short: "Get logs for a workflow execution",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkflowLogs,
}

var workflowCancelCmd = &cobra.Command{
	Use:   "cancel <execution-id>",
	Short: "Cancel a running workflow execution",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkflowCancel,
}

var (
	workflowApproveComment string
	workflowRejectComment  string
	workflowUser           string
	workflowRole           string
	workflowStatusFilter   string
	workflowFile           string
	workflowInputs         []string
	workflowStep           string
)

func init() {
	workflowCmd.AddCommand(workflowListCmd)
	workflowCmd.AddCommand(workflowStatusCmd)
	workflowCmd.AddCommand(workflowApproveCmd)
	workflowCmd.AddCommand(workflowRejectCmd)
	workflowCmd.AddCommand(workflowHistoryCmd)

	// DAG workflow commands
	workflowCmd.AddCommand(workflowRunCmd)
	workflowCmd.AddCommand(workflowApplyCmd)
	workflowCmd.AddCommand(workflowGetCmd)
	workflowCmd.AddCommand(workflowLogsCmd)
	workflowCmd.AddCommand(workflowCancelCmd)

	workflowApproveCmd.Flags().StringVar(&workflowApproveComment, "comment", "", "Approval comment")
	workflowApproveCmd.Flags().StringVar(&workflowUser, "user", "", "Approver username")
	workflowApproveCmd.Flags().StringVar(&workflowRole, "role", "", "Approver role")

	workflowRejectCmd.Flags().StringVar(&workflowRejectComment, "comment", "", "Rejection reason (required)")
	workflowRejectCmd.Flags().StringVar(&workflowUser, "user", "", "Rejector username")
	workflowRejectCmd.Flags().StringVar(&workflowRole, "role", "", "Rejector role")
	workflowRejectCmd.MarkFlagRequired("comment")

	workflowHistoryCmd.Flags().StringVar(&workflowStatusFilter, "status", "", "Filter by status")

	// DAG workflow flags
	workflowRunCmd.Flags().StringArrayVar(&workflowInputs, "input", nil, "Workflow input (key=value)")
	workflowApplyCmd.Flags().StringVarP(&workflowFile, "file", "f", "", "Path to workflow YAML file")
	workflowApplyCmd.MarkFlagRequired("file")
	workflowLogsCmd.Flags().StringVar(&workflowStep, "step", "", "Step ID to get logs for")
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

// DAG Workflow Commands

func runWorkflowRun(cmd *cobra.Command, args []string) error {
	workflowName := args[0]

	// Parse inputs
	inputs := make(map[string]interface{})
	for _, input := range workflowInputs {
		parts := strings.SplitN(input, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid input format: %s (expected key=value)", input)
		}
		inputs[parts[0]] = parts[1]
	}

	ctx := context.Background()
	exec, err := dagRuntime.RunWorkflow(ctx, workflowName, inputs)
	if err != nil {
		return err
	}

	fmt.Printf("Workflow started: %s\n", exec.ID)
	fmt.Printf("  Workflow: %s\n", exec.WorkflowName)
	fmt.Printf("  Status: %s\n", exec.Status)
	fmt.Printf("  Started: %s\n", exec.StartedAt.Format(time.RFC3339))

	return nil
}

func runWorkflowApply(cmd *cobra.Command, args []string) error {
	// Read and parse workflow file
	data, err := os.ReadFile(workflowFile)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	parser := workflow.NewParser()
	wf, err := parser.Parse(data)
	if err != nil {
		return fmt.Errorf("failed to parse workflow: %w", err)
	}

	ctx := context.Background()
	if err := dagRuntime.ApplyWorkflow(ctx, wf); err != nil {
		return err
	}

	fmt.Printf("Workflow applied: %s\n", wf.Metadata.Name)
	fmt.Printf("  Kind: %s\n", wf.Kind)
	fmt.Printf("  Steps: %d\n", len(wf.Spec.Steps))
	if len(wf.Spec.Triggers) > 0 {
		fmt.Printf("  Triggers: %d\n", len(wf.Spec.Triggers))
	}

	return nil
}

func runWorkflowGet(cmd *cobra.Command, args []string) error {
	workflowName := args[0]

	ctx := context.Background()
	wf, err := dagRuntime.GetWorkflow(ctx, workflowName)
	if err != nil {
		return err
	}

	// Output workflow as YAML
	parser := workflow.NewParser()
	data, err := parser.Marshal(wf)
	if err != nil {
		return fmt.Errorf("failed to marshal workflow: %w", err)
	}

	fmt.Println(string(data))
	return nil
}

func runWorkflowLogs(cmd *cobra.Command, args []string) error {
	execID := args[0]
	ctx := context.Background()

	exec, err := dagRuntime.GetExecution(ctx, execID)
	if err != nil {
		return err
	}

	fmt.Printf("Execution: %s\n", exec.ID)
	fmt.Printf("Workflow: %s\n", exec.WorkflowName)
	fmt.Printf("Status: %s\n", exec.Status)
	fmt.Println()

	// If specific step requested
	if workflowStep != "" {
		logs, err := dagRuntime.GetStepLogs(ctx, execID, workflowStep)
		if err != nil {
			return err
		}

		fmt.Printf("Step: %s\n", workflowStep)
		fmt.Println(strings.Repeat("-", 60))
		for _, log := range logs {
			fmt.Printf("[%s] %s: %s\n", log.Time.Format("15:04:05"), log.Level, log.Message)
		}
		return nil
	}

	// Show all steps
	for stepID, stepExec := range exec.Steps {
		fmt.Printf("Step: %s\n", stepID)
		fmt.Printf("  Status: %s\n", stepExec.Status)
		if stepExec.StartedAt != nil {
			fmt.Printf("  Started: %s\n", stepExec.StartedAt.Format(time.RFC3339))
		}
		if stepExec.CompletedAt != nil {
			fmt.Printf("  Completed: %s\n", stepExec.CompletedAt.Format(time.RFC3339))
		}
		if stepExec.Error != "" {
			fmt.Printf("  Error: %s\n", stepExec.Error)
		}
		if len(stepExec.Logs) > 0 {
			fmt.Println("  Logs:")
			for _, log := range stepExec.Logs {
				fmt.Printf("    [%s] %s: %s\n", log.Time.Format("15:04:05"), log.Level, log.Message)
			}
		}
		fmt.Println()
	}

	return nil
}

func runWorkflowCancel(cmd *cobra.Command, args []string) error {
	execID := args[0]
	ctx := context.Background()

	if err := dagRuntime.CancelExecution(ctx, execID); err != nil {
		return err
	}

	fmt.Printf("Workflow execution cancelled: %s\n", execID)
	return nil
}
