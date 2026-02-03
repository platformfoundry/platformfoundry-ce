package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/platformfoundry/pf-ce/internal/events"
	"github.com/platformfoundry/pf-ce/internal/graph"
	"github.com/platformfoundry/pf-ce/internal/telemetry"
	"github.com/platformfoundry/pf-ce/pkg/types"
)

// Server provides the HTTP API for Platform Foundry
type Server struct {
	mu          sync.RWMutex
	server      *http.Server
	mux         *http.ServeMux
	config      ServerConfig
	eventBus    *events.EventBus
	graphEngine *graph.Engine
	telemetry   *telemetry.Provider
	middleware  []Middleware
	routes      []Route
	startTime   time.Time
}

// ServerConfig configures the API server
type ServerConfig struct {
	// Address to listen on (e.g., ":8080")
	Address string `yaml:"address" json:"address"`

	// ReadTimeout for requests
	ReadTimeout time.Duration `yaml:"readTimeout" json:"readTimeout"`

	// WriteTimeout for responses
	WriteTimeout time.Duration `yaml:"writeTimeout" json:"writeTimeout"`

	// IdleTimeout for keep-alive connections
	IdleTimeout time.Duration `yaml:"idleTimeout" json:"idleTimeout"`

	// MaxRequestSize in bytes
	MaxRequestSize int64 `yaml:"maxRequestSize" json:"maxRequestSize"`

	// EnableCORS enables CORS support
	EnableCORS bool `yaml:"enableCors" json:"enableCors"`

	// CORSOrigins allowed origins for CORS
	CORSOrigins []string `yaml:"corsOrigins" json:"corsOrigins"`

	// RateLimitPerSecond limits requests per second
	RateLimitPerSecond int `yaml:"rateLimitPerSecond" json:"rateLimitPerSecond"`

	// EnableMetrics exposes /metrics endpoint
	EnableMetrics bool `yaml:"enableMetrics" json:"enableMetrics"`

	// EnableHealthCheck exposes /health endpoint
	EnableHealthCheck bool `yaml:"enableHealthCheck" json:"enableHealthCheck"`
}

// Route represents an API route
type Route struct {
	Method      string
	Path        string
	Handler     http.HandlerFunc
	Description string
	Auth        bool
}

// Middleware represents an HTTP middleware
type Middleware func(http.Handler) http.Handler

// NewServer creates a new API server
func NewServer(config ServerConfig) *Server {
	if config.Address == "" {
		config.Address = ":8080"
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 30 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 30 * time.Second
	}
	if config.IdleTimeout == 0 {
		config.IdleTimeout = 120 * time.Second
	}
	if config.MaxRequestSize == 0 {
		config.MaxRequestSize = 10 * 1024 * 1024 // 10MB
	}

	s := &Server{
		config:     config,
		mux:        http.NewServeMux(),
		middleware: make([]Middleware, 0),
		routes:     make([]Route, 0),
		startTime:  time.Now(),
	}

	// Register default routes
	s.registerDefaultRoutes()

	return s
}

// SetEventBus sets the event bus for the server
func (s *Server) SetEventBus(bus *events.EventBus) {
	s.eventBus = bus
}

// SetGraphEngine sets the graph engine for the server
func (s *Server) SetGraphEngine(engine *graph.Engine) {
	s.graphEngine = engine
}

// SetTelemetry sets the telemetry provider
func (s *Server) SetTelemetry(provider *telemetry.Provider) {
	s.telemetry = provider
}

// Use adds a middleware to the server
func (s *Server) Use(m Middleware) {
	s.middleware = append(s.middleware, m)
}

// RegisterRoute registers a new route
func (s *Server) RegisterRoute(route Route) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.routes = append(s.routes, route)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	// Build the handler with middleware
	var handler http.Handler = s.buildRouter()

	// Apply middleware in reverse order
	for i := len(s.middleware) - 1; i >= 0; i-- {
		handler = s.middleware[i](handler)
	}

	// Add default middleware
	handler = s.loggingMiddleware(handler)
	handler = s.recoveryMiddleware(handler)

	if s.config.EnableCORS {
		handler = s.corsMiddleware(handler)
	}

	if s.config.RateLimitPerSecond > 0 {
		handler = s.rateLimitMiddleware(handler)
	}

	s.server = &http.Server{
		Addr:         s.config.Address,
		Handler:      handler,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
		IdleTimeout:  s.config.IdleTimeout,
	}

	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

// buildRouter builds the HTTP router
func (s *Server) buildRouter() http.Handler {
	mux := http.NewServeMux()

	// Register all routes
	for _, route := range s.routes {
		pattern := fmt.Sprintf("%s %s", route.Method, route.Path)
		mux.HandleFunc(pattern, route.Handler)
	}

	return mux
}

