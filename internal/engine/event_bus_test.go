package engine

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewEventBus(t *testing.T) {
	bus := NewEventBus()

	if bus == nil {
		t.Fatal("expected non-nil event bus")
	}
	if bus.SubscriberCount() != 0 {
		t.Errorf("expected 0 subscribers, got %d", bus.SubscriberCount())
	}
	if bus.EventCount() != 0 {
		t.Errorf("expected 0 events, got %d", bus.EventCount())
	}
}

func TestEventBusSubscribe(t *testing.T) {
	bus := NewEventBus()

	listener := NewFuncListener(func(event EngineEvent) {})

	bus.Subscribe(listener)

	if bus.SubscriberCount() != 1 {
		t.Errorf("expected 1 subscriber, got %d", bus.SubscriberCount())
	}

	// Subscribe another
	bus.Subscribe(NewFuncListener(func(event EngineEvent) {}))

	if bus.SubscriberCount() != 2 {
		t.Errorf("expected 2 subscribers, got %d", bus.SubscriberCount())
	}
}

func TestEventBusUnsubscribe(t *testing.T) {
	bus := NewEventBus()

	listener := NewFuncListener(func(event EngineEvent) {})

	bus.Subscribe(listener)
	bus.Unsubscribe(listener)

	if bus.SubscriberCount() != 0 {
		t.Errorf("expected 0 subscribers, got %d", bus.SubscriberCount())
	}

	// Unsubscribe non-existent should not panic
	bus.Unsubscribe(listener)
}

func TestEventBusEmit(t *testing.T) {
	bus := NewEventBus()

	var receivedEvent EngineEvent
	var received bool
	var mu sync.Mutex

	listener := NewFuncListener(func(event EngineEvent) {
		mu.Lock()
		receivedEvent = event
		received = true
		mu.Unlock()
	})

	bus.Subscribe(listener)

	event := EngineEvent{
		EngineID:  "test-engine",
		Type:      EventTypeLog,
		Message:   "test message",
		Component: "test",
	}

	bus.Emit(event)

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if !received {
		t.Error("expected event to be received")
	}
	if receivedEvent.Message != "test message" {
		t.Errorf("expected 'test message', got '%s'", receivedEvent.Message)
	}
	if receivedEvent.Timestamp.IsZero() {
		t.Error("expected timestamp to be set")
	}
}

func TestEventBusOnEvent(t *testing.T) {
	bus := NewEventBus()

	var eventCount int32

	listener := NewFuncListener(func(event EngineEvent) {
		atomic.AddInt32(&eventCount, 1)
	})

	bus.Subscribe(listener)

	// OnEvent should work same as Emit but takes event directly
	bus.OnEvent(EngineEvent{Type: EventTypeLog})
	bus.OnEvent(EngineEvent{Type: EventTypeProgress})

	time.Sleep(50 * time.Millisecond)

	if atomic.LoadInt32(&eventCount) != 2 {
		t.Errorf("expected 2 events, got %d", atomic.LoadInt32(&eventCount))
	}
}

func TestEventBusEmitCoordinatorEvent(t *testing.T) {
	bus := NewEventBus()

	var receivedEvent EngineEvent
	var mu sync.Mutex

	listener := NewFuncListener(func(event EngineEvent) {
		mu.Lock()
		receivedEvent = event
		mu.Unlock()
	})

	bus.Subscribe(listener)

	metadata := map[string]interface{}{"key": "value"}
	bus.EmitCoordinatorEvent("test_event", "test message", metadata)

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if receivedEvent.Type != EventType("test_event") {
		t.Errorf("expected type 'test_event', got '%s'", receivedEvent.Type)
	}
	if receivedEvent.Message != "test message" {
		t.Errorf("expected 'test message', got '%s'", receivedEvent.Message)
	}
	if receivedEvent.Metadata["key"] != "value" {
		t.Error("expected metadata to be set")
	}
}

func TestEventBusEventLog(t *testing.T) {
	bus := NewEventBus()

	// Emit some events
	for i := 0; i < 5; i++ {
		bus.Emit(EngineEvent{
			Type:    EventTypeLog,
			Message: "event",
		})
	}

	if bus.EventCount() != 5 {
		t.Errorf("expected 5 events in log, got %d", bus.EventCount())
	}

	log := bus.GetEventLog()
	if len(log) != 5 {
		t.Errorf("expected 5 events in returned log, got %d", len(log))
	}
}

