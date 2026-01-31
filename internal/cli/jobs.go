package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/platformfoundry/platformfoundry-ce/internal/jobs"
	"github.com/spf13/cobra"
)

var jobsCmd = &cobra.Command{
	Use:   "jobs",
	Short: "Manage asynchronous jobs",
	Long:  `View and manage asynchronous jobs submitted to the platform.`,
}

var jobsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all jobs",
	Long:  `List all jobs with their current status.`,
	RunE:  runJobsList,
}

var jobsStatusCmd = &cobra.Command{
	Use:   "status <job-id>",
	Short: "Get status of a specific job",
	Long:  `Get detailed status information for a specific job.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runJobsStatus,
}

var jobsCancelCmd = &cobra.Command{
	Use:   "cancel <job-id>",
	Short: "Cancel a running job",
	Long:  `Cancel a pending or running job.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runJobsCancel,
}

var jobsWatchCmd = &cobra.Command{
	Use:   "watch <job-id>",
	Short: "Watch job progress in real-time",
	Long:  `Monitor a job's progress in real-time with live updates.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runJobsWatch,
}

func init() {
	jobsCmd.AddCommand(jobsListCmd)
	jobsCmd.AddCommand(jobsStatusCmd)
	jobsCmd.AddCommand(jobsCancelCmd)
	jobsCmd.AddCommand(jobsWatchCmd)
}

func runJobsList(cmd *cobra.Command, args []string) error {
	if applyJobQueue == nil {
		fmt.Println("No jobs found.")
		return nil
	}

	jobList := applyJobQueue.List()
	if len(jobList) == 0 {
		fmt.Println("No jobs found.")
		return nil
	}

	fmt.Printf("%-12s %-10s %-12s %-30s %-15s %-10s\n", "ID", "TYPE", "STATUS", "PROGRESS", "STARTED", "DURATION")
	fmt.Println(strings.Repeat("─", 120))

	for _, job := range jobList {
		// Status icon
		statusIcon := getStatusIcon(job.Status)

		// Progress bar
		progressBar := renderProgressBar(job.Progress, 25)

		// Calculate duration
		duration := "N/A"
		if job.StartedAt != nil {
			if job.CompletedAt != nil {
				duration = job.CompletedAt.Sub(*job.StartedAt).Round(time.Second).String()
			} else {
				duration = time.Since(*job.StartedAt).Round(time.Second).String()
			}
		}

		// Time ago
		timeAgo := "N/A"
		if job.StartedAt != nil {
			timeAgo = formatTimeAgo(*job.StartedAt)
		}

		// Truncate ID for display
		shortID := job.ID
		if len(shortID) > 12 {
			shortID = shortID[:12]
		}

		fmt.Printf("%-12s %-10s %s %-10s %-30s %-15s %-10s\n",
			shortID,
			job.Type,
			statusIcon,
			job.Status,
			progressBar,
			timeAgo,
			duration,
		)
	}

	return nil
}

func getStatusIcon(status jobs.JobStatus) string {
	switch status {
	case jobs.JobStatusPending:
		return "⏳"
	case jobs.JobStatusRunning:
		return "🔄"
	case jobs.JobStatusCompleted:
		return "✅"
	case jobs.JobStatusFailed:
		return "❌"
	case jobs.JobStatusCancelled:
		return "🚫"
	default:
		return "•"
	}
}

func renderProgressBar(progress int, width int) string {
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}

	filled := int(float64(width) * float64(progress) / 100.0)
	empty := width - filled

	bar := strings.Repeat("━", filled) + strings.Repeat("░", empty)
	return fmt.Sprintf("%s %3d%%", bar, progress)
}

func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	}
	if duration < time.Hour {
		mins := int(duration.Minutes())
		return fmt.Sprintf("%d min ago", mins)
	}
	if duration < 24*time.Hour {
		hours := int(duration.Hours())
		return fmt.Sprintf("%d hrs ago", hours)
	}
	days := int(duration.Hours() / 24)
	return fmt.Sprintf("%d days ago", days)
}

func runJobsStatus(cmd *cobra.Command, args []string) error {
	jobID := args[0]

	if applyJobQueue == nil {
		return fmt.Errorf("job queue not initialized")
	}

	job, err := applyJobQueue.Get(jobID)
	if err != nil {
		return err
	}

	fmt.Printf("Job ID: %s\n", job.ID)
	fmt.Printf("Type: %s\n", job.Type)
	fmt.Printf("Status: %s\n", job.Status)
	fmt.Printf("Progress: %d%%\n", job.Progress)
	fmt.Printf("Description: %s\n", job.Description)
	fmt.Printf("Created: %s\n", job.CreatedAt.Format(time.RFC3339))

	if job.StartedAt != nil {
		fmt.Printf("Started: %s\n", job.StartedAt.Format(time.RFC3339))
	}

	if job.CompletedAt != nil {
		fmt.Printf("Completed: %s\n", job.CompletedAt.Format(time.RFC3339))
		duration := job.CompletedAt.Sub(*job.StartedAt)
		fmt.Printf("Duration: %s\n", duration.Round(time.Second))
	}

	if job.Error != nil {
		fmt.Printf("\nError: %v\n", job.Error)
	}

	// Metadata
	if len(job.Metadata) > 0 {
		fmt.Println("\nMetadata:")
		for key, value := range job.Metadata {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	// Recent logs
	if len(job.Logs) > 0 {
		fmt.Println("\nRecent Logs:")
		start := 0
		if len(job.Logs) > 10 {
			start = len(job.Logs) - 10
		}
		for _, log := range job.Logs[start:] {
			fmt.Printf("  [%s] [%s] %s: %s\n",
				log.Timestamp.Format("15:04:05"),
				log.Level,
				log.Component,
				log.Message,
			)
		}
	}

	return nil
}

func runJobsCancel(cmd *cobra.Command, args []string) error {
	jobID := args[0]

	if applyJobQueue == nil {
		return fmt.Errorf("job queue not initialized")
	}

	if err := applyJobQueue.Cancel(jobID); err != nil {
		return err
	}

	fmt.Printf("Job %s cancelled successfully\n", jobID)
	return nil
}

func runJobsWatch(cmd *cobra.Command, args []string) error {
	jobID := args[0]

	if applyJobQueue == nil {
		return fmt.Errorf("job queue not initialized")
	}

	fmt.Printf("Watching job %s (Press Ctrl+C to stop)\n\n", jobID)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	lastProgress := -1

	for range ticker.C {
		job, err := applyJobQueue.Get(jobID)
		if err != nil {
			return err
		}

		// Only update if progress changed or job status changed
		if job.Progress != lastProgress || job.Status != jobs.JobStatusRunning {
			// Clear screen and move cursor to top
			fmt.Print("\033[2J\033[H")

			// Header
			fmt.Println("┌─────────────────────────────────────────────────────────────┐")
			fmt.Printf("│  Job: %-52s│\n", truncate(job.ID, 52))
			fmt.Printf("│  Description: %-46s│\n", truncate(job.Description, 46))
			fmt.Println("├─────────────────────────────────────────────────────────────┤")

			// Status
			statusIcon := getStatusIcon(job.Status)
			fmt.Printf("│  Status: %s %-48s│\n", statusIcon, job.Status)

			// Progress bar
			progressBar := renderProgressBar(job.Progress, 50)
			fmt.Printf("│  Progress: %-47s│\n", progressBar)

			// Timing information
			if job.StartedAt != nil {
				elapsed := time.Since(*job.StartedAt).Round(time.Second)
				fmt.Printf("│  Elapsed: %-49s│\n", elapsed)

				if job.Progress > 0 && job.Status == jobs.JobStatusRunning {
					// Estimate remaining time
					rate := float64(job.Progress) / elapsed.Seconds()
					remaining := time.Duration(float64(100-job.Progress)/rate) * time.Second
					fmt.Printf("│  Estimated remaining: %-39s│\n", remaining.Round(time.Second))
				}
			}

			// Recent logs
			if len(job.Logs) > 0 {
				fmt.Println("│                                                             │")
				fmt.Println("│  Recent Logs:                                               │")
				start := 0
				if len(job.Logs) > 5 {
					start = len(job.Logs) - 5
				}
				for _, log := range job.Logs[start:] {
					levelIcon := getLogLevelIcon(log.Level)
					logLine := fmt.Sprintf("[%s] %s %s", log.Timestamp.Format("15:04:05"), levelIcon, log.Message)
					fmt.Printf("│  %-59s│\n", truncate(logLine, 59))
				}
			}

			fmt.Println("└─────────────────────────────────────────────────────────────┘")

			lastProgress = job.Progress

			// Exit if job completed
			if job.Status == jobs.JobStatusCompleted {
				fmt.Println("\n✅ Job completed successfully!")
				return nil
			}

			if job.Status == jobs.JobStatusFailed {
				fmt.Printf("\n❌ Job failed: %v\n", job.Error)
				return job.Error
			}

			if job.Status == jobs.JobStatusCancelled {
				fmt.Println("\n🚫 Job was cancelled")
				return nil
			}
		}
	}

	return nil
}

func getLogLevelIcon(level jobs.LogLevel) string {
	switch level {
	case jobs.LogLevelDebug:
		return "🐛"
	case jobs.LogLevelInfo:
		return "ℹ️ "
	case jobs.LogLevelWarn:
		return "⚠️ "
	case jobs.LogLevelError:
		return "❌"
	default:
		return "•"
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
