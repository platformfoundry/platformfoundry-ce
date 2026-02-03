package server

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"sync"
	"time"
)

//go:embed static
var staticFiles embed.FS

type Config struct {
	Port       int
	EnableCORS bool
}

type Server struct {
	config     Config
	mux        *http.ServeMux
	httpServer *http.Server
	clients    map[string]chan Event
	clientsMu  sync.RWMutex
}

type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

func New(config Config) *Server {
	s := &Server{
		config:  config,
		mux:     http.NewServeMux(),
		clients: make(map[string]chan Event),
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// API routes
	s.mux.HandleFunc("/api/v1/health", s.wrapHandler(s.handleHealth))

	// Environments
	s.mux.HandleFunc("/api/v1/environments", s.wrapHandler(s.handleEnvironments))

	// Workloads
	s.mux.HandleFunc("/api/v1/workloads", s.wrapHandler(s.handleWorkloads))

	// Deployments
	s.mux.HandleFunc("/api/v1/deployments", s.wrapHandler(s.handleDeployments))

	// Resources
	s.mux.HandleFunc("/api/v1/resources", s.wrapHandler(s.handleResources))

	// Catalog
	s.mux.HandleFunc("/api/v1/catalog/templates", s.wrapHandler(s.handleGetTemplates))
	s.mux.HandleFunc("/api/v1/catalog/resource-types", s.wrapHandler(s.handleGetResourceTypes))

	// Apply/Plan/Validate
	s.mux.HandleFunc("/api/v1/apply", s.wrapHandler(s.handleApply))
	s.mux.HandleFunc("/api/v1/plan", s.wrapHandler(s.handlePlan))
	s.mux.HandleFunc("/api/v1/validate", s.wrapHandler(s.handleValidate))

	// SSE for real-time updates
	s.mux.HandleFunc("/api/v1/events", s.handleSSE)

	// Metrics
	s.mux.HandleFunc("/api/v1/metrics/dora", s.wrapHandler(s.handleDORAMetrics))
	s.mux.HandleFunc("/api/v1/metrics/costs", s.wrapHandler(s.handleCostMetrics))

	// Serve static files (embedded React app)
	staticFS, _ := fs.Sub(staticFiles, "static")
	s.mux.Handle("/", http.FileServer(http.FS(staticFS)))
}

func (s *Server) wrapHandler(handler func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.config.EnableCORS {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
		}
		handler(w, r)
	}
}

func (s *Server) Start() error {
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.Port),
		Handler:      s.mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) Broadcast(event Event) {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	for _, ch := range s.clients {
		select {
		case ch <- event:
		default:
		}
	}
}

// Handlers

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
}

func (s *Server) handleEnvironments(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var env map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		writeJSON(w, http.StatusCreated, env)
		return
	}

	// Handle path parameters for single environment
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/environments")
	if path != "" && path != "/" {
		name := strings.TrimPrefix(path, "/")
		env := map[string]interface{}{
			"name":    name,
			"type":    "development",
			"cluster": "dev-cluster",
			"status":  "active",
		}
		writeJSON(w, http.StatusOK, env)
		return
	}

	envs := []map[string]interface{}{
		{"name": "development", "type": "development", "cluster": "dev-cluster", "status": "active"},
		{"name": "staging", "type": "staging", "cluster": "staging-cluster", "status": "active"},
		{"name": "production", "type": "production", "cluster": "prod-cluster", "status": "active"},
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"items": envs})
}

func (s *Server) handleWorkloads(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/workloads")

	// Handle logs endpoint
	if strings.Contains(path, "/logs") {
		parts := strings.Split(path, "/")
		if len(parts) >= 2 {
			name := parts[1]
			logs := fmt.Sprintf("[%s] Starting %s...\n[%s] Service initialized\n[%s] Listening on port 8080\n",
				time.Now().Format(time.RFC3339), name, time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339))
			writeJSON(w, http.StatusOK, map[string]string{"logs": logs})
			return
		}
	}

	// Handle single workload
	if path != "" && path != "/" {
		name := strings.TrimPrefix(path, "/")
		workload := map[string]interface{}{
			"name":          name,
			"environment":   "production",
			"status":        "Running",
			"replicas":      3,
			"readyReplicas": 3,
		}
		writeJSON(w, http.StatusOK, workload)
		return
	}

	workloads := []map[string]interface{}{
		{"name": "api-gateway", "environment": "production", "status": "Running", "replicas": 3, "readyReplicas": 3},
		{"name": "user-service", "environment": "production", "status": "Running", "replicas": 2, "readyReplicas": 2},
		{"name": "order-service", "environment": "staging", "status": "Running", "replicas": 1, "readyReplicas": 1},
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"items": workloads})
}

