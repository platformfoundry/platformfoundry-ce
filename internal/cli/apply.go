package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/platformfoundry/pf-ce/internal/engine"
	"github.com/platformfoundry/pf-ce/internal/jobs"
	"github.com/platformfoundry/pf-ce/internal/mock"
	"github.com/platformfoundry/pf-ce/internal/parser"
	"github.com/platformfoundry/pf-ce/pkg/types"
	"github.com/spf13/cobra"
)

var (
	applyFile    string
	applyEnv     string
	mockMode     bool
	mockDelay    time.Duration
	mockFailRate float64
	parallelism  int
)

// Global job queue for async operations
var applyJobQueue = jobs.NewQueue(4)

var applyCmd = &cobra.Command{
	Use:   "apply -f <file>",
	Short: "Apply resources from a YAML file",
	Long:  `Parse and apply resources defined in a YAML file. Resources are applied in dependency order.`,
	Example: `  pf apply -f platform.yaml
  pf apply -f platform.yaml --mock
  pf apply -f platform.yaml --env production`,
	RunE: runApplyV2,
}

type ApplyOptions struct {
	Parallelism int
	Timeout     time.Duration
	UseMock     bool
	MockConfig  *mock.MockConfig
}

func init() {
	applyCmd.Flags().StringVarP(&applyFile, "file", "f", "", "YAML file containing resources (required)")
	applyCmd.Flags().StringVar(&applyEnv, "env", "", "Environment profile to apply (optional)")
	applyCmd.Flags().BoolVar(&mockMode, "mock", false, "Use mock providers")
	applyCmd.Flags().DurationVar(&mockDelay, "mock-delay", 5*time.Second, "Simulated delay per component")
	applyCmd.Flags().Float64Var(&mockFailRate, "mock-fail-rate", 0, "Simulated failure rate (0-1)")
	applyCmd.Flags().IntVar(&parallelism, "parallelism", 4, "Number of parallel engines to run")
	applyCmd.MarkFlagRequired("file")

	// Add completion for file flag
	applyCmd.RegisterFlagCompletionFunc("file", yamlFileCompletion)
}

func runApplyV2(cmd *cobra.Command, args []string) error {
	p := parser.New()
	resources, err := p.ParseFile(applyFile)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	opts := ApplyOptions{
		Parallelism: parallelism,
		UseMock:     mockMode,
		MockConfig: &mock.MockConfig{
			Mode:         mock.MockModeRealistic,
			DefaultDelay: mockDelay,
			FailureRate:  mockFailRate,
		},
	}

	return ApplyWithEngines(resources, opts)
}

// ApplyWithEngines uses the new engine architecture
func ApplyWithEngines(resources []types.Resource, opts ApplyOptions) error {
	// Find the platform resource
	var platformResource *types.Resource
	var platformSpec map[string]interface{}

	for i, r := range resources {
		if r.Kind == "Platform" {
			platformResource = &resources[i]
			platformSpec = r.Spec
			break
		}
	}

	if platformResource == nil {
		return fmt.Errorf("no platform resource found in the file")
	}

	// Extract component references from platform spec
	components, ok := platformSpec["components"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("platform spec missing components")
	}

	infraRef, _ := components["infrastructure"].(string)
	orchRef, _ := components["orchestrator"].(string)
	obsRef, _ := components["observability"].(string)
	devexRef, _ := components["devex"].(string)

	// Create coordinator with mock config if enabled
	coordConfig := engine.CoordinatorConfig{
		MaxParallelEngines: opts.Parallelism,
		Timeout:            opts.Timeout,
		MockMode:           opts.UseMock,
	}
	if opts.MockConfig != nil {
		coordConfig.MockConfig = &engine.MockConfig{
			Mode:           engine.MockMode(opts.MockConfig.Mode),
			SimulatedDelay: opts.MockConfig.DefaultDelay,
			FailureRate:    opts.MockConfig.FailureRate,
		}
	}

	coordinator := engine.NewCoordinator(coordConfig)

	// Subscribe to events for progress display
	progressDisplay := NewProgressDisplay()
	coordinator.Subscribe(progressDisplay)

	// Build specs map from resources
	specs := make(map[string]map[string]interface{})
	for _, r := range resources {
		switch r.Kind {
		case "Infrastructure":
			if r.Metadata.Name == infraRef {
				specs["Infrastructure"] = r.Spec
			}
		case "Orchestrator":
			if r.Metadata.Name == orchRef {
				specs["Orchestrator"] = r.Spec
			}
		case "Observability":
			if r.Metadata.Name == obsRef {
				specs["Observability"] = r.Spec
			}
		case "DevEx":
			if r.Metadata.Name == devexRef {
				specs["DevEx"] = r.Spec
			}
		}
	}

	// Register engines based on available specs
	if spec, ok := specs["Infrastructure"]; ok {
		provider := getProvider(spec, "terraform")
		infraEngine := engine.NewInfrastructureEngine(provider)
		coordinator.RegisterEngine(infraEngine)
	}
	if spec, ok := specs["Orchestrator"]; ok {
		provider := getProvider(spec, "argocd")
		orchEngine := engine.NewOrchestratorEngine(provider)
		coordinator.RegisterEngine(orchEngine)
	}
	if spec, ok := specs["Observability"]; ok {
		provider := getProvider(spec, "prometheus-stack")
		obsEngine := engine.NewObservabilityEngine(provider)
		coordinator.RegisterEngine(obsEngine)
	}
	if spec, ok := specs["DevEx"]; ok {
		provider := getProvider(spec, "backstage")
		devexEngine := engine.NewDevExEngine(provider)
		coordinator.RegisterEngine(devexEngine)
	}

	// Start progress display
	progressDisplay.Start()
	defer progressDisplay.Stop()

	// Execute
	err := coordinator.Apply(context.Background(), specs)
	if err != nil {
		return fmt.Errorf("apply failed: %w", err)
	}

	return nil
}

// getProvider extracts the provider from a spec, returning the default if not found
func getProvider(spec map[string]interface{}, defaultProvider string) string {
	if provider, ok := spec["provider"].(string); ok && provider != "" {
		return provider
	}
	return defaultProvider
}
