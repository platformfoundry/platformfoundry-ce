package handlers

import (
	"github.com/platformfoundry/platformfoundry-ce/internal/api"
)

// RegisterRoutes sets up all API routes with the handler
func RegisterRoutes(s *api.Server, h *Handler) {
	// Workloads API
	s.RegisterRoute(api.Route{
		Method:      "GET",
		Path:        "/api/v1/workloads",
		Handler:     h.ListWorkloads,
		Description: "List all workloads",
		Auth:        true,
	})
	s.RegisterRoute(api.Route{
		Method:      "POST",
		Path:        "/api/v1/workloads",
		Handler:     h.CreateWorkload,
		Description: "Create and apply a workload",
		Auth:        true,
	})
	s.RegisterRoute(api.Route{
		Method:      "GET",
		Path:        "/api/v1/workloads/{name}",
		Handler:     h.GetWorkload,
		Description: "Get workload details",
		Auth:        true,
	})
	s.RegisterRoute(api.Route{
		Method:      "DELETE",
		Path:        "/api/v1/workloads/{name}",
		Handler:     h.DeleteWorkload,
		Description: "Delete a workload",
		Auth:        true,
	})
	s.RegisterRoute(api.Route{
		Method:      "GET",
		Path:        "/api/v1/workloads/{name}/status",
		Handler:     h.GetWorkloadStatus,
		Description: "Get workload status",
		Auth:        true,
	})

	// Events API
	s.RegisterRoute(api.Route{
		Method:      "GET",
		Path:        "/api/v1/events",
		Handler:     h.ListEvents,
		Description: "List historical events",
		Auth:        true,
	})
	s.RegisterRoute(api.Route{
		Method:      "GET",
		Path:        "/api/v1/events/stream",
		Handler:     h.StreamEvents,
		Description: "Stream events via SSE",
		Auth:        true,
	})
}