func TestEventBusEventLogMaxSize(t *testing.T) {
	bus := NewEventBus()
	bus.SetMaxLogSize(10)

	// Emit more than max
	for i := 0; i < 20; i++ {
		bus.Emit(EngineEvent{
			Type:    EventTypeLog,
			Message: "event",
		})
	}

	if bus.EventCount() != 10 {
		t.Errorf("expected 10 events (max size), got %d", bus.EventCount())
	}
}

func TestEventBusClearLog(t *testing.T) {
	bus := NewEventBus()

	for i := 0; i < 5; i++ {
		bus.Emit(EngineEvent{Type: EventTypeLog})
	}

	bus.ClearLog()

	if bus.EventCount() != 0 {
		t.Errorf("expected 0 events after clear, got %d", bus.EventCount())
	}
}

func TestEventBusGetEventLogSince(t *testing.T) {
	bus := NewEventBus()

	before := time.Now()
	time.Sleep(10 * time.Millisecond)

	// Emit events
	for i := 0; i < 3; i++ {
		bus.Emit(EngineEvent{Type: EventTypeLog})
	}

	events := bus.GetEventLogSince(before)
	if len(events) != 3 {
		t.Errorf("expected 3 events since before, got %d", len(events))
	}

	// Future time should return no events
	events = bus.GetEventLogSince(time.Now().Add(time.Hour))
	if len(events) != 0 {
		t.Errorf("expected 0 events for future time, got %d", len(events))
	}
}

func TestEventBusGetEventsByEngine(t *testing.T) {
	bus := NewEventBus()

	bus.Emit(EngineEvent{EngineID: "engine1", Type: EventTypeLog})
	bus.Emit(EngineEvent{EngineID: "engine1", Type: EventTypeProgress})
	bus.Emit(EngineEvent{EngineID: "engine2", Type: EventTypeLog})

	events := bus.GetEventsByEngine("engine1")
	if len(events) != 2 {
		t.Errorf("expected 2 events for engine1, got %d", len(events))
	}

	events = bus.GetEventsByEngine("engine2")
	if len(events) != 1 {
		t.Errorf("expected 1 event for engine2, got %d", len(events))
	}

	events = bus.GetEventsByEngine("nonexistent")
	if len(events) != 0 {
		t.Errorf("expected 0 events for nonexistent, got %d", len(events))
	}
}

func TestEventBusGetEventsByType(t *testing.T) {
	bus := NewEventBus()

	bus.Emit(EngineEvent{Type: EventTypeLog})
	bus.Emit(EngineEvent{Type: EventTypeLog})
	bus.Emit(EngineEvent{Type: EventTypeProgress})
	bus.Emit(EngineEvent{Type: EventTypeError})

	events := bus.GetEventsByType(EventTypeLog)
	if len(events) != 2 {
		t.Errorf("expected 2 log events, got %d", len(events))
	}

	events = bus.GetEventsByType(EventTypeError)
	if len(events) != 1 {
		t.Errorf("expected 1 error event, got %d", len(events))
	}
}

func TestEventBusSetMaxLogSize(t *testing.T) {
	bus := NewEventBus()

	// Add events
	for i := 0; i < 100; i++ {
		bus.Emit(EngineEvent{Type: EventTypeLog})
	}

	// Reduce max size
	bus.SetMaxLogSize(50)

	// Add more to trigger trim
	bus.Emit(EngineEvent{Type: EventTypeLog})

	if bus.EventCount() > 50 {
		t.Errorf("expected at most 50 events, got %d", bus.EventCount())
	}
}

func TestEventBusConcurrentEmit(t *testing.T) {
	bus := NewEventBus()

	var receivedCount int32

	listener := NewFuncListener(func(event EngineEvent) {
		atomic.AddInt32(&receivedCount, 1)
	})

	bus.Subscribe(listener)

	var wg sync.WaitGroup
	iterations := 100

	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			bus.Emit(EngineEvent{
				Type:    EventTypeLog,
				Message: "concurrent event",
			})
		}(i)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	if bus.EventCount() != iterations {
		t.Errorf("expected %d events in log, got %d", iterations, bus.EventCount())
	}

	if atomic.LoadInt32(&receivedCount) != int32(iterations) {
		t.Errorf("expected %d received events, got %d", iterations, atomic.LoadInt32(&receivedCount))
	}
}

// FuncListener tests

func TestNewFuncListener(t *testing.T) {
	var called bool

	listener := NewFuncListener(func(event EngineEvent) {
		called = true
	})

	listener.OnEvent(EngineEvent{})

	if !called {
		t.Error("expected handler to be called")
	}
}

func TestFuncListenerNilHandler(t *testing.T) {
	listener := NewFuncListener(nil)

	// Should not panic
	listener.OnEvent(EngineEvent{})
}

// ChannelListener tests

