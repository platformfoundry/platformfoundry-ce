package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/platformfoundry/pf-ce/internal/secrets"
	"github.com/platformfoundry/pf-ce/internal/workflow"
	"github.com/platformfoundry/pf-ce/internal/workflow/dag"
)

// SecretsHandler retrieves secrets
type SecretsHandler struct {
	BaseHandler
	manager secrets.Manager
}

// NewSecretsHandler creates a new secrets handler
func NewSecretsHandler(manager secrets.Manager) *SecretsHandler {
	return &SecretsHandler{
		BaseHandler: BaseHandler{stepType: workflow.StepTypeSecrets},
		manager:     manager,
	}
}

// Validate validates the secrets step configuration
func (h *SecretsHandler) Validate(config map[string]interface{}) error {
	path := GetStringConfig(config, "path", "")
	paths := GetStringSliceConfig(config, "paths")

	if path == "" && len(paths) == 0 {
		return fmt.Errorf("secrets step requires 'path' or 'paths' configuration")
	}

	return nil
}

// Execute retrieves the secrets
func (h *SecretsHandler) Execute(ctx context.Context, step *workflow.StepExecution, config map[string]interface{}, resolver dag.OutputResolver) (*workflow.StepResult, error) {
	result := &workflow.StepResult{
		Status:  workflow.StepStatusRunning,
		Outputs: make(map[string]interface{}),
		Logs:    make([]workflow.StepLog, 0),
	}

	path := GetStringConfig(config, "path", "")
	paths := GetStringSliceConfig(config, "paths")
	keys := GetStringSliceConfig(config, "keys") // Specific keys to retrieve

	// Combine single path and multiple paths
	allPaths := make([]string, 0)
	if path != "" {
		allPaths = append(allPaths, path)
	}
	allPaths = append(allPaths, paths...)

	result.Logs = append(result.Logs, workflow.StepLog{
		Time:    time.Now(),
		Level:   "info",
		Message: fmt.Sprintf("Retrieving secrets from %d paths", len(allPaths)),
	})

	if h.manager == nil {
		result.Logs = append(result.Logs, workflow.StepLog{
			Time:    time.Now(),
			Level:   "warn",
			Message: "No secrets manager configured - returning empty secrets",
		})
		result.Status = workflow.StepStatusCompleted
		result.Outputs["secrets"] = map[string]interface{}{}
		result.Outputs["simulated"] = true
		return result, nil
	}

	// Retrieve secrets
	secretsMap := make(map[string]interface{})
	for _, secretPath := range allPaths {
		secret, err := h.manager.GetSecret(ctx, secretPath)
		if err != nil {
			result.Status = workflow.StepStatusFailed
			result.ErrorMsg = fmt.Sprintf("failed to retrieve secret %s: %v", secretPath, err)
			result.Logs = append(result.Logs, workflow.StepLog{
				Time:    time.Now(),
				Level:   "error",
				Message: result.ErrorMsg,
			})
			return result, fmt.Errorf("%s", result.ErrorMsg)
		}

		// Filter keys if specified
		if len(keys) > 0 {
			filtered := make(map[string]string)
			for _, key := range keys {
				if val, ok := secret.Data[key]; ok {
					filtered[key] = val
				}
			}
			secretsMap[secretPath] = filtered
		} else {
			secretsMap[secretPath] = secret.Data
		}

		result.Logs = append(result.Logs, workflow.StepLog{
			Time:    time.Now(),
			Level:   "info",
			Message: fmt.Sprintf("Retrieved secret: %s (%d keys)", secretPath, len(secret.Data)),
		})
	}

	result.Status = workflow.StepStatusCompleted
	result.Outputs["secrets"] = secretsMap
	result.Outputs["pathCount"] = len(allPaths)

	result.Logs = append(result.Logs, workflow.StepLog{
		Time:    time.Now(),
		Level:   "info",
		Message: fmt.Sprintf("Successfully retrieved %d secrets", len(allPaths)),
	})

	return result, nil
}
