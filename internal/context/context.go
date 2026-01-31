package context

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/platformfoundry/platformfoundry-ce/pkg/configloader"
)

// Context represents the current CLI context
type Context struct {
	CurrentUser         string `json:"current_user"`
	CurrentOrganization string `json:"current_organization"`
	CurrentEnvironment  string `json:"current_environment"`
}

// Manager manages CLI context
type Manager struct {
	contextFile string
	context     *Context
}

// NewManager creates a new context manager
func NewManager() (*Manager, error) {
	loader := configloader.Global()
	contextFile := loader.FilePath(configloader.ConfigFileContext)

	m := &Manager{
		contextFile: contextFile,
		context: &Context{
			CurrentUser:         "local-user",
			CurrentOrganization: "default",
			CurrentEnvironment:  "",
		},
	}

	// Load existing context if available
	if err := m.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return m, nil
}

// GetCurrentOrganization returns the current organization
func (m *Manager) GetCurrentOrganization() string {
	return m.context.CurrentOrganization
}

// GetCurrentEnvironment returns the current environment
func (m *Manager) GetCurrentEnvironment() string {
	return m.context.CurrentEnvironment
}

// GetCurrentUser returns the current user
func (m *Manager) GetCurrentUser() string {
	return m.context.CurrentUser
}

// SetOrganization sets the current organization
func (m *Manager) SetOrganization(org string) error {
	m.context.CurrentOrganization = org
	return m.save()
}

// SetEnvironment sets the current environment
func (m *Manager) SetEnvironment(env string) error {
	m.context.CurrentEnvironment = env
	return m.save()
}

// SetUser sets the current user
func (m *Manager) SetUser(user string) error {
	m.context.CurrentUser = user
	return m.save()
}

// load loads context from file
func (m *Manager) load() error {
	data, err := os.ReadFile(m.contextFile)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, m.context)
}

// save saves context to file
func (m *Manager) save() error {
	dir := filepath.Dir(m.contextFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create context directory: %w", err)
	}

	data, err := json.MarshalIndent(m.context, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal context: %w", err)
	}

	return os.WriteFile(m.contextFile, data, 0600) // Secure file permissions
}

// Reset resets context to defaults
func (m *Manager) Reset() error {
	m.context = &Context{
		CurrentUser:         "local-user",
		CurrentOrganization: "default",
		CurrentEnvironment:  "",
	}
	return m.save()
}
