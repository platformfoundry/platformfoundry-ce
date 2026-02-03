package marketplace

import (
	"time"
)

// Plugin represents a plugin in the marketplace
type Plugin struct {
	Name        string            `json:"name" yaml:"name"`
	Version     string            `json:"version" yaml:"version"`
	Description string            `json:"description" yaml:"description"`
	Author      Author            `json:"author" yaml:"author"`
	License     string            `json:"license" yaml:"license"`
	Homepage    string            `json:"homepage,omitempty" yaml:"homepage,omitempty"`
	Repository  string            `json:"repository,omitempty" yaml:"repository,omitempty"`
	Keywords    []string          `json:"keywords,omitempty" yaml:"keywords,omitempty"`
	Categories  []string          `json:"categories,omitempty" yaml:"categories,omitempty"`
	Platforms   []string          `json:"platforms,omitempty" yaml:"platforms,omitempty"`
	MinVersion  string            `json:"minVersion,omitempty" yaml:"minVersion,omitempty"`
	Verified    bool              `json:"verified" yaml:"verified"`
	Downloads   int64             `json:"downloads" yaml:"downloads"`
	Rating      float64           `json:"rating" yaml:"rating"`
	RatingCount int               `json:"ratingCount" yaml:"ratingCount"`
	PublishedAt time.Time         `json:"publishedAt" yaml:"publishedAt"`
	UpdatedAt   time.Time         `json:"updatedAt" yaml:"updatedAt"`
	Readme      string            `json:"readme,omitempty" yaml:"readme,omitempty"`
	Changelog   string            `json:"changelog,omitempty" yaml:"changelog,omitempty"`
	Assets      []Asset           `json:"assets,omitempty" yaml:"assets,omitempty"`
	Dependencies []Dependency     `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
	Capabilities []string         `json:"capabilities,omitempty" yaml:"capabilities,omitempty"`
	Config       PluginConfig     `json:"config,omitempty" yaml:"config,omitempty"`
}

// Author represents plugin author information
type Author struct {
	Name     string `json:"name" yaml:"name"`
	Email    string `json:"email,omitempty" yaml:"email,omitempty"`
	URL      string `json:"url,omitempty" yaml:"url,omitempty"`
	Verified bool   `json:"verified" yaml:"verified"`
}

// Asset represents a downloadable plugin asset
type Asset struct {
	Name        string `json:"name" yaml:"name"`
	Platform    string `json:"platform" yaml:"platform"`
	Arch        string `json:"arch" yaml:"arch"`
	URL         string `json:"url" yaml:"url"`
	Checksum    string `json:"checksum" yaml:"checksum"`
	Size        int64  `json:"size" yaml:"size"`
}

// Dependency represents a plugin dependency
type Dependency struct {
	Name    string `json:"name" yaml:"name"`
	Version string `json:"version" yaml:"version"`
}

// PluginConfig represents plugin configuration schema
type PluginConfig struct {
	Schema     map[string]ConfigField `json:"schema,omitempty" yaml:"schema,omitempty"`
	Defaults   map[string]interface{} `json:"defaults,omitempty" yaml:"defaults,omitempty"`
}

// ConfigField represents a configuration field
type ConfigField struct {
	Type        string      `json:"type" yaml:"type"`
	Description string      `json:"description,omitempty" yaml:"description,omitempty"`
	Default     interface{} `json:"default,omitempty" yaml:"default,omitempty"`
	Required    bool        `json:"required,omitempty" yaml:"required,omitempty"`
	Enum        []string    `json:"enum,omitempty" yaml:"enum,omitempty"`
}

// PluginVersion represents a specific version of a plugin
type PluginVersion struct {
	Version     string    `json:"version" yaml:"version"`
	Changelog   string    `json:"changelog,omitempty" yaml:"changelog,omitempty"`
	MinVersion  string    `json:"minVersion,omitempty" yaml:"minVersion,omitempty"`
	PublishedAt time.Time `json:"publishedAt" yaml:"publishedAt"`
	Deprecated  bool      `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	Assets      []Asset   `json:"assets,omitempty" yaml:"assets,omitempty"`
}

// SearchQuery represents a marketplace search query
type SearchQuery struct {
	Query      string   `json:"query,omitempty"`
	Categories []string `json:"categories,omitempty"`
	Keywords   []string `json:"keywords,omitempty"`
	Author     string   `json:"author,omitempty"`
	Verified   *bool    `json:"verified,omitempty"`
	SortBy     string   `json:"sortBy,omitempty"` // downloads, rating, updated, name
	SortOrder  string   `json:"sortOrder,omitempty"` // asc, desc
	Limit      int      `json:"limit,omitempty"`
	Offset     int      `json:"offset,omitempty"`
}

