package scaling

import (
	"sync"
)

// InMemoryRecorder stores scaling events in memory
type InMemoryRecorder struct {
	events map[string][]ScalingEvent
	mu     sync.RWMutex
	limit  int
}

// NewInMemoryRecorder creates a new in-memory event recorder
func NewInMemoryRecorder(limit int) *InMemoryRecorder {
	if limit <= 0 {
		limit = 1000
	}
	return &InMemoryRecorder{
		events: make(map[string][]ScalingEvent),
		limit:  limit,
	}
}

// Record records a scaling event
func (r *InMemoryRecorder) Record(event ScalingEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()

	events := r.events[event.PolicyName]
	events = append(events, event)

	// Keep only the last N events
	if len(events) > r.limit {
		events = events[len(events)-r.limit:]
	}

	r.events[event.PolicyName] = events
}

// GetHistory returns scaling history for a policy
func (r *InMemoryRecorder) GetHistory(policyName string, limit int) []ScalingEvent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	events, ok := r.events[policyName]
	if !ok {
		return nil
	}

	if limit <= 0 || limit > len(events) {
		limit = len(events)
	}

	// Return most recent events first
	result := make([]ScalingEvent, limit)
	for i := 0; i < limit; i++ {
		result[i] = events[len(events)-1-i]
	}

	return result
}

// GetAllHistory returns all scaling history
func (r *InMemoryRecorder) GetAllHistory(limit int) []ScalingEvent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var allEvents []ScalingEvent
	for _, events := range r.events {
		allEvents = append(allEvents, events...)
	}

	// Sort by timestamp descending
	for i := 0; i < len(allEvents)-1; i++ {
		for j := i + 1; j < len(allEvents); j++ {
			if allEvents[j].Timestamp.After(allEvents[i].Timestamp) {
				allEvents[i], allEvents[j] = allEvents[j], allEvents[i]
			}
		}
	}

	if limit <= 0 || limit > len(allEvents) {
		return allEvents
	}

	return allEvents[:limit]
}

// Clear clears all events for a policy
func (r *InMemoryRecorder) Clear(policyName string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.events, policyName)
}

// ClearAll clears all events
func (r *InMemoryRecorder) ClearAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = make(map[string][]ScalingEvent)
}

// Stats returns statistics about recorded events
func (r *InMemoryRecorder) Stats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	totalEvents := 0
	policyStats := make(map[string]int)

	for name, events := range r.events {
		policyStats[name] = len(events)
		totalEvents += len(events)
	}

	return map[string]interface{}{
		"totalEvents":   totalEvents,
		"totalPolicies": len(r.events),
		"perPolicy":     policyStats,
	}
}
