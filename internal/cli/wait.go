package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var waitCmd = &cobra.Command{
	Use:   "wait <job-id>",
	Short: "Wait for a job to complete",
	Long:  `Wait for an asynchronous job to complete. This command blocks until the job finishes.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runWait,
}

func runWait(cmd *cobra.Command, args []string) error {
	jobID := args[0]

	if applyJobQueue == nil {
		return fmt.Errorf("job queue not initialized")
	}

	fmt.Printf("Waiting for job %s to complete...\n", jobID)

	// Show progress updates
	lastProgress := -1
	for {
		job, err := applyJobQueue.Get(jobID)
		if err != nil {
			return err
		}

		// Show progress updates
		if job.Progress != lastProgress {
			fmt.Printf("[%s] Progress: %d%% - Status: %s\n",
				time.Now().Format("15:04:05"),
				job.Progress,
				job.Status,
			)
			lastProgress = job.Progress
		}

		// Check if job is done
		if job.Status == "completed" {
			fmt.Printf("\nJob completed successfully!\n")
			if job.Result != nil {
				fmt.Printf("Result: %v\n", job.Result)
			}
			return nil
		}

		if job.Status == "failed" {
			fmt.Printf("\nJob failed: %v\n", job.Error)
			return job.Error
		}

		if job.Status == "cancelled" {
			return fmt.Errorf("job was cancelled")
		}

		time.Sleep(500 * time.Millisecond)
	}
}
