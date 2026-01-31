package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewExternalPluginLoader(t *testing.T) {
	loader := NewExternalPluginLoader("/tmp/plugins", "https://registry.example.com")

	if loader == nil {
		t.Fatal("NewExternalPluginLoader returned nil")
	}

	if loader.pluginDir != "/tmp/plugins" {
		t.Errorf("expected pluginDir to be /tmp/plugins, got %s", loader.pluginDir)
	}

	if loader.registry == nil {
		t.Error("expected registry to be initialized")
	}
}

func TestExternalPluginLoader_List(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "plugin-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create some test plugin files
	pluginFiles := []string{"plugin-a-1.0.0", "plugin-b-2.0.0"}
	for _, name := range pluginFiles {
		path := filepath.Join(tempDir, name)
		if err := os.WriteFile(path, []byte("binary"), 0755); err != nil {
			t.Fatalf("failed to create test plugin file: %v", err)
		}
	}

	loader := NewExternalPluginLoader(tempDir, "https://registry.example.com")

	plugins, err := loader.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(plugins) != len(pluginFiles) {
		t.Errorf("expected %d plugins, got %d", len(pluginFiles), len(plugins))
	}
}

func TestExternalPluginLoader_List_EmptyDir(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "plugin-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	loader := NewExternalPluginLoader(tempDir, "https://registry.example.com")

	plugins, err := loader.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(plugins) != 0 {
		t.Errorf("expected 0 plugins for empty directory, got %d", len(plugins))
	}
}

func TestExternalPluginLoader_List_NonExistentDir(t *testing.T) {
	loader := NewExternalPluginLoader("/nonexistent/path", "https://registry.example.com")

	plugins, err := loader.List()
	if err != nil {
		t.Fatalf("List should not fail for non-existent directory: %v", err)
	}

	if len(plugins) != 0 {
		t.Errorf("expected 0 plugins for non-existent directory, got %d", len(plugins))
	}
}

func TestExternalPluginLoader_Uninstall(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "plugin-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test plugin file
	pluginPath := filepath.Join(tempDir, "test-plugin-1.0.0")
	if err := os.WriteFile(pluginPath, []byte("binary"), 0755); err != nil {
		t.Fatalf("failed to create test plugin file: %v", err)
	}

	loader := NewExternalPluginLoader(tempDir, "https://registry.example.com")

	err = loader.Uninstall("test-plugin")
	if err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}

	// Verify plugin was removed
	if _, err := os.Stat(pluginPath); !os.IsNotExist(err) {
		t.Error("plugin file should have been removed")
	}
}

func TestExternalPluginLoader_Uninstall_NotFound(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "plugin-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	loader := NewExternalPluginLoader(tempDir, "https://registry.example.com")

	err = loader.Uninstall("nonexistent-plugin")
	if err == nil {
		t.Error("expected error when uninstalling non-existent plugin")
	}
}

func TestExternalPluginLoader_VerifyChecksum(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "plugin-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test file with known content
	testContent := []byte("test plugin content")
	testFile := filepath.Join(tempDir, "test-file")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	loader := NewExternalPluginLoader(tempDir, "https://registry.example.com")

	// Calculate expected checksum for "test plugin content"
	// SHA256("test plugin content") = 51a64168e1440fdec32c78359055f5d6877163972271fa4fb53cd1c5ba39d215
	expectedChecksum := "51a64168e1440fdec32c78359055f5d6877163972271fa4fb53cd1c5ba39d215"

	err = loader.verifyChecksum(testFile, expectedChecksum)
	if err != nil {
		t.Errorf("checksum verification failed: %v", err)
	}
}

func TestExternalPluginLoader_VerifyChecksum_Mismatch(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "plugin-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test-file")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	loader := NewExternalPluginLoader(tempDir, "https://registry.example.com")

	err = loader.verifyChecksum(testFile, "invalid-checksum")
	if err == nil {
		t.Error("expected checksum verification to fail with invalid checksum")
	}
}

func TestPluginManifest_SaveAndLoad(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "plugin-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	loader := NewExternalPluginLoader(tempDir, "https://registry.example.com")

	// Create a test manifest
	manifest := &PluginManifest{
		Plugins: []InstalledPlugin{
			{
				Name:      "test-plugin",
				Version:   "1.0.0",
				Kind:      "Infrastructure",
				Provider:  "terraform",
				Path:      filepath.Join(tempDir, "test-plugin-1.0.0"),
				Checksum:  "abc123",
				Verified:  true,
				Installed: time.Now(),
			},
		},
		UpdatedAt: time.Now(),
	}

	// Save manifest
	err = loader.saveManifest(manifest)
	if err != nil {
		t.Fatalf("failed to save manifest: %v", err)
	}

	// Load manifest
	loaded, err := loader.loadManifest()
	if err != nil {
		t.Fatalf("failed to load manifest: %v", err)
	}

	if len(loaded.Plugins) != 1 {
		t.Fatalf("expected 1 plugin in manifest, got %d", len(loaded.Plugins))
	}

	if loaded.Plugins[0].Name != "test-plugin" {
		t.Errorf("expected plugin name to be test-plugin, got %s", loaded.Plugins[0].Name)
	}

	if loaded.Plugins[0].Version != "1.0.0" {
		t.Errorf("expected plugin version to be 1.0.0, got %s", loaded.Plugins[0].Version)
	}
}

