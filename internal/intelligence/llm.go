package intelligence

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// LLMProvider represents the LLM provider type
type LLMProvider string

const (
	ProviderOpenAI LLMProvider = "openai"
	ProviderLocal  LLMProvider = "local"
	ProviderMock   LLMProvider = "mock"
)

// LLMConfig represents LLM configuration
type LLMConfig struct {
	Provider   LLMProvider
	APIKey     string
	APIBaseURL string
	Model      string
	MaxTokens  int
	Temperature float64
}

// LLMRecommender provides LLM-powered portal recommendations
type LLMRecommender struct {
	config           LLMConfig
	httpClient       *http.Client
	fallbackRecommender *Recommender
}

// LLMResponse represents the response from LLM
type LLMResponse struct {
	Template     string   `json:"template"`
	Features     []string `json:"features"`
	Integrations []string `json:"integrations"`
	Reason       string   `json:"reason"`
	Confidence   float64  `json:"confidence"`
}

// OpenAIRequest represents OpenAI API request
type OpenAIRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIResponse represents OpenAI API response
type OpenAIResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
}

// Choice represents a response choice
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

var (
	ErrLLMNotAvailable = errors.New("LLM is not available")
	ErrInvalidResponse = errors.New("invalid LLM response")
)

// NewLLMRecommender creates a new LLM-powered recommender
func NewLLMRecommender(config LLMConfig) (*LLMRecommender, error) {
	// Create fallback recommender
	fallback, err := NewRecommender("")
	if err != nil {
		return nil, fmt.Errorf("failed to create fallback recommender: %w", err)
	}

	// Set defaults
	if config.Model == "" {
		config.Model = "gpt-3.5-turbo"
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 500
	}
	if config.Temperature == 0 {
		config.Temperature = 0.7
	}
	if config.APIBaseURL == "" {
		config.APIBaseURL = "https://api.openai.com/v1"
	}

	return &LLMRecommender{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		fallbackRecommender: fallback,
	}, nil
}

// Recommend generates LLM-powered recommendations based on tech stack
func (l *LLMRecommender) Recommend(ts *TechStack) (*Recommendation, error) {
	// Try LLM recommendation
	rec, err := l.recommendWithLLM(ts)
	if err != nil {
		// Fallback to rule-based recommender
		return l.fallbackRecommender.Recommend(ts), nil
	}

	return rec, nil
}

// recommendWithLLM generates recommendations using LLM
func (l *LLMRecommender) recommendWithLLM(ts *TechStack) (*Recommendation, error) {
	switch l.config.Provider {
	case ProviderOpenAI:
		return l.recommendWithOpenAI(ts)
	case ProviderLocal:
		return l.recommendWithLocalLLM(ts)
	case ProviderMock:
		return l.recommendWithMock(ts)
	default:
		return nil, ErrLLMNotAvailable
	}
}

