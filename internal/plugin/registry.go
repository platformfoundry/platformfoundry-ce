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
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// PluginRegistry represents the plugin marketplace registry
type PluginRegistry struct {
	baseURL string
	client  *http.Client
}

// PluginMetadata represents plugin information from the registry
type PluginMetadata struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Author      string   `json:"author"`
	Description string   `json:"description"`
	Kind        string   `json:"kind"`
	Provider    string   `json:"provider"`
	Downloads   int      `json:"downloads"`
	Rating      float64  `json:"rating"`
	Verified    bool     `json:"verified"`
	URL         string   `json:"url"`
	Checksum    string   `json:"checksum"`
	Signature   string   `json:"signature"`
	Tags        []string `json:"tags"`
}

// RegistryResponse represents the API response from the registry
type RegistryResponse struct {
	Plugins []PluginMetadata `json:"plugins"`
	Total   int              `json:"total"`
}

// NewPluginRegistry creates a new plugin registry client
func NewPluginRegistry(baseURL string) *PluginRegistry {
	return &PluginRegistry{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Search searches for plugins in the registry
func (r *PluginRegistry) Search(query string) ([]PluginMetadata, error) {
	url := fmt.Sprintf("%s/api/v1/plugins?q=%s", r.baseURL, query)

	resp, err := r.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to search registry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result RegistryResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Plugins, nil
}

// Get retrieves specific plugin metadata
func (r *PluginRegistry) Get(name string) (*PluginMetadata, error) {
	url := fmt.Sprintf("%s/api/v1/plugins/%s", r.baseURL, name)

	resp, err := r.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get plugin: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("plugin %s not found", name)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var plugin PluginMetadata
	if err := json.Unmarshal(body, &plugin); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &plugin, nil
}

// ListAll lists all available plugins
func (r *PluginRegistry) ListAll() ([]PluginMetadata, error) {
	url := fmt.Sprintf("%s/api/v1/plugins", r.baseURL)

	resp, err := r.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to list plugins: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result RegistryResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Plugins, nil
}

// GetByKind retrieves plugins for a specific resource kind
func (r *PluginRegistry) GetByKind(kind string) ([]PluginMetadata, error) {
	url := fmt.Sprintf("%s/api/v1/plugins?kind=%s", r.baseURL, kind)

	resp, err := r.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get plugins by kind: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result RegistryResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Plugins, nil
}

// Download downloads a plugin binary
func (r *PluginRegistry) Download(plugin *PluginMetadata, destPath string) error {
	resp, err := r.client.Get(plugin.URL)
	if err != nil {
		return fmt.Errorf("failed to download plugin: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Read response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read plugin data: %w", err)
	}

	// Verify checksum before writing
	if plugin.Checksum != "" {
		if err := verifyChecksum(data, plugin.Checksum); err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}
	}

	// Verify signature if provided
	if plugin.Signature != "" {
		if err := verifySignature(data, plugin.Signature); err != nil {
			return fmt.Errorf("signature verification failed: %w", err)
		}
	}

	// Write to destination path
	if err := os.WriteFile(destPath, data, 0755); err != nil {
		return fmt.Errorf("failed to write plugin to %s: %w", destPath, err)
	}

	return nil
}

// verifyChecksum verifies the SHA256 checksum of data
func verifyChecksum(data []byte, expectedChecksum string) error {
	hash := sha256.Sum256(data)
	actualChecksum := hex.EncodeToString(hash[:])
	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}
	return nil
}

// verifySignature verifies the signature of plugin data
// Signature format: base64-encoded ECDSA signature over SHA256 hash
func verifySignature(data []byte, signature string) error {
	// Decode base64 signature
	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("invalid signature encoding: %w", err)
	}

	// Load trusted public key from config
	pubKey, err := loadTrustedPublicKey()
	if err != nil {
		// If no trusted key is configured, skip signature verification with warning
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
func loadTrustedPublicKey() (*ecdsa.PublicKey, error) {
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
