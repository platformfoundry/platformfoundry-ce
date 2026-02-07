package configloader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewLoader(t *testing.T) {
	loader := NewLoader()
	if loader == nil {
		t.Fatal("NewLoader returned nil")
	}

	paths := loader.GetPaths()
	if paths == nil {
		t.Fatal("GetPaths returned nil")
	}

	if paths.ConfigDir == "" {
		t.Error("ConfigDir should not be empty")
	}
}

func TestNewLoaderWithPaths(t *testing.T) {
	customPaths := &Paths{
		ConfigDir:        "/custom/config",
		LegacyConfigDir:  "/custom/legacy",
		SystemConfigDir:  "/custom/system",
		ProjectConfigDir: "/custom/project",
	}

	loader := NewLoaderWithPaths(customPaths)
	paths := loader.GetPaths()

	if paths.ConfigDir != "/custom/config" {
		t.Errorf("Expected ConfigDir /custom/config, got %s", paths.ConfigDir)
	}
}

func TestDefaultPaths(t *testing.T) {
	paths := DefaultPaths()

	if paths.ConfigDir == "" {
		t.Error("ConfigDir should not be empty")
	}

	// Should contain .platformfoundry
	if !contains(paths.ConfigDir, ".platformfoundry") {
		t.Errorf("ConfigDir should contain .platformfoundry, got %s", paths.ConfigDir)
	}

	if !contains(paths.LegacyConfigDir, ".pf") {
		t.Errorf("LegacyConfigDir should contain .pf, got %s", paths.LegacyConfigDir)
	}
}

func TestConfigDir(t *testing.T) {
	tmpDir := t.TempDir()
	paths := &Paths{
		ConfigDir:        filepath.Join(tmpDir, ".platformfoundry"),
		LegacyConfigDir:  filepath.Join(tmpDir, ".pf"),
		SystemConfigDir:  filepath.Join(tmpDir, "etc"),
		ProjectConfigDir: filepath.Join(tmpDir, "config"),
	}

	loader := NewLoaderWithPaths(paths)
	configDir, err := loader.ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir failed: %v", err)
	}

	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Error("ConfigDir should create the directory")
	}
}

func TestFindConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	paths := &Paths{
		ConfigDir:        filepath.Join(tmpDir, ".platformfoundry"),
		LegacyConfigDir:  filepath.Join(tmpDir, ".pf"),
		SystemConfigDir:  filepath.Join(tmpDir, "etc"),
		ProjectConfigDir: filepath.Join(tmpDir, "config"),
	}

	// Create config directory and file
	os.MkdirAll(paths.ConfigDir, 0700)
	testFile := filepath.Join(paths.ConfigDir, "test.yaml")
	os.WriteFile(testFile, []byte("test: value"), 0600)

	loader := NewLoaderWithPaths(paths)

	// Should find the file
	foundPath, found := loader.FindConfigFile("test.yaml")
	if !found {
		t.Error("Expected to find test.yaml")
	}
	if foundPath != testFile {
		t.Errorf("Expected path %s, got %s", testFile, foundPath)
	}

	// Should not find non-existent file
	_, found = loader.FindConfigFile("nonexistent.yaml")
	if found {
		t.Error("Should not find nonexistent.yaml")
	}
}