// recommendWithOpenAI generates recommendations using OpenAI API
func (l *LLMRecommender) recommendWithOpenAI(ts *TechStack) (*Recommendation, error) {
	if l.config.APIKey == "" {
		return nil, ErrLLMNotAvailable
	}

	// Build prompt
	prompt := l.buildPrompt(ts)

	// Create request
	reqBody := OpenAIRequest{
		Model: l.config.Model,
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are an expert in developer platforms and infrastructure. Analyze the tech stack and recommend the best developer portal configuration. Respond only with valid JSON.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   l.config.MaxTokens,
		Temperature: l.config.Temperature,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make API call
	req, err := http.NewRequest("POST", l.config.APIBaseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+l.config.APIKey)

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call OpenAI API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error: %s - %s", resp.Status, string(body))
	}

	// Parse response
	var openAIResp OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(openAIResp.Choices) == 0 {
		return nil, ErrInvalidResponse
	}

	// Extract JSON from response
	content := openAIResp.Choices[0].Message.Content
	llmResp, err := l.parseResponse(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	// Convert to Recommendation
	return &Recommendation{
		Template:     llmResp.Template,
		Features:     llmResp.Features,
		Integrations: llmResp.Integrations,
		Reason:       llmResp.Reason,
		Confidence:   llmResp.Confidence,
		Details: map[string]interface{}{
			"source": "llm",
			"model":  l.config.Model,
		},
	}, nil
}

// recommendWithLocalLLM generates recommendations using local LLM
func (l *LLMRecommender) recommendWithLocalLLM(ts *TechStack) (*Recommendation, error) {
	// In a real implementation, this would connect to a local LLM server
	// For now, fall back to rule-based
	return nil, ErrLLMNotAvailable
}

// recommendWithMock generates mock recommendations for testing
func (l *LLMRecommender) recommendWithMock(ts *TechStack) (*Recommendation, error) {
	// Generate a mock recommendation based on tech stack
	template := "aws-k8s-full"
	if ts.CloudProvider == "gcp" {
		template = "gcp-k8s-full"
	} else if ts.CloudProvider == "azure" {
		template = "azure-k8s-full"
	}

	features := []string{"catalog", "docs", "scaffolder", "techdocs"}
	if ts.HasMonitoring {
		features = append(features, "kubernetes")
	}

	integrations := []string{"github"}
	if ts.Orchestrator != "" {
		integrations = append(integrations, strings.ToLower(ts.Orchestrator))
	}
	if ts.HasMonitoring {
		integrations = append(integrations, "prometheus", "grafana")
	}

	reason := fmt.Sprintf("Based on %s infrastructure with %s orchestrator and comprehensive monitoring, recommend %s template with full feature set.",
		ts.CloudProvider, ts.Orchestrator, template)

	return &Recommendation{
		Template:     template,
		Features:     features,
		Integrations: integrations,
		Reason:       reason,
		Confidence:   0.85,
		Details: map[string]interface{}{
			"source": "mock",
		},
	}, nil
}

// buildPrompt builds the prompt for LLM
func (l *LLMRecommender) buildPrompt(ts *TechStack) string {
	var prompt strings.Builder

	prompt.WriteString("Analyze this platform tech stack and recommend the best developer portal configuration:\n\n")
	prompt.WriteString(fmt.Sprintf("Infrastructure Provider: %s\n", ts.InfrastructureProvider))
	prompt.WriteString(fmt.Sprintf("Cloud Provider: %s\n", ts.CloudProvider))
	prompt.WriteString(fmt.Sprintf("Orchestrator: %s\n", ts.Orchestrator))

	if len(ts.ObservabilityTools) > 0 {
		prompt.WriteString(fmt.Sprintf("Observability Tools: %s\n", strings.Join(ts.ObservabilityTools, ", ")))
	}

	if ts.HasMonitoring {
		prompt.WriteString("Has Monitoring: Yes\n")
	}
	if ts.HasLogging {
		prompt.WriteString("Has Logging: Yes\n")
	}
	if ts.HasTracing {
		prompt.WriteString("Has Tracing: Yes\n")
	}

	prompt.WriteString("\nProvide a JSON response with this structure:\n")
	prompt.WriteString("{\n")
	prompt.WriteString("  \"template\": \"recommended template name\",\n")
	prompt.WriteString("  \"features\": [\"list\", \"of\", \"features\"],\n")
	prompt.WriteString("  \"integrations\": [\"list\", \"of\", \"integrations\"],\n")
	prompt.WriteString("  \"reason\": \"explanation of recommendation\",\n")
	prompt.WriteString("  \"confidence\": 0.9\n")
	prompt.WriteString("}\n\n")
	prompt.WriteString("Template options: aws-k8s-full, gcp-k8s-full, azure-k8s-full, multi-cloud, k8s-basic, minimal\n")
	prompt.WriteString("Features: catalog, docs, scaffolder, techdocs, kubernetes, cost-insights, search\n")
	prompt.WriteString("Integrations: github, gitlab, argocd, flux, prometheus, grafana, aws, gcp, azure")

	return prompt.String()
}

// parseResponse parses LLM response text and extracts JSON
func (l *LLMRecommender) parseResponse(content string) (*LLMResponse, error) {
	// Try to find JSON in the response
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")

	if start == -1 || end == -1 {
		return nil, ErrInvalidResponse
	}

	jsonStr := content[start : end+1]

	var resp LLMResponse
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Validate response
	if resp.Template == "" {
		return nil, ErrInvalidResponse
	}

	// Set defaults
	if resp.Confidence == 0 {
		resp.Confidence = 0.8
	}
	if len(resp.Features) == 0 {
		resp.Features = []string{"catalog", "docs"}
	}
	if len(resp.Integrations) == 0 {
		resp.Integrations = []string{"github"}
	}

	return &resp, nil
}

// SetFallback sets a custom fallback recommender
func (l *LLMRecommender) SetFallback(recommender *Recommender) {
	l.fallbackRecommender = recommender
}

// IsAvailable checks if LLM is available
func (l *LLMRecommender) IsAvailable() bool {
	switch l.config.Provider {
	case ProviderOpenAI:
		return l.config.APIKey != ""
	case ProviderLocal:
		// Would check if local LLM server is running
		return false
	case ProviderMock:
		return true
	default:
		return false
	}
}

// GetProvider returns the current provider
func (l *LLMRecommender) GetProvider() LLMProvider {
	return l.config.Provider
}
