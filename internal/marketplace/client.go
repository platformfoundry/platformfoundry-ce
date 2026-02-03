package marketplace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// Client provides access to the plugin marketplace
type Client struct {
	baseURL    string
	httpClient *http.Client
	cache      *pluginCache
	installed  map[string]*InstalledPlugin
	pluginDir  string
	mu         sync.RWMutex
}

// ClientConfig configures the marketplace client
type ClientConfig struct {
	BaseURL   string
	Timeout   time.Duration
	PluginDir string
	CacheTTL  time.Duration
}

// pluginCache caches marketplace data
type pluginCache struct {
	plugins    []Plugin
	categories []Category
	updatedAt  time.Time
	ttl        time.Duration
	mu         sync.RWMutex
}

// NewClient creates a new marketplace client
func NewClient(config ClientConfig) *Client {
	if config.BaseURL == "" {
		config.BaseURL = "https://marketplace.platformfoundry.io/api/v1"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.PluginDir == "" {
		home, _ := os.UserHomeDir()
		config.PluginDir = filepath.Join(home, ".pf", "plugins")
	}
	if config.CacheTTL == 0 {
		config.CacheTTL = 15 * time.Minute
	}

	c := &Client{
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		cache: &pluginCache{
			ttl: config.CacheTTL,
		},
		installed: make(map[string]*InstalledPlugin),
		pluginDir: config.PluginDir,
	}

	// Initialize with sample data for demo
	c.initializeSampleData()
	c.loadInstalledPlugins()

	return c
}

// initializeSampleData loads sample marketplace data
func (c *Client) initializeSampleData() {
	c.cache.plugins = []Plugin{
		{
			Name:        "terraform-provider",
			Version:     "2.1.0",
			Description: "Terraform infrastructure provisioning plugin",
			Author:      Author{Name: "PlatformFoundry", Verified: true},
			License:     "Apache-2.0",
			Keywords:    []string{"infrastructure", "terraform", "iac"},
			Categories:  []string{"infrastructure", "provisioning"},
			Verified:    true,
			Downloads:   15420,
			Rating:      4.8,
			RatingCount: 234,
			PublishedAt: time.Now().Add(-30 * 24 * time.Hour),
			UpdatedAt:   time.Now().Add(-7 * 24 * time.Hour),
			Capabilities: []string{"provision", "plan", "apply"},
		},
		{
			Name:        "argocd-integration",
			Version:     "1.5.2",
			Description: "ArgoCD GitOps integration for deployments",
			Author:      Author{Name: "PlatformFoundry", Verified: true},
			License:     "Apache-2.0",
			Keywords:    []string{"gitops", "argocd", "kubernetes"},
			Categories:  []string{"gitops", "deployment"},
			Verified:    true,
			Downloads:   12350,
			Rating:      4.7,
			RatingCount: 189,
			PublishedAt: time.Now().Add(-60 * 24 * time.Hour),
			UpdatedAt:   time.Now().Add(-14 * 24 * time.Hour),
			Capabilities: []string{"sync", "deploy", "rollback"},
		},
		{
			Name:        "prometheus-metrics",
			Version:     "3.0.1",
			Description: "Prometheus metrics collection and querying",
			Author:      Author{Name: "PlatformFoundry", Verified: true},
			License:     "Apache-2.0",
			Keywords:    []string{"monitoring", "prometheus", "metrics"},
			Categories:  []string{"observability", "monitoring"},
			Verified:    true,
			Downloads:   18900,
			Rating:      4.9,
			RatingCount: 312,
			PublishedAt: time.Now().Add(-90 * 24 * time.Hour),
			UpdatedAt:   time.Now().Add(-3 * 24 * time.Hour),
			Capabilities: []string{"query", "alert", "dashboard"},
		},
		{
			Name:        "vault-secrets",
			Version:     "2.0.0",
			Description: "HashiCorp Vault secrets management",
			Author:      Author{Name: "PlatformFoundry", Verified: true},
			License:     "Apache-2.0",
			Keywords:    []string{"secrets", "vault", "security"},
			Categories:  []string{"security", "secrets"},
			Verified:    true,
			Downloads:   9800,
			Rating:      4.6,
			RatingCount: 156,
			PublishedAt: time.Now().Add(-45 * 24 * time.Hour),
			UpdatedAt:   time.Now().Add(-10 * 24 * time.Hour),
			Capabilities: []string{"read", "write", "rotate"},
		},
		{
			Name:        "datadog-integration",
			Version:     "1.2.0",
			Description: "Datadog monitoring and APM integration",
			Author:      Author{Name: "Community", Verified: false},
			License:     "MIT",
			Keywords:    []string{"monitoring", "datadog", "apm"},
			Categories:  []string{"observability", "monitoring"},
			Verified:    false,
			Downloads:   5600,
			Rating:      4.3,
			RatingCount: 87,
			PublishedAt: time.Now().Add(-120 * 24 * time.Hour),
			UpdatedAt:   time.Now().Add(-30 * 24 * time.Hour),
			Capabilities: []string{"metrics", "traces", "logs"},
		},
		{
			Name:        "slack-notifications",
			Version:     "1.0.5",
			Description: "Slack notifications for platform events",
			Author:      Author{Name: "PlatformFoundry", Verified: true},
			License:     "Apache-2.0",
			Keywords:    []string{"notifications", "slack", "alerts"},
			Categories:  []string{"notifications", "integration"},
			Verified:    true,
			Downloads:   8200,
			Rating:      4.5,
			RatingCount: 143,
			PublishedAt: time.Now().Add(-75 * 24 * time.Hour),
			UpdatedAt:   time.Now().Add(-5 * 24 * time.Hour),
			Capabilities: []string{"notify", "alert"},
		},
		{
			Name:        "aws-provider",
			Version:     "3.2.1",
			Description: "AWS cloud resource provisioning",
			Author:      Author{Name: "PlatformFoundry", Verified: true},
			License:     "Apache-2.0",
			Keywords:    []string{"aws", "cloud", "infrastructure"},
			Categories:  []string{"cloud", "infrastructure"},
			Verified:    true,
			Downloads:   21500,
			Rating:      4.8,
			RatingCount: 398,
			PublishedAt: time.Now().Add(-180 * 24 * time.Hour),
			UpdatedAt:   time.Now().Add(-2 * 24 * time.Hour),
			Capabilities: []string{"provision", "manage"},
		},
		{
			Name:        "gcp-provider",
			Version:     "2.8.0",
			Description: "Google Cloud Platform resource provisioning",
			Author:      Author{Name: "PlatformFoundry", Verified: true},
			License:     "Apache-2.0",
			Keywords:    []string{"gcp", "google", "cloud"},
			Categories:  []string{"cloud", "infrastructure"},
			Verified:    true,
			Downloads:   14200,
			Rating:      4.7,
			RatingCount: 256,
			PublishedAt: time.Now().Add(-150 * 24 * time.Hour),
			UpdatedAt:   time.Now().Add(-8 * 24 * time.Hour),
			Capabilities: []string{"provision", "manage"},
		},
		{
			Name:        "azure-provider",
			Version:     "2.5.3",
			Description: "Microsoft Azure resource provisioning",
			Author:      Author{Name: "PlatformFoundry", Verified: true},
			License:     "Apache-2.0",
			Keywords:    []string{"azure", "microsoft", "cloud"},
			Categories:  []string{"cloud", "infrastructure"},
			Verified:    true,
			Downloads:   11800,
			Rating:      4.6,
			RatingCount: 198,
			PublishedAt: time.Now().Add(-135 * 24 * time.Hour),
			UpdatedAt:   time.Now().Add(-12 * 24 * time.Hour),
			Capabilities: []string{"provision", "manage"},
		},
		{
			Name:        "backstage-catalog",
			Version:     "1.1.0",
			Description: "Backstage service catalog integration",
			Author:      Author{Name: "Community", Verified: false},
			License:     "Apache-2.0",
			Keywords:    []string{"backstage", "catalog", "portal"},
			Categories:  []string{"catalog", "portal"},
			Verified:    false,
			Downloads:   4300,
			Rating:      4.2,
			RatingCount: 67,
			PublishedAt: time.Now().Add(-60 * 24 * time.Hour),
			UpdatedAt:   time.Now().Add(-20 * 24 * time.Hour),
			Capabilities: []string{"sync", "discover"},
		},
	}

	c.cache.categories = []Category{
		{Name: "infrastructure", Description: "Infrastructure provisioning plugins", Count: 4},
		{Name: "observability", Description: "Monitoring and observability plugins", Count: 3},
		{Name: "security", Description: "Security and secrets management", Count: 2},
		{Name: "gitops", Description: "GitOps and deployment plugins", Count: 2},
		{Name: "cloud", Description: "Cloud provider plugins", Count: 3},
		{Name: "notifications", Description: "Notification integrations", Count: 1},
		{Name: "catalog", Description: "Service catalog plugins", Count: 1},
	}

	c.cache.updatedAt = time.Now()
}

// loadInstalledPlugins loads locally installed plugins
func (c *Client) loadInstalledPlugins() {
	// Load installed plugins from disk
	installedFile := filepath.Join(c.pluginDir, "installed.json")
	data, err := os.ReadFile(installedFile)
	if err != nil {
		return
	}

	var installed map[string]*InstalledPlugin
	if err := json.Unmarshal(data, &installed); err == nil {
		c.installed = installed
	}
}

// saveInstalledPlugins saves installed plugins list
func (c *Client) saveInstalledPlugins() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if err := os.MkdirAll(c.pluginDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c.installed, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(c.pluginDir, "installed.json"), data, 0644)
}

