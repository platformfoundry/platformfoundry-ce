package events

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/platformfoundry/platformfoundry-ce/pkg/types"
)

// EventHandler is a function that handles events
type EventHandler func(event *types.Event) error

// EventBus provides a central event distribution system
type EventBus struct {
	mu            sync.RWMutex
	subscribers   map[string][]subscription
	queue         chan *types.Event
	store         EventStore
	webhookClient *http.Client
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup

	// Configuration
	bufferSize    int
	workerCount   int
	retryAttempts int
	retryDelay    time.Duration
}

// subscription represents an internal subscription
type subscription struct {
	id      string
	filter  types.EventFilter
	handler EventHandler
	webhook *types.EventSubscription
}

// EventStore interface for persisting events
type EventStore interface {
	// Store saves an event
	Store(ctx context.Context, event *types.Event) error

	// Query retrieves events matching a filter
	Query(ctx context.Context, filter types.EventFilter, limit, offset int) ([]*types.Event, error)

	// GetByID retrieves a single event
	GetByID(ctx context.Context, id string) (*types.Event, error)

	// GetByCorrelation retrieves all events with a correlation ID
	GetByCorrelation(ctx context.Context, correlationID string) ([]*types.Event, error)

	// Count returns the number of events matching a filter
	Count(ctx context.Context, filter types.EventFilter) (int64, error)

	// Prune removes events older than the given duration
	Prune(ctx context.Context, olderThan time.Duration) (int64, error)
}

// BusConfig configures the event bus
type BusConfig struct {
	BufferSize    int           `yaml:"bufferSize" json:"bufferSize"`
	WorkerCount   int           `yaml:"workerCount" json:"workerCount"`
	RetryAttempts int           `yaml:"retryAttempts" json:"retryAttempts"`
	RetryDelay    time.Duration `yaml:"retryDelay" json:"retryDelay"`
	Store         EventStore    `yaml:"-" json:"-"`
}

