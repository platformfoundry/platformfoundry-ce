// Package mock provides mock implementations for platform components
// enabling fast iteration and testing without real infrastructure.
package mock

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/platformfoundry/pf-ce/pkg/plugin"
)

// MockMode defines the mock behavior
type MockMode string

const (
	MockModeInstant   MockMode = "instant"   // Immediate completion
	MockModeRealistic MockMode = "realistic" // Simulated delays
	MockModeRecorded  MockMode = "recorded"  // Replay recorded responses
	MockModeChaos     MockMode = "chaos"     // Random failures
)

// MockConfig configures mock behavior
type MockConfig struct {
	Mode             MockMode
	DefaultDelay     time.Duration
	PerToolDelay     map[string]time.Duration
	FailureRate      float64
	FailureTools     []string
	RecordResponses  bool
	PlaybackFile     string
	ResponseOverride map[string]interface{}
}

// DefaultMockConfig returns a sensible default configuration
func DefaultMockConfig() *MockConfig {
	return &MockConfig{
		Mode:         MockModeRealistic,
		DefaultDelay: 2 * time.Second,
		PerToolDelay: map[string]time.Duration{
			"infrastructure": 5 * time.Second,
			"orchestrator":   3 * time.Second,
			"observability":  3 * time.Second,
			"devex":          2 * time.Second,
		},
		FailureRate:      0,
		ResponseOverride: make(map[string]interface{}),
	}
}

// MockPlugin wraps a real plugin with mock capabilities
// Implements plugin.Plugin interface from pkg/plugin
type MockPlugin struct {
	realPlugin plugin.Plugin
	tool       string
	toolType   string
	config     *MockConfig
	recorder   *ResponseRecorder
}

// Ensure MockPlugin implements plugin.Plugin
var _ plugin.Plugin = (*MockPlugin)(nil)

// NewMockPlugin creates a new mock plugin
func NewMockPlugin(tool, toolType string, config *MockConfig) *MockPlugin {
	if config == nil {
		config = DefaultMockConfig()
	}
	return &MockPlugin{
		tool:     tool,
		toolType: toolType,
		config:   config,
		recorder: NewResponseRecorder(),
	}
}

// WrapPlugin wraps an existing plugin with mock capabilities
func WrapPlugin(realPlugin plugin.Plugin, config *MockConfig) *MockPlugin {
	if config == nil {
		config = DefaultMockConfig()
	}
	return &MockPlugin{
		realPlugin: realPlugin,
		tool:       realPlugin.Name(),
		toolType:   realPlugin.Type(),
		config:     config,
		recorder:   NewResponseRecorder(),
	}
}

// Name returns the plugin name
func (m *MockPlugin) Name() string {
	return m.tool + "-mock"
}

// Type returns the plugin type
func (m *MockPlugin) Type() string {
	return m.toolType
}

// Version returns the plugin version
func (m *MockPlugin) Version() string {
	if m.realPlugin != nil {
		return m.realPlugin.Version() + "-mock"
	}
	return "1.0.0-mock"
}

// ConfigType returns the configuration type for this plugin
func (m *MockPlugin) ConfigType() interface{} {
	if m.realPlugin != nil {
		return m.realPlugin.ConfigType()
	}
	return nil
}

// Validate validates the specification
func (m *MockPlugin) Validate(spec map[string]interface{}) error {
	if m.realPlugin != nil {
		return m.realPlugin.Validate(spec)
	}
	return nil
}

// Plan creates an execution plan
func (m *MockPlugin) Plan(spec map[string]interface{}) (*plugin.Plan, error) {
	if m.realPlugin != nil {
		return m.realPlugin.Plan(spec)
	}

	return &plugin.Plan{
		Actions: []string{fmt.Sprintf("Create %s (mock)", m.tool)},
		Changes: map[string]string{
			"resource": m.tool,
			"action":   "create",
		},
	}, nil
}

// Apply executes the mock operation
func (m *MockPlugin) Apply(spec map[string]interface{}) (*plugin.Result, error) {
	switch m.config.Mode {
	case MockModeInstant:
		return m.instantApply(spec)
	case MockModeRealistic:
		return m.realisticApply(spec)
	case MockModeRecorded:
		return m.recordedApply(spec)
	case MockModeChaos:
		return m.chaosApply(spec)
	default:
		return m.instantApply(spec)
	}
}

// instantApply returns immediately with mock success
func (m *MockPlugin) instantApply(spec map[string]interface{}) (*plugin.Result, error) {
	outputs := generateOutputs(m, spec)

	if m.config.RecordResponses {
		m.recorder.Record(m.tool, "apply", spec, outputs, nil)
	}

	return &plugin.Result{
		Status:    "success",
		Message:   fmt.Sprintf("Mock %s applied instantly", m.tool),
		Resources: []string{m.tool},
		Outputs:   outputs,
	}, nil
}

