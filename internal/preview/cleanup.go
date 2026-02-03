package preview

import (
	"context"
	"sync"
	"time"
)

// CleanupWorker manages automatic cleanup of expired preview environments
type CleanupWorker struct {
	manager   *Manager
	interval  time.Duration
	scheduled map[string]time.Time
	mu        sync.RWMutex
	stopCh    chan struct{}
	running   bool
}

// NewCleanupWorker creates a new cleanup worker
func NewCleanupWorker(manager *Manager, interval time.Duration) *CleanupWorker {
	if interval == 0 {
		interval = 5 * time.Minute
	}

	return &CleanupWorker{
		manager:   manager,
		interval:  interval,
		scheduled: make(map[string]time.Time),
		stopCh:    make(chan struct{}),
	}
}

// Start begins the cleanup worker background process
func (w *CleanupWorker) Start(ctx context.Context) {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return
	}
	w.running = true
	w.stopCh = make(chan struct{})
	w.mu.Unlock()

	go w.run(ctx)
}

// Stop stops the cleanup worker
func (w *CleanupWorker) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return
	}

	close(w.stopCh)
	w.running = false
}

// Schedule schedules a preview environment for cleanup at the given time
func (w *CleanupWorker) Schedule(previewID string, expiresAt time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.scheduled[previewID] = expiresAt
}

// Unschedule removes a preview environment from the cleanup schedule
func (w *CleanupWorker) Unschedule(previewID string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	delete(w.scheduled, previewID)
}

// IsScheduled checks if a preview environment is scheduled for cleanup
func (w *CleanupWorker) IsScheduled(previewID string) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	_, ok := w.scheduled[previewID]
	return ok
}

// GetScheduledCount returns the number of scheduled cleanups
func (w *CleanupWorker) GetScheduledCount() int {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return len(w.scheduled)
}

// run is the main cleanup loop
func (w *CleanupWorker) run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.cleanup(ctx)
		}
	}
}

// cleanup performs the actual cleanup of expired environments
func (w *CleanupWorker) cleanup(ctx context.Context) {
	now := time.Now()

	// Get list of expired preview IDs
	w.mu.Lock()
	var expired []string
	for id, expiresAt := range w.scheduled {
		if now.After(expiresAt) {
			expired = append(expired, id)
		}
	}
	// Remove from scheduled
	for _, id := range expired {
		delete(w.scheduled, id)
	}
	w.mu.Unlock()

	// Delete expired environments
	for _, id := range expired {
		if err := w.manager.Delete(ctx, id); err != nil {
			// Log error but continue with other cleanups
			// The environment may have already been deleted
			continue
		}
	}
}

// CleanupAll immediately cleans up all scheduled environments
func (w *CleanupWorker) CleanupAll(ctx context.Context) error {
	w.mu.Lock()
	var ids []string
	for id := range w.scheduled {
		ids = append(ids, id)
	}
	w.scheduled = make(map[string]time.Time)
	w.mu.Unlock()

	var lastErr error
	for _, id := range ids {
		if err := w.manager.Delete(ctx, id); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// CleanupExpiredOnly cleans up only expired environments without waiting for interval
func (w *CleanupWorker) CleanupExpiredOnly(ctx context.Context) {
	w.cleanup(ctx)
}

// GetNextCleanup returns the time of the next scheduled cleanup
func (w *CleanupWorker) GetNextCleanup() *time.Time {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var earliest *time.Time
	for _, expiresAt := range w.scheduled {
		t := expiresAt
		if earliest == nil || t.Before(*earliest) {
			earliest = &t
		}
	}

	return earliest
}

// ListScheduled returns all scheduled cleanup times
func (w *CleanupWorker) ListScheduled() map[string]time.Time {
	w.mu.RLock()
	defer w.mu.RUnlock()

	result := make(map[string]time.Time, len(w.scheduled))
	for id, t := range w.scheduled {
		result[id] = t
	}

	return result
}
