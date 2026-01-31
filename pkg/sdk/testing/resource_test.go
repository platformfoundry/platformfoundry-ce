// Package testing provides test helpers for plugin development.
package testing

import (
	"context"

	"github.com/platformfoundry/platformfoundry-ce/pkg/contracts/ppi"
	"github.com/platformfoundry/platformfoundry-ce/pkg/sdk"
)

// ResourceTestCase defines a test case for a resource
type ResourceTestCase struct {
	// Name is the test case name
	Name string

	// Resource is the resource being tested
	Resource sdk.Resource

	// Config is the resource configuration
	Config map[string]interface{}

	// ExpectError indicates if an error is expected
	ExpectError bool

	// Check is a function to verify the result
	Check func(*ppi.ResourceState) error
}

// ResourceTestStep defines a step in a resource test
type ResourceTestStep struct {
	// Name is the step name
	Name string

	// Config is the resource configuration for this step
	Config map[string]interface{}

	// ExpectError indicates if an error is expected
	ExpectError bool

	// Check is a function to verify the result
	Check func(*ppi.ResourceState) error

	// Destroy indicates this step should destroy the resource
	Destroy bool
}

// ResourceTester provides helper methods for testing resources
type ResourceTester struct {
	resource sdk.Resource
	ctx      context.Context
	state    *ppi.ResourceState
}

// NewResourceTester creates a new resource tester
func NewResourceTester(resource sdk.Resource) *ResourceTester {
	return &ResourceTester{
		resource: resource,
		ctx:      context.Background(),
	}
}

// WithContext sets the context for tests
func (t *ResourceTester) WithContext(ctx context.Context) *ResourceTester {
	t.ctx = ctx
	return t
}

// Validate validates a configuration
func (t *ResourceTester) Validate(config map[string]interface{}) (*ppi.Diagnostics, error) {
	resourceConfig := &ppi.ResourceConfig{
		Values: config,
	}
	return t.resource.Validate(t.ctx, resourceConfig)
}

// Create creates a resource
func (t *ResourceTester) Create(config map[string]interface{}) (*ppi.ResourceState, error) {
	proposed := &ppi.ResourceState{
		Attributes: config,
	}

	plan, err := t.resource.Plan(t.ctx, nil, proposed)
	if err != nil {
		return nil, err
	}

	state, err := t.resource.Create(t.ctx, plan)
	if err != nil {
		return nil, err
	}

	t.state = state
	return state, nil
}

// Update updates a resource
func (t *ResourceTester) Update(config map[string]interface{}) (*ppi.ResourceState, error) {
	if t.state == nil {
		return nil, &sdk.ResourceError{Message: "no existing state"}
	}

	proposed := &ppi.ResourceState{
		ID:         t.state.ID,
		Attributes: config,
	}

	plan, err := t.resource.Plan(t.ctx, t.state, proposed)
	if err != nil {
		return nil, err
	}

	state, err := t.resource.Update(t.ctx, plan)
	if err != nil {
		return nil, err
	}

	t.state = state
	return state, nil
}

// Read reads the current state
func (t *ResourceTester) Read() (*ppi.ResourceState, error) {
	if t.state == nil {
		return nil, &sdk.ResourceError{Message: "no existing state"}
	}

	return t.resource.Read(t.ctx, t.state)
}

// Delete deletes the resource
func (t *ResourceTester) Delete() error {
	if t.state == nil {
		return nil
	}

	err := t.resource.Delete(t.ctx, t.state)
	if err == nil {
		t.state = nil
	}
	return err
}

// Import imports a resource by ID
func (t *ResourceTester) Import(id string) (*ppi.ResourceState, error) {
	state, err := t.resource.Import(t.ctx, id)
	if err != nil {
		return nil, err
	}
	t.state = state
	return state, nil
}

// State returns the current state
func (t *ResourceTester) State() *ppi.ResourceState {
	return t.state
}

// RunSteps runs a series of test steps
func (t *ResourceTester) RunSteps(steps []ResourceTestStep) error {
	for _, step := range steps {
		if step.Destroy {
			if err := t.Delete(); err != nil {
				if !step.ExpectError {
					return &sdk.ResourceError{Message: "destroy failed: " + err.Error()}
				}
			}
			continue
		}

		var state *ppi.ResourceState
		var err error

		if t.state == nil {
			state, err = t.Create(step.Config)
		} else {
			state, err = t.Update(step.Config)
		}

		if err != nil {
			if !step.ExpectError {
				return &sdk.ResourceError{Message: step.Name + ": " + err.Error()}
			}
			continue
		}

		if step.ExpectError {
			return &sdk.ResourceError{Message: step.Name + ": expected error but succeeded"}
		}

		if step.Check != nil {
			if err := step.Check(state); err != nil {
				return &sdk.ResourceError{Message: step.Name + ": check failed: " + err.Error()}
			}
		}
	}

	return nil
}

// CheckFuncs combines multiple check functions
func CheckFuncs(checks ...func(*ppi.ResourceState) error) func(*ppi.ResourceState) error {
	return func(state *ppi.ResourceState) error {
		for _, check := range checks {
			if err := check(state); err != nil {
				return err
			}
		}
		return nil
	}
}

// CheckAttribute checks that an attribute has a specific value
func CheckAttribute(name string, expected interface{}) func(*ppi.ResourceState) error {
	return func(state *ppi.ResourceState) error {
		actual, ok := state.Attributes[name]
		if !ok {
			return &sdk.ResourceError{Message: "attribute " + name + " not found"}
		}
		if actual != expected {
			return &sdk.ResourceError{Message: "attribute " + name + " mismatch"}
		}
		return nil
	}
}

// CheckAttributeSet checks that an attribute is set
func CheckAttributeSet(name string) func(*ppi.ResourceState) error {
	return func(state *ppi.ResourceState) error {
		_, ok := state.Attributes[name]
		if !ok {
			return &sdk.ResourceError{Message: "attribute " + name + " not set"}
		}
		return nil
	}
}

// CheckNoAttribute checks that an attribute is not set
func CheckNoAttribute(name string) func(*ppi.ResourceState) error {
	return func(state *ppi.ResourceState) error {
		_, ok := state.Attributes[name]
		if ok {
			return &sdk.ResourceError{Message: "attribute " + name + " should not be set"}
		}
		return nil
	}
}