func TestFindConfigFilePriority(t *testing.T) {
	tmpDir := t.TempDir()
	paths := &Paths{
		ConfigDir:        filepath.Join(tmpDir, ".platformfoundry"),
		LegacyConfigDir:  filepath.Join(tmpDir, ".pf"),
		SystemConfigDir:  filepath.Join(tmpDir, "etc"),
		ProjectConfigDir: filepath.Join(tmpDir, "config"),
	}

	// Create files in multiple locations
	os.MkdirAll(paths.ProjectConfigDir, 0755)
	os.MkdirAll(paths.ConfigDir, 0700)
	os.MkdirAll(paths.LegacyConfigDir, 0700)

	projectFile := filepath.Join(paths.ProjectConfigDir, "config.yaml")
	userFile := filepath.Join(paths.ConfigDir, "config.yaml")
	legacyFile := filepath.Join(paths.LegacyConfigDir, "config.yaml")

	os.WriteFile(projectFile, []byte("location: project"), 0644)
	os.WriteFile(userFile, []byte("location: user"), 0600)
	os.WriteFile(legacyFile, []byte("location: legacy"), 0600)

	loader := NewLoaderWithPaths(paths)

	// Should prefer project config
	foundPath, found := loader.FindConfigFile("config.yaml")
	if !found {
		t.Fatal("Expected to find config.yaml")
	}
	if foundPath != projectFile {
		t.Errorf("Expected project config %s, got %s", projectFile, foundPath)
	}

	// Remove project file, should find user config
	os.Remove(projectFile)
	foundPath, found = loader.FindConfigFile("config.yaml")
	if !found {
		t.Fatal("Expected to find config.yaml")
	}
	if foundPath != userFile {
		t.Errorf("Expected user config %s, got %s", userFile, foundPath)
	}
}

func TestLoadFromPath(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.yaml")

	content := `
name: test-platform
version: 1.0.0
settings:
  debug: true
`
	os.WriteFile(testFile, []byte(content), 0600)

	type TestConfig struct {
		Name     string `yaml:"name"`
		Version  string `yaml:"version"`
		Settings struct {
			Debug bool `yaml:"debug"`
		} `yaml:"settings"`
	}

	loader := NewLoader()
	var config TestConfig

	if err := loader.LoadFromPath(testFile, &config); err != nil {
		t.Fatalf("LoadFromPath failed: %v", err)
	}

	if config.Name != "test-platform" {
		t.Errorf("Expected name test-platform, got %s", config.Name)
	}
	if config.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", config.Version)
	}
	if !config.Settings.Debug {
		t.Error("Expected debug to be true")
	}
}

func TestLoadFromPathWithEnvExpansion(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.yaml")

	os.Setenv("TEST_NAME", "env-platform")
	defer os.Unsetenv("TEST_NAME")

	content := `
name: $TEST_NAME
`
	os.WriteFile(testFile, []byte(content), 0600)

	type TestConfig struct {
		Name string `yaml:"name"`
	}

	loader := NewLoader()
	var config TestConfig

	if err := loader.LoadFromPath(testFile, &config); err != nil {
		t.Fatalf("LoadFromPath failed: %v", err)
	}

	if config.Name != "env-platform" {
		t.Errorf("Expected name env-platform (from env), got %s", config.Name)
	}
}

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()
	paths := &Paths{
		ConfigDir:        tmpDir,
		LegacyConfigDir:  filepath.Join(tmpDir, ".pf"),
		SystemConfigDir:  filepath.Join(tmpDir, "etc"),
		ProjectConfigDir: filepath.Join(tmpDir, "config"),
	}

	loader := NewLoaderWithPaths(paths)

	type TestConfig struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
	}

	config := TestConfig{
		Name:    "test",
		Version: "1.0.0",
	}

	if err := loader.Save("saved.yaml", config); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists and has correct content
	savedPath := filepath.Join(tmpDir, "saved.yaml")
	data, err := os.ReadFile(savedPath)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}

	if !contains(string(data), "name: test") {
		t.Errorf("Expected saved file to contain 'name: test', got %s", string(data))
	}

	// Verify file exists (permissions vary by platform, skip on Windows)
	_, err = os.Stat(savedPath)
	if err != nil {
		t.Errorf("Expected file to exist: %v", err)
	}
}

