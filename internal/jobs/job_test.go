package jobs

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewQueue(t *testing.T) {
	q := NewQueue(4)
	if q == nil {
		t.Fatal("NewQueue returned nil")
	}
	if q.workers != 4 {
		t.Errorf("Expected 4 workers, got %d", q.workers)
	}
	defer q.Stop()
}

func TestJobSubmit(t *testing.T) {
	q := NewQueue(2)
	defer q.Stop()

	job, err := q.Submit(JobTypeApply, "Test job", func(ctx context.Context, job *Job) (interface{}, error) {
		return "success", nil
	})

	if err != nil {
		t.Fatalf("Failed to submit job: %v", err)
	}

	if job.Status != JobStatusPending {
		t.Errorf("Expected status %s, got %s", JobStatusPending, job.Status)
	}

	if job.Type != JobTypeApply {
		t.Errorf("Expected type %s, got %s", JobTypeApply, job.Type)
	}
}

func TestJobExecution(t *testing.T) {
	q := NewQueue(2)
	defer q.Stop()

	executed := false
	job, err := q.Submit(JobTypeApply, "Test execution", func(ctx context.Context, job *Job) (interface{}, error) {
		executed = true
		job.SetProgress(50)
		return "result", nil
	})

	if err != nil {
		t.Fatalf("Failed to submit job: %v", err)
	}

	// Wait for completion
	if err := q.Wait(job.ID); err != nil {
		t.Fatalf("Job failed: %v", err)
	}

	if !executed {
		t.Error("Job function was not executed")
	}

	if job.Status != JobStatusCompleted {
		t.Errorf("Expected status %s, got %s", JobStatusCompleted, job.Status)
	}

	if job.Result.(string) != "result" {
		t.Errorf("Expected result 'result', got %v", job.Result)
	}
}

func TestJobFailure(t *testing.T) {
	q := NewQueue(2)
	defer q.Stop()

	expectedErr := errors.New("job failed")
	job, err := q.Submit(JobTypeApply, "Test failure", func(ctx context.Context, job *Job) (interface{}, error) {
		return nil, expectedErr
	})

	if err != nil {
		t.Fatalf("Failed to submit job: %v", err)
	}

	// Wait should return error
	err = q.Wait(job.ID)
	if err == nil {
		t.Error("Expected error from Wait, got nil")
	}

	// Get job and check status
	job, err = q.Get(job.ID)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if job.Status != JobStatusFailed {
		t.Errorf("Expected status %s, got %s", JobStatusFailed, job.Status)
	}

	if job.Error == nil || job.Error.Error() != expectedErr.Error() {
		t.Errorf("Expected error %v, got %v", expectedErr, job.Error)
	}
}

func TestJobCancel(t *testing.T) {
	q := NewQueue(1)
	defer q.Stop()

	// Submit a long-running job
	job, err := q.Submit(JobTypeApply, "Test cancel", func(ctx context.Context, job *Job) (interface{}, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(5 * time.Second):
			return "completed", nil
		}
	})

	if err != nil {
		t.Fatalf("Failed to submit job: %v", err)
	}

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Cancel the job
	if err := q.Cancel(job.ID); err != nil {
		t.Fatalf("Failed to cancel job: %v", err)
	}

	// Check status
	job, err = q.Get(job.ID)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if job.Status != JobStatusCancelled {
		t.Errorf("Expected status %s, got %s", JobStatusCancelled, job.Status)
	}
}

func TestJobList(t *testing.T) {
	q := NewQueue(2)
	defer q.Stop()

	// Submit multiple jobs
	for i := 0; i < 3; i++ {
		_, err := q.Submit(JobTypeApply, "Test job", func(ctx context.Context, job *Job) (interface{}, error) {
			return nil, nil
		})
		if err != nil {
			t.Fatalf("Failed to submit job %d: %v", i, err)
		}
	}

	jobs := q.List()
	if len(jobs) != 3 {
		t.Errorf("Expected 3 jobs, got %d", len(jobs))
	}
}

func TestJobLogging(t *testing.T) {
	q := NewQueue(1)
	defer q.Stop()

	job, err := q.Submit(JobTypeApply, "Test logging", func(ctx context.Context, job *Job) (interface{}, error) {
		job.LogInfo("test", "info message")
		job.LogWarn("test", "warn message")
		job.LogError("test", "error message")
		return nil, nil
	})

	if err != nil {
		t.Fatalf("Failed to submit job: %v", err)
	}

	// Wait for completion
	q.Wait(job.ID)

	// Check logs
	job, _ = q.Get(job.ID)
	if len(job.Logs) != 3 {
		t.Errorf("Expected 3 log entries, got %d", len(job.Logs))
	}

	if job.Logs[0].Level != LogLevelInfo {
		t.Errorf("Expected log level %s, got %s", LogLevelInfo, job.Logs[0].Level)
	}
}

func TestJobMetadata(t *testing.T) {
	q := NewQueue(1)
	defer q.Stop()

	job, err := q.Submit(JobTypeApply, "Test metadata", func(ctx context.Context, job *Job) (interface{}, error) {
		job.SetMetadata("key1", "value1")
		job.SetMetadata("key2", "value2")
		return nil, nil
	})

	if err != nil {
		t.Fatalf("Failed to submit job: %v", err)
	}

	// Wait for completion
	q.Wait(job.ID)

	// Check metadata
	job, _ = q.Get(job.ID)
	if len(job.Metadata) != 2 {
		t.Errorf("Expected 2 metadata entries, got %d", len(job.Metadata))
	}

	if job.Metadata["key1"] != "value1" {
		t.Errorf("Expected metadata key1=value1, got %s", job.Metadata["key1"])
	}
}