func TestExternalPluginLoader_RegisterPlugin(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "plugin-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	loader := NewExternalPluginLoader(tempDir, "https://registry.example.com")

	metadata := &PluginMetadata{
		Name:     "new-plugin",
		Version:  "2.0.0",
		Kind:     "Orchestrator",
		Provider: "argocd",
		Checksum: "def456",
		Verified: true,
	}

	pluginPath := filepath.Join(tempDir, "new-plugin-2.0.0")

	err = loader.registerPlugin("new-plugin", metadata, pluginPath)
	if err != nil {
		t.Fatalf("failed to register plugin: %v", err)
	}

	// Verify manifest was created
	manifest, err := loader.loadManifest()
	if err != nil {
		t.Fatalf("failed to load manifest: %v", err)
	}

	if len(manifest.Plugins) != 1 {
		t.Fatalf("expected 1 plugin in manifest, got %d", len(manifest.Plugins))
	}

	if manifest.Plugins[0].Name != "new-plugin" {
		t.Errorf("expected plugin name to be new-plugin, got %s", manifest.Plugins[0].Name)
	}
}

func TestExternalPluginLoader_RegisterPlugin_Update(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "plugin-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	loader := NewExternalPluginLoader(tempDir, "https://registry.example.com")

	// Register initial version
	metadata1 := &PluginMetadata{
		Name:    "test-plugin",
		Version: "1.0.0",
	}
	err = loader.registerPlugin("test-plugin", metadata1, filepath.Join(tempDir, "test-plugin-1.0.0"))
	if err != nil {
		t.Fatalf("failed to register plugin v1: %v", err)
	}

	// Register updated version
	metadata2 := &PluginMetadata{
		Name:    "test-plugin",
		Version: "2.0.0",
	}
	err = loader.registerPlugin("test-plugin", metadata2, filepath.Join(tempDir, "test-plugin-2.0.0"))
	if err != nil {
		t.Fatalf("failed to register plugin v2: %v", err)
	}

	// Verify only one entry exists (updated)
	manifest, err := loader.loadManifest()
	if err != nil {
		t.Fatalf("failed to load manifest: %v", err)
	}

	if len(manifest.Plugins) != 1 {
		t.Fatalf("expected 1 plugin in manifest after update, got %d", len(manifest.Plugins))
	}

	if manifest.Plugins[0].Version != "2.0.0" {
		t.Errorf("expected plugin version to be 2.0.0 after update, got %s", manifest.Plugins[0].Version)
	}
}

func TestExternalPluginLoader_GetInstalledPlugins(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "plugin-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	loader := NewExternalPluginLoader(tempDir, "https://registry.example.com")

	// Initially should return empty list
	plugins, err := loader.GetInstalledPlugins()
	if err != nil {
		t.Fatalf("GetInstalledPlugins failed: %v", err)
	}

	if len(plugins) != 0 {
		t.Errorf("expected 0 plugins initially, got %d", len(plugins))
	}

	// Register a plugin
	metadata := &PluginMetadata{
		Name:    "test-plugin",
		Version: "1.0.0",
	}
	err = loader.registerPlugin("test-plugin", metadata, filepath.Join(tempDir, "test-plugin-1.0.0"))
	if err != nil {
		t.Fatalf("failed to register plugin: %v", err)
	}

	// Should now return the registered plugin
	plugins, err = loader.GetInstalledPlugins()
	if err != nil {
		t.Fatalf("GetInstalledPlugins failed: %v", err)
	}

	if len(plugins) != 1 {
		t.Errorf("expected 1 plugin after registration, got %d", len(plugins))
	}
}

func TestPluginManifest_JSONMarshaling(t *testing.T) {
	manifest := &PluginManifest{
		Plugins: []InstalledPlugin{
			{
				Name:      "test-plugin",
				Version:   "1.0.0",
				Kind:      "Infrastructure",
				Provider:  "terraform",
				Path:      "/path/to/plugin",
				Checksum:  "abc123",
				Verified:  true,
				Installed: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		UpdatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	// Marshal to JSON
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("failed to marshal manifest: %v", err)
	}

	// Unmarshal back
	var loaded PluginManifest
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to unmarshal manifest: %v", err)
	}

	if len(loaded.Plugins) != 1 {
		t.Errorf("expected 1 plugin after round-trip, got %d", len(loaded.Plugins))
	}

	if loaded.Plugins[0].Name != "test-plugin" {
		t.Errorf("expected plugin name to be test-plugin, got %s", loaded.Plugins[0].Name)
	}
}
