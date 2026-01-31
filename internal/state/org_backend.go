package state

import (
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
)

// OrgFilteredBackend wraps Backend with organization filtering
type OrgFilteredBackend struct {
	backend      Backend
	organization string
	environment  string
}

// NewOrgFilteredBackend creates a backend filtered by organization and environment
func NewOrgFilteredBackend(backend Backend, org, env string) *OrgFilteredBackend {
	return &OrgFilteredBackend{
		backend:      backend,
		organization: org,
		environment:  env,
	}
}

// Save stores a resource with org/env context
func (o *OrgFilteredBackend) Save(resource *Resource) error {
	// Add org/env to resource if using local backend
	if lb, ok := o.backend.(*LocalBackend); ok {
		return o.saveWithContext(lb, resource)
	}
	return o.backend.Save(resource)
}

// saveWithContext saves with org/env context in local backend
func (o *OrgFilteredBackend) saveWithContext(lb *LocalBackend, resource *Resource) error {
	// Marshal spec and status
	specJSON, err := json.Marshal(resource.Spec)
	if err != nil {
		return fmt.Errorf("failed to marshal spec: %w", err)
	}

	statusJSON, err := json.Marshal(resource.Status)
	if err != nil {
		return fmt.Errorf("failed to marshal status: %w", err)
	}

	// Check if resource exists in this org/env context
	var existing resourceModel
	query := lb.db.Where("name = ?", resource.Name)
	if o.organization != "" {
		query = query.Where("organization = ?", o.organization)
	}
	if o.environment != "" {
		query = query.Where("environment = ?", o.environment)
	}

	result := query.First(&existing)

	now := resource.UpdatedAt
	if now.IsZero() {
		now = resource.CreatedAt
	}
	if now.IsZero() {
		// No timestamp provided, use current time
		now = lb.db.NowFunc()
	}

	if result.Error == gorm.ErrRecordNotFound {
		// Create new resource
		model := resourceModel{
			Name:         resource.Name,
			Kind:         resource.Kind,
			APIVersion:   resource.APIVersion,
			Spec:         string(specJSON),
			Status:       string(statusJSON),
			Version:      1,
			Organization: o.organization,
			Environment:  o.environment,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		if err := lb.db.Create(&model).Error; err != nil {
			return fmt.Errorf("failed to create resource: %w", err)
		}

		// Create version record
		version := versionModel{
			ResourceID: model.ID,
			Version:    1,
			Spec:       string(specJSON),
			Status:     string(statusJSON),
			CreatedAt:  now,
		}
		if err := lb.db.Create(&version).Error; err != nil {
			return fmt.Errorf("failed to create version: %w", err)
		}

		resource.Version = 1
		resource.CreatedAt = now
		resource.UpdatedAt = now
	} else if result.Error != nil {
		return fmt.Errorf("failed to query resource: %w", result.Error)
	} else {
		// Update existing resource
		existing.Kind = resource.Kind
		existing.APIVersion = resource.APIVersion
		existing.Spec = string(specJSON)
		existing.Status = string(statusJSON)
		existing.Version++
		existing.UpdatedAt = now

		if err := lb.db.Save(&existing).Error; err != nil {
			return fmt.Errorf("failed to update resource: %w", err)
		}

		// Create version record
		version := versionModel{
			ResourceID: existing.ID,
			Version:    existing.Version,
			Spec:       string(specJSON),
			Status:     string(statusJSON),
			CreatedAt:  now,
		}
		if err := lb.db.Create(&version).Error; err != nil {
			return fmt.Errorf("failed to create version: %w", err)
		}

		resource.Version = existing.Version
		resource.CreatedAt = existing.CreatedAt
		resource.UpdatedAt = now
	}

	return nil
}

// List returns resources filtered by org/env
func (o *OrgFilteredBackend) List() ([]*Resource, error) {
	if lb, ok := o.backend.(*LocalBackend); ok {
		return o.listFiltered(lb)
	}
	return o.backend.List()
}

// listFiltered lists resources filtered by org/env
func (o *OrgFilteredBackend) listFiltered(lb *LocalBackend) ([]*Resource, error) {
	var models []resourceModel

	query := lb.db
	if o.organization != "" {
		query = query.Where("organization = ?", o.organization)
	}
	if o.environment != "" {
		query = query.Where("environment = ?", o.environment)
	}

	if err := query.Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	resources := make([]*Resource, len(models))
	for i, model := range models {
		resource, err := lb.modelToResource(&model)
		if err != nil {
			return nil, err
		}
		resources[i] = resource
	}

	return resources, nil
}

// ListSharedPlatforms returns platforms shared with this organization
func (o *OrgFilteredBackend) ListSharedPlatforms() ([]*Resource, error) {
	if lb, ok := o.backend.(*LocalBackend); ok {
		var models []resourceModel

		// Query platforms where:
		// 1. Organization owns them, OR
		// 2. They are shared (we'll check sharing config in spec)
		query := lb.db.Where("kind = ?", "Platform")
		if o.organization != "" {
			query = query.Where("organization = ? OR organization = ''", o.organization)
		}

		if err := query.Find(&models).Error; err != nil {
			return nil, fmt.Errorf("failed to list shared platforms: %w", err)
		}

		resources := make([]*Resource, 0)
		for _, model := range models {
			resource, err := lb.modelToResource(&model)
			if err != nil {
				return nil, err
			}

			// Check if owned by this org or shared with this org
			if model.Organization == o.organization || o.isSharedWithOrg(resource) {
				resources = append(resources, resource)
			}
		}

		return resources, nil
	}

	return []*Resource{}, nil
}

// isSharedWithOrg checks if a resource is shared with the current organization
func (o *OrgFilteredBackend) isSharedWithOrg(resource *Resource) bool {
	// Check sharing configuration in metadata
	if metadata, ok := resource.Spec["metadata"].(map[string]interface{}); ok {
		if sharing, ok := metadata["sharing"].(map[string]interface{}); ok {
			if enabled, ok := sharing["enabled"].(bool); ok && enabled {
				if orgs, ok := sharing["organizations"].([]interface{}); ok {
					for _, org := range orgs {
						if orgMap, ok := org.(map[string]interface{}); ok {
							if name, ok := orgMap["name"].(string); ok && name == o.organization {
								return true
							}
						}
					}
				}
			}
		}
	}

	return false
}

// Get retrieves a resource by name in current org/env context
func (o *OrgFilteredBackend) Get(name string) (*Resource, error) {
	if lb, ok := o.backend.(*LocalBackend); ok {
		return o.getFiltered(lb, name)
	}
	return o.backend.Get(name)
}

// getFiltered gets resource with org/env filter
func (o *OrgFilteredBackend) getFiltered(lb *LocalBackend, name string) (*Resource, error) {
	var model resourceModel
	query := lb.db.Where("name = ?", name)

	if o.organization != "" {
		query = query.Where("organization = ?", o.organization)
	}
	if o.environment != "" {
		query = query.Where("environment = ?", o.environment)
	}

	if err := query.First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("resource not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get resource: %w", err)
	}

	return lb.modelToResource(&model)
}