func TestGetEnv(t *testing.T) {
	os.Setenv("PF_TEST_KEY", "test_value")
	defer os.Unsetenv("PF_TEST_KEY")

	loader := NewLoader()

	value := loader.GetEnv("test_key")
	if value != "test_value" {
		t.Errorf("Expected test_value, got %s", value)
	}

	// Non-existent key
	value = loader.GetEnv("nonexistent")
	if value != "" {
		t.Errorf("Expected empty string for nonexistent key, got %s", value)
	}
}

func TestGetEnvOr(t *testing.T) {
	os.Setenv("PF_EXISTING", "exists")
	defer os.Unsetenv("PF_EXISTING")

	loader := NewLoader()

	// Existing key
	value := loader.GetEnvOr("existing", "default")
	if value != "exists" {
		t.Errorf("Expected exists, got %s", value)
	}

	// Non-existent key
	value = loader.GetEnvOr("nonexistent", "default")
	if value != "default" {
		t.Errorf("Expected default, got %s", value)
	}
}

func TestFilePath(t *testing.T) {
	tmpDir := t.TempDir()
	paths := &Paths{
		ConfigDir:        tmpDir,
		LegacyConfigDir:  filepath.Join(tmpDir, ".pf"),
		SystemConfigDir:  filepath.Join(tmpDir, "etc"),
		ProjectConfigDir: filepath.Join(tmpDir, "config"),
	}

	loader := NewLoaderWithPaths(paths)

	path := loader.FilePath("test.yaml")
	expected := filepath.Join(tmpDir, "test.yaml")
	if path != expected {
		t.Errorf("Expected %s, got %s", expected, path)
	}
}

func TestLegacyFilePath(t *testing.T) {
	tmpDir := t.TempDir()
	legacyDir := filepath.Join(tmpDir, ".pf")
	paths := &Paths{
		ConfigDir:        tmpDir,
		LegacyConfigDir:  legacyDir,
		SystemConfigDir:  filepath.Join(tmpDir, "etc"),
		ProjectConfigDir: filepath.Join(tmpDir, "config"),
	}

	loader := NewLoaderWithPaths(paths)

	path := loader.LegacyFilePath("test.yaml")
	expected := filepath.Join(legacyDir, "test.yaml")
	if path != expected {
		t.Errorf("Expected %s, got %s", expected, path)
	}
}

func TestExists(t *testing.T) {
	tmpDir := t.TempDir()
	paths := &Paths{
		ConfigDir:        tmpDir,
		LegacyConfigDir:  filepath.Join(tmpDir, ".pf"),
		SystemConfigDir:  filepath.Join(tmpDir, "etc"),
		ProjectConfigDir: filepath.Join(tmpDir, "config"),
	}

	// Create test file
	testFile := filepath.Join(tmpDir, "exists.yaml")
	os.WriteFile(testFile, []byte("test"), 0600)

	loader := NewLoaderWithPaths(paths)

	if !loader.Exists("exists.yaml") {
		t.Error("Expected exists.yaml to exist")
	}

	if loader.Exists("nonexistent.yaml") {
		t.Error("Expected nonexistent.yaml to not exist")
	}
}

func TestMigrateFromLegacy(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".platformfoundry")
	legacyDir := filepath.Join(tmpDir, ".pf")
	os.MkdirAll(legacyDir, 0700)

	paths := &Paths{
		ConfigDir:        configDir,
		LegacyConfigDir:  legacyDir,
		SystemConfigDir:  filepath.Join(tmpDir, "etc"),
		ProjectConfigDir: filepath.Join(tmpDir, "config"),
	}

	// Create legacy file
	legacyFile := filepath.Join(legacyDir, "config.yaml")
	os.WriteFile(legacyFile, []byte("legacy: true"), 0600)

	loader := NewLoaderWithPaths(paths)

	if err := loader.MigrateFromLegacy("config.yaml"); err != nil {
		t.Fatalf("MigrateFromLegacy failed: %v", err)
	}

	// Verify new file exists
	newFile := filepath.Join(configDir, "config.yaml")
	data, err := os.ReadFile(newFile)
	if err != nil {
		t.Fatalf("Failed to read migrated file: %v", err)
	}

	if string(data) != "legacy: true" {
		t.Errorf("Expected migrated content 'legacy: true', got %s", string(data))
	}
}

