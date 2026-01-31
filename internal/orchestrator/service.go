package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/platformfoundry/platformfoundry-ce/internal/engine"
	"github.com/platformfoundry/platformfoundry-ce/internal/plugin"
	"github.com/platformfoundry/platformfoundry-ce/internal/state"
	"github.com/platformfoundry/platformfoundry-ce/internal/workload"
	"github.com/platformfoundry/platformfoundry-ce/pkg/types"
)

// Service orchestrates workload deployment using the coordinator and plugins
type Service struct {
	pluginManager *plugin.Manager
	stateBackend  state.Backend
	eventBus      *engine.EventBus
	config        Config
}

// Config configures the orchestrator service
type Config struct {
	MaxParallel       int
	Timeout           time.Duration
	RollbackOnFailure bool
	DryRun            bool
	RetryCount        int
	RetryDelay        time.Duration
}

// DefaultConfig returns default orchestrator configuration
func DefaultConfig() Config {
	return Config{
		MaxParallel:       4,
		Timeout:           30 * time.Minute,
		RollbackOnFailure: true,
		DryRun:            false,
		RetryCount:        3,
		RetryDelay:        5 * time.Second,
	}
}

// ApplyResult contains the result of applying a workload
type ApplyResult struct {
	WorkloadName string                 `json:"workloadName"`
	Namespace    string                 `json:"namespace"`
	Success      bool                   `json:"success"`
	Message      string                 `json:"message"`
	Outputs      map[string]interface{} `json:"outputs"`
	Resources    []ResourceResult       `json:"resources"`
	Duration     time.Duration          `json:"duration"`
	Error        error                  `json:"error,omitempty"`
}

// ResourceResult contains the result of applying a single resource
type ResourceResult struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// WorkloadState represents stored workload state
type WorkloadState struct {
	Name        string                 `json:"name"`
	Namespace   string                 `json:"namespace"`
	Version     string                 `json:"version"`
	Outputs     map[string]interface{} `json:"outputs"`
	AppliedAt   time.Time              `json:"appliedAt"`
	Resources   []ResourceResult       `json:"resources"`
	Translation *workload.TranslationResult `json:"translation,omitempty"`
}

// NewService creates a new orchestrator service
func NewService(cfg Config, pm *plugin.Manager, sb state.Backend) *Service {
	if cfg.MaxParallel == 0 {
		cfg.MaxParallel = 4
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Minute
	}

	return &Service{
		pluginManager: pm,
		stateBackend:  sb,
		eventBus:      engine.NewEventBus(),
		config:        cfg,
	}
}