// registerDefaultRoutes registers built-in routes
func (s *Server) registerDefaultRoutes() {
	// Health check
	if s.config.EnableHealthCheck {
		s.RegisterRoute(Route{
			Method:      "GET",
			Path:        "/health",
			Handler:     s.handleHealth,
			Description: "Health check endpoint",
			Auth:        false,
		})
		s.RegisterRoute(Route{
			Method:      "GET",
			Path:        "/ready",
			Handler:     s.handleReady,
			Description: "Readiness check endpoint",
			Auth:        false,
		})
	}

	// Metrics
	if s.config.EnableMetrics {
		s.RegisterRoute(Route{
			Method:      "GET",
			Path:        "/metrics",
			Handler:     s.handleMetrics,
			Description: "Prometheus metrics endpoint",
			Auth:        false,
		})
	}

	// API version
	s.RegisterRoute(Route{
		Method:      "GET",
		Path:        "/api/v1/version",
		Handler:     s.handleVersion,
		Description: "API version information",
		Auth:        false,
	})

	// Platform operations
	s.RegisterRoute(Route{
		Method:      "GET",
		Path:        "/api/v1/platforms",
		Handler:     s.handleListPlatforms,
		Description: "List all platforms",
		Auth:        true,
	})
	s.RegisterRoute(Route{
		Method:      "GET",
		Path:        "/api/v1/platforms/{name}",
		Handler:     s.handleGetPlatform,
		Description: "Get platform details",
		Auth:        true,
	})
	s.RegisterRoute(Route{
		Method:      "POST",
		Path:        "/api/v1/platforms/{name}/apply",
		Handler:     s.handleApplyPlatform,
		Description: "Apply platform configuration",
		Auth:        true,
	})
	s.RegisterRoute(Route{
		Method:      "POST",
		Path:        "/api/v1/platforms/{name}/plan",
		Handler:     s.handlePlanPlatform,
		Description: "Plan platform changes",
		Auth:        true,
	})

	// Graph operations
	s.RegisterRoute(Route{
		Method:      "GET",
		Path:        "/api/v1/graph",
		Handler:     s.handleGetGraph,
		Description: "Get resource graph",
		Auth:        true,
	})
	s.RegisterRoute(Route{
		Method:      "GET",
		Path:        "/api/v1/graph/{resource}/impact",
		Handler:     s.handleImpactAnalysis,
		Description: "Get impact analysis for a resource",
		Auth:        true,
	})

	// Events
	s.RegisterRoute(Route{
		Method:      "GET",
		Path:        "/api/v1/events",
		Handler:     s.handleListEvents,
		Description: "List events",
		Auth:        true,
	})
	s.RegisterRoute(Route{
		Method:      "GET",
		Path:        "/api/v1/events/stream",
		Handler:     s.handleEventStream,
		Description: "Server-sent events stream",
		Auth:        true,
	})

	// Promises
	s.RegisterRoute(Route{
		Method:      "GET",
		Path:        "/api/v1/promises",
		Handler:     s.handleListPromises,
		Description: "List available promises",
		Auth:        true,
	})
	s.RegisterRoute(Route{
		Method:      "POST",
		Path:        "/api/v1/promises/{name}/request",
		Handler:     s.handleRequestPromise,
		Description: "Request a promise",
		Auth:        true,
	})

	// Workloads
	s.RegisterRoute(Route{
		Method:      "GET",
		Path:        "/api/v1/workloads",
		Handler:     s.handleListWorkloads,
		Description: "List workloads",
		Auth:        true,
	})
	s.RegisterRoute(Route{
		Method:      "POST",
		Path:        "/api/v1/workloads",
		Handler:     s.handleCreateWorkload,
		Description: "Create a workload",
		Auth:        true,
	})
}

// Handler implementations

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	// Check if all dependencies are ready
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"ready":     true,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	// Return Prometheus-format metrics
	if s.telemetry != nil {
		stats := s.telemetry.Stats()
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "# HELP pf_traces_collected Total traces collected\n")
		fmt.Fprintf(w, "# TYPE pf_traces_collected counter\n")
		fmt.Fprintf(w, "pf_traces_collected %d\n", stats.TracesCollected)
		fmt.Fprintf(w, "# HELP pf_metrics_collected Total metrics collected\n")
		fmt.Fprintf(w, "# TYPE pf_metrics_collected counter\n")
		fmt.Fprintf(w, "pf_metrics_collected %d\n", stats.MetricsCollected)
	}
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"version":    "1.0.0",
		"apiVersion": "v1",
		"uptime":     time.Since(s.startTime).String(),
	})
}

func (s *Server) handleListPlatforms(w http.ResponseWriter, r *http.Request) {
	// Placeholder - would integrate with store
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"platforms": []interface{}{},
	})
}

func (s *Server) handleGetPlatform(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		s.writeError(w, http.StatusBadRequest, "platform name is required")
		return
	}
	// Placeholder
	s.writeJSON(w, http.StatusNotFound, map[string]interface{}{
		"error": fmt.Sprintf("platform %s not found", name),
	})
}

