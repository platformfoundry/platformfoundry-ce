package web

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"sync"

	"github.com/platformfoundry/pf-ce/internal/audit"
	"github.com/platformfoundry/pf-ce/internal/auth"
	configPkg "github.com/platformfoundry/pf-ce/internal/config"
	"github.com/platformfoundry/pf-ce/internal/rbac"
	"github.com/platformfoundry/pf-ce/internal/telemetry"
	"github.com/platformfoundry/pf-ce/pkg/log"
)

// Server represents the web server
type Server struct {
	port             int
	httpsPort        int
	staticDir        string
	rbac             *rbac.RBAC
	auditLogger      *audit.Logger
	metricsCollector *telemetry.MetricsCollector
	authMiddleware   *auth.AuthMiddleware
	securityConfig   *configPkg.SecurityConfig
	platforms        map[string]interface{}
	jobs             map[string]interface{}
	mu               sync.RWMutex
}

// Config represents server configuration
type Config struct {
	Port             int
	HTTPSPort        int
	StaticDir        string
	RBAC             *rbac.RBAC
	AuditLogger      *audit.Logger
	MetricsCollector *telemetry.MetricsCollector
	AuthMiddleware   *auth.AuthMiddleware
	SecurityConfig   *configPkg.SecurityConfig
}

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// PlatformInfo represents platform information for the dashboard
type PlatformInfo struct {
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Status    string                 `json:"status"`
	Resources int                    `json:"resources"`
	CreatedAt string                 `json:"created_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// JobInfo represents job information for the dashboard
type JobInfo struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Status   string `json:"status"`
	Progress int    `json:"progress"`
	Message  string `json:"message,omitempty"`
}

// NewServer creates a new web server
func NewServer(config Config) *Server {
	if config.Port == 0 {
		config.Port = 8080
	}

	if config.HTTPSPort == 0 {
		config.HTTPSPort = 8443
	}

	if config.StaticDir == "" {
		config.StaticDir = "web/static"
	}

	return &Server{
		port:             config.Port,
		httpsPort:        config.HTTPSPort,
		staticDir:        config.StaticDir,
		rbac:             config.RBAC,
		auditLogger:      config.AuditLogger,
		metricsCollector: config.MetricsCollector,
		authMiddleware:   config.AuthMiddleware,
		securityConfig:   config.SecurityConfig,
		platforms:        make(map[string]interface{}),
		jobs:             make(map[string]interface{}),
	}
}

// Start starts the web server with TLS support
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Static files (no auth required)
	fs := http.FileServer(http.Dir(s.staticDir))
	mux.Handle("/", fs)

	// Public endpoints (no auth required)
	mux.HandleFunc("/api/health", s.handleHealth)

	// Protected API routes (require authentication)
	protectedRoutes := http.NewServeMux()
	protectedRoutes.HandleFunc("/api/platforms", s.handlePlatforms)
	protectedRoutes.HandleFunc("/api/platforms/", s.handlePlatform)
	protectedRoutes.HandleFunc("/api/jobs", s.handleJobs)
	protectedRoutes.HandleFunc("/api/jobs/", s.handleJob)
	protectedRoutes.HandleFunc("/api/validate", s.handleValidate)
	protectedRoutes.HandleFunc("/api/apply", s.handleApply)
	protectedRoutes.HandleFunc("/api/plan", s.handlePlan)
	protectedRoutes.HandleFunc("/api/rollback", s.handleRollback)
	protectedRoutes.HandleFunc("/api/stats", s.handleStats)

	// Service endpoints
	protectedRoutes.HandleFunc("/api/services", s.handleServices)
	protectedRoutes.HandleFunc("/api/services/", s.handleService)

	// Template endpoints
	protectedRoutes.HandleFunc("/api/templates", s.handleTemplates)
	protectedRoutes.HandleFunc("/api/templates/", s.handleTemplate)

	// Scorecard endpoints
	protectedRoutes.HandleFunc("/api/scorecards", s.handleScorecards)
	protectedRoutes.HandleFunc("/api/scorecards/stats", s.handleScorecards)

	// Apply authentication middleware to protected routes
	var apiHandler http.Handler = protectedRoutes
	if s.authMiddleware != nil && s.securityConfig != nil && s.securityConfig.Server.RequireAuth {
		apiHandler = s.authMiddleware.Authenticate(protectedRoutes)
	}

	// Mount protected routes
	for _, route := range []string{
		"/api/platforms", "/api/platforms/", "/api/jobs", "/api/jobs/",
		"/api/validate", "/api/apply", "/api/plan", "/api/rollback", "/api/stats",
		"/api/services", "/api/services/",
		"/api/templates", "/api/templates/",
	} {
		mux.Handle(route, apiHandler)
	}

	// Metrics endpoint (optionally protected)
	if s.metricsCollector != nil {
		mux.HandleFunc("/metrics", s.metricsCollector.Handler())
	}

	// Apply middleware chain
	handler := s.securityHeadersMiddleware(
		s.corsMiddleware(
			s.loggingMiddleware(mux),
		),
	)

	// Check if TLS is enabled
	useTLS := s.securityConfig != nil && s.securityConfig.TLS.Enabled

	if useTLS {
		// Start HTTPS server
		return s.startHTTPS(handler)
	}

	// Start HTTP server
	addr := fmt.Sprintf("%s:%d", getServerAddress(s.securityConfig), s.port)
	log.Info("Starting web server",
		log.String("protocol", "http"),
		log.String("address", addr),
	)
	log.Warn("TLS is disabled - not recommended for production")

	return http.ListenAndServe(addr, handler)
}

// startHTTPS starts the HTTPS server with TLS configuration
func (s *Server) startHTTPS(handler http.Handler) error {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
		PreferServerCipherSuites: true,
	}

	addr := fmt.Sprintf("%s:%d", getServerAddress(s.securityConfig), s.httpsPort)
	server := &http.Server{
		Addr:      addr,
		Handler:   handler,
		TLSConfig: tlsConfig,
	}

	// Get certificate paths from security config
	certFile := s.securityConfig.TLS.Manual.CertFile
	keyFile := s.securityConfig.TLS.Manual.KeyFile

	log.Info("Starting secure web server",
		log.String("protocol", "https"),
		log.String("address", addr),
		log.String("certificate", certFile),
	)

	// Start HTTP->HTTPS redirect server if enabled
	if s.securityConfig.Server.RedirectHTTPS {
		go s.startHTTPRedirect()
	}

	return server.ListenAndServeTLS(certFile, keyFile)
}

// startHTTPRedirect starts HTTP server that redirects to HTTPS
func (s *Server) startHTTPRedirect() {
	redirectHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpsURL := fmt.Sprintf("https://%s:%d%s", r.Host, s.httpsPort, r.RequestURI)
		http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
	})

	httpAddr := fmt.Sprintf("%s:%d", getServerAddress(s.securityConfig), s.port)
	log.Info("Starting HTTP redirect server",
		log.String("from", httpAddr),
		log.String("to", "https"),
	)

	if err := http.ListenAndServe(httpAddr, redirectHandler); err != nil {
		log.Error("HTTP redirect server error", log.Err(err))
	}
}

// getServerAddress returns the server address from config or default
func getServerAddress(cfg *configPkg.SecurityConfig) string {
	if cfg != nil && cfg.Server.Address != "" {
		return cfg.Server.Address
	}
	return "0.0.0.0"
}

// Middleware

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log API requests to metrics
		if s.metricsCollector != nil {
			s.metricsCollector.IncrementCounter("pf_api_requests_total", map[string]string{
				"method": r.Method,
				"path":   r.URL.Path,
			})
		}

		// Log to audit logger if available
		if s.auditLogger != nil {
			user := "anonymous"
			if authUser, ok := auth.GetUserFromContext(r.Context()); ok {
				user = authUser.Username
			}
			s.auditLogger.LogAccess(user, r.Method, r.URL.Path, "")
		}

		next.ServeHTTP(w, r)
	})
}

// securityHeadersMiddleware adds security headers to responses
func (s *Server) securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.securityConfig == nil {
			next.ServeHTTP(w, r)
			return
		}

		headers := s.securityConfig.Security.Headers

		// HSTS (HTTP Strict Transport Security)
		if headers.HSTS.Enabled && s.securityConfig.TLS.Enabled {
			hstsValue := fmt.Sprintf("max-age=%s", headers.HSTS.MaxAge)
			if headers.HSTS.IncludeSubDomains {
				hstsValue += "; includeSubDomains"
			}
			if headers.HSTS.Preload {
				hstsValue += "; preload"
			}
			w.Header().Set("Strict-Transport-Security", hstsValue)
		}

		// Content Security Policy
		if headers.CSP.Enabled {
			w.Header().Set("Content-Security-Policy", headers.CSP.Policy)
		}

		// X-Frame-Options
		if headers.FrameOptions != "" {
			w.Header().Set("X-Frame-Options", headers.FrameOptions)
		}

		// X-Content-Type-Options
		if headers.ContentTypeNoSniff {
			w.Header().Set("X-Content-Type-Options", "nosniff")
		}

		// X-XSS-Protection
		if headers.XSSProtection {
			w.Header().Set("X-XSS-Protection", "1; mode=block")
		}

		// Referrer-Policy
		if headers.ReferrerPolicy != "" {
			w.Header().Set("Referrer-Policy", headers.ReferrerPolicy)
		}

		next.ServeHTTP(w, r)
	})
}

// API Handlers

func (s *Server) handlePlatforms(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.listPlatforms(w, r)
	case "POST":
		s.createPlatform(w, r)
	default:
		s.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) handlePlatform(w http.ResponseWriter, r *http.Request) {
	// Extract platform name from path
	name := filepath.Base(r.URL.Path)

	switch r.Method {
	case "GET":
		s.getPlatform(w, r, name)
	case "PUT":
		s.updatePlatform(w, r, name)
	case "DELETE":
		s.deletePlatform(w, r, name)
	default:
		s.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) listPlatforms(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	platforms := make([]PlatformInfo, 0, len(s.platforms))
	for name := range s.platforms {
		platforms = append(platforms, PlatformInfo{
			Name:      name,
			Type:      "Platform",
			Status:    "Ready",
			Resources: 5,
			CreatedAt: "2024-01-01T00:00:00Z",
		})
	}

	s.sendSuccess(w, platforms)
}

func (s *Server) getPlatform(w http.ResponseWriter, r *http.Request, name string) {
	s.mu.RLock()
	platform, exists := s.platforms[name]
	s.mu.RUnlock()

	if !exists {
		s.sendError(w, http.StatusNotFound, "Platform not found")
		return
	}

	s.sendSuccess(w, platform)
}

func (s *Server) createPlatform(w http.ResponseWriter, r *http.Request) {
	var platform map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&platform); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	name, ok := platform["name"].(string)
	if !ok || name == "" {
		s.sendError(w, http.StatusBadRequest, "Platform name is required")
		return
	}

	s.mu.Lock()
	s.platforms[name] = platform
	s.mu.Unlock()

	if s.auditLogger != nil {
		s.auditLogger.LogCreate("web-user", audit.ResourcePlatform, name, "success", "Platform created via web UI")
	}

	s.sendSuccess(w, platform)
}

func (s *Server) updatePlatform(w http.ResponseWriter, r *http.Request, name string) {
	var platform map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&platform); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	s.mu.Lock()
	s.platforms[name] = platform
	s.mu.Unlock()

	if s.auditLogger != nil {
		s.auditLogger.LogUpdate("web-user", audit.ResourcePlatform, name, "success", "Platform updated via web UI")
	}

	s.sendSuccess(w, platform)
}

func (s *Server) deletePlatform(w http.ResponseWriter, r *http.Request, name string) {
	s.mu.Lock()
	delete(s.platforms, name)
	s.mu.Unlock()

	if s.auditLogger != nil {
		s.auditLogger.LogDelete("web-user", audit.ResourcePlatform, name, "success", "Platform deleted via web UI")
	}

	s.sendSuccess(w, map[string]string{"message": "Platform deleted successfully"})
}

func (s *Server) handleJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]JobInfo, 0, len(s.jobs))
	for id := range s.jobs {
		jobs = append(jobs, JobInfo{
			ID:       id,
			Type:     "apply",
			Status:   "running",
			Progress: 50,
		})
	}

	s.sendSuccess(w, jobs)
}

func (s *Server) handleJob(w http.ResponseWriter, r *http.Request) {
	id := filepath.Base(r.URL.Path)

	s.mu.RLock()
	job, exists := s.jobs[id]
	s.mu.RUnlock()

	if !exists {
		s.sendError(w, http.StatusNotFound, "Job not found")
		return
	}

	s.sendSuccess(w, job)
}

func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		s.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Validation logic would go here
	s.sendSuccess(w, map[string]interface{}{
		"valid":  true,
		"errors": []string{},
	})
}

func (s *Server) handleApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		s.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Create a job
	jobID := fmt.Sprintf("job-%d", len(s.jobs)+1)

	s.mu.Lock()
	s.jobs[jobID] = map[string]interface{}{
		"id":       jobID,
		"type":     "apply",
		"status":   "pending",
		"progress": 0,
	}
	s.mu.Unlock()

	s.sendSuccess(w, map[string]interface{}{
		"job_id": jobID,
		"status": "pending",
	})
}

func (s *Server) handlePlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		s.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Plan logic would go here
	s.sendSuccess(w, map[string]interface{}{
		"changes": map[string]interface{}{
			"create": []string{"platform-1"},
			"update": []string{},
			"delete": []string{},
		},
	})
}

func (s *Server) handleRollback(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		s.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	jobID, _ := payload["job_id"].(string)

	s.sendSuccess(w, map[string]interface{}{
		"job_id": jobID,
		"status": "rolled_back",
	})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	s.mu.RLock()
	platformCount := len(s.platforms)
	jobCount := len(s.jobs)
	s.mu.RUnlock()

	stats := map[string]interface{}{
		"platforms": platformCount,
		"jobs":      jobCount,
		"resources": platformCount * 5,
		"uptime":    "24h",
	}

	s.sendSuccess(w, stats)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	health := map[string]interface{}{
		"status":  "healthy",
		"version": "1.0.0",
	}

	s.sendSuccess(w, health)
}

// Helper methods

func (s *Server) sendSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    data,
	})
}

func (s *Server) sendError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(APIResponse{
		Success: false,
		Error:   message,
	})
}

// GetPort returns the server port
func (s *Server) GetPort() int {
	return s.port
}
