package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/platformfoundry/platformfoundry-ce/internal/auth"
	"github.com/platformfoundry/platformfoundry-ce/internal/rbac"
	"github.com/platformfoundry/platformfoundry-ce/internal/service"
	"github.com/platformfoundry/platformfoundry-ce/pkg/types"
)

// handleServiceScorecard handles GET and POST for /api/services/{name}/scorecard
func (s *Server) handleServiceScorecard(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetServiceScorecard(w, r)
	case http.MethodPost:
		s.handleCalculateServiceScorecard(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleScorecards handles GET for /api/scorecards and /api/scorecards/stats
func (s *Server) handleScorecards(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if stats endpoint
	if strings.HasSuffix(r.URL.Path, "/stats") {
		s.handleScorecardStats(w, r)
	} else {
		s.handleListScorecards(w, r)
	}
}

// handleGetServiceScorecard retrieves a service's scorecard
func (s *Server) handleGetServiceScorecard(w http.ResponseWriter, r *http.Request) {
	// Get user context
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		s.sendError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Extract service name from path
	serviceName := extractNameFromPath(r.URL.Path, "/api/services/", "/scorecard")

	// Get organization from query
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

	// Get scorecard calculator
	calculator := s.getScorecardCalculator()
	if calculator == nil {
		s.sendError(w, http.StatusInternalServerError, "scorecard calculator not initialized")
		return
	}

	// Get scorecard
	scorecard, err := calculator.Get(serviceName, organization)
	if err != nil {
		s.sendError(w, http.StatusNotFound, fmt.Sprintf("scorecard not found: %v", err))
		return
	}

	s.sendSuccess(w, scorecard)
}

// handleCalculateServiceScorecard calculates/recalculates a service's scorecard
func (s *Server) handleCalculateServiceScorecard(w http.ResponseWriter, r *http.Request) {
	// Get user context
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		s.sendError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Extract service name from path
	serviceName := extractNameFromPath(r.URL.Path, "/api/services/", "/scorecard")

	// Get organization from query
	organization := r.URL.Query().Get("organization")
	if organization == "" {
		organization = user.Organization
	}

	// Check permission - need operator or admin role to trigger calculations
	if err := s.rbac.CheckOrganizationPermission(
		user.Username,
		organization,
		rbac.ResourceService,
		rbac.ActionUpdate,
	); err != nil {
		s.sendError(w, http.StatusForbidden, "permission denied")
		return
	}

	// Parse optional context from request body
	var contextReq struct {
		Context *service.CheckContext `json:"context,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&contextReq); err != nil && err.Error() != "EOF" {
		s.sendError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	// Get scorecard calculator
	calculator := s.getScorecardCalculator()
	if calculator == nil {
		s.sendError(w, http.StatusInternalServerError, "scorecard calculator not initialized")
		return
	}

	// Calculate scorecard
	scorecard, err := calculator.Calculate(serviceName, organization, contextReq.Context)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, fmt.Sprintf("failed to calculate scorecard: %v", err))
		return
	}

	s.sendSuccess(w, scorecard)
}

// handleListScorecards lists all scorecards for an organization
func (s *Server) handleListScorecards(w http.ResponseWriter, r *http.Request) {
	// Get user context
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		s.sendError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get organization from query
	organization := r.URL.Query().Get("organization")
	if organization == "" {
		organization = user.Organization
	}

	// Check permission
	if err := s.rbac.CheckOrganizationPermission(
		user.Username,
		organization,
		rbac.ResourceService,
		rbac.ActionList,
	); err != nil {
		s.sendError(w, http.StatusForbidden, "permission denied")
		return
	}

	// Get scorecard calculator
	calculator := s.getScorecardCalculator()
	if calculator == nil {
		s.sendError(w, http.StatusInternalServerError, "scorecard calculator not initialized")
		return
	}

	// Get optional grade filter
	gradeFilter := r.URL.Query().Get("grade")

	var scorecards []*types.ServiceScorecard
	var err error

	if gradeFilter != "" {
		// List by grade
		grade := types.ScorecardGrade(gradeFilter)
		scorecards, err = calculator.ListByGrade(grade, organization)
	} else {
		// List all
		scorecards, err = calculator.List(organization)
	}

	if err != nil {
		s.sendError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list scorecards: %v", err))
		return
	}

	s.sendSuccess(w, scorecards)
}

// handleScorecardStats returns aggregated scorecard statistics
func (s *Server) handleScorecardStats(w http.ResponseWriter, r *http.Request) {
	// Get user context
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		s.sendError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get organization from query
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

	// Get scorecard calculator
	calculator := s.getScorecardCalculator()
	if calculator == nil {
		s.sendError(w, http.StatusInternalServerError, "scorecard calculator not initialized")
		return
	}

	// Get statistics
	stats, err := calculator.GetStats(organization)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get stats: %v", err))
		return
	}

	s.sendSuccess(w, stats)
}

// getScorecardCalculator returns the scorecard calculator instance
func (s *Server) getScorecardCalculator() *service.ScorecardCalculator {
	s.mu.RLock()
	if calc, ok := s.platforms["scorecardCalculator"]; ok {
		s.mu.RUnlock()
		return calc.(*service.ScorecardCalculator)
	}
	s.mu.RUnlock()
	return nil
}

// SetScorecardCalculator sets the scorecard calculator
func (s *Server) SetScorecardCalculator(calculator *service.ScorecardCalculator) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.platforms["scorecardCalculator"] = calculator
}

// extractNameFromPath extracts a name from a path with prefix and suffix
func extractNameFromPath(path, prefix, suffix string) string {
	// Remove prefix
	name := strings.TrimPrefix(path, prefix)
	// Remove suffix
	name = strings.TrimSuffix(name, suffix)
	return name
}
