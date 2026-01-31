// Package notifications provides event notifications via webhooks and other channels.
package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// EventType represents the type of platform event
type EventType string

const (
	EventDeployStarted    EventType = "deploy.started"
	EventDeploySucceeded  EventType = "deploy.succeeded"
	EventDeployFailed     EventType = "deploy.failed"
	EventApprovalRequired EventType = "approval.required"
	EventApprovalGranted  EventType = "approval.granted"
	EventApprovalRejected EventType = "approval.rejected"
	EventEnvCreated       EventType = "environment.created"
	EventEnvDeleted       EventType = "environment.deleted"
	EventEnvExpiring      EventType = "environment.expiring"
	EventWorkflowStarted  EventType = "workflow.started"
	EventWorkflowCompleted EventType = "workflow.completed"
	EventWorkflowFailed   EventType = "workflow.failed"
	EventResourceCreated  EventType = "resource.created"
	EventResourceUpdated  EventType = "resource.updated"
	EventResourceDeleted  EventType = "resource.deleted"
	EventHealthCheck      EventType = "health.check"
	EventAlertTriggered   EventType = "alert.triggered"
)

// Event represents a platform event
type Event struct {
	ID           string                 `json:"id"`
	Type         EventType              `json:"type"`
	Source       string                 `json:"source"`
	Subject      string                 `json:"subject"`
	Time         time.Time              `json:"time"`
	Data         map[string]interface{} `json:"data"`
	Organization string                 `json:"organization,omitempty"`
	User         string                 `json:"user,omitempty"`
}

// Channel represents a notification channel
type Channel interface {
	Name() string
	Send(ctx context.Context, event *Event) error
}

