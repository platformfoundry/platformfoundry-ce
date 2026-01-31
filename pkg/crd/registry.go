// Package crd provides Custom Resource Definition support for Platform Foundry.
package crd

import (
	"fmt"
	"strings"
	"sync"
)

// Registry manages Custom Resource Definitions
type Registry struct {
	mu   sync.RWMutex
	crds map[string]*CustomResourceDefinition
	// Index by group/version/kind
	byGVK map[GroupVersionKind]*CustomResourceDefinition
	// Index by group/version/resource
	byGVR map[GroupVersionResource]*CustomResourceDefinition
}

// NewRegistry creates a new CRD registry
func NewRegistry() *Registry {
	return &Registry{
		crds:  make(map[string]*CustomResourceDefinition),
		byGVK: make(map[GroupVersionKind]*CustomResourceDefinition),
		byGVR: make(map[GroupVersionResource]*CustomResourceDefinition),
	}
}

// Register registers a CRD
func (r *Registry) Register(crd *CustomResourceDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate CRD
	if err := r.validate(crd); err != nil {
		return err
	}

	// Check for conflicts
	name := crd.Metadata.Name
	if existing, ok := r.crds[name]; ok {
		return &RegistryError{
			CRD:     name,
			Message: fmt.Sprintf("CRD already registered with version %s", existing.Spec.Version),
		}
	}

	// Build indexes
	gvk := GroupVersionKind{
		Group:   crd.Spec.Group,
		Version: crd.Spec.Version,
		Kind:    crd.Spec.Kind,
	}

	if _, ok := r.byGVK[gvk]; ok {
		return &RegistryError{
			CRD:     name,
			Message: fmt.Sprintf("GVK %s already registered", gvk),
		}
	}

	plural := crd.Spec.Plural
	if plural == "" {
		plural = strings.ToLower(crd.Spec.Kind) + "s"
	}

	gvr := GroupVersionResource{
		Group:    crd.Spec.Group,
		Version:  crd.Spec.Version,
		Resource: plural,
	}

	if _, ok := r.byGVR[gvr]; ok {
		return &RegistryError{
			CRD:     name,
			Message: fmt.Sprintf("GVR %s already registered", gvr),
		}
	}

	// Store CRD
	r.crds[name] = crd
	r.byGVK[gvk] = crd
	r.byGVR[gvr] = crd

	return nil
}

// Unregister removes a CRD
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	crd, ok := r.crds[name]
	if !ok {
		return &RegistryError{
			CRD:     name,
			Message: "CRD not found",
		}
	}

	// Remove from indexes
	gvk := GroupVersionKind{
		Group:   crd.Spec.Group,
		Version: crd.Spec.Version,
		Kind:    crd.Spec.Kind,
	}
	delete(r.byGVK, gvk)

	plural := crd.Spec.Plural
	if plural == "" {
		plural = strings.ToLower(crd.Spec.Kind) + "s"
	}
	gvr := GroupVersionResource{
		Group:    crd.Spec.Group,
		Version:  crd.Spec.Version,
		Resource: plural,
	}
	delete(r.byGVR, gvr)

	delete(r.crds, name)

	return nil
}

// Get returns a CRD by name
func (r *Registry) Get(name string) (*CustomResourceDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	crd, ok := r.crds[name]
	if !ok {
		return nil, &RegistryError{
			CRD:     name,
			Message: "CRD not found",
		}
	}

	return crd, nil
}

// GetByGVK returns a CRD by group/version/kind
func (r *Registry) GetByGVK(gvk GroupVersionKind) (*CustomResourceDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	crd, ok := r.byGVK[gvk]
	if !ok {
		return nil, &RegistryError{
			CRD:     gvk.String(),
			Message: "CRD not found",
		}
	}

	return crd, nil
}

// GetByGVR returns a CRD by group/version/resource
func (r *Registry) GetByGVR(gvr GroupVersionResource) (*CustomResourceDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	crd, ok := r.byGVR[gvr]
	if !ok {
		return nil, &RegistryError{
			CRD:     gvr.String(),
			Message: "CRD not found",
		}
	}

	return crd, nil
}

// List returns all registered CRDs
func (r *Registry) List() []*CustomResourceDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*CustomResourceDefinition, 0, len(r.crds))
	for _, crd := range r.crds {
		result = append(result, crd)
	}

	return result
}

// ListByGroup returns CRDs in a specific group
func (r *Registry) ListByGroup(group string) []*CustomResourceDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*CustomResourceDefinition
	for _, crd := range r.crds {
		if crd.Spec.Group == group {
			result = append(result, crd)
		}
	}

	return result
}

// Has checks if a CRD is registered
func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.crds[name]
	return ok
}

// HasGVK checks if a GVK is registered
func (r *Registry) HasGVK(gvk GroupVersionKind) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.byGVK[gvk]
	return ok
}

// validate validates a CRD
func (r *Registry) validate(crd *CustomResourceDefinition) error {
	if crd.Metadata.Name == "" {
		return &RegistryError{
			CRD:     "",
			Message: "CRD name is required",
		}
	}

	if crd.Spec.Group == "" {
		return &RegistryError{
			CRD:     crd.Metadata.Name,
			Message: "CRD group is required",
		}
	}

	if crd.Spec.Version == "" {
		return &RegistryError{
			CRD:     crd.Metadata.Name,
			Message: "CRD version is required",
		}
	}

	if crd.Spec.Kind == "" {
		return &RegistryError{
			CRD:     crd.Metadata.Name,
			Message: "CRD kind is required",
		}
	}

	if crd.Spec.Scope != ClusterScope && crd.Spec.Scope != NamespacedScope {
		return &RegistryError{
			CRD:     crd.Metadata.Name,
			Message: fmt.Sprintf("CRD scope must be %q or %q", ClusterScope, NamespacedScope),
		}
	}

	return nil
}

// RegistryError represents a registry error
type RegistryError struct {
	CRD     string
	Message string
}

func (e *RegistryError) Error() string {
	if e.CRD != "" {
		return fmt.Sprintf("CRD %q: %s", e.CRD, e.Message)
	}
	return e.Message
}

// DefaultRegistry is the default global CRD registry
var DefaultRegistry = NewRegistry()

// Register registers a CRD in the default registry
func Register(crd *CustomResourceDefinition) error {
	return DefaultRegistry.Register(crd)
}

// Get returns a CRD from the default registry
func Get(name string) (*CustomResourceDefinition, error) {
	return DefaultRegistry.Get(name)
}

// GetByGVK returns a CRD from the default registry by GVK
func GetByGVK(gvk GroupVersionKind) (*CustomResourceDefinition, error) {
	return DefaultRegistry.GetByGVK(gvk)
}
