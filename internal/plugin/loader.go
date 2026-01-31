package plugin

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// ExternalPluginLoader manages external plugin loading
type ExternalPluginLoader struct {
	pluginDir string
	registry  *PluginRegistry
}

// NewExternalPluginLoader creates a new external plugin loader
func NewExternalPluginLoader(pluginDir string, registryURL string) *ExternalPluginLoader {
	return &ExternalPluginLoader{
		pluginDir: pluginDir,
		registry:  NewPluginRegistry(registryURL),
	}
}

// Install installs a plugin from the registry
func (l *ExternalPluginLoader) Install(pluginName string) error {
	// Get plugin metadata from registry
	metadata, err := l.registry.Get(pluginName)
	if err != nil {
		return fmt.Errorf("failed to get plugin metadata: %w", err)
	}

	// Create plugin directory if it doesn't exist
	if err := os.MkdirAll(l.pluginDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugin directory: %w", err)
	}

	// Download plugin
	destPath := filepath.Join(l.pluginDir, fmt.Sprintf("%s-%s", pluginName, metadata.Version))
	if err := l.registry.Download(metadata, destPath); err != nil {
		return fmt.Errorf("failed to download plugin: %w", err)
	}

	// Verify checksum
	if err := l.verifyChecksum(destPath, metadata.Checksum); err != nil {
		os.Remove(destPath)
		return fmt.Errorf("checksum verification failed: %w", err)
	}

	// Verify signature if provided
	if metadata.Signature != "" {
		if err := l.verifySignature(destPath, metadata.Signature); err != nil {
			os.Remove(destPath)
			return fmt.Errorf("signature verification failed: %w", err)
		}
	}

	// Make binary executable
	if err := os.Chmod(destPath, 0755); err != nil {
		os.Remove(destPath)
		return fmt.Errorf("failed to make plugin executable: %w", err)
	}

	// Register plugin in manifest
	if err := l.registerPlugin(pluginName, metadata, destPath); err != nil {
		os.Remove(destPath)
		return fmt.Errorf("failed to register plugin: %w", err)
	}

	fmt.Printf("Plugin %s installed successfully\n", pluginName)
	return nil
}

// Uninstall removes a plugin
func (l *ExternalPluginLoader) Uninstall(pluginName string) error {
	// Find plugin file
	pattern := filepath.Join(l.pluginDir, fmt.Sprintf("%s-*", pluginName))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to find plugin: %w", err)
	}

	if len(matches) == 0 {
		return fmt.Errorf("plugin %s not found", pluginName)
	}

	// Remove all versions
	for _, match := range matches {
		if err := os.Remove(match); err != nil {
			return fmt.Errorf("failed to remove plugin file: %w", err)
		}
	}

	fmt.Printf("Plugin %s uninstalled successfully\n", pluginName)
	return nil
}

// List lists installed plugins
func (l *ExternalPluginLoader) List() ([]string, error) {
	files, err := os.ReadDir(l.pluginDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read plugin directory: %w", err)
	}

	var plugins []string
	for _, file := range files {
		if !file.IsDir() {
			plugins = append(plugins, file.Name())
		}
	}

	return plugins, nil
}

// Upgrade upgrades a plugin to the latest version
func (l *ExternalPluginLoader) Upgrade(pluginName string) error {
	// Uninstall current version
	if err := l.Uninstall(pluginName); err != nil {
		return fmt.Errorf("failed to uninstall old version: %w", err)
	}

	// Install latest version
	if err := l.Install(pluginName); err != nil {
		return fmt.Errorf("failed to install new version: %w", err)
	}

	return nil
}

// verifyChecksum verifies the SHA256 checksum of a file
func (l *ExternalPluginLoader) verifyChecksum(filePath, expectedChecksum string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	actualChecksum := hex.EncodeToString(hash.Sum(nil))
	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return nil
}

// verifySignature verifies the signature of a plugin file
func (l *ExternalPluginLoader) verifySignature(filePath, signature string) error {
	// Read file data
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read plugin file: %w", err)
	}

	// Decode base64 signature
	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("invalid signature encoding: %w", err)
	}

	// Load trusted public key
	pubKey, err := l.loadTrustedPublicKey()
	if err != nil {
		// If no trusted key is configured, warn but allow installation
		fmt.Println("Warning: No trusted public key configured, skipping signature verification")
		return nil
	}

	// Compute hash of data
	hash := sha256.Sum256(data)

	// Verify ECDSA signature
	if !ecdsa.VerifyASN1(pubKey, hash[:], sigBytes) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