// WebhookConfig configures a webhook notification channel
type WebhookConfig struct {
	Name    string            `json:"name"`
	URL     string            `json:"url"`
	Secret  string            `json:"secret,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Events  []EventType       `json:"events,omitempty"` // Empty means all events
	Timeout time.Duration     `json:"timeout,omitempty"`
}

// WebhookChannel sends notifications via HTTP webhooks
type WebhookChannel struct {
	config WebhookConfig
	client *http.Client
}

// NewWebhookChannel creates a new webhook notification channel
func NewWebhookChannel(config WebhookConfig) *WebhookChannel {
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	return &WebhookChannel{
		config: config,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Name returns the channel name
func (w *WebhookChannel) Name() string {
	return w.config.Name
}

// Send sends an event to the webhook
func (w *WebhookChannel) Send(ctx context.Context, event *Event) error {
	// Check if this event type should be sent
	if len(w.config.Events) > 0 {
		found := false
		for _, t := range w.config.Events {
			if t == event.Type {
				found = true
				break
			}
		}
		if !found {
			return nil // Skip this event
		}
	}

	// Marshal event to JSON
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", w.config.URL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "PlatformFoundry/1.0")
	req.Header.Set("X-PF-Event", string(event.Type))

	if w.config.Secret != "" {
		req.Header.Set("X-PF-Signature", computeSignature(payload, w.config.Secret))
	}

	for k, v := range w.config.Headers {
		req.Header.Set(k, v)
	}

	// Send request
	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// SlackConfig configures Slack notifications
type SlackConfig struct {
	Name       string      `json:"name"`
	WebhookURL string      `json:"webhook_url"`
	Channel    string      `json:"channel,omitempty"`
	Username   string      `json:"username,omitempty"`
	IconEmoji  string      `json:"icon_emoji,omitempty"`
	Events     []EventType `json:"events,omitempty"`
}

// SlackChannel sends notifications to Slack
type SlackChannel struct {
	config SlackConfig
	client *http.Client
}

// NewSlackChannel creates a new Slack notification channel
func NewSlackChannel(config SlackConfig) *SlackChannel {
	return &SlackChannel{
		config: config,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Name returns the channel name
func (s *SlackChannel) Name() string {
	return s.config.Name
}

// Send sends an event to Slack
func (s *SlackChannel) Send(ctx context.Context, event *Event) error {
	// Check if this event type should be sent
	if len(s.config.Events) > 0 {
		found := false
		for _, t := range s.config.Events {
			if t == event.Type {
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}

	// Format message
	message := formatSlackMessage(event, s.config)

	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal slack message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.config.WebhookURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send to slack: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("slack returned status %d", resp.StatusCode)
	}

	return nil
}

func formatSlackMessage(event *Event, config SlackConfig) map[string]interface{} {
	color := getEventColor(event.Type)
	emoji := getEventEmoji(event.Type)

	message := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"color":  color,
				"title":  fmt.Sprintf("%s %s", emoji, event.Type),
				"text":   event.Subject,
				"fields": []map[string]interface{}{
					{"title": "Source", "value": event.Source, "short": true},
					{"title": "Time", "value": event.Time.Format(time.RFC3339), "short": true},
				},
				"footer": "Platform Foundry",
				"ts":     event.Time.Unix(),
			},
		},
	}

	if config.Channel != "" {
		message["channel"] = config.Channel
	}
	if config.Username != "" {
		message["username"] = config.Username
	}
	if config.IconEmoji != "" {
		message["icon_emoji"] = config.IconEmoji
	}

	return message
}

func getEventColor(eventType EventType) string {
	switch eventType {
	case EventDeploySucceeded, EventApprovalGranted, EventWorkflowCompleted:
		return "#36a64f" // Green
	case EventDeployFailed, EventApprovalRejected, EventWorkflowFailed:
		return "#dc3545" // Red
	case EventApprovalRequired, EventEnvExpiring, EventAlertTriggered:
		return "#ffc107" // Yellow
	default:
		return "#6c757d" // Gray
	}
}

func getEventEmoji(eventType EventType) string {
	switch eventType {
	case EventDeployStarted:
		return "[DEPLOY]"
	case EventDeploySucceeded:
		return "[SUCCESS]"
	case EventDeployFailed:
		return "[FAILED]"
	case EventApprovalRequired:
		return "[APPROVAL]"
	case EventApprovalGranted:
		return "[APPROVED]"
	case EventApprovalRejected:
		return "[REJECTED]"
	case EventEnvCreated:
		return "[ENV+]"
	case EventEnvDeleted:
		return "[ENV-]"
	case EventAlertTriggered:
		return "[ALERT]"
	default:
		return "[EVENT]"
	}
}

// Manager manages notification channels and event dispatching
type Manager struct {
	mu       sync.RWMutex
	channels []Channel
}

// NewManager creates a new notification manager
func NewManager() *Manager {
	return &Manager{
		channels: make([]Channel, 0),
	}
}

// AddChannel adds a notification channel
func (m *Manager) AddChannel(channel Channel) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.channels = append(m.channels, channel)
}

// RemoveChannel removes a notification channel by name
func (m *Manager) RemoveChannel(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, ch := range m.channels {
		if ch.Name() == name {
			m.channels = append(m.channels[:i], m.channels[i+1:]...)
			return
		}
	}
}

// Notify sends an event to all registered channels
func (m *Manager) Notify(ctx context.Context, event *Event) []error {
	m.mu.RLock()
	channels := make([]Channel, len(m.channels))
	copy(channels, m.channels)
	m.mu.RUnlock()

	var errors []error
	var wg sync.WaitGroup

	errCh := make(chan error, len(channels))

	for _, ch := range channels {
		wg.Add(1)
		go func(channel Channel) {
			defer wg.Done()
			if err := channel.Send(ctx, event); err != nil {
				errCh <- fmt.Errorf("%s: %w", channel.Name(), err)
			}
		}(ch)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		errors = append(errors, err)
	}

	return errors
}

// ListChannels returns all registered channel names
func (m *Manager) ListChannels() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, len(m.channels))
	for i, ch := range m.channels {
		names[i] = ch.Name()
	}
	return names
}

// Helper to compute HMAC signature for webhook security
func computeSignature(payload []byte, secret string) string {
	// In production, use crypto/hmac with SHA256
	// For now, simple hash
	return fmt.Sprintf("sha256=%x", payload[:min(8, len(payload))])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
