package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/platformfoundry/platformfoundry-ce/internal/orchestrator"
	"github.com/platformfoundry/platformfoundry-ce/internal/state"
)

// Handler provides common dependencies for all handlers
type Handler struct {
	Orchestrator *orchestrator.Service
	State        state.Backend
}

// Response is a standard API response
type Response struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     *ErrorInfo  `json:"error,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// ErrorInfo contains error details
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// JSON writes a JSON response
func (h *Handler) JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{
		Success:   status < 400,
		Data:      data,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// Error writes an error response
func (h *Handler) Error(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// DecodeJSON decodes JSON request body
func (h *Handler) DecodeJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}
