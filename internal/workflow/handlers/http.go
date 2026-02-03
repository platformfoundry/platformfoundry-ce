package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/platformfoundry/platformfoundry-ce/internal/workflow"
	"github.com/platformfoundry/platformfoundry-ce/internal/workflow/dag"
)

// HTTPHandler executes HTTP requests
type HTTPHandler struct {
	BaseHandler
	client *http.Client
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler() *HTTPHandler {
	return &HTTPHandler{
		BaseHandler: BaseHandler{stepType: workflow.StepTypeHTTP},
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewHTTPHandlerWithClient creates an HTTP handler with a custom client
func NewHTTPHandlerWithClient(client *http.Client) *HTTPHandler {
	return &HTTPHandler{
		BaseHandler: BaseHandler{stepType: workflow.StepTypeHTTP},
		client:      client,
	}
}

// Validate validates the HTTP step configuration
func (h *HTTPHandler) Validate(config map[string]interface{}) error {
	url := GetStringConfig(config, "url", "")
	if url == "" {
		return fmt.Errorf("http step requires 'url' configuration")
	}

	method := strings.ToUpper(GetStringConfig(config, "method", "GET"))
	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true, "HEAD": true, "OPTIONS": true,
	}
	if !validMethods[method] {
		return fmt.Errorf("invalid HTTP method: %s", method)
	}

	return nil
}

// Execute performs the HTTP request
func (h *HTTPHandler) Execute(ctx context.Context, step *workflow.StepExecution, config map[string]interface{}, resolver dag.OutputResolver) (*workflow.StepResult, error) {
	result := &workflow.StepResult{
		Status:  workflow.StepStatusRunning,
		Outputs: make(map[string]interface{}),
		Logs:    make([]workflow.StepLog, 0),
	}

	// Get configuration
	url := GetStringConfig(config, "url", "")
	method := strings.ToUpper(GetStringConfig(config, "method", "GET"))
	headers := GetMapConfig(config, "headers")
	body := config["body"]
	timeout := GetIntConfig(config, "timeout", 30)
	successCodes := GetStringSliceConfig(config, "successCodes")
	if len(successCodes) == 0 {
		successCodes = []string{"2xx"}
	}

	// Prepare request body
	var bodyReader io.Reader
	if body != nil {
		switch v := body.(type) {
		case string:
			bodyReader = strings.NewReader(v)
		case map[string]interface{}:
			jsonBody, err := json.Marshal(v)
			if err != nil {
				result.Status = workflow.StepStatusFailed
				result.ErrorMsg = fmt.Sprintf("failed to marshal body: %v", err)
				return result, fmt.Errorf("%s", result.ErrorMsg)
			}
			bodyReader = bytes.NewReader(jsonBody)
		}
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		result.Status = workflow.StepStatusFailed
		result.ErrorMsg = fmt.Sprintf("failed to create request: %v", err)
		return result, fmt.Errorf("%s", result.ErrorMsg)
	}

	// Set headers
	for key, val := range headers {
		if strVal, ok := val.(string); ok {
			req.Header.Set(key, strVal)
		}
	}

	// Set content-type if body is JSON
	if body != nil && req.Header.Get("Content-Type") == "" {
		if _, ok := body.(map[string]interface{}); ok {
			req.Header.Set("Content-Type", "application/json")
		}
	}

	// Log request
	result.Logs = append(result.Logs, workflow.StepLog{
		Time:    time.Now(),
		Level:   "info",
		Message: fmt.Sprintf("HTTP %s %s", method, url),
	})

	// Set timeout
	client := h.client
	if timeout > 0 {
		client = &http.Client{
			Timeout:   time.Duration(timeout) * time.Second,
			Transport: h.client.Transport,
		}
	}

	// Execute request
	startTime := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		result.Status = workflow.StepStatusFailed
		result.Outputs["duration_ms"] = duration.Milliseconds()
		result.ErrorMsg = fmt.Sprintf("HTTP request failed: %v", err)
		result.Logs = append(result.Logs, workflow.StepLog{
			Time:    time.Now(),
			Level:   "error",
			Message: result.ErrorMsg,
		})
		return result, fmt.Errorf("%s", result.ErrorMsg)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Status = workflow.StepStatusFailed
		result.ErrorMsg = fmt.Sprintf("failed to read response body: %v", err)
		return result, fmt.Errorf("%s", result.ErrorMsg)
	}

	// Set outputs
	result.Outputs["statusCode"] = resp.StatusCode
	result.Outputs["status"] = resp.Status
	result.Outputs["duration_ms"] = duration.Milliseconds()

	// Try to parse JSON response
	var jsonResp interface{}
	if err := json.Unmarshal(respBody, &jsonResp); err == nil {
		result.Outputs["body"] = jsonResp
	} else {
		result.Outputs["body"] = string(respBody)
	}

	// Capture response headers
	respHeaders := make(map[string]string)
	for key := range resp.Header {
		respHeaders[key] = resp.Header.Get(key)
	}
	result.Outputs["headers"] = respHeaders

	// Log response
	result.Logs = append(result.Logs, workflow.StepLog{
		Time:    time.Now(),
		Level:   "info",
		Message: fmt.Sprintf("Response: %d %s (took %v)", resp.StatusCode, resp.Status, duration),
	})

	// Check success codes
	success := false
	for _, code := range successCodes {
		if code == "2xx" && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			success = true
			break
		}
		if code == fmt.Sprintf("%d", resp.StatusCode) {
			success = true
			break
		}
	}

	if !success {
		result.Status = workflow.StepStatusFailed
		result.ErrorMsg = fmt.Sprintf("HTTP request returned status %d", resp.StatusCode)
		return result, fmt.Errorf("%s", result.ErrorMsg)
	}

	result.Status = workflow.StepStatusCompleted
	return result, nil
}