// loadTrustedPublicKey loads the trusted public key for plugin verification
func (l *ExternalPluginLoader) loadTrustedPublicKey() (*ecdsa.PublicKey, error) {
	// Check for public key file in standard locations
	keyPaths := []string{
		os.Getenv("PF_PLUGIN_PUBLIC_KEY"),
		filepath.Join(os.Getenv("HOME"), ".pf", "plugin-signing-key.pub"),
		"/etc/pf/plugin-signing-key.pub",
	}

	for _, keyPath := range keyPaths {
		if keyPath == "" {
			continue
		}

		data, err := os.ReadFile(keyPath)
		if err != nil {
			continue
		}

		// Parse PEM block
		block, _ := pem.Decode(data)
		if block == nil {
			continue
		}

		// Parse public key
		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			continue
		}

		ecdsaPub, ok := pub.(*ecdsa.PublicKey)
		if !ok {
			continue
		}

		return ecdsaPub, nil
	}

	return nil, fmt.Errorf("no trusted public key found")
}

// InstalledPlugin represents a registered plugin
type InstalledPlugin struct {
	Name      string    `json:"name"`
	Version   string    `json:"version"`
	Kind      string    `json:"kind"`
	Provider  string    `json:"provider"`
	Path      string    `json:"path"`
	Checksum  string    `json:"checksum"`
	Verified  bool      `json:"verified"`
	Installed time.Time `json:"installed"`
}

// PluginManifest represents the installed plugins manifest
type PluginManifest struct {
	Plugins   []InstalledPlugin `json:"plugins"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// registerPlugin registers a plugin in the manifest
func (l *ExternalPluginLoader) registerPlugin(name string, metadata *PluginMetadata, path string) error {
	manifest, err := l.loadManifest()
	if err != nil {
		// Create new manifest if it doesn't exist
		manifest = &PluginManifest{
			Plugins: []InstalledPlugin{},
		}
	}

	// Check if plugin already exists and update it
	found := false
	for i, p := range manifest.Plugins {
		if p.Name == name {
			manifest.Plugins[i] = InstalledPlugin{
				Name:      name,
				Version:   metadata.Version,
				Kind:      metadata.Kind,
				Provider:  metadata.Provider,
				Path:      path,
				Checksum:  metadata.Checksum,
				Verified:  metadata.Verified,
				Installed: time.Now(),
			}
			found = true
			break
		}
	}

	// Add new plugin if not found
	if !found {
		manifest.Plugins = append(manifest.Plugins, InstalledPlugin{
			Name:      name,
			Version:   metadata.Version,
			Kind:      metadata.Kind,
			Provider:  metadata.Provider,
			Path:      path,
			Checksum:  metadata.Checksum,
			Verified:  metadata.Verified,
			Installed: time.Now(),
		})
	}

	manifest.UpdatedAt = time.Now()

	return l.saveManifest(manifest)
}

// loadManifest loads the plugin manifest from disk
func (l *ExternalPluginLoader) loadManifest() (*PluginManifest, error) {
	manifestPath := filepath.Join(l.pluginDir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	var manifest PluginManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &manifest, nil
}

// saveManifest saves the plugin manifest to disk
func (l *ExternalPluginLoader) saveManifest(manifest *PluginManifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	manifestPath := filepath.Join(l.pluginDir, "manifest.json")
	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}

// GetInstalledPlugins returns all installed plugins from the manifest
func (l *ExternalPluginLoader) GetInstalledPlugins() ([]InstalledPlugin, error) {
	manifest, err := l.loadManifest()
	if err != nil {
		if os.IsNotExist(err) {
			return []InstalledPlugin{}, nil
		}
		return nil, err
	}
	return manifest.Plugins, nil
}

// Search searches for plugins in the registry
func (l *ExternalPluginLoader) Search(query string) ([]PluginMetadata, error) {
	return l.registry.Search(query)
}