// Search searches for plugins in the marketplace
func (c *Client) Search(ctx context.Context, query SearchQuery) (*SearchResult, error) {
	c.cache.mu.RLock()
	plugins := c.cache.plugins
	c.cache.mu.RUnlock()

	// Filter plugins
	var filtered []Plugin
	for _, p := range plugins {
		if matchesQuery(p, query) {
			filtered = append(filtered, p)
		}
	}

	// Sort
	sortPlugins(filtered, query.SortBy, query.SortOrder)

	// Pagination
	total := len(filtered)
	offset := query.Offset
	limit := query.Limit
	if limit == 0 {
		limit = 20
	}
	if offset >= len(filtered) {
		filtered = []Plugin{}
	} else {
		end := offset + limit
		if end > len(filtered) {
			end = len(filtered)
		}
		filtered = filtered[offset:end]
	}

	return &SearchResult{
		Plugins: filtered,
		Total:   total,
		Query:   query.Query,
	}, nil
}

// GetPlugin retrieves a plugin by name
func (c *Client) GetPlugin(ctx context.Context, name string) (*Plugin, error) {
	c.cache.mu.RLock()
	defer c.cache.mu.RUnlock()

	for _, p := range c.cache.plugins {
		if p.Name == name {
			return &p, nil
		}
	}

	return nil, fmt.Errorf("plugin not found: %s", name)
}