// ApplyWorkload applies a translated workload
func (s *Service) ApplyWorkload(ctx context.Context, w *types.Workload, translation *workload.TranslationResult) (*ApplyResult, error) {
	startTime := time.Now()

	result := &ApplyResult{
		WorkloadName: w.Metadata.Name,
		Namespace:    getNamespace(w, translation),
		Outputs:      make(map[string]interface{}),
		Resources:    make([]ResourceResult, 0),
	}

	// Create coordinator
	coordinator := engine.NewCoordinator(engine.CoordinatorConfig{
		MaxParallelEngines: s.config.MaxParallel,
		Timeout:            s.config.Timeout,
		RollbackOnFailure:  s.config.RollbackOnFailure,
		RetryCount:         s.config.RetryCount,
		RetryDelay:         s.config.RetryDelay,
	})

	// Subscribe to events
	coordinator.Subscribe(s.eventBus)

	// Create and register engines based on translation
	specs, err := s.createEngines(coordinator, translation)
	if err != nil {
		result.Error = err
		result.Message = fmt.Sprintf("failed to create engines: %v", err)
		return result, err
	}

	// Execute
	s.eventBus.EmitCoordinatorEvent("workload_apply_started",
		fmt.Sprintf("Applying workload: %s", w.Metadata.Name),
		map[string]interface{}{
			"workload":  w.Metadata.Name,
			"namespace": result.Namespace,
		})

	if err := coordinator.Apply(ctx, specs); err != nil {
		result.Error = err
		result.Success = false
		result.Message = fmt.Sprintf("apply failed: %v", err)

		s.eventBus.EmitCoordinatorEvent("workload_apply_failed",
			fmt.Sprintf("Workload %s failed: %v", w.Metadata.Name, err),
			map[string]interface{}{
				"workload": w.Metadata.Name,
				"error":    err.Error(),
			})

		return result, err
	}

	// Collect results
	for _, r := range coordinator.GetResults() {
		for _, res := range r.Resources {
			result.Resources = append(result.Resources, ResourceResult{
				Name:   res,
				Status: "created",
			})
		}
		for k, v := range r.Outputs {
			result.Outputs[k] = v
		}
	}

	result.Success = true
	result.Duration = time.Since(startTime)
	result.Message = fmt.Sprintf("Successfully applied %d resources", len(result.Resources))

	// Store state
	if s.stateBackend != nil {
		if err := s.storeState(ctx, w, result, translation); err != nil {
			// Non-fatal error
			s.eventBus.EmitCoordinatorEvent("state_store_warning",
				fmt.Sprintf("Failed to store state: %v", err),
				nil)
		}
	}

	s.eventBus.EmitCoordinatorEvent("workload_apply_completed",
		fmt.Sprintf("Workload %s applied successfully", w.Metadata.Name),
		map[string]interface{}{
			"workload":  w.Metadata.Name,
			"resources": len(result.Resources),
			"duration":  result.Duration.String(),
		})

	return result, nil
}

// createEngines creates engines based on translation result
func (s *Service) createEngines(coordinator *engine.Coordinator, translation *workload.TranslationResult) (map[string]map[string]interface{}, error) {
	specs := make(map[string]map[string]interface{})

	// Create infrastructure engine if there are infra resources
	if len(translation.InfraResources) > 0 {
		infraEngine := NewInfrastructureEngine("infra-engine", s.pluginManager)
		if err := coordinator.RegisterEngine(infraEngine); err != nil {
			return nil, fmt.Errorf("failed to register infrastructure engine: %w", err)
		}

		specs["infra-engine"] = map[string]interface{}{
			"resources": translation.InfraResources,
		}
	}

	// Create Kubernetes engine if there are K8s resources
	if translation.Deployment != nil || translation.Service != nil {
		k8sEngine := NewKubernetesEngine("k8s-engine", s.pluginManager)

		// K8s depends on infrastructure
		if len(translation.InfraResources) > 0 {
			k8sEngine.SetDependencies([]string{"infra-engine"})
		}

		if err := coordinator.RegisterEngine(k8sEngine); err != nil {
			return nil, fmt.Errorf("failed to register kubernetes engine: %w", err)
		}

		namespace := "default"
		if translation.Deployment != nil {
			namespace = translation.Deployment.Namespace
		}

		specs["k8s-engine"] = map[string]interface{}{
			"namespace":  namespace,
			"deployment": translation.Deployment,
			"service":    translation.Service,
			"hpa":        translation.HPA,
			"ingress":    translation.Ingress,
			"configMaps": translation.ConfigMaps,
			"secrets":    translation.Secrets,
		}
	}

	return specs, nil
}

