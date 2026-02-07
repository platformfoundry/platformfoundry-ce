package triggers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"sync"
)

// WebhookTrigger implements webhook-based workflow triggering
type WebhookTrigger struct {
	workflowName string
	path         string
	secret       string
	callback     TriggerCallback
	server       *http.Server
	running      bool
	mu           sync.Mutex
}

// NewWebhookTrigger creates a new webhook trigger
func NewWebhookTrigger(workflowName, path, secret string) *WebhookTrigger {
	return &WebhookTrigger{
		workflowName: workflowName,
		path:         path,
		secret:       secret,
	}
}

// Type returns the trigger type
func (w *WebhookTrigger) Type() string {
	return "webhook"
}

// OnTrigger sets the callback
func (w *WebhookTrigger) OnTrigger(callback TriggerCallback) {
	w.callback = callback
}

// Start starts the webhook listener
// Note: In production, webhooks would be registered with a central HTTP server
func (w *WebhookTrigger) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.running {
		return nil
	}

	w.running = true
	return nil
}

// Stop stops the webhook listener
func (w *WebhookTrigger) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.running = false
	return nil
}

// HandleRequest handles an incoming webhook request
func (w *WebhookTrigger) HandleRequest(rw http.ResponseWriter, req *http.Request) {
	if !w.running {
		http.Error(rw, "Webhook not running", http.StatusServiceUnavailable)
		return
	}

	// Read body
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(rw, "Failed to read body", http.StatusBadRequest)
		return
	}

	// Verify signature if secret is set
	if w.secret != "" {
		signature := req.Header.Get("X-Webhook-Signature")
		if signature == "" {
			signature = req.Header.Get("X-Hub-Signature-256")
		}

		if !w.verifySignature(body, signature) {
			http.Error(rw, "Invalid signature", http.StatusUnauthorized)
			return
		}
	}

	// Parse payload
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		// If not JSON, use raw body
		payload = map[string]interface{}{
			"body": string(body),
		}
	}

	// Add webhook metadata
	payload["_webhook"] = map[string]interface{}{
		"path":   w.path,
		"method": req.Method,
		"headers": func() map[string]string {
			h := make(map[string]string)
			for k := range req.Header {
				h[k] = req.Header.Get(k)
			}
			return h
		}(),
	}

	// Trigger callback
	if w.callback != nil {
		go w.callback(w.workflowName, payload)
	}

	rw.WriteHeader(http.StatusAccepted)
	json.NewEncoder(rw).Encode(map[string]interface{}{
		"status":   "accepted",
		"workflow": w.workflowName,
	})
}

// verifySignature verifies the webhook signature
func (w *WebhookTrigger) verifySignature(payload []byte, signature string) bool {
	if signature == "" {
		return false
	}

	// Handle sha256=xxx format
	expectedPrefix := "sha256="
	if len(signature) > len(expectedPrefix) && signature[:len(expectedPrefix)] == expectedPrefix {
		signature = signature[len(expectedPrefix):]
	}

	// Compute expected signature
	mac := hmac.New(sha256.New, []byte(w.secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expected))
}

// Path returns the webhook path
func (w *WebhookTrigger) Path() string {
	return w.path
}

// WebhookRouter routes incoming webhooks to the appropriate trigger
type WebhookRouter struct {
	triggers map[string]*WebhookTrigger
	mu       sync.RWMutex
}

// NewWebhookRouter creates a new webhook router
func NewWebhookRouter() *WebhookRouter {
	return &WebhookRouter{
		triggers: make(map[string]*WebhookTrigger),
	}
}

// Register registers a webhook trigger
func (r *WebhookRouter) Register(trigger *WebhookTrigger) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.triggers[trigger.path] = trigger
}

// Unregister removes a webhook trigger
func (r *WebhookRouter) Unregister(path string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.triggers, path)
}

// ServeHTTP implements http.Handler
func (r *WebhookRouter) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	r.mu.RLock()
	trigger, ok := r.triggers[req.URL.Path]
	r.mu.RUnlock()

	if !ok {
		http.NotFound(rw, req)
		return
	}

	trigger.HandleRequest(rw, req)
}
