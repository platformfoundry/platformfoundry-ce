package triggers

import (
	"context"
	"sync"
	"time"
)

// Scheduler implements cron-based workflow triggering
type Scheduler struct {
	workflowName string
	cronExpr     string
	callback     TriggerCallback
	ctx          context.Context
	cancel       context.CancelFunc
	running      bool
	mu           sync.Mutex
}

// NewScheduler creates a new scheduler trigger
func NewScheduler(workflowName, cronExpr string) *Scheduler {
	return &Scheduler{
		workflowName: workflowName,
		cronExpr:     cronExpr,
	}
}

// Type returns the trigger type
func (s *Scheduler) Type() string {
	return "schedule"
}

// OnTrigger sets the callback
func (s *Scheduler) OnTrigger(callback TriggerCallback) {
	s.callback = callback
}

// Start starts the scheduler
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	s.ctx, s.cancel = context.WithCancel(ctx)
	s.running = true

	go s.run()

	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	if s.cancel != nil {
		s.cancel()
	}
	s.running = false

	return nil
}

// run is the main scheduler loop
func (s *Scheduler) run() {
	// Parse cron expression and calculate next run time
	// For simplicity, we implement a basic cron parser here
	// In production, use a library like robfig/cron/v3

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case t := <-ticker.C:
			if s.shouldRun(t) {
				if s.callback != nil {
					s.callback(s.workflowName, map[string]interface{}{
						"trigger":   "schedule",
						"cron":      s.cronExpr,
						"timestamp": t.Format(time.RFC3339),
					})
				}
			}
		}
	}
}

// shouldRun checks if the scheduler should trigger based on cron expression
func (s *Scheduler) shouldRun(t time.Time) bool {
	// Basic cron parsing for common patterns
	// Format: minute hour day month weekday
	// Supports: *, */N, N

	parts := parseCronExpr(s.cronExpr)
	if parts == nil {
		return false
	}

	// Check each field
	if !matchesCronField(parts[0], t.Minute()) {
		return false
	}
	if !matchesCronField(parts[1], t.Hour()) {
		return false
	}
	if !matchesCronField(parts[2], t.Day()) {
		return false
	}
	if !matchesCronField(parts[3], int(t.Month())) {
		return false
	}
	if !matchesCronField(parts[4], int(t.Weekday())) {
		return false
	}

	return true
}

// parseCronExpr parses a cron expression into fields
func parseCronExpr(expr string) []string {
	// Split by whitespace
	parts := make([]string, 0)
	current := ""
	for _, c := range expr {
		if c == ' ' || c == '\t' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}

	if len(parts) != 5 {
		return nil
	}

	return parts
}

// matchesCronField checks if a value matches a cron field
func matchesCronField(field string, value int) bool {
	if field == "*" {
		return true
	}

	// Handle */N pattern
	if len(field) > 2 && field[0:2] == "*/" {
		var step int
		if _, err := parseField(field[2:], &step); err == nil {
			return value%step == 0
		}
	}

	// Handle specific value
	var expected int
	if _, err := parseField(field, &expected); err == nil {
		return value == expected
	}

	// Handle range: N-M
	for i := 0; i < len(field); i++ {
		if field[i] == '-' {
			var start, end int
			if _, err := parseField(field[:i], &start); err == nil {
				if _, err := parseField(field[i+1:], &end); err == nil {
					return value >= start && value <= end
				}
			}
		}
	}

	// Handle list: N,M,O
	for i := 0; i < len(field); i++ {
		if field[i] == ',' {
			start := 0
			for j := 0; j < len(field); j++ {
				if field[j] == ',' || j == len(field)-1 {
					end := j
					if j == len(field)-1 {
						end = len(field)
					}
					var val int
					if _, err := parseField(field[start:end], &val); err == nil {
						if val == value {
							return true
						}
					}
					start = j + 1
				}
			}
			return false
		}
	}

	return false
}

// parseField parses a numeric field
func parseField(s string, out *int) (bool, error) {
	val := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return false, nil
		}
		val = val*10 + int(c-'0')
	}
	*out = val
	return true, nil
}

// CronScheduler manages multiple workflow schedules using robfig/cron
type CronScheduler struct {
	schedules map[string]*Scheduler
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewCronScheduler creates a new cron scheduler
func NewCronScheduler() *CronScheduler {
	return &CronScheduler{
		schedules: make(map[string]*Scheduler),
	}
}

// AddSchedule adds a workflow schedule
func (cs *CronScheduler) AddSchedule(workflowName, cronExpr string, callback TriggerCallback) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	scheduler := NewScheduler(workflowName, cronExpr)
	scheduler.OnTrigger(callback)
	cs.schedules[workflowName] = scheduler

	// If already running, start this scheduler too
	if cs.ctx != nil {
		scheduler.Start(cs.ctx)
	}

	return nil
}

// RemoveSchedule removes a workflow schedule
func (cs *CronScheduler) RemoveSchedule(workflowName string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if scheduler, ok := cs.schedules[workflowName]; ok {
		scheduler.Stop()
		delete(cs.schedules, workflowName)
	}
}

// Start starts all schedulers
func (cs *CronScheduler) Start(ctx context.Context) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.ctx, cs.cancel = context.WithCancel(ctx)

	for _, scheduler := range cs.schedules {
		if err := scheduler.Start(cs.ctx); err != nil {
			return err
		}
	}

	return nil
}

// Stop stops all schedulers
func (cs *CronScheduler) Stop() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.cancel != nil {
		cs.cancel()
	}

	for _, scheduler := range cs.schedules {
		scheduler.Stop()
	}

	return nil
}
