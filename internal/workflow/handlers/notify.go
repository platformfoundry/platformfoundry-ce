package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/platformfoundry/platformfoundry-ce/internal/notifications"
	"github.com/platformfoundry/platformfoundry-ce/internal/workflow"
	"github.com/platformfoundry/platformfoundry-ce/internal/workflow/dag"
)

// NotifyHandler sends notifications
type NotifyHandler struct {
	BaseHandler
	notifyManager *notifications.Manager
}

// NewNotifyHandler creates a new notify handler
func NewNotifyHandler(manager *notifications.Manager) *NotifyHandler {
	return &NotifyHandler{
		BaseHandler:   BaseHandler{stepType: workflow.StepTypeNotify},
		notifyManager: manager,
	}
}

// Validate validates the notify step configuration
func (h *NotifyHandler) Validate(config map[string]interface{}) error {
	channel := GetStringConfig(config, "channel", "")
	if channel == "" {
		return fmt.Errorf("notify step requires 'channel' configuration (slack, webhook, etc.)")
	}

	message := GetStringConfig(config, "message", "")
	if message == "" {
		return fmt.Errorf("notify step requires 'message' configuration")
	}

	return nil
}

// Execute sends the notification
func (h *NotifyHandler) Execute(ctx context.Context, step *workflow.StepExecution, config map[string]interface{}, resolver dag.OutputResolver) (*workflow.StepResult, error) {
	result := &workflow.StepResult{
		Status:  workflow.StepStatusRunning,
		Outputs: make(map[string]interface{}),
		Logs:    make([]workflow.StepLog, 0),
	}

	// Get configuration
	channel := GetStringConfig(config, "channel", "")
	message := GetStringConfig(config, "message", "")
	title := GetStringConfig(config, "title", "Workflow Notification")
	severity := GetStringConfig(config, "severity", "info")

	result.Logs = append(result.Logs, workflow.StepLog{
		Time:    time.Now(),
		Level:   "info",
		Message: fmt.Sprintf("Sending notification to %s: %s", channel, title),
	})

	// Create notification event
	event := &notifications.Event{
		ID:      step.ID,
		Type:    notifications.EventWorkflowStarted, // Generic workflow event
		Source:  "workflow",
		Subject: message,
		Time:    time.Now(),
		Data: map[string]interface{}{
			"title":    title,
			"message":  message,
			"severity": severity,
			"channel":  channel,
		},
	}

	// Send notification if manager is available
	if h.notifyManager != nil {
		errs := h.notifyManager.Notify(ctx, event)
		if len(errs) > 0 {
			result.Status = workflow.StepStatusFailed
			result.ErrorMsg = fmt.Sprintf("notification errors: %v", errs)
			result.Logs = append(result.Logs, workflow.StepLog{
				Time:    time.Now(),
				Level:   "error",
				Message: result.ErrorMsg,
			})
			return result, fmt.Errorf("%s", result.ErrorMsg)
		}
	} else {
		// Log that we would send a notification
		result.Logs = append(result.Logs, workflow.StepLog{
			Time:    time.Now(),
			Level:   "info",
			Message: fmt.Sprintf("Would notify %s: [%s] %s - %s", channel, severity, title, message),
		})
	}

	result.Status = workflow.StepStatusCompleted
	result.Outputs["channel"] = channel
	result.Outputs["message"] = message
	result.Outputs["sent"] = h.notifyManager != nil

	result.Logs = append(result.Logs, workflow.StepLog{
		Time:    time.Now(),
		Level:   "info",
		Message: "Notification sent successfully",
	})

	return result, nil
}
