package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/platformfoundry/pf-ce/internal/engine"
)

// StreamEvents provides Server-Sent Events for real-time updates
func (h *Handler) StreamEvents(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		h.Error(w, http.StatusInternalServerError, "SSE_ERROR", "streaming not supported")
		return
	}

	// Create event channel
	events := make(chan engine.EngineEvent, 100)

	// Create channel listener
	listener := &channelListener{ch: events}

	// Subscribe to event bus
	h.Orchestrator.Subscribe(listener)
	defer h.Orchestrator.Unsubscribe(listener)

	// Keep-alive ticker
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Send initial connection message
	fmt.Fprintf(w, "event: connected\n")
	fmt.Fprintf(w, "data: {\"message\": \"SSE connection established\"}\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return

		case event := <-events:
			data, _ := json.Marshal(map[string]interface{}{
				"engineId":  event.EngineID,
				"type":      event.Type,
				"component": event.Component,
				"progress":  event.Progress,
				"message":   event.Message,
				"timestamp": event.Timestamp.Format(time.RFC3339),
			})
			fmt.Fprintf(w, "event: %s\n", event.Type)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()

		case <-ticker.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

// ListEvents returns historical events
func (h *Handler) ListEvents(w http.ResponseWriter, r *http.Request) {
	// Get event bus from orchestrator
	eventBus := h.Orchestrator.GetEventBus()
	if eventBus == nil {
		h.JSON(w, http.StatusOK, map[string]interface{}{
			"events": []interface{}{},
			"count":  0,
		})
		return
	}

	// Get query parameters
	since := r.URL.Query().Get("since")
	engineID := r.URL.Query().Get("engineId")
	eventType := r.URL.Query().Get("type")

	var events []engine.EngineEvent

	if engineID != "" {
		events = eventBus.GetEventsByEngine(engineID)
	} else if eventType != "" {
		events = eventBus.GetEventsByType(engine.EventType(eventType))
	} else if since != "" {
		t, err := time.Parse(time.RFC3339, since)
		if err != nil {
			h.Error(w, http.StatusBadRequest, "INVALID_TIME", "invalid since parameter, expected RFC3339 format")
			return
		}
		events = eventBus.GetEventLogSince(t)
	} else {
		events = eventBus.GetEventLog()
	}

	// Convert to response format
	response := make([]map[string]interface{}, len(events))
	for i, e := range events {
		response[i] = map[string]interface{}{
			"engineId":  e.EngineID,
			"type":      e.Type,
			"component": e.Component,
			"progress":  e.Progress,
			"message":   e.Message,
			"timestamp": e.Timestamp.Format(time.RFC3339),
		}
		if e.Error != nil {
			response[i]["error"] = e.Error.Error()
		}
	}

	h.JSON(w, http.StatusOK, map[string]interface{}{
		"events": response,
		"count":  len(events),
	})
}

// channelListener implements engine.EventListener by sending events to a channel
type channelListener struct {
	ch chan engine.EngineEvent
}

func (l *channelListener) OnEvent(event engine.EngineEvent) {
	select {
	case l.ch <- event:
	default:
		// Channel full, drop event
	}
}
