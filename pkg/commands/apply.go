// Package commands provides testable command implementations for Platform Foundry.
// These commands encapsulate business logic separate from CLI parsing.
package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/platformfoundry/platformfoundry-ce/pkg/log"
	"github.com/platformfoundry/platformfoundry-ce/pkg/types"
)

// ApplyCommand handles applying platform resources
type ApplyCommand struct {
	Parser      Parser
	Executor    Executor
	Parallelism int
	Logger      *log.Logger
}

// Parser interface for resource parsing
type Parser interface {
	ParseFile(path string) ([]types.Resource, error)
}

// Executor interface for applying resources
type Executor interface {
	Apply(ctx context.Context, resources []types.Resource) ([]ComponentResult, error)
}

// ApplyInput contains input parameters for the apply command
type ApplyInput struct {
	FilePath    string
	Environment string
	DryRun      bool
	Timeout     time.Duration
}

// ApplyOutput contains the result of the apply command
type ApplyOutput struct {
	Platform    string
	Components  []ComponentResult
	Duration    time.Duration
	Success     bool
	ErrorCount  int
}

// ComponentResult represents the result of applying a component
type ComponentResult struct {
	Name    string
	Type    string
	Status  string
	Message string
	Outputs map[string]string
}

// NewApplyCommand creates a new apply command
func NewApplyCommand(parser Parser, executor Executor) *ApplyCommand {
	return &ApplyCommand{
		Parser:      parser,
		Executor:    executor,
		Parallelism: 4,
		Logger:      log.Default().WithPrefix("apply"),
	}
}

// Execute runs the apply command
func (c *ApplyCommand) Execute(ctx context.Context, input ApplyInput) (*ApplyOutput, error) {
	start := time.Now()

	// Validate input
	if input.FilePath == "" {
		return nil, fmt.Errorf("file path is required")
	}

	// Parse resources
	resources, err := c.Parser.ParseFile(input.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	if c.Logger != nil {
		c.Logger.Info("Parsed resources",
			log.String("file", input.FilePath),
			log.Int("count", len(resources)),
		)
	}

	// Find platform resource
	platform, err := c.findPlatform(resources)
	if err != nil {
		return nil, err
	}

	output := &ApplyOutput{
		Platform:   platform.Metadata.Name,
		Components: make([]ComponentResult, 0),
	}

	// Dry run check
	if input.DryRun {
		if c.Logger != nil {
			c.Logger.Info("Dry run mode - no changes will be made")
		}
		output.Success = true
		output.Duration = time.Since(start)
		return output, nil
	}

	// Apply timeout if specified
	if input.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, input.Timeout)
		defer cancel()
	}

	// Execute apply via executor
	if c.Executor != nil {
		results, err := c.Executor.Apply(ctx, resources)
		if err != nil {
			return nil, fmt.Errorf("apply failed: %w", err)
		}

		for _, result := range results {
			output.Components = append(output.Components, result)
			if result.Status != "success" {
				output.ErrorCount++
			}
		}
	}

	output.Success = output.ErrorCount == 0
	output.Duration = time.Since(start)

	if c.Logger != nil {
		c.Logger.Info("Apply completed",
			log.String("platform", output.Platform),
			log.Int("components", len(output.Components)),
			log.Int("errors", output.ErrorCount),
			log.Duration("duration", output.Duration),
		)
	}

	return output, nil
}

// findPlatform finds and validates the platform resource
func (c *ApplyCommand) findPlatform(resources []types.Resource) (*types.Resource, error) {
	for i, r := range resources {
		if r.Kind == "Platform" {
			return &resources[i], nil
		}
	}
	return nil, fmt.Errorf("no platform resource found")
}

// Validate validates the resources without applying them
func (c *ApplyCommand) Validate(ctx context.Context, input ApplyInput) error {
	// Parse resources
	resources, err := c.Parser.ParseFile(input.FilePath)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	// Find platform
	_, err = c.findPlatform(resources)
	if err != nil {
		return err
	}

	// Additional validation could go here
	return nil
}
