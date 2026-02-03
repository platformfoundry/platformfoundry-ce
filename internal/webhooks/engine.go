// Package webhooks provides webhook-driven extensibility for platform events.
package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Event represents a platform event that can trigger webhooks
type Event struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"` // resource.created, deployment.started, etc.
	Source    string                 `json:"source"`
	Subject   string                 `json:"subject,omitempty"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// WebhookSubscription represents a webhook subscription
type WebhookSubscription struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	URL         string            `json:"url"`
	Events      []string          `json:"events"` // Event types to subscribe to
	Secret      string            `json:"secret"` // For HMAC signing
	Headers     map[string]string `json:"headers,omitempty"`
	Filters     []Filter          `json:"filters,omitempty"`
	Enabled     bool              `json:"enabled"`
	RetryPolicy RetryPolicy       `json:"retryPolicy,omitempty"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

// Filter defines criteria for filtering events
type Filter struct {
	Field    string `json:"field"`    // e.g., "data.environment"
	Operator string `json:"operator"` // eq, ne, contains, matches
	Value    string `json:"value"`
}

// RetryPolicy defines retry behavior for failed deliveries
type RetryPolicy struct {
	MaxRetries    int           `json:"maxRetries"`
	InitialDelay  time.Duration `json:"initialDelay"`
	MaxDelay      time.Duration `json:"maxDelay"`
	BackoffFactor float64       `json:"backoffFactor"`
}

// DeliveryResult represents the result of a webhook delivery attempt
type DeliveryResult struct {
	SubscriptionID string        `json:"subscriptionId"`
	EventID        string        `json:"eventId"`
	Status         string        `json:"status"` // success, failed, pending
	StatusCode     int           `json:"statusCode,omitempty"`
	Response       string        `json:"response,omitempty"`
	Error          string        `json:"error,omitempty"`
	Attempts       int           `json:"attempts"`
	Duration       time.Duration `json:"duration"`
	Timestamp      time.Time     `json:"timestamp"`
}

// StateBackend interface for persistence
type StateBackend interface {
	Get(ctx context.Context, kind, id string) (interface{}, error)
	Put(ctx context.Context, kind, id string, value interface{}) error
	Delete(ctx context.Context, kind, id string) error
	List(ctx context.Context, kind string) ([]interface{}, error)
}

// EngineConfig contains configuration for the webhook engine
type EngineConfig struct {
	DefaultTimeout     time.Duration
	MaxConcurrent      int
	DefaultRetryPolicy RetryPolicy
}

// Engine manages webhook subscriptions and event delivery
type Engine struct {
	subscriptions map[string]*WebhookSubscription
	stateBackend  StateBackend
	httpClient    *http.Client
	config        EngineConfig
	deliveryQueue chan deliveryTask
	mu            sync.RWMutex
	wg            sync.WaitGroup
	stopCh        chan struct{}
}

type deliveryTask struct {
	subscription *WebhookSubscription
	event        *Event
	attempt      int
}

// NewEngine creates a new webhook engine
func NewEngine(backend StateBackend, config EngineConfig) *Engine {
	if config.DefaultTimeout == 0 {
		config.DefaultTimeout = 30 * time.Second
	}
	if config.MaxConcurrent == 0 {
		config.MaxConcurrent = 10
	}
	if config.DefaultRetryPolicy.MaxRetries == 0 {
		config.DefaultRetryPolicy = RetryPolicy{
			MaxRetries:    3,
			InitialDelay:  time.Second,
			MaxDelay:      time.Minute,
			BackoffFactor: 2.0,
		}
	}

	e := &Engine{
		subscriptions: make(map[string]*WebhookSubscription),
		stateBackend:  backend,
		httpClient:    &http.Client{Timeout: config.DefaultTimeout},
		config:        config,
		deliveryQueue: make(chan deliveryTask, 1000),
		stopCh:        make(chan struct{}),
	}

	return e
}

// Start starts the webhook delivery workers
func (e *Engine) Start() {
	for i := 0; i < e.config.MaxConcurrent; i++ {
		e.wg.Add(1)
		go e.deliveryWorker()
	}
}

// Stop stops the webhook engine
func (e *Engine) Stop() {
	close(e.stopCh)
	e.wg.Wait()
}

