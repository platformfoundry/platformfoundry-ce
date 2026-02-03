# Development Guide

This guide covers plugin development, component engines, mocking, and testing for Platform Foundry.

## Table of Contents

- [Plugin Development](#plugin-development)
- [Component Engines](#component-engines)
- [Mocking and Testing](#mocking-and-testing)
- [Contributing](#contributing)

---

## Plugin Development

### Plugin Architecture

Plugins implement provider-specific logic for resource types:

```
Platform Foundry Core
        ↓
  Plugin Interface
        ↓
├── Infrastructure Plugins (AWS, GCP, Terraform)
├── Orchestrator Plugins (ArgoCD, Flux)
├── Observability Plugins (Prometheus, Datadog)
└── DevEx Plugins (Backstage, Port)
```

### Plugin Interface

Every plugin must implement:

```go
type Plugin interface {
    Name() string                                    // Plugin name
    Type() string                                    // Resource type
    Version() string                                 // Plugin version
    Validate(spec map[string]interface{}) error     // Validate resource spec
    Plan(spec map[string]interface{}) (*Plan, error) // Generate execution plan
    Apply(spec map[string]interface{}) (*Result, error) // Provision resource
    Delete(name string) error                        // Destroy resource
    Status(name string) (*Status, error)             // Get resource status
}
```

### Creating a Plugin

#### 1. Directory Structure

```
pf-pipeline-myprovider/
├── cmd/
│   └── pf-pipeline-myprovider/
│       └── main.go              # Plugin entrypoint
├── internal/
│   ├── client/
│   │   └── client.go            # Provider API client
│   └── plugin.go                # Plugin implementation
├── plugin.yaml                  # Plugin metadata
├── go.mod
└── README.md
```

#### 2. Plugin Metadata (plugin.yaml)

```yaml
apiVersion: platformfoundry.io/v1
kind: PluginMetadata
metadata:
  name: pf-pipeline-myprovider
  version: 1.0.0
  description: My custom CI/CD provider integration

spec:
  resourceType: Pipeline
  provider: myprovider

  supports:
    - validate
    - plan
    - apply
    - delete
    - status

  requires:
    - resourceType: Cluster
      optional: false

  schema:
    url:
      type: string
      required: true
    credentials:
      type: object
      required: true
```

#### 3. Implement Plugin

```go
package internal

import (
    "errors"
    "github.com/platformfoundry/pf/pkg/plugin"
)

type MyProviderPlugin struct {
    client *ProviderClient
}

func New() *MyProviderPlugin {
    return &MyProviderPlugin{}
}

func (p *MyProviderPlugin) Name() string    { return "myprovider" }
func (p *MyProviderPlugin) Type() string    { return "Pipeline" }
func (p *MyProviderPlugin) Version() string { return "1.0.0" }

func (p *MyProviderPlugin) Validate(spec map[string]interface{}) error {
    url, ok := spec["url"].(string)
    if !ok || url == "" {
        return errors.New("url is required")
    }
    return nil
}

func (p *MyProviderPlugin) Apply(spec map[string]interface{}) (*plugin.Result, error) {
    url := spec["url"].(string)
    client, err := NewClient(url)
    if err != nil {
        return nil, err
    }

    // Provision resources...

    return &plugin.Result{
        Status:  "success",
        Message: "Pipeline provisioned",
    }, nil
}
```

### Building and Testing Plugins

```bash
# Build
go build -o pf-pipeline-myprovider cmd/pf-pipeline-myprovider/main.go

# Validate implementation
pf plugin validate .

# Install locally
pf plugin install ./pf-pipeline-myprovider

# Test
pf apply -f test-pipeline.yaml
```

---

## Component Engines

### Engine Types

Platform Foundry uses specialized engines for different platform concerns:

| Engine | Package | Responsibility |
|--------|---------|---------------|
| Infrastructure | `internal/engine` | VPCs, clusters, registries |
| Orchestrator | `internal/engine` | GitOps, deployments |
| Observability | `internal/engine` | Metrics, logs, traces |
| DevEx | `internal/engine` | Developer portals |
| Workflow | `internal/workflow` | Approvals, change windows |

### BaseEngine Interface

```go
type BaseEngine interface {
    Name() string
    Type() EngineType
    Priority() int

    Initialize(ctx context.Context, config EngineConfig) error
    Validate(spec map[string]interface{}) error
    Plan(ctx context.Context, spec map[string]interface{}) (*Plan, error)
    Apply(ctx context.Context, spec map[string]interface{}) (*Result, error)
    Destroy(ctx context.Context) error

    Status() EngineStatus
    Capabilities() []Capability
}
```

### Coordinator

The Coordinator orchestrates engine execution:

```go
coordinator := engine.NewCoordinator()

// Register engines
coordinator.RegisterEngine(engine.NewInfrastructureEngine("terraform"))
coordinator.RegisterEngine(engine.NewOrchestratorEngine("argocd"))
coordinator.RegisterEngine(engine.NewObservabilityEngine("prometheus"))

// Execute
result, err := coordinator.Execute(ctx, platformSpec)
```

### Orchestrator Service

The `orchestrator.Service` connects CLI/API to the coordinator with plugin management:

```go
import (
    "github.com/platformfoundry/pf-ce/internal/orchestrator"
    "github.com/platformfoundry/pf-ce/internal/plugin"
    "github.com/platformfoundry/pf-ce/internal/plugin/providers"
)

// Initialize plugin manager with builtin providers
pluginManager := plugin.NewManager()
providers.RegisterBuiltins(pluginManager)

// Create orchestrator service
svc := orchestrator.NewService(orchestrator.Config{
    Namespace: "production",
    Cloud:     "aws",
    Region:    "us-east-1",
}, pluginManager, stateBackend)

// Apply workload
result, err := svc.ApplyWorkload(ctx, workload, translationResult)

// Subscribe to events
svc.Subscribe(myEventListener)
```

#### Event Listening

Implement `engine.EventListener` to receive real-time updates:

```go
type MyListener struct{}

func (l *MyListener) OnEvent(event engine.EngineEvent) {
    fmt.Printf("[%s] %s: %s (%d%%)\n",
        event.Type, event.Component, event.Message, event.Progress)
}

svc.Subscribe(&MyListener{})
defer svc.Unsubscribe(&MyListener{})
```

### Dependency Graph

Engines respect dependency ordering:

```go
graph := engine.NewDependencyGraph()
graph.AddNode("infrastructure", nil)
graph.AddNode("orchestrator", []string{"infrastructure"})
graph.AddNode("observability", []string{"orchestrator"})

order := graph.TopologicalSort()
// Returns: ["infrastructure", "orchestrator", "observability"]
```

### Event Bus

Inter-engine communication:

```go
bus := engine.NewEventBus()

// Subscribe to events
bus.Subscribe(engine.EventTypeResourceCreated, func(e engine.Event) {
    log.Printf("Resource created: %s", e.Resource)
})

// Publish events
bus.Publish(engine.Event{
    Type:     engine.EventTypeResourceCreated,
    Source:   "infrastructure",
    Resource: "vpc-123",
})
```

---

## Mocking and Testing

### Mock Plugin

The MockPlugin enables testing without real infrastructure:

```go
import "github.com/platformfoundry/pf-ce/internal/mock"

mockPlugin := mock.NewMockPlugin(mock.MockConfig{
    Mode:           mock.MockModeInstant,  // instant, simulated, recorded
    SimulatedDelay: 100 * time.Millisecond,
    FailureRate:    0.0,
})
```

### Mock Modes

| Mode | Description | Use Case |
|------|-------------|----------|
| `instant` | Returns immediately | Unit tests |
| `simulated` | Adds realistic delays | Integration tests |
| `recorded` | Replays recorded responses | CI/CD pipelines |

### Using Mock Mode

```yaml
# examples/platform-mock.yaml
apiVersion: platformfoundry.io/v1
kind: Platform
metadata:
  name: test-platform
spec:
  mock:
    enabled: true
    mode: simulated
    delay: 100ms
```

```bash
pf apply -f examples/platform-mock.yaml --mock
```

### Writing Tests

#### Unit Tests

```go
func TestPluginValidate(t *testing.T) {
    plugin := NewMyPlugin()

    tests := []struct {
        name    string
        spec    map[string]interface{}
        wantErr bool
    }{
        {
            name: "valid spec",
            spec: map[string]interface{}{
                "url": "https://example.com",
            },
            wantErr: false,
        },
        {
            name:    "missing url",
            spec:    map[string]interface{}{},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := plugin.Validate(tt.spec)
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

#### Integration Tests

```go
func TestCoordinatorExecute(t *testing.T) {
    coordinator := engine.NewCoordinator()

    // Use mock engines
    mockInfra := engine.NewInfrastructureEngine("mock")
    coordinator.RegisterEngine(mockInfra)

    spec := map[string]interface{}{
        "Infrastructure": map[string]interface{}{
            "provider": "mock",
        },
    }

    result, err := coordinator.Execute(context.Background(), spec)
    if err != nil {
        t.Fatalf("Execute failed: %v", err)
    }

    if result.Status != engine.StatusSuccess {
        t.Errorf("expected success, got %s", result.Status)
    }
}
```

### Running Tests

```bash
# All tests
go test ./...

# Specific package
go test ./internal/engine/...

# With coverage
go test -cover ./...

# Verbose output
go test -v ./internal/workflow/...
```

---

## Contributing

### Development Setup

```bash
git clone https://github.com/platformfoundry/pf-ce.git
cd PlatformFoundry
go mod download
```

### Build

```bash
go run build.go build
# Binary: ./bin/pf
```

### Code Style

- Use `gofmt` for formatting
- Follow [Effective Go](https://go.dev/doc/effective_go) guidelines
- Add tests for new functionality
- Keep functions focused and small

### Pull Request Process

1. Fork the repository
2. Create feature branch: `git checkout -b feature/my-feature`
3. Make changes with tests
4. Run tests: `go test ./...`
5. Commit: `git commit -m "feat: add my feature"`
6. Push: `git push origin feature/my-feature`
7. Open Pull Request

### Commit Message Format

```
type: subject

body (optional)
```

Types: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`

### Adding a New Engine

1. Create engine file in `internal/engine/`
2. Implement `BaseEngine` interface
3. Register in coordinator
4. Add tests in `*_test.go`
5. Update documentation

### Adding a New Plugin

1. Create plugin directory in `internal/plugins/`
2. Implement `Plugin` interface
3. Add plugin metadata
4. Register in plugin manager
5. Add integration tests

---

## SDK Reference

### Plugin SDK (`pkg/plugin/`)

- `interface.go` - Plugin interface definition
- `types.go` - Common types (Plan, Result, Status)
- `validation.go` - Spec validation helpers

### Builtin Providers (`internal/plugin/providers/`)

- `aws/plugin.go` - AWS infrastructure plugin (VPC, RDS, ElastiCache, S3)
- `kubernetes/plugin.go` - Kubernetes orchestrator plugin (Deployment, Service, HPA)
- `terraform/plugin.go` - Terraform CLI adapter plugin
- `registry.go` - Builtin provider registration

### Engine SDK (`internal/engine/`)

- `base.go` - BaseEngine interface
- `coordinator.go` - Engine coordinator
- `dependency.go` - Dependency graph
- `events.go` - Event bus

### Orchestrator Service (`internal/orchestrator/`)

- `service.go` - Service connecting workloads to coordinator
- `engines.go` - InfrastructureEngine and KubernetesEngine wrappers

### API Handlers (`internal/api/handlers/`)

- `base.go` - Handler base with JSON/Error response helpers
- `workloads.go` - Workload CRUD handlers
- `events.go` - SSE streaming and event listing
- `routes.go` - Route registration

### Type Definitions (`pkg/types/`)

- `resource.go` - Resource types
- `platform.go` - Platform configuration
- `metadata.go` - Resource metadata
- `workload.go` - Workload specification
