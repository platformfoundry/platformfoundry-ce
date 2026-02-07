package handlers

import (
	"net/http"

	"github.com/platformfoundry/pf-ce/internal/workload"
	"github.com/platformfoundry/pf-ce/pkg/types"
	"gopkg.in/yaml.v3"
)

// WorkloadRequest represents a workload creation request
type WorkloadRequest struct {
	Config string `json:"config"` // YAML content
	DryRun bool   `json:"dryRun"`
}

// ListWorkloads returns all workloads
func (h *Handler) ListWorkloads(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	workloads, err := h.Orchestrator.ListWorkloads(ctx)
	if err != nil {
		h.Error(w, http.StatusInternalServerError, "STATE_ERROR", err.Error())
		return
	}

	h.JSON(w, http.StatusOK, map[string]interface{}{
		"workloads": workloads,
		"count":     len(workloads),
	})
}

// GetWorkload returns a specific workload
func (h *Handler) GetWorkload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := r.PathValue("name")

	if name == "" {
		h.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "workload name is required")
		return
	}

	ws, err := h.Orchestrator.GetWorkloadStatus(ctx, name)
	if err != nil {
		h.Error(w, http.StatusNotFound, "NOT_FOUND", "workload not found")
		return
	}

	h.JSON(w, http.StatusOK, ws)
}

// CreateWorkload creates and applies a new workload
func (h *Handler) CreateWorkload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req WorkloadRequest
	if err := h.DecodeJSON(r, &req); err != nil {
		h.Error(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	// Parse workload YAML
	var wl types.Workload
	if err := yaml.Unmarshal([]byte(req.Config), &wl); err != nil {
		h.Error(w, http.StatusBadRequest, "INVALID_YAML", err.Error())
		return
	}

	// Validate workload
	if err := wl.Validate(); err != nil {
		h.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	// Translate
	translator := workload.NewTranslator("aws", "us-east-1", "default")
	result, err := translator.Translate(&wl)
	if err != nil {
		h.Error(w, http.StatusBadRequest, "TRANSLATION_ERROR", err.Error())
		return
	}

	// If dry run, return plan
	if req.DryRun {
		h.JSON(w, http.StatusOK, map[string]interface{}{
			"dryRun":         true,
			"deployment":     result.Deployment,
			"service":        result.Service,
			"hpa":            result.HPA,
			"ingress":        result.Ingress,
			"infraResources": result.InfraResources,
			"outputs":        result.Outputs,
		})
		return
	}

	// Apply
	applyResult, err := h.Orchestrator.ApplyWorkload(ctx, &wl, result)
	if err != nil {
		h.Error(w, http.StatusInternalServerError, "APPLY_ERROR", err.Error())
		return
	}

	h.JSON(w, http.StatusCreated, applyResult)
}

// DeleteWorkload removes a workload
func (h *Handler) DeleteWorkload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := r.PathValue("name")

	if name == "" {
		h.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "workload name is required")
		return
	}

	if err := h.Orchestrator.DeleteWorkload(ctx, name); err != nil {
		h.Error(w, http.StatusInternalServerError, "DELETE_ERROR", err.Error())
		return
	}

	h.JSON(w, http.StatusOK, map[string]string{
		"message": "workload deleted",
		"deleted": name,
	})
}

// GetWorkloadStatus returns workload status
func (h *Handler) GetWorkloadStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := r.PathValue("name")

	if name == "" {
		h.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "workload name is required")
		return
	}

	status, err := h.Orchestrator.GetWorkloadStatus(ctx, name)
	if err != nil {
		h.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}

	h.JSON(w, http.StatusOK, status)
}