// deliveryWorker processes delivery tasks
func (e *Engine) deliveryWorker() {
	defer e.wg.Done()

	for {
		select {
		case <-e.stopCh:
			return
		case task := <-e.deliveryQueue:
			e.deliver(context.Background(), task)
		}
	}
}

// Subscribe creates a new webhook subscription
func (e *Engine) Subscribe(ctx context.Context, sub *WebhookSubscription) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if sub.ID == "" {
		sub.ID = fmt.Sprintf("webhook-%d", time.Now().UnixNano())
	}
	sub.CreatedAt = time.Now()
	sub.UpdatedAt = time.Now()

	if sub.RetryPolicy.MaxRetries == 0 {
		sub.RetryPolicy = e.config.DefaultRetryPolicy
	}

	e.subscriptions[sub.ID] = sub

	if e.stateBackend != nil {
		return e.stateBackend.Put(ctx, "WebhookSubscription", sub.ID, sub)
	}

	return nil
}

// Unsubscribe removes a webhook subscription
func (e *Engine) Unsubscribe(ctx context.Context, id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.subscriptions, id)

	if e.stateBackend != nil {
		return e.stateBackend.Delete(ctx, "WebhookSubscription", id)
	}

	return nil
}

// GetSubscription returns a subscription by ID
func (e *Engine) GetSubscription(id string) *WebhookSubscription {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.subscriptions[id]
}

// ListSubscriptions returns all subscriptions
func (e *Engine) ListSubscriptions() []*WebhookSubscription {
	e.mu.RLock()
	defer e.mu.RUnlock()

	subs := make([]*WebhookSubscription, 0, len(e.subscriptions))
	for _, s := range e.subscriptions {
		subs = append(subs, s)
	}
	return subs
}

// Dispatch dispatches an event to matching subscribers
func (e *Engine) Dispatch(ctx context.Context, event *Event) error {
	if event.ID == "" {
		event.ID = fmt.Sprintf("evt-%d", time.Now().UnixNano())
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Find matching subscriptions
	subs := e.findMatchingSubscriptions(event)

	// Queue deliveries
	for _, sub := range subs {
		select {
		case e.deliveryQueue <- deliveryTask{subscription: sub, event: event, attempt: 1}:
		default:
			// Queue full, log and skip
			fmt.Printf("webhook delivery queue full, skipping %s for event %s\n", sub.ID, event.ID)
		}
	}

	return nil
}

// findMatchingSubscriptions finds subscriptions that match the event
func (e *Engine) findMatchingSubscriptions(event *Event) []*WebhookSubscription {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var matching []*WebhookSubscription

	for _, sub := range e.subscriptions {
		if !sub.Enabled {
			continue
		}

		// Check event type match
		if !e.matchesEventType(sub.Events, event.Type) {
			continue
		}

		// Check filters
		if !e.matchesFilters(sub.Filters, event) {
			continue
		}

		matching = append(matching, sub)
	}

	return matching
}

// matchesEventType checks if event type matches subscription events
func (e *Engine) matchesEventType(events []string, eventType string) bool {
	for _, et := range events {
		if et == "*" || et == eventType {
			return true
		}
		// Support wildcards like "resource.*"
		if len(et) > 1 && et[len(et)-1] == '*' {
			prefix := et[:len(et)-1]
			if len(eventType) >= len(prefix) && eventType[:len(prefix)] == prefix {
				return true
			}
		}
	}
	return false
}

// matchesFilters checks if event matches all filters
func (e *Engine) matchesFilters(filters []Filter, event *Event) bool {
	for _, filter := range filters {
		if !e.matchesFilter(filter, event) {
			return false
		}
	}
	return true
}

// matchesFilter checks if event matches a single filter
func (e *Engine) matchesFilter(filter Filter, event *Event) bool {
	value := e.getFieldValue(filter.Field, event)
	strValue := fmt.Sprintf("%v", value)

	switch filter.Operator {
	case "eq", "==":
		return strValue == filter.Value
	case "ne", "!=":
		return strValue != filter.Value
	case "contains":
		return contains(strValue, filter.Value)
	default:
		return true
	}
}

// getFieldValue extracts a field value from the event
func (e *Engine) getFieldValue(field string, event *Event) interface{} {
	parts := splitField(field)
	if len(parts) == 0 {
		return nil
	}

	// Handle top-level fields
	switch parts[0] {
	case "type":
		return event.Type
	case "source":
		return event.Source
	case "subject":
		return event.Subject
	case "data":
		if len(parts) > 1 {
			return getNestedValue(event.Data, parts[1:])
		}
		return event.Data
	}

	return nil
}

// deliver delivers an event to a subscriber
func (e *Engine) deliver(ctx context.Context, task deliveryTask) {
	start := time.Now()
	result := &DeliveryResult{
		SubscriptionID: task.subscription.ID,
		EventID:        task.event.ID,
		Attempts:       task.attempt,
		Timestamp:      time.Now(),
	}

	// Build payload
	payload, err := json.Marshal(task.event)
	if err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		return
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", task.subscription.URL, bytes.NewReader(payload))
	if err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		return
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-PlatformFoundry-Event", task.event.Type)
	req.Header.Set("X-PlatformFoundry-Event-ID", task.event.ID)
	req.Header.Set("X-PlatformFoundry-Delivery-Attempt", fmt.Sprintf("%d", task.attempt))

	// Add custom headers
	for k, v := range task.subscription.Headers {
		req.Header.Set(k, v)
	}

	// Sign payload
	if task.subscription.Secret != "" {
		signature := e.sign(payload, task.subscription.Secret)
		req.Header.Set("X-PlatformFoundry-Signature", signature)
	}

	// Send request
	resp, err := e.httpClient.Do(req)
	result.Duration = time.Since(start)

	if err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		e.handleFailure(task, result)
		return
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Status = "success"
	} else {
		result.Status = "failed"
		result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		e.handleFailure(task, result)
	}
}

