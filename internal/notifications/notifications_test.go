package notifications

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("NewManager returned nil")
	}
	if len(m.channels) != 0 {
		t.Error("New manager should have no channels")
	}
}

func TestAddRemoveChannel(t *testing.T) {
	m := NewManager()

	webhook := NewWebhookChannel(WebhookConfig{
		Name: "test-webhook",
		URL:  "https://example.com/webhook",
	})

	m.AddChannel(webhook)

	if len(m.ListChannels()) != 1 {
		t.Error("Expected 1 channel after add")
	}

	m.RemoveChannel("test-webhook")

	if len(m.ListChannels()) != 0 {
		t.Error("Expected 0 channels after remove")
	}
}

func TestWebhookChannelSend(t *testing.T) {
	// Create test server
	var receivedEvent Event
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("Expected Content-Type: application/json")
		}
		if r.Header.Get("X-PF-Event") == "" {
			t.Error("Expected X-PF-Event header")
		}

		json.NewDecoder(r.Body).Decode(&receivedEvent)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	webhook := NewWebhookChannel(WebhookConfig{
		Name: "test",
		URL:  server.URL,
	})

	event := &Event{
		ID:      "test-123",
		Type:    EventDeploySucceeded,
		Source:  "test",
		Subject: "Test deployment succeeded",
		Time:    time.Now(),
		Data:    map[string]interface{}{"key": "value"},
	}

	err := webhook.Send(context.Background(), event)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	if receivedEvent.ID != event.ID {
		t.Errorf("Event ID mismatch: got %s, want %s", receivedEvent.ID, event.ID)
	}
}

func TestWebhookChannelEventFilter(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	webhook := NewWebhookChannel(WebhookConfig{
		Name:   "filtered",
		URL:    server.URL,
		Events: []EventType{EventDeploySucceeded}, // Only this event type
	})

	// Send matching event
	err := webhook.Send(context.Background(), &Event{
		ID:   "1",
		Type: EventDeploySucceeded,
		Time: time.Now(),
	})
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Send non-matching event
	err = webhook.Send(context.Background(), &Event{
		ID:   "2",
		Type: EventDeployFailed, // Different type
		Time: time.Now(),
	})
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Only one call should have been made
	if callCount != 1 {
		t.Errorf("Expected 1 webhook call, got %d", callCount)
	}
}

func TestWebhookChannelWithSecret(t *testing.T) {
	var signature string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		signature = r.Header.Get("X-PF-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	webhook := NewWebhookChannel(WebhookConfig{
		Name:   "secure",
		URL:    server.URL,
		Secret: "my-secret-key",
	})

	err := webhook.Send(context.Background(), &Event{
		ID:   "1",
		Type: EventDeploySucceeded,
		Time: time.Now(),
	})
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	if signature == "" {
		t.Error("Expected signature header to be set")
	}
}

func TestManagerNotify(t *testing.T) {
	m := NewManager()

	// Create two test servers
	call1 := false
	call2 := false

	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call1 = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call2 = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server2.Close()

	m.AddChannel(NewWebhookChannel(WebhookConfig{Name: "hook1", URL: server1.URL}))
	m.AddChannel(NewWebhookChannel(WebhookConfig{Name: "hook2", URL: server2.URL}))

	event := &Event{
		ID:   "notify-test",
		Type: EventResourceCreated,
		Time: time.Now(),
	}

	errors := m.Notify(context.Background(), event)

	if len(errors) > 0 {
		t.Errorf("Notify returned errors: %v", errors)
	}

	if !call1 || !call2 {
		t.Error("Not all channels were called")
	}
}

func TestManagerNotifyWithFailure(t *testing.T) {
	m := NewManager()

	// One working server, one that returns error
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server2.Close()

	m.AddChannel(NewWebhookChannel(WebhookConfig{Name: "ok", URL: server1.URL}))
	m.AddChannel(NewWebhookChannel(WebhookConfig{Name: "fail", URL: server2.URL}))

	errors := m.Notify(context.Background(), &Event{
		ID:   "test",
		Type: EventDeployFailed,
		Time: time.Now(),
	})

	if len(errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(errors))
	}
}

func TestEventColors(t *testing.T) {
	tests := []struct {
		eventType EventType
		wantColor string
	}{
		{EventDeploySucceeded, "#36a64f"},
		{EventDeployFailed, "#dc3545"},
		{EventApprovalRequired, "#ffc107"},
		{EventResourceCreated, "#6c757d"},
	}

	for _, tt := range tests {
		color := getEventColor(tt.eventType)
		if color != tt.wantColor {
			t.Errorf("getEventColor(%s) = %s, want %s", tt.eventType, color, tt.wantColor)
		}
	}
}

func TestSlackMessageFormat(t *testing.T) {
	event := &Event{
		ID:      "test-123",
		Type:    EventDeploySucceeded,
		Source:  "ci/cd",
		Subject: "Deployed app v1.2.3",
		Time:    time.Now(),
	}

	config := SlackConfig{
		Name:      "test-slack",
		Channel:   "#deployments",
		Username:  "PF Bot",
		IconEmoji: ":rocket:",
	}

	message := formatSlackMessage(event, config)

	if message["channel"] != "#deployments" {
		t.Error("Channel not set correctly")
	}
	if message["username"] != "PF Bot" {
		t.Error("Username not set correctly")
	}

	attachments, ok := message["attachments"].([]map[string]interface{})
	if !ok || len(attachments) == 0 {
		t.Fatal("Attachments not set correctly")
	}

	if attachments[0]["color"] != "#36a64f" {
		t.Error("Color not set correctly for success event")
	}
}
