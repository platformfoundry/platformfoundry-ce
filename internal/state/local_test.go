package state

import (
	"os"
	"testing"
)

func TestNewLocalBackend(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-state-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	backend, err := NewLocalBackend(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create local backend: %v", err)
	}
	defer backend.Close()

	if backend == nil {
		t.Fatal("NewLocalBackend returned nil")
	}
}

func TestLocalBackend_SaveAndGet(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-state-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	backend, err := NewLocalBackend(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create local backend: %v", err)
	}
	defer backend.Close()

	// Save a resource
	resource := &Resource{
		Name: "test-resource",
		Kind: "Cluster",
		Spec: map[string]interface{}{
			"provider": "existing",
		},
	}

	err = backend.Save(resource)
	if err != nil {
		t.Fatalf("Failed to save resource: %v", err)
	}

	// Get the resource
	retrieved, err := backend.Get("test-resource")
	if err != nil {
		t.Fatalf("Failed to get resource: %v", err)
	}

	if retrieved.Name != "test-resource" {
		t.Errorf("Expected name 'test-resource', got %s", retrieved.Name)
	}

	if retrieved.Kind != "Cluster" {
		t.Errorf("Expected kind 'Cluster', got %s", retrieved.Kind)
	}
}

func TestLocalBackend_List(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-state-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	backend, err := NewLocalBackend(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create local backend: %v", err)
	}
	defer backend.Close()

	// Save multiple resources
	for i := 1; i <= 3; i++ {
		resource := &Resource{
			Name: "resource" + string(rune('0'+i)),
			Kind: "Cluster",
			Spec: map[string]interface{}{},
		}
		backend.Save(resource)
	}

	// List resources
	resources, err := backend.List()
	if err != nil {
		t.Fatalf("Failed to list resources: %v", err)
	}

	if len(resources) != 3 {
		t.Errorf("Expected 3 resources, got %d", len(resources))
	}
}

func TestLocalBackend_Delete(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-state-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	backend, err := NewLocalBackend(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create local backend: %v", err)
	}
	defer backend.Close()

	// Save a resource
	resource := &Resource{
		Name: "test-delete",
		Kind: "Cluster",
		Spec: map[string]interface{}{},
	}
	backend.Save(resource)

	// Delete the resource
	err = backend.Delete("test-delete")
	if err != nil {
		t.Fatalf("Failed to delete resource: %v", err)
	}

	// Try to get the deleted resource
	_, err = backend.Get("test-delete")
	if err == nil {
		t.Error("Expected error when getting deleted resource, got nil")
	}
}

func TestLocalBackend_LockAndUnlock(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-state-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	backend, err := NewLocalBackend(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create local backend: %v", err)
	}
	defer backend.Close()

	// Lock a resource
	err = backend.Lock("test-resource")
	if err != nil {
		t.Fatalf("Failed to lock resource: %v", err)
	}

	// Unlock the resource
	err = backend.Unlock("test-resource")
	if err != nil {
		t.Fatalf("Failed to unlock resource: %v", err)
	}
}
