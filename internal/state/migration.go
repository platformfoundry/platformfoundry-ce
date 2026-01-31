package state

import (
	"fmt"

	"gorm.io/gorm"
)

// MigrateToOrgModel migrates existing resources to org/env model
func MigrateToOrgModel(db *gorm.DB, defaultOrg string) error {
	// Check if migration is needed
	var count int64
	if err := db.Model(&resourceModel{}).Where("organization = '' OR organization IS NULL").Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check migration status: %w", err)
	}

	if count == 0 {
		return nil // Already migrated
	}

	fmt.Printf("Migrating %d resources to organization model...\n", count)

	// Update all resources without organization to default org
	result := db.Model(&resourceModel{}).
		Where("organization = '' OR organization IS NULL").
		Updates(map[string]interface{}{
			"organization": defaultOrg,
			"environment":  "", // Global scope
		})

	if result.Error != nil {
		return fmt.Errorf("migration failed: %w", result.Error)
	}

	fmt.Printf("Migration complete: %d resources assigned to organization '%s'\n",
		result.RowsAffected, defaultOrg)

	return nil
}

// CreateDefaultOrganization creates a default organization if none exists
func CreateDefaultOrganization(backend Backend) error {
	// Check if default org exists
	_, err := backend.Get("default")
	if err == nil {
		return nil // Already exists
	}

	// Create default organization
	defaultOrg := &Resource{
		Name:       "default",
		Kind:       "Organization",
		APIVersion: "platformfoundry.io/v1",
		Spec: map[string]interface{}{
			"displayName": "Default Organization",
			"description": "Default organization for migrated resources",
			"owner":       "admin",
		},
		Status: map[string]interface{}{
			"phase": "Ready",
		},
		Version: 1,
	}

	if err := backend.Save(defaultOrg); err != nil {
		return fmt.Errorf("failed to create default organization: %w", err)
	}

	fmt.Println("Created default organization")
	return nil
}