// GetCategories returns all plugin categories
func (c *Client) GetCategories(ctx context.Context) ([]Category, error) {
	c.cache.mu.RLock()
	defer c.cache.mu.RUnlock()

	return c.cache.categories, nil
}

// Install installs a plugin
func (c *Client) Install(ctx context.Context, name, version string) error {
	plugin, err := c.GetPlugin(ctx, name)
	if err != nil {
		return err
	}

	if version == "" {
		version = plugin.Version
	}

	// Create plugin directory
	pluginPath := filepath.Join(c.pluginDir, name)
	if err := os.MkdirAll(pluginPath, 0755); err != nil {
		return fmt.Errorf("failed to create plugin directory: %w", err)
	}

	// Download plugin (simulated)
	fmt.Printf("Downloading %s@%s...\n", name, version)

	// Record installation
	c.mu.Lock()
	c.installed[name] = &InstalledPlugin{
		Name:        name,
		Version:     version,
		Enabled:     true,
		InstalledAt: time.Now(),
		UpdatedAt:   time.Now(),
		AutoUpdate:  true,
	}
	c.mu.Unlock()

	return c.saveInstalledPlugins()
}

// Uninstall removes a plugin
func (c *Client) Uninstall(ctx context.Context, name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.installed[name]; !exists {
		return fmt.Errorf("plugin not installed: %s", name)
	}

	// Remove plugin directory
	pluginPath := filepath.Join(c.pluginDir, name)
	if err := os.RemoveAll(pluginPath); err != nil {
		return fmt.Errorf("failed to remove plugin: %w", err)
	}

	delete(c.installed, name)
	return c.saveInstalledPlugins()
}

// Enable enables a plugin
func (c *Client) Enable(ctx context.Context, name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	installed, exists := c.installed[name]
	if !exists {
		return fmt.Errorf("plugin not installed: %s", name)
	}

	installed.Enabled = true
	installed.UpdatedAt = time.Now()
	return c.saveInstalledPlugins()
}