// realisticApply simulates realistic delays
func (m *MockPlugin) realisticApply(spec map[string]interface{}) (*plugin.Result, error) {
	delay := m.config.DefaultDelay
	if toolDelay, ok := m.config.PerToolDelay[m.tool]; ok {
		delay = toolDelay
	}

	// Simulate progress over time
	steps := 5
	stepDelay := delay / time.Duration(steps)

	for i := 0; i < steps; i++ {
		time.Sleep(stepDelay)
	}

	outputs := generateOutputs(m, spec)

	if m.config.RecordResponses {
		m.recorder.Record(m.tool, "apply", spec, outputs, nil)
	}

	return &plugin.Result{
		Status:    "success",
		Message:   fmt.Sprintf("Mock %s applied with realistic delay", m.tool),
		Resources: []string{m.tool},
		Outputs:   outputs,
	}, nil
}

// recordedApply replays recorded responses
func (m *MockPlugin) recordedApply(spec map[string]interface{}) (*plugin.Result, error) {
	if m.recorder == nil {
		return m.instantApply(spec)
	}

	response, err := m.recorder.Playback(m.tool, "apply")
	if err != nil {
		return m.instantApply(spec)
	}

	return &plugin.Result{
		Status:    "success",
		Message:   fmt.Sprintf("Mock %s applied from recording", m.tool),
		Resources: []string{m.tool},
		Outputs:   response.Outputs,
	}, nil
}

// chaosApply introduces random failures
func (m *MockPlugin) chaosApply(spec map[string]interface{}) (*plugin.Result, error) {
	// Check if this tool should fail
	for _, failTool := range m.config.FailureTools {
		if failTool == m.tool {
			err := fmt.Errorf("simulated failure for %s", m.tool)
			if m.config.RecordResponses {
				m.recorder.Record(m.tool, "apply", spec, nil, err)
			}
			return nil, err
		}
	}

	// Random failure based on rate
	if m.config.FailureRate > 0 && rand.Float64() < m.config.FailureRate {
		err := fmt.Errorf("random chaos failure for %s", m.tool)
		if m.config.RecordResponses {
			m.recorder.Record(m.tool, "apply", spec, nil, err)
		}
		return nil, err
	}

	return m.realisticApply(spec)
}

// Delete removes the mock resource
func (m *MockPlugin) Delete(name string) error {
	if m.config.RecordResponses {
		m.recorder.Record(m.tool, "delete", map[string]interface{}{"name": name}, nil, nil)
	}
	return nil
}

// Status returns the mock status
func (m *MockPlugin) Status(name string) (*plugin.Status, error) {
	return &plugin.Status{
		State:   "ready",
		Ready:   true,
		Message: fmt.Sprintf("Mock %s is healthy", m.tool),
		Details: map[string]string{
			"mock": "true",
			"name": name,
		},
	}, nil
}

// ResponseRecorder records and replays mock responses
type ResponseRecorder struct {
	responses map[string][]RecordedResponse
	mu        sync.RWMutex
}

// RecordedResponse represents a recorded response
type RecordedResponse struct {
	Tool      string
	Operation string
	Input     map[string]interface{}
	Outputs   map[string]string
	Error     error
	Timestamp time.Time
}

// NewResponseRecorder creates a new response recorder
func NewResponseRecorder() *ResponseRecorder {
	return &ResponseRecorder{
		responses: make(map[string][]RecordedResponse),
	}
}

// Record stores a response
func (r *ResponseRecorder) Record(tool, operation string, input map[string]interface{}, outputs map[string]string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := fmt.Sprintf("%s:%s", tool, operation)
	r.responses[key] = append(r.responses[key], RecordedResponse{
		Tool:      tool,
		Operation: operation,
		Input:     input,
		Outputs:   outputs,
		Error:     err,
		Timestamp: time.Now(),
	})
}

// Playback retrieves a recorded response
func (r *ResponseRecorder) Playback(tool, operation string) (*RecordedResponse, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", tool, operation)
	responses, ok := r.responses[key]
	if !ok || len(responses) == 0 {
		return nil, fmt.Errorf("no recorded response for %s", key)
	}

	// Return the most recent response
	return &responses[len(responses)-1], nil
}

// Clear removes all recorded responses
func (r *ResponseRecorder) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.responses = make(map[string][]RecordedResponse)
}

// GetAll returns all recorded responses
func (r *ResponseRecorder) GetAll() map[string][]RecordedResponse {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string][]RecordedResponse)
	for k, v := range r.responses {
		result[k] = append([]RecordedResponse{}, v...)
	}
	return result
}
