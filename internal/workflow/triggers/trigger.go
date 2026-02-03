package triggers

import (
	"context"

	"github.com/platformfoundry/platformfoundry-ce/internal/workflow"
)

// Trigger interface for workflow triggers
type Trigger interface {
	// Type returns the trigger type
	Type() string

	// Start starts the trigger
	Start(ctx context.Context) error

	// Stop stops the trigger
	Stop() error

	// OnTrigger sets the callback for when the trigger fires
	OnTrigger(callback TriggerCallback)
}

// TriggerCallback is called when a trigger fires
type TriggerCallback func(workflowName string, inputs map[string]interface{})

// TriggerEvent represents a trigger event
type TriggerEvent struct {
	WorkflowName string
	TriggerName  string
	TriggerType  string
	Inputs       map[string]interface{}
}

// TriggerManager manages workflow triggers
type TriggerManager struct {
	triggers  map[string]Trigger
	callbacks []TriggerCallback
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewTriggerManager creates a new trigger manager
func NewTriggerManager() *TriggerManager {
	return &TriggerManager{
		triggers:  make(map[string]Trigger),
		callbacks: make([]TriggerCallback, 0),
	}
}

// RegisterTrigger registers a trigger
func (m *TriggerManager) RegisterTrigger(name string, trigger Trigger) {
	m.triggers[name] = trigger
	trigger.OnTrigger(m.onTrigger)
}

// UnregisterTrigger removes a trigger
func (m *TriggerManager) UnregisterTrigger(name string) {
	if trigger, ok := m.triggers[name]; ok {
		trigger.Stop()
		delete(m.triggers, name)
	}
}

// OnTrigger adds a global callback for all triggers
func (m *TriggerManager) OnTrigger(callback TriggerCallback) {
	m.callbacks = append(m.callbacks, callback)
}

// onTrigger is called when any trigger fires
func (m *TriggerManager) onTrigger(workflowName string, inputs map[string]interface{}) {
	for _, callback := range m.callbacks {
		go callback(workflowName, inputs)
	}
}

// Start starts all triggers
func (m *TriggerManager) Start(ctx context.Context) error {
	m.ctx, m.cancel = context.WithCancel(ctx)

	for _, trigger := range m.triggers {
		if err := trigger.Start(m.ctx); err != nil {
			return err
		}
	}

	return nil
}

// Stop stops all triggers
func (m *TriggerManager) Stop() error {
	if m.cancel != nil {
		m.cancel()
	}

	for _, trigger := range m.triggers {
		trigger.Stop()
	}

	return nil
}

// SetupTriggersForWorkflow sets up triggers for a workflow
func (m *TriggerManager) SetupTriggersForWorkflow(wf *workflow.DAGWorkflow) error {
	for _, triggerSpec := range wf.Spec.Triggers {
		if triggerSpec.Disabled {
			continue
		}

		var trigger Trigger
		triggerName := triggerSpec.Name
		if triggerName == "" {
			triggerName = wf.Metadata.Name + "-" + triggerSpec.Type
		}

		switch triggerSpec.Type {
		case "schedule":
			if triggerSpec.Cron != "" {
				trigger = NewScheduler(wf.Metadata.Name, triggerSpec.Cron)
			}
		case "webhook":
			path := "/webhook/" + wf.Metadata.Name
			if triggerSpec.Webhook != nil && triggerSpec.Webhook.Path != "" {
				path = triggerSpec.Webhook.Path
			}
			secret := ""
			if triggerSpec.Webhook != nil {
				secret = triggerSpec.Webhook.Secret
			}
			trigger = NewWebhookTrigger(wf.Metadata.Name, path, secret)
		case "manual":
			// Manual triggers don't need setup
			continue
		}

		if trigger != nil {
			m.RegisterTrigger(triggerName, trigger)
		}
	}

	return nil
}