func TestMigrateFromLegacy_NoLegacyFile(t *testing.T) {
	tmpDir := t.TempDir()
	paths := &Paths{
		ConfigDir:        filepath.Join(tmpDir, ".platformfoundry"),
		LegacyConfigDir:  filepath.Join(tmpDir, ".pf"),
		SystemConfigDir:  filepath.Join(tmpDir, "etc"),
		ProjectConfigDir: filepath.Join(tmpDir, "config"),
	}

	loader := NewLoaderWithPaths(paths)

	// Should not error when no legacy file exists
	if err := loader.MigrateFromLegacy("nonexistent.yaml"); err != nil {
		t.Errorf("Expected no error for non-existent legacy file, got %v", err)
	}
}

func TestMigrateFromLegacy_NewFileExists(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".platformfoundry")
	legacyDir := filepath.Join(tmpDir, ".pf")
	os.MkdirAll(configDir, 0700)
	os.MkdirAll(legacyDir, 0700)

	paths := &Paths{
		ConfigDir:        configDir,
		LegacyConfigDir:  legacyDir,
		SystemConfigDir:  filepath.Join(tmpDir, "etc"),
		ProjectConfigDir: filepath.Join(tmpDir, "config"),
	}

	// Create both files
	legacyFile := filepath.Join(legacyDir, "config.yaml")
	newFile := filepath.Join(configDir, "config.yaml")
	os.WriteFile(legacyFile, []byte("legacy: true"), 0600)
	os.WriteFile(newFile, []byte("new: true"), 0600)

	loader := NewLoaderWithPaths(paths)

	// Should not overwrite existing new file
	if err := loader.MigrateFromLegacy("config.yaml"); err != nil {
		t.Fatalf("MigrateFromLegacy failed: %v", err)
	}

	// New file should be unchanged
	data, _ := os.ReadFile(newFile)
	if string(data) != "new: true" {
		t.Errorf("Expected new file to be unchanged, got %s", string(data))
	}
}

func TestGlobal(t *testing.T) {
	loader := Global()
	if loader == nil {
		t.Fatal("Global returned nil")
	}

	// Should always return the same instance
	loader2 := Global()
	if loader != loader2 {
		t.Error("Global should return the same instance")
	}
}

func TestSetGlobal(t *testing.T) {
	original := Global()

	customPaths := &Paths{
		ConfigDir:        "/custom",
		LegacyConfigDir:  "/custom/legacy",
		SystemConfigDir:  "/custom/system",
		ProjectConfigDir: "/custom/project",
	}
	customLoader := NewLoaderWithPaths(customPaths)

	SetGlobal(customLoader)
	if Global() != customLoader {
		t.Error("SetGlobal did not update the global loader")
	}

	// Restore original
	SetGlobal(original)
}

func TestConfigFileConstants(t *testing.T) {
	if ConfigFileClient != "config.yaml" {
		t.Errorf("Expected ConfigFileClient to be config.yaml, got %s", ConfigFileClient)
	}
	if ConfigFileSecurity != "security.yaml" {
		t.Errorf("Expected ConfigFileSecurity to be security.yaml, got %s", ConfigFileSecurity)
	}
	if ConfigFileContext != "context.json" {
		t.Errorf("Expected ConfigFileContext to be context.json, got %s", ConfigFileContext)
	}
	if ConfigFileAPIKeys != "api_keys.json" {
		t.Errorf("Expected ConfigFileAPIKeys to be api_keys.json, got %s", ConfigFileAPIKeys)
	}
	if ConfigFileCreds != "credentials" {
		t.Errorf("Expected ConfigFileCreds to be credentials, got %s", ConfigFileCreds)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