func TestNewChannelListener(t *testing.T) {
	listener := NewChannelListener(10, false)

	if listener == nil {
		t.Fatal("expected non-nil listener")
	}

	ch := listener.Channel()
	if ch == nil {
		t.Error("expected non-nil channel")
	}
}

func TestChannelListenerNonBlocking(t *testing.T) {
	listener := NewChannelListener(1, false) // Buffer of 1

	// Fill buffer
	listener.OnEvent(EngineEvent{Message: "first"})

	// Should not block on full buffer
	done := make(chan bool)
	go func() {
		listener.OnEvent(EngineEvent{Message: "second"})
		done <- true
	}()

	select {
	case <-done:
		// Good, didn't block
	case <-time.After(100 * time.Millisecond):
		t.Error("expected non-blocking listener to not block")
	}

	// Read the first event
	event := <-listener.Channel()
	if event.Message != "first" {
		t.Errorf("expected 'first', got '%s'", event.Message)
	}
}

func TestChannelListenerBlocking(t *testing.T) {
	listener := NewChannelListener(1, true) // Buffer of 1, blocking

	// Fill buffer
	listener.OnEvent(EngineEvent{Message: "first"})

	// Should block on full buffer
	blocked := make(chan bool)
	go func() {
		listener.OnEvent(EngineEvent{Message: "second"}) // This will block
		blocked <- true
	}()

	select {
	case <-blocked:
		t.Error("expected blocking listener to block")
	case <-time.After(50 * time.Millisecond):
		// Good, it's blocking
	}

	// Read to unblock
	<-listener.Channel()

	// Now the second event should complete
	select {
	case <-blocked:
		// Good
	case <-time.After(100 * time.Millisecond):
		t.Error("expected listener to unblock after read")
	}
}

func TestChannelListenerClose(t *testing.T) {
	listener := NewChannelListener(10, false)

	listener.Close()

	// Reading from closed channel should not block
	_, ok := <-listener.Channel()
	if ok {
		t.Error("expected channel to be closed")
	}
}

// FilteredListener tests

func TestNewFilteredListener(t *testing.T) {
	var receivedEvents []EngineEvent
	var mu sync.Mutex

	inner := NewFuncListener(func(event EngineEvent) {
		mu.Lock()
		receivedEvents = append(receivedEvents, event)
		mu.Unlock()
	})

	// Filter only log events
	filter := func(event EngineEvent) bool {
		return event.Type == EventTypeLog
	}

	listener := NewFilteredListener(inner, filter)

	listener.OnEvent(EngineEvent{Type: EventTypeLog, Message: "log"})
	listener.OnEvent(EngineEvent{Type: EventTypeProgress, Message: "progress"})
	listener.OnEvent(EngineEvent{Type: EventTypeLog, Message: "log2"})

	mu.Lock()
	defer mu.Unlock()

	if len(receivedEvents) != 2 {
		t.Errorf("expected 2 filtered events, got %d", len(receivedEvents))
	}
}

func TestFilteredListenerNilFilter(t *testing.T) {
	var called bool

	inner := NewFuncListener(func(event EngineEvent) {
		called = true
	})

	// Nil filter should pass all events
	listener := NewFilteredListener(inner, nil)

	listener.OnEvent(EngineEvent{})

	if !called {
		t.Error("expected event to pass through with nil filter")
	}
}

func TestFilterByEngine(t *testing.T) {
	filter := FilterByEngine("engine1")

	if !filter(EngineEvent{EngineID: "engine1"}) {
		t.Error("expected filter to pass engine1")
	}
	if filter(EngineEvent{EngineID: "engine2"}) {
		t.Error("expected filter to block engine2")
	}
}

func TestFilterByType(t *testing.T) {
	filter := FilterByType(EventTypeLog)

	if !filter(EngineEvent{Type: EventTypeLog}) {
		t.Error("expected filter to pass log events")
	}
	if filter(EngineEvent{Type: EventTypeError}) {
		t.Error("expected filter to block error events")
	}
}

func TestFilterByComponent(t *testing.T) {
	filter := FilterByComponent("infrastructure")

	if !filter(EngineEvent{Component: "infrastructure"}) {
		t.Error("expected filter to pass infrastructure component")
	}
	if filter(EngineEvent{Component: "orchestrator"}) {
		t.Error("expected filter to block orchestrator component")
	}
}

// AggregatingListener tests