// SearchResult represents marketplace search results
type SearchResult struct {
	Plugins    []Plugin `json:"plugins"`
	Total      int      `json:"total"`
	Query      string   `json:"query"`
	Categories []string `json:"categories,omitempty"`
}

// Review represents a plugin review
type Review struct {
	ID        string    `json:"id"`
	PluginName string   `json:"pluginName"`
	Version   string    `json:"version"`
	Author    string    `json:"author"`
	Rating    int       `json:"rating"` // 1-5
	Title     string    `json:"title,omitempty"`
	Body      string    `json:"body,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	Helpful   int       `json:"helpful"`
}

// InstalledPlugin represents a locally installed plugin
type InstalledPlugin struct {
	Name         string                 `json:"name"`
	Version      string                 `json:"version"`
	Enabled      bool                   `json:"enabled"`
	InstalledAt  time.Time              `json:"installedAt"`
	UpdatedAt    time.Time              `json:"updatedAt"`
	Config       map[string]interface{} `json:"config,omitempty"`
	AutoUpdate   bool                   `json:"autoUpdate"`
	LatestVersion string                `json:"latestVersion,omitempty"`
	HasUpdate    bool                   `json:"hasUpdate"`
}

// PluginManifest represents a plugin manifest file
type PluginManifest struct {
	APIVersion   string            `json:"apiVersion" yaml:"apiVersion"`
	Kind         string            `json:"kind" yaml:"kind"`
	Name         string            `json:"name" yaml:"name"`
	Version      string            `json:"version" yaml:"version"`
	Description  string            `json:"description" yaml:"description"`
	Author       Author            `json:"author" yaml:"author"`
	License      string            `json:"license" yaml:"license"`
	Homepage     string            `json:"homepage,omitempty" yaml:"homepage,omitempty"`
	Repository   string            `json:"repository,omitempty" yaml:"repository,omitempty"`
	Keywords     []string          `json:"keywords,omitempty" yaml:"keywords,omitempty"`
	Categories   []string          `json:"categories,omitempty" yaml:"categories,omitempty"`
	MinVersion   string            `json:"minVersion,omitempty" yaml:"minVersion,omitempty"`
	Capabilities []string          `json:"capabilities,omitempty" yaml:"capabilities,omitempty"`
	Hooks        PluginHooks       `json:"hooks,omitempty" yaml:"hooks,omitempty"`
	Commands     []PluginCommand   `json:"commands,omitempty" yaml:"commands,omitempty"`
	Config       PluginConfig      `json:"config,omitempty" yaml:"config,omitempty"`
}

// PluginHooks defines plugin lifecycle hooks
type PluginHooks struct {
	Install   string `json:"install,omitempty" yaml:"install,omitempty"`
	Uninstall string `json:"uninstall,omitempty" yaml:"uninstall,omitempty"`
	Upgrade   string `json:"upgrade,omitempty" yaml:"upgrade,omitempty"`
	Enable    string `json:"enable,omitempty" yaml:"enable,omitempty"`
	Disable   string `json:"disable,omitempty" yaml:"disable,omitempty"`
}

// PluginCommand defines a command provided by the plugin
type PluginCommand struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description" yaml:"description"`
	Usage       string `json:"usage,omitempty" yaml:"usage,omitempty"`
	Binary      string `json:"binary" yaml:"binary"`
}

// Category represents a plugin category
type Category struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon,omitempty"`
	Count       int    `json:"count"`
}

// Publisher represents a verified plugin publisher
type Publisher struct {
	Name        string    `json:"name"`
	DisplayName string    `json:"displayName"`
	Email       string    `json:"email,omitempty"`
	URL         string    `json:"url,omitempty"`
	Verified    bool      `json:"verified"`
	Plugins     []string  `json:"plugins"`
	JoinedAt    time.Time `json:"joinedAt"`
}

// SecurityScan represents plugin security scan results
type SecurityScan struct {
	PluginName  string    `json:"pluginName"`
	Version     string    `json:"version"`
	ScannedAt   time.Time `json:"scannedAt"`
	Status      string    `json:"status"` // passed, warning, failed
	Issues      []SecurityIssue `json:"issues,omitempty"`
}

// SecurityIssue represents a security issue found in a plugin
type SecurityIssue struct {
	Severity    string `json:"severity"` // critical, high, medium, low
	Type        string `json:"type"`
	Description string `json:"description"`
	Location    string `json:"location,omitempty"`
	Remediation string `json:"remediation,omitempty"`
}
