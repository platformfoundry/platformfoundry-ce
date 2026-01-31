package integration

import (
	"context"
	"testing"
	"time"

	"github.com/platformfoundry/platformfoundry-ce/internal/jobs"
)

func TestJobQueue_Integration(t *testing.T) {
	queue := jobs.NewQueue(2)
	defer queue.Stop()

	// Submit multiple jobs
	jobIDs := []string{}
	for i := 0; i < 5; i++ {
		job, err := queue.Submit(
			jobs.JobTypeApply,
			"Integration test job",
			func(ctx context.Context, job *jobs.Job) (interface{}, error) {
				job.LogInfo("test", "Starting job")
				job.SetProgress(25)
				time.Sleep(100 * time.Millisecond)
				job.SetProgress(50)
				time.Sleep(100 * time.Millisecond)
				job.SetProgress(75)
				time.Sleep(100 * time.Millisecond)
				job.LogInfo("test", "Job completed")
				return "success", nil
			},
		)

		if err != nil {
			t.Fatalf("Failed to submit job: %v", err)
		}

		jobIDs = append(jobIDs, job.ID)
	}

	// Wait for all jobs to complete
	for _, jobID := range jobIDs {
		if err := queue.Wait(jobID); err != nil {
			t.Errorf("Job %s failed: %v", jobID, err)
		}
	}

	// Verify all jobs completed
	for _, jobID := range jobIDs {
		job, err := queue.Get(jobID)
		if err != nil {
			t.Errorf("Failed to get job %s: %v", jobID, err)
			continue
		}

		if job.Status != jobs.JobStatusCompleted {
			t.Errorf("Job %s has status %s, expected %s", jobID, job.Status, jobs.JobStatusCompleted)
		}

		if job.Progress != 100 {
			t.Errorf("Job %s has progress %d, expected 100", jobID, job.Progress)
		}

		if len(job.Logs) < 2 {
			t.Errorf("Job %s has %d logs, expected at least 2", jobID, len(job.Logs))
		}
	}
}

func TestJobQueue_ConcurrentExecution(t *testing.T) {
	workers := 3
	queue := jobs.NewQueue(workers)
	defer queue.Stop()

	jobCount := 10
	jobIDs := make([]string, jobCount)

	// Submit jobs
	for i := 0; i < jobCount; i++ {
		job, err := queue.Submit(
			jobs.JobTypeApply,
			"Concurrent test job",
			func(ctx context.Context, job *jobs.Job) (interface{}, error) {
				time.Sleep(50 * time.Millisecond)
				return "done", nil
			},
		)

		if err != nil {
			t.Fatalf("Failed to submit job: %v", err)
		}

		jobIDs[i] = job.ID
	}

	// Wait for all to complete
	for _, jobID := range jobIDs {
		if err := queue.Wait(jobID); err != nil {
			t.Errorf("Job failed: %v", err)
		}
	}

	// All jobs should be completed
	jobs := queue.List()
	completedCount := 0
	for _, job := range jobs {
		if job.Status == "completed" {
			completedCount++
		}
	}

	if completedCount != jobCount {
		t.Errorf("Expected %d completed jobs, got %d", jobCount, completedCount)
	}
}

func TestJobQueue_CancellationFlow(t *testing.T) {
	queue := jobs.NewQueue(1)
	defer queue.Stop()

	// Submit a long-running job
	job, err := queue.Submit(
		jobs.JobTypeApply,
		"Long running job",
		func(ctx context.Context, job *jobs.Job) (interface{}, error) {
			for i := 0; i < 100; i++ {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				default:
					time.Sleep(10 * time.Millisecond)
					job.SetProgress(i)
				}
			}
			return "completed", nil
		},
	)

	if err != nil {
		t.Fatalf("Failed to submit job: %v", err)
	}

	// Give it time to start
	time.Sleep(100 * time.Millisecond)

	// Cancel the job
	err = queue.Cancel(job.ID)
	if err != nil {
		t.Fatalf("Failed to cancel job: %v", err)
	}

	// Wait a bit for cancellation to take effect
	time.Sleep(100 * time.Millisecond)

	// Check status
	job, err = queue.Get(job.ID)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if job.Status != jobs.JobStatusCancelled {
		t.Errorf("Expected status %s, got %s", jobs.JobStatusCancelled, job.Status)
	}
}
