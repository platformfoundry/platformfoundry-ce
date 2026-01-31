package plugin

// Plugin interface that all plugins must implement
type Plugin interface {
	// Name returns the plugin name (e.g., "jenkins", "eks")
	Name() string

	// Type returns the resource type this plugin handles (e.g., "Pipeline", "Cluster")
	Type() string

	// Version returns the plugin version
	Version() string

	// ConfigType returns an instance of the plugin's configuration struct
	// Platform Foundry uses this to validate and bind YAML to strongly-typed config
	ConfigType() interface{}

	// Validate validates the resource specification
	Validate(spec map[string]interface{}) error

	// Plan generates an execution plan for the resource
	Plan(spec map[string]interface{}) (*Plan, error)

	// Apply provisions or updates the resource
	Apply(spec map[string]interface{}) (*Result, error)

	// Delete destroys the resource
	Delete(name string) error

	// Status gets the current status of the resource
	Status(name string) (*Status, error)
}

// Plan represents the execution plan for a resource
type Plan struct {
	Actions []string          `json:"actions"`
	Changes map[string]string `json:"changes,omitempty"`
}

// Result represents the result of applying a resource
type Result struct {
	Status    string            `json:"status"`  // success, failed, partial
	Message   string            `json:"message"` // Human-readable message
	Resources []string          `json:"resources,omitempty"`
	Outputs   map[string]string `json:"outputs,omitempty"`
}

// Status represents the current status of a resource
type Status struct {
	State   string            `json:"state"`  // pending, running, ready, failed
	Ready   bool              `json:"ready"`
	Message string            `json:"message,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}