func (s *Server) handleApplyPlatform(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		s.writeError(w, http.StatusBadRequest, "platform name is required")
		return
	}
	// Placeholder
	s.writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"message": fmt.Sprintf("apply started for platform %s", name),
		"status":  "pending",
	})
}

func (s *Server) handlePlanPlatform(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		s.writeError(w, http.StatusBadRequest, "platform name is required")
		return
	}
	// Placeholder
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"platform": name,
		"plan":     "no changes",
		"toCreate": 0,
		"toUpdate": 0,
		"toDelete": 0,
	})
}

func (s *Server) handleGetGraph(w http.ResponseWriter, r *http.Request) {
	if s.graphEngine == nil {
		s.writeError(w, http.StatusServiceUnavailable, "graph engine not available")
		return
	}

	graph := s.graphEngine.GetGraph()
	s.writeJSON(w, http.StatusOK, graph)
}

func (s *Server) handleImpactAnalysis(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	if resource == "" {
		s.writeError(w, http.StatusBadRequest, "resource is required")
		return
	}

	if s.graphEngine == nil {
		s.writeError(w, http.StatusServiceUnavailable, "graph engine not available")
		return
	}

	impact, err := s.graphEngine.ImpactAnalysis(r.Context(), resource)
	if err != nil {
		s.writeError(w, http.StatusNotFound, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, impact)
}

func (s *Server) handleListEvents(w http.ResponseWriter, r *http.Request) {
	if s.eventBus == nil {
		s.writeError(w, http.StatusServiceUnavailable, "event bus not available")
		return
	}

	filter := types.EventFilter{}
	events, err := s.eventBus.Query(r.Context(), filter, 100, 0)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"events": events,
		"count":  len(events),
	})
}

func (s *Server) handleEventStream(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		s.writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	// Subscribe to events
	ctx := r.Context()
	eventCh := make(chan *types.Event, 100)

	if s.eventBus != nil {
		subscriptionID := fmt.Sprintf("sse-%d", time.Now().UnixNano())
		s.eventBus.Subscribe(subscriptionID, types.EventFilter{}, func(event *types.Event) error {
			select {
			case eventCh <- event:
			default:
				// Channel full, skip event
			}
			return nil
		})
		defer s.eventBus.Unsubscribe(subscriptionID)
	}

	// Send events
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-eventCh:
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "event: %s\n", event.Type)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-time.After(30 * time.Second):
			// Keep-alive
			fmt.Fprintf(w, ": keep-alive\n\n")
			flusher.Flush()
		}
	}
}

func (s *Server) handleListPromises(w http.ResponseWriter, r *http.Request) {
	// Placeholder
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"promises": []interface{}{},
	})
}

func (s *Server) handleRequestPromise(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		s.writeError(w, http.StatusBadRequest, "promise name is required")
		return
	}
	// Placeholder
	s.writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"message":    fmt.Sprintf("promise %s requested", name),
		"status":     "pending",
		"request_id": fmt.Sprintf("req-%d", time.Now().UnixNano()),
	})
}

func (s *Server) handleListWorkloads(w http.ResponseWriter, r *http.Request) {
	// Placeholder
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"workloads": []interface{}{},
	})
}

func (s *Server) handleCreateWorkload(w http.ResponseWriter, r *http.Request) {
	// Placeholder
	s.writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "workload created",
		"status":  "pending",
	})
}

// Middleware implementations

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		if s.telemetry != nil && s.telemetry.Logger() != nil {
			s.telemetry.Logger().Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapped.statusCode,
				"duration", time.Since(start).String(),
			)
		}
	})
}

func (s *Server) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("internal error: %v", err))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := false

		if len(s.config.CORSOrigins) == 0 {
			allowed = true
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else {
			for _, o := range s.config.CORSOrigins {
				if o == origin || o == "*" {
					allowed = true
					w.Header().Set("Access-Control-Allow-Origin", origin)
					break
				}
			}
		}

		if allowed {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "86400")
		}

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) rateLimitMiddleware(next http.Handler) http.Handler {
	// Simple token bucket rate limiter
	var mu sync.Mutex
	tokens := float64(s.config.RateLimitPerSecond)
	lastRefill := time.Now()
	maxTokens := float64(s.config.RateLimitPerSecond)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()

		// Refill tokens
		now := time.Now()
		elapsed := now.Sub(lastRefill).Seconds()
		tokens += elapsed * float64(s.config.RateLimitPerSecond)
		if tokens > maxTokens {
			tokens = maxTokens
		}
		lastRefill = now

		if tokens < 1 {
			mu.Unlock()
			w.Header().Set("Retry-After", "1")
			s.writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}

		tokens--
		mu.Unlock()

		next.ServeHTTP(w, r)
	})
}

// Helper functions

func (s *Server) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, map[string]interface{}{
		"error":     message,
		"status":    status,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
