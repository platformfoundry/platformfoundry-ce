package intelligence

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewLLMRecommender(t *testing.T) {
	config := LLMConfig{
		Provider: ProviderMock,
	}

	recommender, err := NewLLMRecommender(config)
	if err != nil {
		t.Fatalf("NewLLMRecommender() error = %v", err)
	}

	if recommender == nil {
		t.Fatal("NewLLMRecommender() returned nil")
	}

	if recommender.fallbackRecommender == nil {
		t.Error("Fallback recommender should be initialized")
	}

	// Check defaults
	if recommender.config.Model != "gpt-3.5-turbo" {
		t.Errorf("Expected default model gpt-3.5-turbo, got %s", recommender.config.Model)
	}

	if recommender.config.MaxTokens != 500 {
		t.Errorf("Expected default max_tokens 500, got %d", recommender.config.MaxTokens)
	}

	if recommender.config.Temperature != 0.7 {
		t.Errorf("Expected default temperature 0.7, got %.1f", recommender.config.Temperature)
	}
}

func TestRecommendWithMock(t *testing.T) {
	config := LLMConfig{
		Provider: ProviderMock,
	}

	recommender, _ := NewLLMRecommender(config)

	ts := &TechStack{
		InfrastructureProvider: "terraform",
		CloudProvider:          "aws",
		Orchestrator:           "argocd",
		ObservabilityTools:     []string{"prometheus", "grafana"},
		HasMonitoring:          true,
	}

	rec, err := recommender.Recommend(ts)
	if err != nil {
		t.Fatalf("Recommend() error = %v", err)
	}

	if rec == nil {
		t.Fatal("Recommend() returned nil")
	}

	if rec.Template != "aws-k8s-full" {
		t.Errorf("Expected template aws-k8s-full, got %s", rec.Template)
	}

	if rec.Confidence == 0 {
		t.Error("Confidence should be set")
	}

	if len(rec.Features) == 0 {
		t.Error("Features should not be empty")
	}

	if len(rec.Integrations) == 0 {
		t.Error("Integrations should not be empty")
	}

	// Check that GitHub is included
	githubFound := false
	for _, integ := range rec.Integrations {
		if integ == "github" {
			githubFound = true
			break
		}
	}
	if !githubFound {
		t.Error("GitHub should be in integrations")
	}
}

func TestRecommendWithMockGCP(t *testing.T) {
	config := LLMConfig{
		Provider: ProviderMock,
	}

	recommender, _ := NewLLMRecommender(config)

	ts := &TechStack{
		CloudProvider: "gcp",
		Orchestrator:  "flux",
		HasMonitoring: true,
	}

	rec, err := recommender.Recommend(ts)
	if err != nil {
		t.Fatalf("Recommend() error = %v", err)
	}

	if rec.Template != "gcp-k8s-full" {
		t.Errorf("Expected template gcp-k8s-full, got %s", rec.Template)
	}
}

func TestRecommendWithMockAzure(t *testing.T) {
	config := LLMConfig{
		Provider: ProviderMock,
	}

	recommender, _ := NewLLMRecommender(config)

	ts := &TechStack{
		CloudProvider: "azure",
		HasMonitoring: false,
	}

	rec, err := recommender.Recommend(ts)
	if err != nil {
		t.Fatalf("Recommend() error = %v", err)
	}

	if rec.Template != "azure-k8s-full" {
		t.Errorf("Expected template azure-k8s-full, got %s", rec.Template)
	}
}

func TestRecommendFallback(t *testing.T) {
	config := LLMConfig{
		Provider: ProviderLocal, // This will fail and fall back
	}

	recommender, _ := NewLLMRecommender(config)

	ts := &TechStack{
		CloudProvider: "aws",
		HasMonitoring: true,
	}

	rec, err := recommender.Recommend(ts)
	if err != nil {
		t.Fatalf("Recommend() should not error with fallback, got: %v", err)
	}

	if rec == nil {
		t.Fatal("Recommend() should return fallback recommendation")
	}

	// Verify it's from the fallback recommender
	if rec.Template == "" {
		t.Error("Fallback should provide a template")
	}
}

func TestBuildPrompt(t *testing.T) {
	config := LLMConfig{
		Provider: ProviderMock,
	}

	recommender, _ := NewLLMRecommender(config)

	ts := &TechStack{
		InfrastructureProvider: "terraform",
		CloudProvider:          "aws",
		Orchestrator:           "argocd",
		ObservabilityTools:     []string{"prometheus"},
		HasMonitoring:          true,
		HasLogging:             true,
	}

	prompt := recommender.buildPrompt(ts)

	if prompt == "" {
		t.Error("buildPrompt() should return non-empty string")
	}

	// Check key elements are in prompt
	if !contains(prompt, "terraform") {
		t.Error("Prompt should mention infrastructure provider")
	}

	if !contains(prompt, "aws") {
		t.Error("Prompt should mention cloud provider")
	}

	if !contains(prompt, "argocd") {
		t.Error("Prompt should mention orchestrator")
	}

	if !contains(prompt, "prometheus") {
		t.Error("Prompt should mention observability tools")
	}

	if !contains(prompt, "JSON") {
		t.Error("Prompt should mention JSON format")
	}
}