// Delete removes a resource by name in current org/env context
func (o *OrgFilteredBackend) Delete(name string) error {
	if lb, ok := o.backend.(*LocalBackend); ok {
		return o.deleteFiltered(lb, name)
	}
	return o.backend.Delete(name)
}

// deleteFiltered deletes resource with org/env filter
func (o *OrgFilteredBackend) deleteFiltered(lb *LocalBackend, name string) error {
	var model resourceModel
	query := lb.db.Where("name = ?", name)

	if o.organization != "" {
		query = query.Where("organization = ?", o.organization)
	}
	if o.environment != "" {
		query = query.Where("environment = ?", o.environment)
	}

	if err := query.First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("resource not found: %s", name)
		}
		return fmt.Errorf("failed to find resource: %w", err)
	}

	// Delete versions
	if err := lb.db.Where("resource_id = ?", model.ID).Delete(&versionModel{}).Error; err != nil {
		return fmt.Errorf("failed to delete versions: %w", err)
	}

	// Delete resource
	if err := lb.db.Delete(&model).Error; err != nil {
		return fmt.Errorf("failed to delete resource: %w", err)
	}

	return nil
}

// Lock acquires a lock on a resource
func (o *OrgFilteredBackend) Lock(name string) error {
	return o.backend.Lock(name)
}