func TestNewAggregatingListener(t *testing.T) {
	var aggregatedEvents []EngineEvent
	var mu sync.Mutex

	handler := func(events []EngineEvent) {
		mu.Lock()
		aggregatedEvents = append(aggregatedEvents, events...)
		mu.Unlock()
	}

	listener := NewAggregatingListener(3, handler)

	// Send events less than max
	listener.OnEvent(EngineEvent{Message: "1"})
	listener.OnEvent(EngineEvent{Message: "2"})

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if len(aggregatedEvents) != 0 {
		t.Error("expected no events until max reached")
	}
	mu.Unlock()

	// Send third event to trigger flush
	listener.OnEvent(EngineEvent{Message: "3"})

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if len(aggregatedEvents) != 3 {
		t.Errorf("expected 3 aggregated events, got %d", len(aggregatedEvents))
	}
	mu.Unlock()
}

func TestAggregatingListenerFlush(t *testing.T) {
	var aggregatedEvents []EngineEvent
	var mu sync.Mutex

	handler := func(events []EngineEvent) {
		mu.Lock()
		aggregatedEvents = append(aggregatedEvents, events...)
		mu.Unlock()
	}

	listener := NewAggregatingListener(10, handler) // Max of 10

	// Send fewer than max
	listener.OnEvent(EngineEvent{Message: "1"})
	listener.OnEvent(EngineEvent{Message: "2"})

	// Manual flush
	listener.Flush()

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(aggregatedEvents) != 2 {
		t.Errorf("expected 2 events after flush, got %d", len(aggregatedEvents))
	}
}

func TestAggregatingListenerFlushEmpty(t *testing.T) {
	var handlerCalled bool

	handler := func(events []EngineEvent) {
		handlerCalled = true
	}

	listener := NewAggregatingListener(10, handler)

	// Flush without events
	listener.Flush()

	time.Sleep(50 * time.Millisecond)

	if handlerCalled {
		t.Error("expected handler not to be called for empty flush")
	}
}

func TestAggregatingListenerNilHandler(t *testing.T) {
	listener := NewAggregatingListener(2, nil)

	// Should not panic
	listener.OnEvent(EngineEvent{})
	listener.OnEvent(EngineEvent{})
}

func TestAggregatingListenerConcurrent(t *testing.T) {
	var totalEvents int32

	handler := func(events []EngineEvent) {
		atomic.AddInt32(&totalEvents, int32(len(events)))
	}

	listener := NewAggregatingListener(5, handler)

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			listener.OnEvent(EngineEvent{})
		}()
	}

	wg.Wait()
	listener.Flush()

	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&totalEvents) != 100 {
		t.Errorf("expected 100 total events, got %d", atomic.LoadInt32(&totalEvents))
	}
}

// Integration test for event flow

func TestEventBusIntegration(t *testing.T) {
	bus := NewEventBus()

	// Track events through different listeners
	var funcEvents, channelEvents, filteredEvents []EngineEvent
	var mu sync.Mutex

	// Func listener
	funcListener := NewFuncListener(func(event EngineEvent) {
		mu.Lock()
		funcEvents = append(funcEvents, event)
		mu.Unlock()
	})

	// Channel listener
	channelListener := NewChannelListener(100, false)

	// Filtered listener (only errors)
	filteredInner := NewFuncListener(func(event EngineEvent) {
		mu.Lock()
		filteredEvents = append(filteredEvents, event)
		mu.Unlock()
	})
	filteredListener := NewFilteredListener(filteredInner, FilterByType(EventTypeError))

	bus.Subscribe(funcListener)
	bus.Subscribe(channelListener)
	bus.Subscribe(filteredListener)

	// Emit various events
	bus.Emit(EngineEvent{Type: EventTypeLog, Message: "log1"})
	bus.Emit(EngineEvent{Type: EventTypeProgress, Message: "progress1"})
	bus.Emit(EngineEvent{Type: EventTypeError, Message: "error1"})
	bus.Emit(EngineEvent{Type: EventTypeLog, Message: "log2"})

	// Collect channel events
	go func() {
		for event := range channelListener.Channel() {
			mu.Lock()
			channelEvents = append(channelEvents, event)
			mu.Unlock()
		}
	}()

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	// Func listener should receive all
	if len(funcEvents) != 4 {
		t.Errorf("expected 4 func events, got %d", len(funcEvents))
	}

	// Channel listener should receive all
	if len(channelEvents) != 4 {
		t.Errorf("expected 4 channel events, got %d", len(channelEvents))
	}

	// Filtered listener should receive only errors
	if len(filteredEvents) != 1 {
		t.Errorf("expected 1 filtered event, got %d", len(filteredEvents))
	}

	// Event log should have all
	if bus.EventCount() != 4 {
		t.Errorf("expected 4 events in log, got %d", bus.EventCount())
	}
}