func TestParseResponse(t *testing.T) {
	config := LLMConfig{
		Provider: ProviderMock,
	}

	recommender, _ := NewLLMRecommender(config)

	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name: "valid JSON",
			content: `{
				"template": "aws-k8s-full",
				"features": ["catalog", "docs"],
				"integrations": ["github", "argocd"],
				"reason": "Test reason",
				"confidence": 0.9
			}`,
			wantErr: false,
		},
		{
			name: "JSON with surrounding text",
			content: `Here's my recommendation:
			{
				"template": "gcp-k8s-full",
				"features": ["catalog"],
				"integrations": ["github"],
				"reason": "Test",
				"confidence": 0.8
			}
			This should work well.`,
			wantErr: false,
		},
		{
			name:    "no JSON",
			content: "This is just text without JSON",
			wantErr: true,
		},
		{
			name: "invalid JSON",
			content: `{
				"template": "aws-k8s-full",
				"invalid
			}`,
			wantErr: true,
		},
		{
			name: "missing template",
			content: `{
				"features": ["catalog"],
				"reason": "Test"
			}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := recommender.parseResponse(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if resp == nil {
					t.Error("parseResponse() should return response for valid JSON")
				}
				if resp.Template == "" {
					t.Error("Response should have template")
				}
			}
		})
	}
}

func TestParseResponseDefaults(t *testing.T) {
	config := LLMConfig{
		Provider: ProviderMock,
	}

	recommender, _ := NewLLMRecommender(config)

	content := `{
		"template": "minimal",
		"reason": "Basic setup"
	}`

	resp, err := recommender.parseResponse(content)
	if err != nil {
		t.Fatalf("parseResponse() error = %v", err)
	}

	// Check defaults are applied
	if resp.Confidence != 0.8 {
		t.Errorf("Expected default confidence 0.8, got %.1f", resp.Confidence)
	}

	if len(resp.Features) == 0 {
		t.Error("Default features should be set")
	}

	if len(resp.Integrations) == 0 {
		t.Error("Default integrations should be set")
	}
}

func TestRecommendWithOpenAIMock(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("Authorization header should be set")
		}

		// Return mock response
		response := OpenAIResponse{
			ID:      "test-id",
			Object:  "chat.completion",
			Created: 1234567890,
			Model:   "gpt-3.5-turbo",
			Choices: []Choice{
				{
					Index: 0,
					Message: Message{
						Role: "assistant",
						Content: `{
							"template": "aws-k8s-full",
							"features": ["catalog", "docs", "scaffolder"],
							"integrations": ["github", "argocd", "prometheus"],
							"reason": "AWS with full monitoring stack",
							"confidence": 0.95
						}`,
					},
					FinishReason: "stop",
				},
			},
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := LLMConfig{
		Provider:   ProviderOpenAI,
		APIKey:     "test-key",
		APIBaseURL: server.URL,
	}

	recommender, _ := NewLLMRecommender(config)

	ts := &TechStack{
		CloudProvider: "aws",
		HasMonitoring: true,
	}

	rec, err := recommender.Recommend(ts)
	if err != nil {
		t.Fatalf("Recommend() error = %v", err)
	}

	if rec.Template != "aws-k8s-full" {
		t.Errorf("Expected template aws-k8s-full, got %s", rec.Template)
	}

	if rec.Confidence != 0.95 {
		t.Errorf("Expected confidence 0.95, got %.2f", rec.Confidence)
	}

	if len(rec.Features) != 3 {
		t.Errorf("Expected 3 features, got %d", len(rec.Features))
	}
}

func TestRecommendWithOpenAINoKey(t *testing.T) {
	config := LLMConfig{
		Provider: ProviderOpenAI,
		APIKey:   "", // No key
	}

	recommender, _ := NewLLMRecommender(config)

	ts := &TechStack{
		CloudProvider: "aws",
	}

	// Should fall back to rule-based
	rec, err := recommender.Recommend(ts)
	if err != nil {
		t.Fatalf("Recommend() should not error with fallback, got: %v", err)
	}

	if rec == nil {
		t.Fatal("Should return fallback recommendation")
	}
}

func TestIsAvailable(t *testing.T) {
	tests := []struct {
		name     string
		config   LLMConfig
		expected bool
	}{
		{
			name: "OpenAI with key",
			config: LLMConfig{
				Provider: ProviderOpenAI,
				APIKey:   "test-key",
			},
			expected: true,
		},
		{
			name: "OpenAI without key",
			config: LLMConfig{
				Provider: ProviderOpenAI,
				APIKey:   "",
			},
			expected: false,
		},
		{
			name: "Mock provider",
			config: LLMConfig{
				Provider: ProviderMock,
			},
			expected: true,
		},
		{
			name: "Local provider",
			config: LLMConfig{
				Provider: ProviderLocal,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recommender, _ := NewLLMRecommender(tt.config)
			if recommender.IsAvailable() != tt.expected {
				t.Errorf("IsAvailable() = %v, want %v", recommender.IsAvailable(), tt.expected)
			}
		})
	}
}

func TestGetProvider(t *testing.T) {
	config := LLMConfig{
		Provider: ProviderMock,
	}

	recommender, _ := NewLLMRecommender(config)

	if recommender.GetProvider() != ProviderMock {
		t.Errorf("Expected provider %s, got %s", ProviderMock, recommender.GetProvider())
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsMiddle(s, substr))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