func (s *Server) handleDeployments(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		deployment := map[string]interface{}{
			"id":          fmt.Sprintf("dep-%d", time.Now().Unix()),
			"workload":    req["workload"],
			"environment": req["environment"],
			"version":     "latest",
			"status":      "running",
			"startedAt":   time.Now().Format(time.RFC3339),
		}

		s.Broadcast(Event{Type: "deployment.created", Data: deployment})
		writeJSON(w, http.StatusCreated, deployment)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/deployments")
	if path != "" && path != "/" {
		id := strings.TrimPrefix(path, "/")
		deployment := map[string]interface{}{
			"id":          id,
			"workload":    "api-gateway",
			"environment": "production",
			"version":     "v1.2.0",
			"status":      "succeeded",
			"startedAt":   time.Now().Format(time.RFC3339),
		}
		writeJSON(w, http.StatusOK, deployment)
		return
	}

	deployments := []map[string]interface{}{
		{"id": "dep-001", "workload": "api-gateway", "environment": "production", "version": "v1.2.0", "status": "succeeded", "startedAt": time.Now().Add(-1 * time.Hour).Format(time.RFC3339)},
		{"id": "dep-002", "workload": "user-service", "environment": "staging", "version": "v2.0.1", "status": "running", "startedAt": time.Now().Format(time.RFC3339)},
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"items": deployments})
}

func (s *Server) handleResources(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/resources")
	if path != "" && path != "/" {
		name := strings.TrimPrefix(path, "/")
		resource := map[string]interface{}{
			"name":     name,
			"type":     "postgres",
			"status":   "available",
			"provider": "aws-rds",
		}
		writeJSON(w, http.StatusOK, resource)
		return
	}

	resources := []map[string]interface{}{
		{"name": "main-db", "type": "postgres", "status": "available", "provider": "aws-rds"},
		{"name": "cache", "type": "redis", "status": "available", "provider": "elasticache"},
		{"name": "queue", "type": "sqs", "status": "available", "provider": "aws-sqs"},
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"items": resources})
}

func (s *Server) handleGetTemplates(w http.ResponseWriter, r *http.Request) {
	templates := []map[string]interface{}{
		{"name": "microservice-go", "description": "Go microservice template", "category": "backend"},
		{"name": "microservice-node", "description": "Node.js microservice template", "category": "backend"},
		{"name": "frontend-react", "description": "React frontend template", "category": "frontend"},
		{"name": "data-pipeline", "description": "Data pipeline template", "category": "data"},
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"items": templates})
}

func (s *Server) handleGetResourceTypes(w http.ResponseWriter, r *http.Request) {
	types := []map[string]interface{}{
		{"type": "postgres", "description": "PostgreSQL database", "providers": []string{"aws-rds", "gcp-sql", "azure-db"}},
		{"type": "redis", "description": "Redis cache", "providers": []string{"elasticache", "memorystore"}},
		{"type": "s3", "description": "Object storage", "providers": []string{"aws-s3", "gcs", "azure-blob"}},
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"items": types})
}

func (s *Server) handleApply(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"message": "Configuration applied successfully"})
}

func (s *Server) handlePlan(w http.ResponseWriter, r *http.Request) {
	plan := map[string]interface{}{
		"toCreate":  []string{"deployment/api-gateway"},
		"toUpdate":  []string{"service/api-gateway"},
		"toDelete":  []string{},
		"noChanges": false,
	}
	writeJSON(w, http.StatusOK, plan)
}

func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{"valid": true, "errors": []string{}})
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	if s.config.EnableCORS {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	}

	clientID := fmt.Sprintf("%d", time.Now().UnixNano())
	events := make(chan Event, 10)

	s.clientsMu.Lock()
	s.clients[clientID] = events
	s.clientsMu.Unlock()

	defer func() {
		s.clientsMu.Lock()
		delete(s.clients, clientID)
		close(events)
		s.clientsMu.Unlock()
	}()

	for {
		select {
		case event := <-events:
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (s *Server) handleDORAMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := map[string]interface{}{
		"deploymentFrequency": map[string]interface{}{
			"value":  15.2,
			"unit":   "deploys/day",
			"rating": "elite",
		},
		"leadTime": map[string]interface{}{
			"value":  2.5,
			"unit":   "hours",
			"rating": "elite",
		},
		"changeFailureRate": map[string]interface{}{
			"value":  3.2,
			"unit":   "percent",
			"rating": "high",
		},
		"mttr": map[string]interface{}{
			"value":  45,
			"unit":   "minutes",
			"rating": "elite",
		},
	}
	writeJSON(w, http.StatusOK, metrics)
}

func (s *Server) handleCostMetrics(w http.ResponseWriter, r *http.Request) {
	costs := map[string]interface{}{
		"total":   12500.50,
		"compute": 8000.00,
		"storage": 2500.00,
		"network": 1500.50,
		"other":   500.00,
		"trend":   -5.2,
		"byTeam": map[string]float64{
			"platform": 3500.00,
			"backend":  5000.00,
			"frontend": 2500.00,
			"data":     1500.50,
		},
	}
	writeJSON(w, http.StatusOK, costs)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
