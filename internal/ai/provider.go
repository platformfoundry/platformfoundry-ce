package ai

import (
	"context"
	"time"
)

// LLMProvider defines the interface for language model providers
type LLMProvider interface {
	// Name returns the provider name (e.g., "claude", "openai")
	Name() string

	// Complete sends a prompt and returns a complete response
	Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)

	// Stream sends a prompt and streams the response
	Stream(ctx context.Context, req *CompletionRequest) (<-chan StreamChunk, error)

	// SupportsTools returns whether the provider supports tool/function calling
	SupportsTools() bool

	// MaxTokens returns the maximum tokens supported by this provider
	MaxTokens() int
}

// CompletionRequest represents a request to the LLM
type CompletionRequest struct {
	// Messages is the conversation history
	Messages []Message `json:"messages"`

	// Tools available for the model to use
	Tools []ToolDefinition `json:"tools,omitempty"`

	// SystemPrompt is the system instruction
	SystemPrompt string `json:"system_prompt,omitempty"`

	// MaxTokens limits the response length
	MaxTokens int `json:"max_tokens,omitempty"`

	// Temperature controls randomness (0-1)
	Temperature float64 `json:"temperature,omitempty"`

	// TopP controls nucleus sampling
	TopP float64 `json:"top_p,omitempty"`

	// StopSequences are strings that stop generation
	StopSequences []string `json:"stop_sequences,omitempty"`
}

// Message represents a conversation message
type Message struct {
	Role      string     `json:"role"` // system, user, assistant, tool
	Content   string     `json:"content"`
	Name      string     `json:"name,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string    `json:"tool_call_id,omitempty"` // For tool responses
}

// ToolDefinition defines a tool the LLM can use
type ToolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"` // JSON Schema
}

// ToolCall represents a tool invocation by the LLM
type ToolCall struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// CompletionResponse represents the LLM response
type CompletionResponse struct {
	Content      string     `json:"content"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	Usage        TokenUsage `json:"usage"`
	Model        string     `json:"model"`
	FinishReason string     `json:"finish_reason"` // stop, tool_calls, length, etc.
}

// TokenUsage tracks token consumption
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamChunk represents a streaming response chunk
type StreamChunk struct {
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Done      bool       `json:"done"`
	Error     error      `json:"error,omitempty"`
}

// ProviderConfig contains common provider configuration
type ProviderConfig struct {
	APIKey      string        `yaml:"apiKey" json:"apiKey"`
	Model       string        `yaml:"model" json:"model"`
	BaseURL     string        `yaml:"baseURL" json:"baseURL"`
	Timeout     time.Duration `yaml:"timeout" json:"timeout"`
	MaxRetries  int           `yaml:"maxRetries" json:"maxRetries"`
	Temperature float64       `yaml:"temperature" json:"temperature"`
	MaxTokens   int           `yaml:"maxTokens" json:"maxTokens"`
}

// DefaultProviderConfig returns sensible defaults
func DefaultProviderConfig() ProviderConfig {
	return ProviderConfig{
		Timeout:     120 * time.Second,
		MaxRetries:  3,
		Temperature: 0.7,
		MaxTokens:   4096,
	}
}

// ParameterSchema defines the JSON Schema for tool parameters
type ParameterSchema struct {
	Type        string              `json:"type"`
	Description string              `json:"description,omitempty"`
	Properties  map[string]Property `json:"properties,omitempty"`
	Required    []string            `json:"required,omitempty"`
}

// Property defines a single parameter property
type Property struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Default     interface{} `json:"default,omitempty"`
}

// NewMessage creates a new message
func NewMessage(role, content string) Message {
	return Message{
		Role:    role,
		Content: content,
	}
}

// NewUserMessage creates a user message
func NewUserMessage(content string) Message {
	return NewMessage("user", content)
}

// NewAssistantMessage creates an assistant message
func NewAssistantMessage(content string) Message {
	return NewMessage("assistant", content)
}

// NewToolResultMessage creates a tool result message
func NewToolResultMessage(toolCallID, content string) Message {
	return Message{
		Role:       "tool",
		Content:    content,
		ToolCallID: toolCallID,
	}
}

// IsToolCallResponse checks if this is a response with tool calls
func (r *CompletionResponse) IsToolCallResponse() bool {
	return len(r.ToolCalls) > 0
}

// HasContent checks if the response has text content
func (r *CompletionResponse) HasContent() bool {
	return r.Content != ""
}
