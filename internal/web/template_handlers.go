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

// handleTemplates handles listing and creating templates
// GET /api/templates - List templates
// POST /api/templates - Create template
func (s *Server) handleTemplates(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListTemplates(w, r)
	case http.MethodPost:
		s.handleCreateTemplate(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleTemplate handles getting, updating, and deleting a template
// GET /api/templates/{name} - Get template
// PUT /api/templates/{name} - Update template
// DELETE /api/templates/{name} - Delete template
func (s *Server) handleTemplate(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetTemplate(w, r)
	case http.MethodPut:
		s.handleUpdateTemplate(w, r)
	case http.MethodDelete:
		s.handleDeleteTemplate(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleListTemplates lists service templates
func (s *Server) handleListTemplates(w http.ResponseWriter, r *http.Request) {
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
		rbac.ResourceServiceTemplate,
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
	category := r.URL.Query().Get("category")
	tag := r.URL.Query().Get("tag")

	// Get template manager
	manager := s.getTemplateManager()
	if manager == nil {
		s.sendError(w, http.StatusInternalServerError, "template manager not initialized")
		return
	}

	// List templates based on filters
	var templates []*types.ServiceTemplate
	var err error

	if category != "" {
		templates, err = manager.ListByCategory(types.TemplateCategory(category), organization)
	} else if tag != "" {
		templates, err = manager.SearchByTag(tag, organization)
	} else {
		templates, err = manager.List(organization)
	}

	if err != nil {
		s.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.sendSuccess(w, templates)
}

// handleCreateTemplate creates a new template
func (s *Server) handleCreateTemplate(w http.ResponseWriter, r *http.Request) {
	// Get user context
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		s.sendError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse request body
	var template types.ServiceTemplate
	if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
		s.sendError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	// Set organization from user context if not provided
	if template.Metadata.Organization == "" {
		template.Metadata.Organization = user.Organization
	}

	// Check permission
	if err := s.rbac.CheckOrganizationPermission(
		user.Username,
		template.Metadata.Organization,
		rbac.ResourceServiceTemplate,
		rbac.ActionCreate,
	); err != nil {
		s.sendError(w, http.StatusForbidden, "permission denied")
		return
	}

	// Get template manager
	manager := s.getTemplateManager()
	if manager == nil {
		s.sendError(w, http.StatusInternalServerError, "template manager not initialized")
		return
	}

	// Create template
	if err := manager.Create(&template); err != nil {
		s.sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Audit log
	if s.auditLogger != nil {
		s.auditLogger.LogCreate(user.Username, "ServiceTemplate", template.Metadata.Name, "success", "Template created via API")
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    template,
		Message: "Template created successfully",
	})
}

// handleGetTemplate gets a template
func (s *Server) handleGetTemplate(w http.ResponseWriter, r *http.Request) {
	// Get user context
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		s.sendError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Extract template name from path
	name := strings.TrimPrefix(r.URL.Path, "/api/templates/")
	if name == "" {
		s.sendError(w, http.StatusBadRequest, "template name is required")
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
		rbac.ResourceServiceTemplate,
		rbac.ActionRead,
	); err != nil {
		s.sendError(w, http.StatusForbidden, "permission denied")
		return
	}

	// Get template manager
	manager := s.getTemplateManager()
	if manager == nil {
		s.sendError(w, http.StatusInternalServerError, "template manager not initialized")
		return
	}

	// Get template
	template, err := manager.Get(name, organization)
	if err != nil {
		s.sendError(w, http.StatusNotFound, err.Error())
		return
	}

	s.sendSuccess(w, template)
}

// handleUpdateTemplate updates a template
func (s *Server) handleUpdateTemplate(w http.ResponseWriter, r *http.Request) {
	// Get user context
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		s.sendError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Extract template name from path
	name := strings.TrimPrefix(r.URL.Path, "/api/templates/")
	if name == "" {
		s.sendError(w, http.StatusBadRequest, "template name is required")
		return
	}

	// Parse request body
	var template types.ServiceTemplate
	if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
		s.sendError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	// Set organization from user context if not provided
	if template.Metadata.Organization == "" {
		template.Metadata.Organization = user.Organization
	}

	// Ensure name matches
	if template.Metadata.Name != name {
		s.sendError(w, http.StatusBadRequest, "template name mismatch")
		return
	}

	// Check permission
	if err := s.rbac.CheckOrganizationPermission(
		user.Username,
		template.Metadata.Organization,
		rbac.ResourceServiceTemplate,
		rbac.ActionUpdate,
	); err != nil {
		s.sendError(w, http.StatusForbidden, "permission denied")
		return
	}

	// Get template manager
	manager := s.getTemplateManager()
	if manager == nil {
		s.sendError(w, http.StatusInternalServerError, "template manager not initialized")
		return
	}

	// Update template
	if err := manager.Update(&template); err != nil {
		s.sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Audit log
	if s.auditLogger != nil {
		s.auditLogger.LogUpdate(user.Username, "ServiceTemplate", template.Metadata.Name, "success", "Template updated via API")
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    template,
		Message: "Template updated successfully",
	})
}

// handleDeleteTemplate deletes a template
func (s *Server) handleDeleteTemplate(w http.ResponseWriter, r *http.Request) {
	// Get user context
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		s.sendError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Extract template name from path
	name := strings.TrimPrefix(r.URL.Path, "/api/templates/")
	if name == "" {
		s.sendError(w, http.StatusBadRequest, "template name is required")
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
		rbac.ResourceServiceTemplate,
		rbac.ActionDelete,
	); err != nil {
		s.sendError(w, http.StatusForbidden, "permission denied")
		return
	}

	// Get template manager
	manager := s.getTemplateManager()
	if manager == nil {
		s.sendError(w, http.StatusInternalServerError, "template manager not initialized")
		return
	}

	// Delete template
	if err := manager.Delete(name, organization); err != nil {
		s.sendError(w, http.StatusNotFound, err.Error())
		return
	}

	// Audit log
	if s.auditLogger != nil {
		s.auditLogger.LogDelete(user.Username, "ServiceTemplate", name, "success", "Template deleted via API")
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Message: "Template deleted successfully",
	})
}

// getTemplateManager gets or creates the template manager
func (s *Server) getTemplateManager() *service.TemplateManager {
	s.mu.RLock()
	if mgr, ok := s.platforms["templateManager"]; ok {
		s.mu.RUnlock()
		return mgr.(*service.TemplateManager)
	}
	s.mu.RUnlock()

	// Template manager should be initialized when server starts
	// For now, return nil to indicate it's not initialized
	return nil
}

// SetTemplateManager sets the template manager
func (s *Server) SetTemplateManager(manager *service.TemplateManager) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.platforms["templateManager"] = manager
}