// sign creates HMAC signature for payload
func (e *Engine) sign(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// handleFailure handles failed delivery with retry logic
func (e *Engine) handleFailure(task deliveryTask, result *DeliveryResult) {
	if task.attempt >= task.subscription.RetryPolicy.MaxRetries {
		return // Max retries reached
	}

	// Calculate delay with exponential backoff
	delay := task.subscription.RetryPolicy.InitialDelay
	for i := 1; i < task.attempt; i++ {
		delay = time.Duration(float64(delay) * task.subscription.RetryPolicy.BackoffFactor)
		if delay > task.subscription.RetryPolicy.MaxDelay {
			delay = task.subscription.RetryPolicy.MaxDelay
			break
		}
	}

	// Schedule retry
	go func() {
		time.Sleep(delay)
		select {
		case e.deliveryQueue <- deliveryTask{
			subscription: task.subscription,
			event:        task.event,
			attempt:      task.attempt + 1,
		}:
		default:
			// Queue full
		}
	}()
}

// VerifySignature verifies a webhook signature
func VerifySignature(payload []byte, signature, secret string) bool {
	expected := "sha256=" + hex.EncodeToString(hmacSHA256(payload, secret))
	return hmac.Equal([]byte(signature), []byte(expected))
}

func hmacSHA256(data []byte, secret string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(data)
	return mac.Sum(nil)
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr) >= 0))
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func splitField(field string) []string {
	var parts []string
	current := ""
	for _, c := range field {
		if c == '.' {
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
	return parts
}

func getNestedValue(data map[string]interface{}, path []string) interface{} {
	if len(path) == 0 {
		return data
	}

	value, ok := data[path[0]]
	if !ok {
		return nil
	}

	if len(path) == 1 {
		return value
	}

	if nested, ok := value.(map[string]interface{}); ok {
		return getNestedValue(nested, path[1:])
	}

	return nil
}

// Common event types
const (
	EventResourceCreated     = "resource.created"
	EventResourceUpdated     = "resource.updated"
	EventResourceDeleted     = "resource.deleted"
	EventDeploymentStarted   = "deployment.started"
	EventDeploymentCompleted = "deployment.completed"
	EventDeploymentFailed    = "deployment.failed"
	EventAlertFired          = "alert.fired"
	EventAlertResolved       = "alert.resolved"
	EventPolicyViolation     = "policy.violation"
	EventComplianceCheck     = "compliance.check"
)