// Disable disables a plugin
func (c *Client) Disable(ctx context.Context, name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	installed, exists := c.installed[name]
	if !exists {
		return fmt.Errorf("plugin not installed: %s", name)
	}

	installed.Enabled = false
	installed.UpdatedAt = time.Now()
	return c.saveInstalledPlugins()
}

// ListInstalled returns all installed plugins
func (c *Client) ListInstalled(ctx context.Context) ([]*InstalledPlugin, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]*InstalledPlugin, 0, len(c.installed))
	for _, p := range c.installed {
		// Check for updates
		if marketPlugin, err := c.GetPlugin(ctx, p.Name); err == nil {
			if marketPlugin.Version != p.Version {
				p.LatestVersion = marketPlugin.Version
				p.HasUpdate = true
			}
		}
		result = append(result, p)
	}

	return result, nil
}

// Update updates a plugin to the latest version
func (c *Client) Update(ctx context.Context, name string) error {
	plugin, err := c.GetPlugin(ctx, name)
	if err != nil {
		return err
	}

	c.mu.Lock()
	installed, exists := c.installed[name]
	if !exists {
		c.mu.Unlock()
		return fmt.Errorf("plugin not installed: %s", name)
	}

	installed.Version = plugin.Version
	installed.UpdatedAt = time.Now()
	installed.HasUpdate = false
	installed.LatestVersion = ""
	c.mu.Unlock()

	return c.saveInstalledPlugins()
}

// CheckUpdates checks for plugin updates
func (c *Client) CheckUpdates(ctx context.Context) (map[string]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	updates := make(map[string]string)
	for name, installed := range c.installed {
		if plugin, err := c.GetPlugin(ctx, name); err == nil {
			if plugin.Version != installed.Version {
				updates[name] = plugin.Version
			}
		}
	}

	return updates, nil
}

// VerifyChecksum verifies a plugin's checksum
func (c *Client) VerifyChecksum(filePath, expected string) (bool, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false, err
	}

	actual := hex.EncodeToString(h.Sum(nil))
	return actual == expected, nil
}

// GetPlatformAsset returns the appropriate asset for the current platform
func (c *Client) GetPlatformAsset(plugin *Plugin) *Asset {
	platform := runtime.GOOS
	arch := runtime.GOARCH

	for _, asset := range plugin.Assets {
		if asset.Platform == platform && asset.Arch == arch {
			return &asset
		}
	}

	return nil
}

// Helper functions

func matchesQuery(plugin Plugin, query SearchQuery) bool {
	// Text search
	if query.Query != "" {
		q := strings.ToLower(query.Query)
		if !strings.Contains(strings.ToLower(plugin.Name), q) &&
			!strings.Contains(strings.ToLower(plugin.Description), q) {
			matched := false
			for _, kw := range plugin.Keywords {
				if strings.Contains(strings.ToLower(kw), q) {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}
		}
	}

	// Category filter
	if len(query.Categories) > 0 {
		matched := false
		for _, cat := range query.Categories {
			for _, pluginCat := range plugin.Categories {
				if cat == pluginCat {
					matched = true
					break
				}
			}
		}
		if !matched {
			return false
		}
	}

	// Verified filter
	if query.Verified != nil && *query.Verified != plugin.Verified {
		return false
	}

	// Author filter
	if query.Author != "" && !strings.EqualFold(plugin.Author.Name, query.Author) {
		return false
	}

	return true
}

func sortPlugins(plugins []Plugin, sortBy, sortOrder string) {
	if sortBy == "" {
		sortBy = "downloads"
	}

	sort.Slice(plugins, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "downloads":
			less = plugins[i].Downloads > plugins[j].Downloads
		case "rating":
			less = plugins[i].Rating > plugins[j].Rating
		case "updated":
			less = plugins[i].UpdatedAt.After(plugins[j].UpdatedAt)
		case "name":
			less = plugins[i].Name < plugins[j].Name
		default:
			less = plugins[i].Downloads > plugins[j].Downloads
		}

		if sortOrder == "asc" {
			return !less
		}
		return less
	})
}

// buildURL builds a URL with query parameters
func buildURL(base string, params map[string]string) string {
	u, _ := url.Parse(base)
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String()
}
