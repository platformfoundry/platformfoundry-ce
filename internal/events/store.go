package events

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/platformfoundry/platformfoundry-ce/pkg/types"
)

// MemoryEventStore provides an in-memory event store implementation
type MemoryEventStore struct {
	mu     sync.RWMutex
	events []*types.Event
	byID   map[string]*types.Event
	maxAge time.Duration
	maxSize int
}

// MemoryStoreConfig configures the memory store
type MemoryStoreConfig struct {
	MaxAge  time.Duration `yaml:"maxAge" json:"maxAge"`
	MaxSize int           `yaml:"maxSize" json:"maxSize"`
}

// NewMemoryEventStore creates a new in-memory event store
func NewMemoryEventStore(config MemoryStoreConfig) *MemoryEventStore {
	if config.MaxSize == 0 {
		config.MaxSize = 10000
	}
	if config.MaxAge == 0 {
		config.MaxAge = 24 * time.Hour
	}

	store := &MemoryEventStore{
		events:  make([]*types.Event, 0, config.MaxSize),
		byID:    make(map[string]*types.Event),
		maxAge:  config.MaxAge,
		maxSize: config.MaxSize,
	}

	// Start background cleanup
	go store.cleanup()

	return store
}

// Store saves an event
func (s *MemoryEventStore) Store(ctx context.Context, event *types.Event) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}
	if event.ID == "" {
		return fmt.Errorf("event ID cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicate
	if _, exists := s.byID[event.ID]; exists {
		return fmt.Errorf("event with ID %s already exists", event.ID)
	}

	// Enforce max size
	if len(s.events) >= s.maxSize {
		// Remove oldest events (first 10%)
		removeCount := s.maxSize / 10
		for i := 0; i < removeCount; i++ {
			delete(s.byID, s.events[i].ID)
		}
		s.events = s.events[removeCount:]
	}

	s.events = append(s.events, event)
	s.byID[event.ID] = event

	return nil
}

// Query retrieves events matching a filter
func (s *MemoryEventStore) Query(ctx context.Context, filter types.EventFilter, limit, offset int) ([]*types.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*types.Event

	// Iterate in reverse order (newest first)
	for i := len(s.events) - 1; i >= 0; i-- {
		event := s.events[i]
		if filter.Matches(event) {
			results = append(results, event)
		}
	}

	// Apply offset and limit
	if offset > 0 {
		if offset >= len(results) {
			return []*types.Event{}, nil
		}
		results = results[offset:]
	}
	if limit > 0 && limit < len(results) {
		results = results[:limit]
	}

	return results, nil
}

// GetByID retrieves a single event
func (s *MemoryEventStore) GetByID(ctx context.Context, id string) (*types.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	event, ok := s.byID[id]
	if !ok {
		return nil, fmt.Errorf("event %s not found", id)
	}
	return event, nil
}

// GetByCorrelation retrieves all events with a correlation ID
func (s *MemoryEventStore) GetByCorrelation(ctx context.Context, correlationID string) ([]*types.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*types.Event
	for _, event := range s.events {
		if event.CorrelationID == correlationID {
			results = append(results, event)
		}
	}

	// Sort by timestamp
	sort.Slice(results, func(i, j int) bool {
		return results[i].Timestamp.Before(results[j].Timestamp)
	})

	return results, nil
}

// Count returns the number of events matching a filter
func (s *MemoryEventStore) Count(ctx context.Context, filter types.EventFilter) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int64
	for _, event := range s.events {
		if filter.Matches(event) {
			count++
		}
	}
	return count, nil
}

// Prune removes events older than the given duration
func (s *MemoryEventStore) Prune(ctx context.Context, olderThan time.Duration) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	var pruned int64
	var remaining []*types.Event

	for _, event := range s.events {
		if event.Timestamp.Before(cutoff) {
			delete(s.byID, event.ID)
			pruned++
		} else {
			remaining = append(remaining, event)
		}
	}

	s.events = remaining
	return pruned, nil
}

// cleanup periodically removes old events
func (s *MemoryEventStore) cleanup() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		s.Prune(context.Background(), s.maxAge)
	}
}

// Size returns the current number of stored events
func (s *MemoryEventStore) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.events)
}

// Clear removes all events
func (s *MemoryEventStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = make([]*types.Event, 0, s.maxSize)
	s.byID = make(map[string]*types.Event)
}
