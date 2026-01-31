package types

import (
	"fmt"
)

// ServiceAction represents an action to be executed on a service
type ServiceAction struct {
	APIVersion string            `yaml:"apiVersion" json:"apiVersion"`
	Kind       string            `yaml:"kind" json:"kind"`
	Metadata   Metadata          `yaml:"metadata" json:"metadata"`
	Spec       ServiceActionSpec `yaml:"spec" json:"spec"`
}

// ServiceActionSpec defines the service action specification
type ServiceActionSpec struct {
	Service     string            `yaml:"service" json:"service"`
	Action      ActionType        `yaml:"action" json:"action"`
	Environment string            `yaml:"environment" json:"environment"`
	Config      map[string]interface{} `yaml:"config,omitempty" json:"config,omitempty"`
	DryRun      bool              `yaml:"dryRun,omitempty" json:"dryRun,omitempty"`
	Force       bool              `yaml:"force,omitempty" json:"force,omitempty"`
}

// ActionType represents the type of action to perform
type ActionType string

const (
	ActionDeploy    ActionType = "deploy"
	ActionRollback  ActionType = "rollback"
	ActionScale     ActionType = "scale"
	ActionRestart   ActionType = "restart"
	ActionStop      ActionType = "stop"
	ActionStart     ActionType = "start"
	ActionDelete    ActionType = "delete"
	ActionPromote   ActionType = "promote"
	ActionCanary    ActionType = "canary"
	ActionBlueGreen ActionType = "bluegreen"
)

// Validate validates the service action with security checks
func (sa *ServiceAction) Validate() error {
	if sa.APIVersion == "" {
		return ErrMissingAPIVersion
	}
	if sa.Kind != "ServiceAction" {
		return ErrInvalidKind
	}
	if sa.Metadata.Name == "" {
		return ErrMissingName
	}

	// Validate service reference
	if sa.Spec.Service == "" {
		return fmt.Errorf("service is required")
	}
	if len(sa.Spec.Service) > 253 {
		return fmt.Errorf("service reference must be 253 characters or less")
	}

	// Validate action type
	if sa.Spec.Action == "" {
		return fmt.Errorf("action is required")
	}
	if !IsValidActionType(sa.Spec.Action) {
		return fmt.Errorf("invalid action type: %s", sa.Spec.Action)
	}

	// Validate environment
	if sa.Spec.Environment == "" {
		return fmt.Errorf("environment is required")
	}
	if len(sa.Spec.Environment) > 63 {
		return fmt.Errorf("environment must be 63 characters or less")
	}

	// Security: Limit config size
	if len(sa.Spec.Config) > 100 {
		return fmt.Errorf("too many config parameters (max 100)")
	}

	// Validate specific actions
	switch sa.Spec.Action {
	case ActionScale:
		// Replicas parameter required for scale
		if sa.Spec.Config != nil {
			if replicas, ok := sa.Spec.Config["replicas"]; ok {
				switch v := replicas.(type) {
				case int:
					if v < 0 || v > 1000 {
						return fmt.Errorf("replicas must be between 0 and 1000")
					}
				case float64:
					if v < 0 || v > 1000 {
						return fmt.Errorf("replicas must be between 0 and 1000")
					}
				default:
					return fmt.Errorf("replicas must be a number")
				}
			}
		}
	case ActionCanary, ActionBlueGreen:
		// Version required for canary/bluegreen deployments
		if sa.Spec.Config != nil {
			if version, ok := sa.Spec.Config["version"]; ok {
				if str, ok := version.(string); ok {
					if len(str) > 128 {
						return fmt.Errorf("version must be 128 characters or less")
					}
				}
			}
		}
	}

	return nil
}

// IsValidActionType checks if an action type is valid
func IsValidActionType(actionType ActionType) bool {
	return actionType == ActionDeploy ||
		actionType == ActionRollback ||
		actionType == ActionScale ||
		actionType == ActionRestart ||
		actionType == ActionStop ||
		actionType == ActionStart ||
		actionType == ActionDelete ||
		actionType == ActionPromote ||
		actionType == ActionCanary ||
		actionType == ActionBlueGreen
}
