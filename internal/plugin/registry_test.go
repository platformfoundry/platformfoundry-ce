package plugin

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNewPluginRegistry(t *testing.T) {
	registry := NewPluginRegistry("https://registry.example.com")

	if registry == nil {
		t.Fatal("NewPluginRegistry returned nil")
	}

	if registry.baseURL != "https://registry.example.com" {
		t.Errorf("expected baseURL to be https://registry.example.com, got %s", registry.baseURL)
	}

	if registry.client == nil {
		t.Error("expected client to be initialized")
	}
}

func TestVerifyChecksum(t *testing.T) {
	tests := []struct {
		name             string
		data             []byte
		expectedChecksum string
		shouldPass       bool
	}{
		{
			name:             "valid checksum",
			data:             []byte("test data"),
			expectedChecksum: func() string {
				h := sha256.Sum256([]byte("test data"))
				return hex.EncodeToString(h[:])
			}(),
			shouldPass: true,
		},
		{
			name:             "invalid checksum",
			data:             []byte("test data"),
			expectedChecksum: "invalid-checksum",
			shouldPass:       false,
		},
		{
			name:             "empty data",
			data:             []byte{},
			expectedChecksum: func() string {
				h := sha256.Sum256([]byte{})
				return hex.EncodeToString(h[:])
			}(),
			shouldPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := verifyChecksum(tt.data, tt.expectedChecksum)
			if tt.shouldPass && err != nil {
				t.Errorf("expected checksum verification to pass, got error: %v", err)
			}
			if !tt.shouldPass && err == nil {
				t.Error("expected checksum verification to fail, but it passed")
			}
		})
	}
}

func TestPluginRegistry_Download(t *testing.T) {
	// Create a test server
	testData := []byte("plugin binary content")
	checksum := sha256.Sum256(testData)
	checksumStr := hex.EncodeToString(checksum[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(testData)
	}))
	defer server.Close()

	// Create temp directory for download
	tempDir, err := os.MkdirTemp("", "plugin-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	registry := NewPluginRegistry(server.URL)

	plugin := &PluginMetadata{
		Name:     "test-plugin",
		Version:  "1.0.0",
		URL:      server.URL + "/plugin.tar.gz",
		Checksum: checksumStr,
	}

	destPath := filepath.Join(tempDir, "test-plugin")
	err = registry.Download(plugin, destPath)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	// Verify file was written
	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}

	if string(data) != string(testData) {
		t.Errorf("downloaded content mismatch: expected %s, got %s", testData, data)
	}

	// Verify file was created with correct permissions
	info, err := os.Stat(destPath)
	if err != nil {
		t.Fatalf("failed to stat downloaded file: %v", err)
	}

	// On Windows, executable permissions work differently, so we only check on Unix
	if runtime.GOOS != "windows" {
		if info.Mode().Perm()&0100 == 0 {
			t.Error("expected file to be executable")
		}
	}
}

func TestPluginRegistry_Download_ChecksumMismatch(t *testing.T) {
	testData := []byte("plugin binary content")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(testData)
	}))
	defer server.Close()

	tempDir, err := os.MkdirTemp("", "plugin-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	registry := NewPluginRegistry(server.URL)

	plugin := &PluginMetadata{
		Name:     "test-plugin",
		Version:  "1.0.0",
		URL:      server.URL + "/plugin.tar.gz",
		Checksum: "invalid-checksum-that-wont-match",
	}

	destPath := filepath.Join(tempDir, "test-plugin")
	err = registry.Download(plugin, destPath)
	if err == nil {
		t.Error("expected download to fail with checksum mismatch")
	}

	// File should not exist after failed download
	if _, err := os.Stat(destPath); !os.IsNotExist(err) {
		t.Error("file should not exist after failed checksum verification")
	}
}

func TestPluginRegistry_Download_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	tempDir, err := os.MkdirTemp("", "plugin-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	registry := NewPluginRegistry(server.URL)

	plugin := &PluginMetadata{
		Name:    "test-plugin",
		Version: "1.0.0",
		URL:     server.URL + "/plugin.tar.gz",
	}

	destPath := filepath.Join(tempDir, "test-plugin")
	err = registry.Download(plugin, destPath)
	if err == nil {
		t.Error("expected download to fail with server error")
	}
}

func TestLoadTrustedPublicKey_NoKeyFound(t *testing.T) {
	// Ensure no key files exist in standard locations for this test
	originalEnv := os.Getenv("PF_PLUGIN_PUBLIC_KEY")
	os.Setenv("PF_PLUGIN_PUBLIC_KEY", "")
	defer os.Setenv("PF_PLUGIN_PUBLIC_KEY", originalEnv)

	_, err := loadTrustedPublicKey()
	if err == nil {
		t.Error("expected error when no public key is found")
	}
}