// Unlock releases a lock on a resource
func (o *OrgFilteredBackend) Unlock(name string) error {
	return o.backend.Unlock(name)
}

// GetVersion retrieves a specific version of a resource
func (o *OrgFilteredBackend) GetVersion(name string, version int) (*Resource, error) {
	if lb, ok := o.backend.(*LocalBackend); ok {
		return o.getVersionFiltered(lb, name, version)
	}
	return o.backend.GetVersion(name, version)
}

// getVersionFiltered gets a specific version with org/env filter
func (o *OrgFilteredBackend) getVersionFiltered(lb *LocalBackend, name string, version int) (*Resource, error) {
	var resource resourceModel
	query := lb.db.Where("name = ?", name)

	if o.organization != "" {
		query = query.Where("organization = ?", o.organization)
	}
	if o.environment != "" {
		query = query.Where("environment = ?", o.environment)
	}

	if err := query.First(&resource).Error; err != nil {
		return nil, fmt.Errorf("resource not found: %s", name)
	}

	var versionModel versionModel
	if err := lb.db.Where("resource_id = ? AND version = ?", resource.ID, version).First(&versionModel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("version not found: %d", version)
		}
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	var spec map[string]interface{}
	if err := json.Unmarshal([]byte(versionModel.Spec), &spec); err != nil {
		return nil, fmt.Errorf("failed to unmarshal spec: %w", err)
	}

	var status map[string]interface{}
	if versionModel.Status != "" {
		if err := json.Unmarshal([]byte(versionModel.Status), &status); err != nil {
			return nil, fmt.Errorf("failed to unmarshal status: %w", err)
		}
	}

	return &Resource{
		Name:       name,
		Kind:       resource.Kind,
		APIVersion: resource.APIVersion,
		Spec:       spec,
		Status:     status,
		Version:    versionModel.Version,
		CreatedAt:  versionModel.CreatedAt,
		UpdatedAt:  versionModel.CreatedAt,
	}, nil
}

// ListVersions returns all versions of a resource
func (o *OrgFilteredBackend) ListVersions(name string) ([]*ResourceVersion, error) {
	if lb, ok := o.backend.(*LocalBackend); ok {
		return o.listVersionsFiltered(lb, name)
	}
	return o.backend.ListVersions(name)
}

// listVersionsFiltered lists all versions with org/env filter
func (o *OrgFilteredBackend) listVersionsFiltered(lb *LocalBackend, name string) ([]*ResourceVersion, error) {
	var resource resourceModel
	query := lb.db.Where("name = ?", name)

	if o.organization != "" {
		query = query.Where("organization = ?", o.organization)
	}
	if o.environment != "" {
		query = query.Where("environment = ?", o.environment)
	}

	if err := query.First(&resource).Error; err != nil {
		return nil, fmt.Errorf("resource not found: %s", name)
	}

	var versions []versionModel
	if err := lb.db.Where("resource_id = ?", resource.ID).Order("version ASC").Find(&versions).Error; err != nil {
		return nil, fmt.Errorf("failed to list versions: %w", err)
	}

	result := make([]*ResourceVersion, len(versions))
	for i, v := range versions {
		var spec map[string]interface{}
		if err := json.Unmarshal([]byte(v.Spec), &spec); err != nil {
			return nil, fmt.Errorf("failed to unmarshal spec: %w", err)
		}

		var status map[string]interface{}
		if v.Status != "" {
			if err := json.Unmarshal([]byte(v.Status), &status); err != nil {
				return nil, fmt.Errorf("failed to unmarshal status: %w", err)
			}
		}

		result[i] = &ResourceVersion{
			Version:   v.Version,
			Spec:      spec,
			Status:    status,
			CreatedAt: v.CreatedAt,
		}
	}

	return result, nil
}

// Close closes the backend connection
func (o *OrgFilteredBackend) Close() error {
	return o.backend.Close()
}
