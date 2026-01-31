package web

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/platformfoundry/platformfoundry-ce/internal/auth"
	"github.com/platformfoundry/platformfoundry-ce/internal/rbac"
	"github.com/platformfoundry/platformfoundry-ce/internal/service"
	"github.com/platformfoundry/platformfoundry-ce/pkg/types"
)

// handleServices handles listing and creating services
// GET /api/services - List services
// POST /api/services - Create service
func (s *Server) handleServices(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListServices(w, r)
	case http.MethodPost:
		s.handleCreateService(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleService handles getting, updating, and deleting a service
// GET /api/services/{name} - Get service
// PUT /api/services/{name} - Update service
// DELETE /api/services/{name} - Delete service
// GET/POST /api/services/{name}/scorecard - Service scorecard
func (s *Server) handleService(w http.ResponseWriter, r *http.Request) {
	// Check if this is a scorecard request
	if strings.HasSuffix(r.URL.Path, "/scorecard") {
		s.handleServiceScorecard(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetService(w, r)
	case http.MethodPut:
		s.handleUpdateService(w, r)
	case http.MethodDelete:
		s.handleDeleteService(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleListServices lists services
func (s *Server) handleListServices(w http.ResponseWriter, r *http.Request) {
	// Get user context
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		s.sendError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Check permission
	if err := s.rbac.CheckOrganizationPermission(
		user.Username,
		user.Organization,
		rbac.ResourceService,
		rbac.ActionList,
	); err != nil {
		s.sendError(w, http.StatusForbidden, "permission denied")
		return
	}

	// Get query parameters
	organization := r.URL.Query().Get("organization")
	if organization == "" {
		organization = user.Organization
	}
	team := r.URL.Query().Get("team")
	serviceType := r.URL.Query().Get("type")
	state := r.URL.Query().Get("state")
	health := r.URL.Query().Get("health")

	// Get service manager
	manager := s.getServiceManager()
	if manager == nil {
		s.sendError(w, http.StatusInternalServerError, "service manager not initialized")
		return
	}

	// List services based on filters
	var services []*types.Service
	var err error

	if team != "" {
		services, err = manager.ListByTeam(team, organization)
	} else if serviceType != "" {
		services, err = manager.ListByType(types.ServiceType(serviceType), organization)
	} else {
		services, err = manager.List(organization)
	}

	if err != nil {
		s.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Apply additional filters
	if state != "" {
		filtered := make([]*types.Service, 0)
		for _, svc := range services {
			if string(svc.Status.State) == state {
				filtered = append(filtered, svc)
			}
		}
		services = filtered
	}

	if health != "" {
		filtered := make([]*types.Service, 0)
		for _, svc := range services {
			if string(svc.Status.Health) == health {
				filtered = append(filtered, svc)
			}
		}
		services = filtered
	}

	s.sendSuccess(w, services)
}

// handleCreateService creates a new service
func (s *Server) handleCreateService(w http.ResponseWriter, r *http.Request) {
	// Get user context
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		s.sendError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse request body
	var svc types.Service
	if err := json.NewDecoder(r.Body).Decode(&svc); err != nil {
		s.sendError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	// Set organization from user context if not provided
	if svc.Metadata.Organization == "" {
		svc.Metadata.Organization = user.Organization
	}

	// Check permission
	if err := s.rbac.CheckOrganizationPermission(
		user.Username,
		svc.Metadata.Organization,
		rbac.ResourceService,
		rbac.ActionCreate,
	); err != nil {
		s.sendError(w, http.StatusForbidden, "permission denied")
		return
	}

	// Get service manager
	manager := s.getServiceManager()
	if manager == nil {
		s.sendError(w, http.StatusInternalServerError, "service manager not initialized")
		return
	}

	// Create service
	if err := manager.Create(&svc); err != nil {
		s.sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Audit log
	if s.auditLogger != nil {
		s.auditLogger.LogCreate(user.Username, "Service", svc.Metadata.Name, "success", "Service created via API")
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    svc,
		Message: "Service created successfully",
	})
}

// handleGetService gets a service
func (s *Server) handleGetService(w http.ResponseWriter, r *http.Request) {
	// Get user context
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		s.sendError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Extract service name from path
	name := strings.TrimPrefix(r.URL.Path, "/api/services/")
	if name == "" {
		s.sendError(w, http.StatusBadRequest, "service name is required")
		return
	}

	// Get organization from query or user context
	organization := r.URL.Query().Get("organization")
	if organization == "" {
		organization = user.Organization
	}

	// Check permission
	if err := s.rbac.CheckOrganizationPermission(
		user.Username,
		organization,
		rbac.ResourceService,
		rbac.ActionRead,
	); err != nil {
		s.sendError(w, http.StatusForbidden, "permission denied")
		return
	}

	// Get service manager
	manager := s.getServiceManager()
	if manager == nil {
		s.sendError(w, http.StatusInternalServerError, "service manager not initialized")
		return
	}

	// Get service
	svc, err := manager.Get(name, organization)
	if err != nil {
		s.sendError(w, http.StatusNotFound, err.Error())
		return
	}

	s.sendSuccess(w, svc)
}

// handleUpdateService updates a service
func (s *Server) handleUpdateService(w http.ResponseWriter, r *http.Request) {
	// Get user context
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		s.sendError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Extract service name from path
	name := strings.TrimPrefix(r.URL.Path, "/api/services/")
	if name == "" {
		s.sendError(w, http.StatusBadRequest, "service name is required")
		return
	}

	// Parse request body
	var svc types.Service
	if err := json.NewDecoder(r.Body).Decode(&svc); err != nil {
		s.sendError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	// Set organization from user context if not provided
	if svc.Metadata.Organization == "" {
		svc.Metadata.Organization = user.Organization
	}

	// Ensure name matches
	if svc.Metadata.Name != name {
		s.sendError(w, http.StatusBadRequest, "service name mismatch")
		return
	}

	// Check permission
	if err := s.rbac.CheckOrganizationPermission(
		user.Username,
		svc.Metadata.Organization,
		rbac.ResourceService,
		rbac.ActionUpdate,
	); err != nil {
		s.sendError(w, http.StatusForbidden, "permission denied")
		return
	}

	// Get service manager
	manager := s.getServiceManager()
	if manager == nil {
		s.sendError(w, http.StatusInternalServerError, "service manager not initialized")
		return
	}

	// Update service
	if err := manager.Update(&svc); err != nil {
		s.sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Audit log
	if s.auditLogger != nil {
		s.auditLogger.LogUpdate(user.Username, "Service", svc.Metadata.Name, "success", "Service updated via API")
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    svc,
		Message: "Service updated successfully",
	})
}

// handleDeleteService deletes a service
func (s *Server) handleDeleteService(w http.ResponseWriter, r *http.Request) {
	// Get user context
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		s.sendError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Extract service name from path
	name := strings.TrimPrefix(r.URL.Path, "/api/services/")
	if name == "" {
		s.sendError(w, http.StatusBadRequest, "service name is required")
		return
	}

	// Get organization from query or user context
	organization := r.URL.Query().Get("organization")
	if organization == "" {
		organization = user.Organization
	}

	// Check permission
	if err := s.rbac.CheckOrganizationPermission(
		user.Username,
		organization,
		rbac.ResourceService,
		rbac.ActionDelete,
	); err != nil {
		s.sendError(w, http.StatusForbidden, "permission denied")
		return
	}

	// Get service manager
	manager := s.getServiceManager()
	if manager == nil {
		s.sendError(w, http.StatusInternalServerError, "service manager not initialized")
		return
	}

	// Delete service
	if err := manager.Delete(name, organization); err != nil {
		s.sendError(w, http.StatusNotFound, err.Error())
		return
	}

	// Audit log
	if s.auditLogger != nil {
		s.auditLogger.LogDelete(user.Username, "Service", name, "success", "Service deleted via API")
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Message: "Service deleted successfully",
	})
}

// getServiceManager gets or creates the service manager
func (s *Server) getServiceManager() *service.Manager {
	s.mu.RLock()
	if mgr, ok := s.platforms["serviceManager"]; ok {
		s.mu.RUnlock()
		return mgr.(*service.Manager)
	}
	s.mu.RUnlock()

	// Service manager should be initialized when server starts
	// For now, return nil to indicate it's not initialized
	return nil
}

// SetServiceManager sets the service manager
func (s *Server) SetServiceManager(manager *service.Manager) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.platforms["serviceManager"] = manager
}
