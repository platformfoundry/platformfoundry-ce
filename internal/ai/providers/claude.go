package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/platformfoundry/pf-ce/internal/ai"
)

const (
	claudeDefaultBaseURL = "https://api.anthropic.com/v1"
	claudeDefaultModel   = "claude-3-sonnet-20240229"
	claudeAPIVersion     = "2023-06-01"
)

// ClaudeProvider implements the LLM provider interface for Anthropic Claude
type ClaudeProvider struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
	maxTokens  int
}

// ClaudeConfig configures the Claude provider
type ClaudeConfig struct {
	APIKey    string        `yaml:"apiKey" json:"apiKey"`
	Model     string        `yaml:"model" json:"model"`
	BaseURL   string        `yaml:"baseURL" json:"baseURL"`
	Timeout   time.Duration `yaml:"timeout" json:"timeout"`
	MaxTokens int           `yaml:"maxTokens" json:"maxTokens"`
}

// NewClaudeProvider creates a new Claude provider
func NewClaudeProvider(config ClaudeConfig) (*ClaudeProvider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("Claude API key is required")
	}

	model := config.Model
	if model == "" {
		model = claudeDefaultModel
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = claudeDefaultBaseURL
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second
	}

	maxTokens := config.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	return &ClaudeProvider{
		apiKey:    config.APIKey,
		model:     model,
		baseURL:   baseURL,
		maxTokens: maxTokens,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func (p *ClaudeProvider) Name() string {
	return "claude"
}

func (p *ClaudeProvider) SupportsTools() bool {
	return true
}

func (p *ClaudeProvider) MaxTokens() int {
	return p.maxTokens
}

// Claude API request/response structures
type claudeRequest struct {
	Model       string          `json:"model"`
	Messages    []claudeMessage `json:"messages"`
	System      string          `json:"system,omitempty"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float64         `json:"temperature,omitempty"`
	TopP        float64         `json:"top_p,omitempty"`
	Tools       []claudeTool    `json:"tools,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}

type claudeMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // string or []claudeContentBlock
}

type claudeContentBlock struct {
	Type      string          `json:"type"` // text, tool_use, tool_result
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   string          `json:"content,omitempty"`
}

type claudeTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"input_schema"`
}

type claudeResponse struct {
	ID           string               `json:"id"`
	Type         string               `json:"type"`
	Role         string               `json:"role"`
	Content      []claudeContentBlock `json:"content"`
	Model        string               `json:"model"`
	StopReason   string               `json:"stop_reason"`
	StopSequence *string              `json:"stop_sequence"`
	Usage        claudeUsage          `json:"usage"`
}

type claudeUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type claudeError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type claudeErrorResponse struct {
	Type  string      `json:"type"`
	Error claudeError `json:"error"`
}

func (p *ClaudeProvider) Complete(ctx context.Context, req *ai.CompletionRequest) (*ai.CompletionResponse, error) {
	// Convert messages to Claude format
	claudeMessages, err := p.convertMessages(req.Messages)
	if err != nil {
		return nil, fmt.Errorf("failed to convert messages: %w", err)
	}

	// Build Claude request
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = p.maxTokens
	}

	claudeReq := claudeRequest{
		Model:       p.model,
		Messages:    claudeMessages,
		System:      req.SystemPrompt,
		MaxTokens:   maxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
	}

	// Convert tools
	if len(req.Tools) > 0 {
		claudeReq.Tools = p.convertTools(req.Tools)
	}

	// Make HTTP request
	resp, err := p.doRequest(ctx, "/messages", claudeReq)
	if err != nil {
		return nil, err
	}

	// Convert response
	return p.convertResponse(resp), nil
}

func (p *ClaudeProvider) Stream(ctx context.Context, req *ai.CompletionRequest) (<-chan ai.StreamChunk, error) {
	ch := make(chan ai.StreamChunk, 100)

	go func() {
		defer close(ch)

		// For now, use non-streaming and send as single chunk
		// Full streaming implementation would use SSE
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

func (p *ClaudeProvider) convertMessages(messages []ai.Message) ([]claudeMessage, error) {
	var result []claudeMessage

	for _, msg := range messages {
		switch msg.Role {
		case "user":
			if msg.ToolCallID != "" {
				// This is a tool result
				result = append(result, claudeMessage{
					Role: "user",
					Content: []claudeContentBlock{{
						Type:      "tool_result",
						ToolUseID: msg.ToolCallID,
						Content:   msg.Content,
					}},
				})
			} else {
				result = append(result, claudeMessage{
					Role:    "user",
					Content: msg.Content,
				})
			}

		case "assistant":
			if len(msg.ToolCalls) > 0 {
				// Assistant message with tool calls
				var blocks []claudeContentBlock
				if msg.Content != "" {
					blocks = append(blocks, claudeContentBlock{
						Type: "text",
						Text: msg.Content,
					})
				}
				for _, tc := range msg.ToolCalls {
					inputJSON, _ := json.Marshal(tc.Arguments)
					blocks = append(blocks, claudeContentBlock{
						Type:  "tool_use",
						ID:    tc.ID,
						Name:  tc.Name,
						Input: inputJSON,
					})
				}
				result = append(result, claudeMessage{
					Role:    "assistant",
					Content: blocks,
				})
			} else {
				result = append(result, claudeMessage{
					Role:    "assistant",
					Content: msg.Content,
				})
			}

		case "tool":
			// Tool results are handled as user messages in Claude
			result = append(result, claudeMessage{
				Role: "user",
				Content: []claudeContentBlock{{
					Type:      "tool_result",
					ToolUseID: msg.ToolCallID,
					Content:   msg.Content,
				}},
			})

		case "system":
			// System messages are handled separately in Claude
			continue

		default:
			return nil, fmt.Errorf("unsupported message role: %s", msg.Role)
		}
	}

	return result, nil
}

func (p *ClaudeProvider) convertTools(tools []ai.ToolDefinition) []claudeTool {
	var result []claudeTool
	for _, t := range tools {
		result = append(result, claudeTool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.Parameters,
		})
	}
	return result
}

func (p *ClaudeProvider) convertResponse(resp *claudeResponse) *ai.CompletionResponse {
	result := &ai.CompletionResponse{
		Model:        resp.Model,
		FinishReason: resp.StopReason,
		Usage: ai.TokenUsage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}

	// Extract content and tool calls
	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			result.Content += block.Text
		case "tool_use":
			var args map[string]interface{}
			if len(block.Input) > 0 {
				json.Unmarshal(block.Input, &args)
			}
			result.ToolCalls = append(result.ToolCalls, ai.ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: args,
			})
		}
	}

	return result
}

func (p *ClaudeProvider) doRequest(ctx context.Context, endpoint string, body interface{}) (*claudeResponse, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", claudeAPIVersion)

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
		var errResp claudeErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil {
			return nil, fmt.Errorf("Claude API error (%d): %s - %s",
				resp.StatusCode, errResp.Error.Type, errResp.Error.Message)
		}
		return nil, fmt.Errorf("Claude API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var claudeResp claudeResponse
	if err := json.Unmarshal(respBody, &claudeResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &claudeResp, nil
}
