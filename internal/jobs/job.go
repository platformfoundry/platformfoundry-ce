package jobs

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Job represents an asynchronous job
type Job struct {
	ID          string
	Type        JobType
	Description string
	Status      JobStatus
	Progress    int // 0-100
	CreatedAt   time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
	Error       error
	Result      interface{}
	Logs        []LogEntry
	Metadata    map[string]string

	// Cancellation
	cancel context.CancelFunc
	ctx    context.Context
}

// JobType defines job types
type JobType string

const (
	JobTypeApply    JobType = "apply"
	JobTypeDelete   JobType = "delete"
	JobTypeValidate JobType = "validate"
	JobTypePlan     JobType = "plan"
)

// JobStatus defines job status
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

// LogEntry represents a log entry
type LogEntry struct {
	Timestamp time.Time
	Level     LogLevel
	Message   string
	Component string
}

// LogLevel defines log levels
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// JobFunc is the function signature for job execution
type JobFunc func(ctx context.Context, job *Job) (interface{}, error)

// Queue manages asynchronous jobs
type Queue struct {
	jobs      map[string]*Job
	mu        sync.RWMutex
	workers   int
	workChan  chan *jobWork
	stopChan  chan struct{}
	wg        sync.WaitGroup
	listeners []JobListener
}

type jobWork struct {
	job *Job
	fn  JobFunc
}

// JobListener interface for job events
type JobListener interface {
	OnJobStarted(job *Job)
	OnJobProgress(job *Job)
	OnJobCompleted(job *Job)
	OnJobFailed(job *Job, err error)
}

// NewQueue creates a new job queue
func NewQueue(workers int) *Queue {
	q := &Queue{
		jobs:      make(map[string]*Job),
		workers:   workers,
		workChan:  make(chan *jobWork, 100),
		stopChan:  make(chan struct{}),
		listeners: []JobListener{},
	}

	// Start worker pool
	for i := 0; i < workers; i++ {
		q.wg.Add(1)
		go q.worker()
	}

	return q
}

// Submit submits a new job to the queue
func (q *Queue) Submit(jobType JobType, description string, fn JobFunc) (*Job, error) {
	ctx, cancel := context.WithCancel(context.Background())

	job := &Job{
		ID:          uuid.New().String(),
		Type:        jobType,
		Description: description,
		Status:      JobStatusPending,
		Progress:    0,
		CreatedAt:   time.Now(),
		Logs:        []LogEntry{},
		Metadata:    make(map[string]string),
		cancel:      cancel,
		ctx:         ctx,
	}

	q.mu.Lock()
	q.jobs[job.ID] = job
	q.mu.Unlock()

	// Add to work queue
	q.workChan <- &jobWork{
		job: job,
		fn:  fn,
	}

	return job, nil
}

// Get retrieves a job by ID
func (q *Queue) Get(jobID string) (*Job, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	job, exists := q.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job %s not found", jobID)
	}

	return job, nil
}

// List returns all jobs
func (q *Queue) List() []*Job {
	q.mu.RLock()
	defer q.mu.RUnlock()

	jobs := make([]*Job, 0, len(q.jobs))
	for _, job := range q.jobs {
		jobs = append(jobs, job)
	}

	return jobs
}

// Cancel cancels a job
func (q *Queue) Cancel(jobID string) error {
	q.mu.RLock()
	job, exists := q.jobs[jobID]
	q.mu.RUnlock()

	if !exists {
		return fmt.Errorf("job %s not found", jobID)
	}

	if job.Status != JobStatusPending && job.Status != JobStatusRunning {
		return fmt.Errorf("job %s cannot be cancelled (status: %s)", jobID, job.Status)
	}

	job.cancel()

	q.mu.Lock()
	job.Status = JobStatusCancelled
	now := time.Now()
	job.CompletedAt = &now
	q.mu.Unlock()

	return nil
}

// Wait waits for a job to complete
func (q *Queue) Wait(jobID string) error {
	for {
		job, err := q.Get(jobID)
		if err != nil {
			return err
		}

		if job.Status == JobStatusCompleted {
			return nil
		}

		if job.Status == JobStatusFailed {
			return job.Error
		}

		if job.Status == JobStatusCancelled {
			return fmt.Errorf("job was cancelled")
		}

		time.Sleep(500 * time.Millisecond)
	}
}

// AddListener adds a job listener
func (q *Queue) AddListener(listener JobListener) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.listeners = append(q.listeners, listener)
}

// Stop stops the job queue
func (q *Queue) Stop() {
	close(q.stopChan)
	q.wg.Wait()
}

// worker processes jobs from the queue
func (q *Queue) worker() {
	defer q.wg.Done()

	for {
		select {
		case <-q.stopChan:
			return

		case work := <-q.workChan:
			q.executeJob(work)
		}
	}
}

// executeJob executes a job
func (q *Queue) executeJob(work *jobWork) {
	job := work.job
	now := time.Now()

	// Update status to running
	q.mu.Lock()
	job.Status = JobStatusRunning
	job.StartedAt = &now
	q.mu.Unlock()

	q.notifyStarted(job)

	// Execute job function
	result, err := work.fn(job.ctx, job)

	// Update job with result
	completedAt := time.Now()
	q.mu.Lock()

	// Don't overwrite status if already cancelled
	if job.Status == JobStatusCancelled {
		job.CompletedAt = &completedAt
		q.mu.Unlock()
		return
	}

	job.CompletedAt = &completedAt
	job.Result = result

	if err != nil {
		job.Status = JobStatusFailed
		job.Error = err
		q.mu.Unlock()
		q.notifyFailed(job, err)
	} else {
		job.Status = JobStatusCompleted
		job.Progress = 100
		q.mu.Unlock()
		q.notifyCompleted(job)
	}
}

// Helper methods for logging and progress
func (j *Job) Log(level LogLevel, component, message string) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Component: component,
		Message:   message,
	}
	j.Logs = append(j.Logs, entry)
}

func (j *Job) LogInfo(component, message string) {
	j.Log(LogLevelInfo, component, message)
}

func (j *Job) LogError(component, message string) {
	j.Log(LogLevelError, component, message)
}

func (j *Job) LogWarn(component, message string) {
	j.Log(LogLevelWarn, component, message)
}

func (j *Job) SetProgress(progress int) {
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	j.Progress = progress
}

func (j *Job) SetMetadata(key, value string) {
	j.Metadata[key] = value
}

// Notification methods
func (q *Queue) notifyStarted(job *Job) {
	for _, listener := range q.listeners {
		go listener.OnJobStarted(job)
	}
}

func (q *Queue) notifyCompleted(job *Job) {
	for _, listener := range q.listeners {
		go listener.OnJobCompleted(job)
	}
}

func (q *Queue) notifyFailed(job *Job, err error) {
	for _, listener := range q.listeners {
		go listener.OnJobFailed(job, err)
	}
}

// DefaultJobListener is a simple console logger
type DefaultJobListener struct{}

func (l *DefaultJobListener) OnJobStarted(job *Job) {
	fmt.Printf("[Job %s] Started: %s\n", job.ID, job.Description)
}

func (l *DefaultJobListener) OnJobProgress(job *Job) {
	fmt.Printf("[Job %s] Progress: %d%%\n", job.ID, job.Progress)
}

func (l *DefaultJobListener) OnJobCompleted(job *Job) {
	fmt.Printf("[Job %s] Completed successfully\n", job.ID)
}

func (l *DefaultJobListener) OnJobFailed(job *Job, err error) {
	fmt.Printf("[Job %s] Failed: %v\n", job.ID, err)
}
