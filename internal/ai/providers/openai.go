package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/platformfoundry/platformfoundry-ce/internal/ai"
)

const (
	openaiDefaultBaseURL = "https://api.openai.com/v1"
	openaiDefaultModel   = "gpt-4-turbo-preview"
)

// OpenAIProvider implements the LLM provider interface for OpenAI
type OpenAIProvider struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
	maxTokens  int
	orgID      string
}

// OpenAIConfig configures the OpenAI provider
type OpenAIConfig struct {
	APIKey    string        `yaml:"apiKey" json:"apiKey"`
	Model     string        `yaml:"model" json:"model"`
	BaseURL   string        `yaml:"baseURL" json:"baseURL"`
	OrgID     string        `yaml:"orgId" json:"orgId"`
	Timeout   time.Duration `yaml:"timeout" json:"timeout"`
	MaxTokens int           `yaml:"maxTokens" json:"maxTokens"`
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(config OpenAIConfig) (*OpenAIProvider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	model := config.Model
	if model == "" {
		model = openaiDefaultModel
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = openaiDefaultBaseURL
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second
	}

	maxTokens := config.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	return &OpenAIProvider{
		apiKey:    config.APIKey,
		model:     model,
		baseURL:   baseURL,
		orgID:     config.OrgID,
		maxTokens: maxTokens,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func (p *OpenAIProvider) Name() string {
	return "openai"
}

func (p *OpenAIProvider) SupportsTools() bool {
	return true
}

func (p *OpenAIProvider) MaxTokens() int {
	return p.maxTokens
}

// OpenAI API structures
type openaiRequest struct {
	Model       string          `json:"model"`
	Messages    []openaiMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	TopP        float64         `json:"top_p,omitempty"`
	Tools       []openaiTool    `json:"tools,omitempty"`
	ToolChoice  interface{}     `json:"tool_choice,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}

type openaiMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	Name       string           `json:"name,omitempty"`
	ToolCalls  []openaiToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openaiTool struct {
	Type     string         `json:"type"`
	Function openaiFunction `json:"function"`
}

type openaiFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type openaiToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function openaiToolFunction `json:"function"`
}

type openaiToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openaiResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []openaiChoice `json:"choices"`
	Usage   openaiUsage    `json:"usage"`
}

type openaiChoice struct {
	Index        int           `json:"index"`
	Message      openaiMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openaiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type openaiErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

func (p *OpenAIProvider) Complete(ctx context.Context, req *ai.CompletionRequest) (*ai.CompletionResponse, error) {
	// Convert messages to OpenAI format
	openaiMessages := p.convertMessages(req.Messages, req.SystemPrompt)

	// Build OpenAI request
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = p.maxTokens
	}

	openaiReq := openaiRequest{
		Model:       p.model,
		Messages:    openaiMessages,
		MaxTokens:   maxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
	}

	// Convert tools
	if len(req.Tools) > 0 {
		openaiReq.Tools = p.convertTools(req.Tools)
		openaiReq.ToolChoice = "auto"
	}

	// Make HTTP request
	resp, err := p.doRequest(ctx, "/chat/completions", openaiReq)
	if err != nil {
		return nil, err
	}

	// Convert response
	return p.convertResponse(resp), nil
}

func (p *OpenAIProvider) Stream(ctx context.Context, req *ai.CompletionRequest) (<-chan ai.StreamChunk, error) {
	ch := make(chan ai.StreamChunk, 100)

	go func() {
		defer close(ch)

		// For now, use non-streaming and send as single chunk
		resp, err := p.Complete(ctx, req)
		if err != nil {
			ch <- ai.StreamChunk{Error: err, Done: true}
			return
		}

		ch <- ai.StreamChunk{
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
			Done:      true,
		}
	}()

	return ch, nil
}

func (p *OpenAIProvider) convertMessages(messages []ai.Message, systemPrompt string) []openaiMessage {
	var result []openaiMessage

	// Add system prompt if provided
	if systemPrompt != "" {
		result = append(result, openaiMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	for _, msg := range messages {
		switch msg.Role {
		case "user":
			result = append(result, openaiMessage{
				Role:    "user",
				Content: msg.Content,
			})

		case "assistant":
			openaiMsg := openaiMessage{
				Role:    "assistant",
				Content: msg.Content,
			}
			if len(msg.ToolCalls) > 0 {
				for _, tc := range msg.ToolCalls {
					argsJSON, _ := json.Marshal(tc.Arguments)
					openaiMsg.ToolCalls = append(openaiMsg.ToolCalls, openaiToolCall{
						ID:   tc.ID,
						Type: "function",
						Function: openaiToolFunction{
							Name:      tc.Name,
							Arguments: string(argsJSON),
						},
					})
				}
			}
			result = append(result, openaiMsg)

		case "tool":
			result = append(result, openaiMessage{
				Role:       "tool",
				Content:    msg.Content,
				ToolCallID: msg.ToolCallID,
			})

		case "system":
			result = append(result, openaiMessage{
				Role:    "system",
				Content: msg.Content,
			})
		}
	}

	return result
}

func (p *OpenAIProvider) convertTools(tools []ai.ToolDefinition) []openaiTool {
	var result []openaiTool
	for _, t := range tools {
		result = append(result, openaiTool{
			Type: "function",
			Function: openaiFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		})
	}
	return result
}

func (p *OpenAIProvider) convertResponse(resp *openaiResponse) *ai.CompletionResponse {
	if len(resp.Choices) == 0 {
		return &ai.CompletionResponse{
			Model: resp.Model,
			Usage: ai.TokenUsage{
				PromptTokens:     resp.Usage.PromptTokens,
				CompletionTokens: resp.Usage.CompletionTokens,
				TotalTokens:      resp.Usage.TotalTokens,
			},
		}
	}

	choice := resp.Choices[0]
	result := &ai.CompletionResponse{
		Content:      choice.Message.Content,
		Model:        resp.Model,
		FinishReason: choice.FinishReason,
		Usage: ai.TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}

	// Convert tool calls
	for _, tc := range choice.Message.ToolCalls {
		var args map[string]interface{}
		json.Unmarshal([]byte(tc.Function.Arguments), &args)
		result.ToolCalls = append(result.ToolCalls, ai.ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: args,
		})
	}

	return result
}

func (p *OpenAIProvider) doRequest(ctx context.Context, endpoint string, body interface{}) (*openaiResponse, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	if p.orgID != "" {
		req.Header.Set("OpenAI-Organization", p.orgID)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp openaiErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("OpenAI API error (%d): %s - %s",
				resp.StatusCode, errResp.Error.Type, errResp.Error.Message)
		}
		return nil, fmt.Errorf("OpenAI API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var openaiResp openaiResponse
	if err := json.Unmarshal(respBody, &openaiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &openaiResp, nil
}
