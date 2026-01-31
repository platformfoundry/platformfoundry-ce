package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// LocalBackend implements Backend using SQLite
// Implements US-3.2: Local State Backend
type LocalBackend struct {
	db    *gorm.DB
	locks map[string]*Lock
	mu    sync.RWMutex
}

// resourceModel represents a resource in the database
type resourceModel struct {
	ID           uint      `gorm:"primaryKey"`
	Name         string    `gorm:"uniqueIndex:idx_org_env_name;not null"`
	Kind         string    `gorm:"not null;index"`
	APIVersion   string    `gorm:"not null"`
	Spec         string    `gorm:"type:text;not null"`
	Status       string    `gorm:"type:text"`
	Version      int       `gorm:"not null"`
	Organization string    `gorm:"uniqueIndex:idx_org_env_name;index"`
	Environment  string    `gorm:"uniqueIndex:idx_org_env_name;index"`
	CreatedAt    time.Time `gorm:"not null"`
	UpdatedAt    time.Time `gorm:"not null"`
}

// versionModel represents a resource version in the database
type versionModel struct {
	ID         uint      `gorm:"primaryKey"`
	ResourceID uint      `gorm:"not null;index"`
	Version    int       `gorm:"not null"`
	Spec       string    `gorm:"type:text;not null"`
	Status     string    `gorm:"type:text"`
	CreatedAt  time.Time `gorm:"not null"`
}

// NewLocalBackend creates a new local backend using bbolt (pure Go, no CGO)
func NewLocalBackend(dbPath string) (Backend, error) {
	// For backwards compatibility, convert .db to .bbolt
	if filepath.Ext(dbPath) == ".db" {
		dbPath = dbPath[:len(dbPath)-3] + ".bbolt"
	}

	// Use bbolt backend (pure Go, no CGO required)
	return NewBboltBackend(dbPath)
}

// NewLocalBackendSQLite creates a new local SQLite backend (requires CGO)
// This is kept for backwards compatibility but not recommended for new installations
func NewLocalBackendSQLite(dbPath string) (*LocalBackend, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Open SQLite database
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Auto-migrate schemas
	if err := db.AutoMigrate(&resourceModel{}, &versionModel{}); err != nil {
		return nil, fmt.Errorf("failed to migrate schemas: %w", err)
	}

	return &LocalBackend{
		db:    db,
		locks: make(map[string]*Lock),
	}, nil
}

// Save stores a resource
func (lb *LocalBackend) Save(resource *Resource) error {
	// Marshal spec and status
	specJSON, err := json.Marshal(resource.Spec)
	if err != nil {
		return fmt.Errorf("failed to marshal spec: %w", err)
	}

	statusJSON, err := json.Marshal(resource.Status)
	if err != nil {
		return fmt.Errorf("failed to marshal status: %w", err)
	}

	// Check if resource exists
	var existing resourceModel
	result := lb.db.Where("name = ?", resource.Name).First(&existing)

	now := time.Now()
	if result.Error == gorm.ErrRecordNotFound {
		// Create new resource
		model := resourceModel{
			Name:       resource.Name,
			Kind:       resource.Kind,
			APIVersion: resource.APIVersion,
			Spec:       string(specJSON),
			Status:     string(statusJSON),
			Version:    1,
			CreatedAt:  now,
			UpdatedAt:  now,
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

// Get retrieves a resource by name
func (lb *LocalBackend) Get(name string) (*Resource, error) {
	var model resourceModel
	if err := lb.db.Where("name = ?", name).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("resource not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get resource: %w", err)
	}

	return lb.modelToResource(&model)
}

// List returns all resources
func (lb *LocalBackend) List() ([]*Resource, error) {
	var models []resourceModel
	if err := lb.db.Find(&models).Error; err != nil {
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

// Delete removes a resource by name
func (lb *LocalBackend) Delete(name string) error {
	var model resourceModel
	if err := lb.db.Where("name = ?", name).First(&model).Error; err != nil {
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
func (lb *LocalBackend) Lock(name string) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	// Check if already locked
	if lock, exists := lb.locks[name]; exists {
		if time.Now().Before(lock.ExpiresAt) {
			return fmt.Errorf("resource is locked by %s", lock.Owner)
		}
		// Lock expired, remove it
		delete(lb.locks, name)
	}

	// Acquire lock
	lock := &Lock{
		ResourceName: name,
		Owner:        "local",
		AcquiredAt:   time.Now(),
		ExpiresAt:    time.Now().Add(5 * time.Minute),
	}
	lb.locks[name] = lock

	return nil
}

// Unlock releases a lock on a resource
func (lb *LocalBackend) Unlock(name string) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	delete(lb.locks, name)
	return nil
}

// GetVersion retrieves a specific version of a resource
func (lb *LocalBackend) GetVersion(name string, version int) (*Resource, error) {
	var resource resourceModel
	if err := lb.db.Where("name = ?", name).First(&resource).Error; err != nil {
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
func (lb *LocalBackend) ListVersions(name string) ([]*ResourceVersion, error) {
	var resource resourceModel
	if err := lb.db.Where("name = ?", name).First(&resource).Error; err != nil {
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
func (lb *LocalBackend) Close() error {
	sqlDB, err := lb.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}
	return sqlDB.Close()
}

// DB returns the underlying GORM database instance (for migrations)
func (lb *LocalBackend) DB() *gorm.DB {
	return lb.db
}

// modelToResource converts a database model to a Resource
func (lb *LocalBackend) modelToResource(model *resourceModel) (*Resource, error) {
	var spec map[string]interface{}
	if err := json.Unmarshal([]byte(model.Spec), &spec); err != nil {
		return nil, fmt.Errorf("failed to unmarshal spec: %w", err)
	}

	var status map[string]interface{}
	if model.Status != "" {
		if err := json.Unmarshal([]byte(model.Status), &status); err != nil {
			return nil, fmt.Errorf("failed to unmarshal status: %w", err)
		}
	}

	return &Resource{
		Name:       model.Name,
		Kind:       model.Kind,
		APIVersion: model.APIVersion,
		Spec:       spec,
		Status:     status,
		Version:    model.Version,
		CreatedAt:  model.CreatedAt,
		UpdatedAt:  model.UpdatedAt,
	}, nil
}
