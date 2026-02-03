package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// APIClientConfig represents API client configuration
type APIClientConfig struct {
	BaseURL      string        `yaml:"baseURL" json:"baseURL"`
	Token        string        `yaml:"token" json:"token"`
	Timeout      time.Duration `yaml:"timeout" json:"timeout"`
	Insecure     bool          `yaml:"insecure" json:"insecure"`
	Organization string        `yaml:"organization" json:"organization"`
}

// APIClient is a simple HTTP client for API calls
type APIClient struct {
	baseURL      string
	token        string
	organization string
	httpClient   *http.Client
}

// NewAPIClient creates a new API client
func NewAPIClient(baseURL, token string) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewAPIClientFromConfig creates a new API client from configuration
func NewAPIClientFromConfig(cfg *APIClientConfig) *APIClient {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &APIClient{
		baseURL:      cfg.BaseURL,
		token:        cfg.Token,
		organization: cfg.Organization,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// LoadAPIClientConfig loads API client configuration from file and environment
func LoadAPIClientConfig() (*APIClientConfig, error) {
	config := &APIClientConfig{
		Timeout: 30 * time.Second,
	}

	// Try to load from config file
	configPaths := []string{
		filepath.Join(os.Getenv("HOME"), ".pf", "config.yaml"),
		filepath.Join(os.Getenv("HOME"), ".platformfoundry", "config.yaml"),
		"config/client.yaml",
	}

	for _, path := range configPaths {
		if data, err := os.ReadFile(path); err == nil {
			// Expand environment variables in config
			expanded := os.ExpandEnv(string(data))
			if err := yaml.Unmarshal([]byte(expanded), config); err == nil {
				break
			}
		}
	}

	// Environment variables override config file
	if url := os.Getenv("PF_API_URL"); url != "" {
		config.BaseURL = url
	}
	if token := os.Getenv("PF_API_TOKEN"); token != "" {
		config.Token = token
	}
	if org := os.Getenv("PF_ORGANIZATION"); org != "" {
		config.Organization = org
	}
	if timeout := os.Getenv("PF_API_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			config.Timeout = d
		}
	}

	// Set defaults if not configured
	if config.BaseURL == "" {
		config.BaseURL = "http://localhost:8080"
	}

	return config, nil
}

// SaveAPIClientConfig saves API client configuration to file
func SaveAPIClientConfig(cfg *APIClientConfig) error {
	configDir := filepath.Join(os.Getenv("HOME"), ".pf")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

func (c *APIClient) Get(path string, params map[string]string) ([]byte, error) {
	req, err := http.NewRequest("GET", c.buildURL(path, params), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	return c.do(req)
}

func (c *APIClient) Post(path string, data interface{}) ([]byte, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	req, err := http.NewRequest("POST", c.buildURL(path, nil), bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	return c.do(req)
}

func (c *APIClient) Put(path string, data interface{}) ([]byte, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	req, err := http.NewRequest("PUT", c.buildURL(path, nil), bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	return c.do(req)
}

func (c *APIClient) Delete(path string, params map[string]string) ([]byte, error) {
	req, err := http.NewRequest("DELETE", c.buildURL(path, params), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	return c.do(req)
}

func (c *APIClient) buildURL(path string, params map[string]string) string {
	fullURL := fmt.Sprintf("%s%s", c.baseURL, path)

	if len(params) > 0 {
		queryParams := url.Values{}
		for key, value := range params {
			queryParams.Add(key, value)
		}
		fullURL = fmt.Sprintf("%s?%s", fullURL, queryParams.Encode())
	}

	return fullURL
}

func (c *APIClient) do(req *http.Request) ([]byte, error) {
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("api error (status %d): %s", resp.StatusCode, string(body))
	}

	return body, nil
}
