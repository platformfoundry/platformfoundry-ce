package engine

import (
	"sync"
	"time"
)

// EventBus distributes events to subscribers
type EventBus struct {
	subscribers []EventListener
	mu          sync.RWMutex
	eventLog    []EngineEvent
	logMu       sync.Mutex
	maxLogSize  int
}

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make([]EventListener, 0),
		eventLog:    make([]EngineEvent, 0),
		maxLogSize:  10000, // Keep last 10000 events
	}
}

// Subscribe adds a listener to the event bus
func (b *EventBus) Subscribe(listener EventListener) {
	b.mu.Lock()
	b.subscribers = append(b.subscribers, listener)
	b.mu.Unlock()
}

// Unsubscribe removes a listener from the event bus
func (b *EventBus) Unsubscribe(listener EventListener) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i, sub := range b.subscribers {
		if sub == listener {
			b.subscribers = append(b.subscribers[:i], b.subscribers[i+1:]...)
			return
		}
	}
}

// OnEvent receives and broadcasts an event
func (b *EventBus) OnEvent(event EngineEvent) {
	// Log event
	b.logEvent(event)

	// Broadcast to subscribers
	b.broadcast(event)
}

// Emit emits an event to the bus
func (b *EventBus) Emit(event EngineEvent) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	b.OnEvent(event)
}

// EmitCoordinatorEvent emits a coordinator-level event
func (b *EventBus) EmitCoordinatorEvent(eventType, message string, metadata map[string]interface{}) {
	b.Emit(EngineEvent{
		Type:      EventType(eventType),
		Message:   message,
		Timestamp: time.Now(),
		Metadata:  metadata,
	})
}

// logEvent adds an event to the log
func (b *EventBus) logEvent(event EngineEvent) {
	b.logMu.Lock()
	defer b.logMu.Unlock()

	b.eventLog = append(b.eventLog, event)

	// Trim log if it exceeds max size
	if len(b.eventLog) > b.maxLogSize {
		b.eventLog = b.eventLog[len(b.eventLog)-b.maxLogSize:]
	}
}

// broadcast sends an event to all subscribers
func (b *EventBus) broadcast(event EngineEvent) {
	b.mu.RLock()
	subscribers := make([]EventListener, len(b.subscribers))
	copy(subscribers, b.subscribers)
	b.mu.RUnlock()

	for _, sub := range subscribers {
		// Send to each subscriber in a goroutine to avoid blocking
		go sub.OnEvent(event)
	}
}

// GetEventLog returns a copy of the event log
func (b *EventBus) GetEventLog() []EngineEvent {
	b.logMu.Lock()
	defer b.logMu.Unlock()

	result := make([]EngineEvent, len(b.eventLog))
	copy(result, b.eventLog)
	return result
}

// GetEventLogSince returns events since a given time
func (b *EventBus) GetEventLogSince(since time.Time) []EngineEvent {
	b.logMu.Lock()
	defer b.logMu.Unlock()

	var result []EngineEvent
	for _, event := range b.eventLog {
		if event.Timestamp.After(since) || event.Timestamp.Equal(since) {
			result = append(result, event)
		}
	}
	return result
}

// GetEventsByEngine returns events for a specific engine
func (b *EventBus) GetEventsByEngine(engineID string) []EngineEvent {
	b.logMu.Lock()
	defer b.logMu.Unlock()

	var result []EngineEvent
	for _, event := range b.eventLog {
		if event.EngineID == engineID {
			result = append(result, event)
		}
	}
	return result
}

// GetEventsByType returns events of a specific type
func (b *EventBus) GetEventsByType(eventType EventType) []EngineEvent {
	b.logMu.Lock()
	defer b.logMu.Unlock()

	var result []EngineEvent
	for _, event := range b.eventLog {
		if event.Type == eventType {
			result = append(result, event)
		}
	}
	return result
}

// ClearLog clears the event log
func (b *EventBus) ClearLog() {
	b.logMu.Lock()
	defer b.logMu.Unlock()
	b.eventLog = make([]EngineEvent, 0)
}

