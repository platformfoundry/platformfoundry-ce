// Package configloader provides centralized configuration loading for Platform Foundry.
// This loader handles config file discovery, environment variable expansion,
// validation, and default values.
package configloader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Paths defines standard configuration paths
type Paths struct {
	// ConfigDir is the main configuration directory (~/.platformfoundry)
	ConfigDir string

	// LegacyConfigDir is the legacy config directory (~/.pf)
	LegacyConfigDir string

	// SystemConfigDir is the system-wide config directory (/etc/platformfoundry)
	SystemConfigDir string

	// ProjectConfigDir is the project-local config directory (./config)
	ProjectConfigDir string
}

// DefaultPaths returns the default configuration paths
func DefaultPaths() *Paths {
	home := os.Getenv("HOME")
	if home == "" {
		home = os.Getenv("USERPROFILE") // Windows fallback
	}

	return &Paths{
		ConfigDir:        filepath.Join(home, ".platformfoundry"),
		LegacyConfigDir:  filepath.Join(home, ".pf"),
		SystemConfigDir:  "/etc/platformfoundry",
		ProjectConfigDir: "config",
	}
}

// Loader provides centralized configuration loading
type Loader struct {
	paths     *Paths
	envPrefix string
}

// NewLoader creates a new configuration loader
func NewLoader() *Loader {
	return &Loader{
		paths:     DefaultPaths(),
		envPrefix: "PF_",
	}
}

// NewLoaderWithPaths creates a loader with custom paths
func NewLoaderWithPaths(paths *Paths) *Loader {
	return &Loader{
		paths:     paths,
		envPrefix: "PF_",
	}
}

// GetPaths returns the configuration paths
func (l *Loader) GetPaths() *Paths {
	return l.paths
}

// ConfigDir returns the main config directory, creating it if needed
func (l *Loader) ConfigDir() (string, error) {
	if err := os.MkdirAll(l.paths.ConfigDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}
	return l.paths.ConfigDir, nil
}

// FindConfigFile searches for a config file in standard locations
// Priority: project -> user -> legacy -> system
func (l *Loader) FindConfigFile(filename string) (string, bool) {
	searchPaths := []string{
		filepath.Join(l.paths.ProjectConfigDir, filename),
		filepath.Join(l.paths.ConfigDir, filename),
		filepath.Join(l.paths.LegacyConfigDir, filename),
		filepath.Join(l.paths.SystemConfigDir, filename),
	}

	for _, path := range searchPaths {
		if _, err := os.Stat(path); err == nil {
			return path, true
		}
	}

	return "", false
}

// Load loads configuration from a file into the target struct
// It handles environment variable expansion in the YAML content
func (l *Loader) Load(filename string, target interface{}) error {
	path, found := l.FindConfigFile(filename)
	if !found {
		return fmt.Errorf("config file not found: %s", filename)
	}

	return l.LoadFromPath(path, target)
}

// LoadFromPath loads configuration from a specific path
func (l *Loader) LoadFromPath(path string, target interface{}) error {
	// Expand environment variables in path
	path = os.ExpandEnv(path)

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	// Expand environment variables in content
	expanded := os.ExpandEnv(string(data))

	// Parse YAML
	if err := yaml.Unmarshal([]byte(expanded), target); err != nil {
		return fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	return nil
}

// LoadOrDefault loads configuration or returns a default-initialized struct
func (l *Loader) LoadOrDefault(filename string, target interface{}, defaults func(interface{})) error {
	if err := l.Load(filename, target); err != nil {
		if defaults != nil {
			defaults(target)
		}
		return nil
	}
	return nil
}

// Save saves configuration to a file in the config directory
func (l *Loader) Save(filename string, config interface{}) error {
	configDir, err := l.ConfigDir()
	if err != nil {
		return err
	}

	path := filepath.Join(configDir, filename)

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Use restrictive permissions for config files
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetEnv gets an environment variable with the loader's prefix
func (l *Loader) GetEnv(key string) string {
	return os.Getenv(l.envPrefix + strings.ToUpper(key))
}

// GetEnvOr gets an environment variable or returns a default
func (l *Loader) GetEnvOr(key, defaultValue string) string {
	if v := l.GetEnv(key); v != "" {
		return v
	}
	return defaultValue
}

// FilePath returns the full path for a file in the config directory
func (l *Loader) FilePath(filename string) string {
	return filepath.Join(l.paths.ConfigDir, filename)
}

// LegacyFilePath returns the full path for a file in the legacy config directory
func (l *Loader) LegacyFilePath(filename string) string {
	return filepath.Join(l.paths.LegacyConfigDir, filename)
}

// Exists checks if a config file exists in any standard location
func (l *Loader) Exists(filename string) bool {
	_, found := l.FindConfigFile(filename)
	return found
}

// MigrateFromLegacy migrates a config file from legacy to new location
func (l *Loader) MigrateFromLegacy(filename string) error {
	legacyPath := l.LegacyFilePath(filename)
	newPath := l.FilePath(filename)

	// Check if legacy exists and new doesn't
	if _, err := os.Stat(legacyPath); err != nil {
		return nil // No legacy file to migrate
	}
	if _, err := os.Stat(newPath); err == nil {
		return nil // New file already exists
	}

	// Ensure new config dir exists
	if _, err := l.ConfigDir(); err != nil {
		return err
	}

	// Copy legacy to new location
	data, err := os.ReadFile(legacyPath)
	if err != nil {
		return fmt.Errorf("failed to read legacy config: %w", err)
	}

	if err := os.WriteFile(newPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write migrated config: %w", err)
	}

	return nil
}

// Standard config filenames
const (
	ConfigFileClient   = "config.yaml"
	ConfigFileSecurity = "security.yaml"
	ConfigFileContext  = "context.json"
	ConfigFileAPIKeys  = "api_keys.json"
	ConfigFileCreds    = "credentials"
)

// Global loader instance
var globalLoader = NewLoader()

// Global returns the global configuration loader
func Global() *Loader {
	return globalLoader
}

// SetGlobal sets the global configuration loader
func SetGlobal(loader *Loader) {
	globalLoader = loader
}