// NewEventBus creates a new event bus
func NewEventBus(config BusConfig) *EventBus {
	if config.BufferSize == 0 {
		config.BufferSize = 1000
	}
	if config.WorkerCount == 0 {
		config.WorkerCount = 4
	}
	if config.RetryAttempts == 0 {
		config.RetryAttempts = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	bus := &EventBus{
		subscribers:   make(map[string][]subscription),
		queue:         make(chan *types.Event, config.BufferSize),
		store:         config.Store,
		webhookClient: &http.Client{Timeout: 30 * time.Second},
		ctx:           ctx,
		cancel:        cancel,
		bufferSize:    config.BufferSize,
		workerCount:   config.WorkerCount,
		retryAttempts: config.RetryAttempts,
		retryDelay:    config.RetryDelay,
	}

	return bus
}

// Start begins processing events
func (b *EventBus) Start() {
	for i := 0; i < b.workerCount; i++ {
		b.wg.Add(1)
		go b.worker(i)
	}
}

// Stop gracefully shuts down the event bus
func (b *EventBus) Stop() {
	b.cancel()
	close(b.queue)
	b.wg.Wait()
}

// Publish sends an event to all matching subscribers
func (b *EventBus) Publish(event *types.Event) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	// Ensure event has an ID and timestamp
	if event.ID == "" {
		event.ID = generateID("evt")
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Store event if persistence is enabled
	if b.store != nil {
		if err := b.store.Store(b.ctx, event); err != nil {
			return fmt.Errorf("failed to store event: %w", err)
		}
	}

	// Queue for async delivery
	select {
	case b.queue <- event:
		return nil
	case <-b.ctx.Done():
		return fmt.Errorf("event bus is shutting down")
	default:
		return fmt.Errorf("event queue is full")
	}
}

// PublishSync synchronously delivers an event to all subscribers
func (b *EventBus) PublishSync(ctx context.Context, event *types.Event) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	if event.ID == "" {
		event.ID = generateID("evt")
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Store event
	if b.store != nil {
		if err := b.store.Store(ctx, event); err != nil {
			return fmt.Errorf("failed to store event: %w", err)
		}
	}

	// Deliver synchronously
	return b.deliver(ctx, event)
}

// Subscribe registers a handler for events matching the filter
func (b *EventBus) Subscribe(id string, filter types.EventFilter, handler EventHandler) error {
	if id == "" {
		return fmt.Errorf("subscription ID cannot be empty")
	}
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Check for duplicate
	for _, subs := range b.subscribers {
		for _, sub := range subs {
			if sub.id == id {
				return fmt.Errorf("subscription with ID %s already exists", id)
			}
		}
	}

	sub := subscription{
		id:      id,
		filter:  filter,
		handler: handler,
	}

	// Index by event types for faster lookup
	if len(filter.Types) > 0 {
		for _, t := range filter.Types {
			key := string(t)
			b.subscribers[key] = append(b.subscribers[key], sub)
		}
	} else {
		// Subscribe to all events
		b.subscribers["*"] = append(b.subscribers["*"], sub)
	}

	return nil
}

// SubscribeWebhook registers a webhook subscription
func (b *EventBus) SubscribeWebhook(sub *types.EventSubscription) error {
	if sub == nil {
		return fmt.Errorf("subscription cannot be nil")
	}
	if sub.ID == "" {
		return fmt.Errorf("subscription ID cannot be empty")
	}
	if sub.WebhookURL == "" {
		return fmt.Errorf("webhook URL cannot be empty")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	subscription := subscription{
		id:      sub.ID,
		filter:  sub.Filter,
		webhook: sub,
	}

	if len(sub.Filter.Types) > 0 {
		for _, t := range sub.Filter.Types {
			key := string(t)
			b.subscribers[key] = append(b.subscribers[key], subscription)
		}
	} else {
		b.subscribers["*"] = append(b.subscribers["*"], subscription)
	}

	return nil
}

// Unsubscribe removes a subscription
func (b *EventBus) Unsubscribe(id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	found := false
	for key, subs := range b.subscribers {
		filtered := make([]subscription, 0, len(subs))
		for _, sub := range subs {
			if sub.id != id {
				filtered = append(filtered, sub)
			} else {
				found = true
			}
		}
		if len(filtered) == 0 {
			delete(b.subscribers, key)
		} else {
			b.subscribers[key] = filtered
		}
	}

	if !found {
		return fmt.Errorf("subscription %s not found", id)
	}
	return nil
}

// Query retrieves historical events
func (b *EventBus) Query(ctx context.Context, filter types.EventFilter, limit, offset int) ([]*types.Event, error) {
	if b.store == nil {
		return nil, fmt.Errorf("event store not configured")
	}
	return b.store.Query(ctx, filter, limit, offset)
}

// GetEvent retrieves a single event by ID
func (b *EventBus) GetEvent(ctx context.Context, id string) (*types.Event, error) {
	if b.store == nil {
		return nil, fmt.Errorf("event store not configured")
	}
	return b.store.GetByID(ctx, id)
}

// GetCorrelatedEvents retrieves all events with the same correlation ID
func (b *EventBus) GetCorrelatedEvents(ctx context.Context, correlationID string) ([]*types.Event, error) {
	if b.store == nil {
		return nil, fmt.Errorf("event store not configured")
	}
	return b.store.GetByCorrelation(ctx, correlationID)
}

// worker processes events from the queue
func (b *EventBus) worker(id int) {
	defer b.wg.Done()

	for event := range b.queue {
		if err := b.deliver(b.ctx, event); err != nil {
			// Log error but don't fail
			fmt.Printf("worker %d: failed to deliver event %s: %v\n", id, event.ID, err)
		}
	}
}

// deliver sends an event to all matching subscribers
func (b *EventBus) deliver(ctx context.Context, event *types.Event) error {
	b.mu.RLock()

	// Collect matching subscribers
	var matches []subscription

	// Check type-specific subscribers
	if subs, ok := b.subscribers[string(event.Type)]; ok {
		for _, sub := range subs {
			if sub.filter.Matches(event) {
				matches = append(matches, sub)
			}
		}
	}

	// Check wildcard subscribers
	if subs, ok := b.subscribers["*"]; ok {
		for _, sub := range subs {
			if sub.filter.Matches(event) {
				matches = append(matches, sub)
			}
		}
	}

	b.mu.RUnlock()

	// Deliver to all matching subscribers
	var errs []error
	for _, sub := range matches {
		if sub.handler != nil {
			if err := b.deliverToHandler(ctx, event, sub); err != nil {
				errs = append(errs, err)
			}
		}
		if sub.webhook != nil {
			if err := b.deliverToWebhook(ctx, event, sub.webhook); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("delivery failed for %d subscribers", len(errs))
	}
	return nil
}

// deliverToHandler calls a handler with retry
func (b *EventBus) deliverToHandler(ctx context.Context, event *types.Event, sub subscription) error {
	var lastErr error
	for attempt := 0; attempt < b.retryAttempts; attempt++ {
		if err := sub.handler(event); err != nil {
			lastErr = err
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(b.retryDelay * time.Duration(attempt+1)):
				continue
			}
		}
		return nil
	}
	return fmt.Errorf("handler %s failed after %d attempts: %w", sub.id, b.retryAttempts, lastErr)
}

// deliverToWebhook sends an event to a webhook endpoint
func (b *EventBus) deliverToWebhook(ctx context.Context, event *types.Event, sub *types.EventSubscription) error {
	if !sub.Active {
		return nil
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < b.retryAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "POST", sub.WebhookURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-PF-Event-ID", event.ID)
		req.Header.Set("X-PF-Event-Type", string(event.Type))

		// Sign the payload if secret is configured
		if sub.WebhookSecret != "" {
			signature := signPayload(payload, sub.WebhookSecret)
			req.Header.Set("X-PF-Signature", signature)
		}

		resp, err := b.webhookClient.Do(req)
		if err != nil {
			lastErr = err
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(b.retryDelay * time.Duration(attempt+1)):
				continue
			}
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}

		lastErr = fmt.Errorf("webhook returned status %d", resp.StatusCode)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			// Client error, don't retry
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(b.retryDelay * time.Duration(attempt+1)):
			continue
		}
	}

	return fmt.Errorf("webhook %s failed after %d attempts: %w", sub.WebhookURL, b.retryAttempts, lastErr)
}

// signPayload creates an HMAC-SHA256 signature
func signPayload(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}

// generateID generates a unique ID with a prefix
func generateID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

// Stats returns event bus statistics
type BusStats struct {
	QueueSize       int `json:"queue_size"`
	QueueCapacity   int `json:"queue_capacity"`
	SubscriberCount int `json:"subscriber_count"`
	WorkerCount     int `json:"worker_count"`
}

// Stats returns current statistics
func (b *EventBus) Stats() BusStats {
	b.mu.RLock()
	defer b.mu.RUnlock()

	totalSubs := 0
	seen := make(map[string]bool)
	for _, subs := range b.subscribers {
		for _, sub := range subs {
			if !seen[sub.id] {
				seen[sub.id] = true
				totalSubs++
			}
		}
	}

	return BusStats{
		QueueSize:       len(b.queue),
		QueueCapacity:   b.bufferSize,
		SubscriberCount: totalSubs,
		WorkerCount:     b.workerCount,
	}
}