// SetMaxLogSize sets the maximum log size
func (b *EventBus) SetMaxLogSize(size int) {
	b.logMu.Lock()
	defer b.logMu.Unlock()
	b.maxLogSize = size
}

// SubscriberCount returns the number of subscribers
func (b *EventBus) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}

// EventCount returns the number of events in the log
func (b *EventBus) EventCount() int {
	b.logMu.Lock()
	defer b.logMu.Unlock()
	return len(b.eventLog)
}

// FuncListener is a simple event listener that calls a function
type FuncListener struct {
	handler func(event EngineEvent)
}

// NewFuncListener creates a new function-based listener
func NewFuncListener(handler func(event EngineEvent)) *FuncListener {
	return &FuncListener{handler: handler}
}

// OnEvent calls the handler function
func (l *FuncListener) OnEvent(event EngineEvent) {
	if l.handler != nil {
		l.handler(event)
	}
}

// ChannelListener sends events to a channel
type ChannelListener struct {
	ch       chan EngineEvent
	blocking bool
}

// NewChannelListener creates a new channel-based listener
func NewChannelListener(bufferSize int, blocking bool) *ChannelListener {
	return &ChannelListener{
		ch:       make(chan EngineEvent, bufferSize),
		blocking: blocking,
	}
}

// OnEvent sends the event to the channel
func (l *ChannelListener) OnEvent(event EngineEvent) {
	if l.blocking {
		l.ch <- event
	} else {
		select {
		case l.ch <- event:
		default:
			// Channel full, drop event
		}
	}
}

// Channel returns the event channel
func (l *ChannelListener) Channel() <-chan EngineEvent {
	return l.ch
}

// Close closes the event channel
func (l *ChannelListener) Close() {
	close(l.ch)
}

// FilteredListener wraps a listener and filters events
type FilteredListener struct {
	inner  EventListener
	filter func(event EngineEvent) bool
}

// NewFilteredListener creates a listener that filters events
func NewFilteredListener(inner EventListener, filter func(event EngineEvent) bool) *FilteredListener {
	return &FilteredListener{
		inner:  inner,
		filter: filter,
	}
}

// OnEvent filters and forwards events
func (l *FilteredListener) OnEvent(event EngineEvent) {
	if l.filter == nil || l.filter(event) {
		l.inner.OnEvent(event)
	}
}

// FilterByEngine creates a filter for a specific engine
func FilterByEngine(engineID string) func(event EngineEvent) bool {
	return func(event EngineEvent) bool {
		return event.EngineID == engineID
	}
}

// FilterByType creates a filter for a specific event type
func FilterByType(eventType EventType) func(event EngineEvent) bool {
	return func(event EngineEvent) bool {
		return event.Type == eventType
	}
}

// FilterByComponent creates a filter for a specific component
func FilterByComponent(component string) func(event EngineEvent) bool {
	return func(event EngineEvent) bool {
		return event.Component == component
	}
}

// AggregatingListener aggregates events for batch processing
type AggregatingListener struct {
	events    []EngineEvent
	mu        sync.Mutex
	maxEvents int
	handler   func(events []EngineEvent)
}

// NewAggregatingListener creates a listener that aggregates events
func NewAggregatingListener(maxEvents int, handler func(events []EngineEvent)) *AggregatingListener {
	return &AggregatingListener{
		events:    make([]EngineEvent, 0),
		maxEvents: maxEvents,
		handler:   handler,
	}
}

// OnEvent adds an event to the aggregate
func (l *AggregatingListener) OnEvent(event EngineEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.events = append(l.events, event)

	if len(l.events) >= l.maxEvents {
		l.flush()
	}
}

// Flush processes all aggregated events
func (l *AggregatingListener) Flush() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.flush()
}

// flush is the internal flush implementation (must be called with lock held)
func (l *AggregatingListener) flush() {
	if len(l.events) == 0 {
		return
	}

	events := make([]EngineEvent, len(l.events))
	copy(events, l.events)
	l.events = l.events[:0]

	if l.handler != nil {
		go l.handler(events)
	}
}