// storeState persists workload state
func (s *Service) storeState(ctx context.Context, w *types.Workload, result *ApplyResult, translation *workload.TranslationResult) error {
	ws := &WorkloadState{
		Name:        w.Metadata.Name,
		Namespace:   result.Namespace,
		Outputs:     result.Outputs,
		AppliedAt:   time.Now(),
		Resources:   result.Resources,
		Translation: translation,
	}

	// Convert to state.Resource
	resource := &state.Resource{
		Name:       fmt.Sprintf("workloads/%s", w.Metadata.Name),
		Kind:       "Workload",
		APIVersion: "platformfoundry.io/v1",
		Spec: map[string]interface{}{
			"name":        ws.Name,
			"namespace":   ws.Namespace,
			"outputs":     ws.Outputs,
			"resources":   ws.Resources,
			"translation": ws.Translation,
		},
		Status: map[string]interface{}{
			"appliedAt": ws.AppliedAt,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return s.stateBackend.Save(resource)
}

// GetWorkloadStatus returns the status of a deployed workload
func (s *Service) GetWorkloadStatus(ctx context.Context, name string) (*WorkloadState, error) {
	key := fmt.Sprintf("workloads/%s", name)

	resource, err := s.stateBackend.Get(key)
	if err != nil {
		return nil, fmt.Errorf("workload not found: %s", name)
	}

	ws := &WorkloadState{
		Name: name,
	}

	if ns, ok := resource.Spec["namespace"].(string); ok {
		ws.Namespace = ns
	}
	if outputs, ok := resource.Spec["outputs"].(map[string]interface{}); ok {
		ws.Outputs = outputs
	}
	if appliedAt, ok := resource.Status["appliedAt"].(time.Time); ok {
		ws.AppliedAt = appliedAt
	}

	return ws, nil
}

// ListWorkloads returns all deployed workloads
func (s *Service) ListWorkloads(ctx context.Context) ([]WorkloadState, error) {
	resources, err := s.stateBackend.List()
	if err != nil {
		return nil, err
	}

	workloads := make([]WorkloadState, 0)
	for _, res := range resources {
		if res.Kind != "Workload" {
			continue
		}

		ws := WorkloadState{
			Name: res.Name,
		}
		if ns, ok := res.Spec["namespace"].(string); ok {
			ws.Namespace = ns
		}
		if outputs, ok := res.Spec["outputs"].(map[string]interface{}); ok {
			ws.Outputs = outputs
		}
		workloads = append(workloads, ws)
	}

	return workloads, nil
}

// DeleteWorkload removes a deployed workload
func (s *Service) DeleteWorkload(ctx context.Context, name string) error {
	// Get current state
	ws, err := s.GetWorkloadStatus(ctx, name)
	if err != nil {
		return err
	}

	// Create coordinator for deletion
	coordinator := engine.NewCoordinator(engine.CoordinatorConfig{
		MaxParallelEngines: s.config.MaxParallel,
		Timeout:            s.config.Timeout,
	})

	// Delete in reverse order (K8s first, then infrastructure)
	if ws.Translation != nil {
		if ws.Translation.Deployment != nil {
			k8sEngine := NewKubernetesEngine("k8s-delete", s.pluginManager)
			coordinator.RegisterEngine(k8sEngine)
		}

		if len(ws.Translation.InfraResources) > 0 {
			infraEngine := NewInfrastructureEngine("infra-delete", s.pluginManager)
			infraEngine.SetDependencies([]string{"k8s-delete"})
			coordinator.RegisterEngine(infraEngine)
		}
	}

	// Delete state
	key := fmt.Sprintf("workloads/%s", name)
	return s.stateBackend.Delete(key)
}

// Subscribe adds an event listener
func (s *Service) Subscribe(listener engine.EventListener) {
	s.eventBus.Subscribe(listener)
}

// Unsubscribe removes an event listener
func (s *Service) Unsubscribe(listener engine.EventListener) {
	s.eventBus.Unsubscribe(listener)
}

// GetEventBus returns the event bus for external subscriptions
func (s *Service) GetEventBus() *engine.EventBus {
	return s.eventBus
}

func getNamespace(w *types.Workload, t *workload.TranslationResult) string {
	if t.Deployment != nil && t.Deployment.Namespace != "" {
		return t.Deployment.Namespace
	}
	if w.Metadata.Environment != "" {
		return w.Metadata.Environment
	}
	return "default"
}
