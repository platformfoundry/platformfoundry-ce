# Plugins Guide

PlatformFoundry's plugin architecture allows extending functionality with custom providers.

## Plugin Types

| Type | Description | Examples |
|------|-------------|----------|
| `infrastructure` | Infrastructure provisioning | Terraform, Pulumi, CDK |
| `orchestrator` | Container orchestration | ArgoCD, Flux, Kubernetes |
| `observability` | Monitoring & alerting | Prometheus, Datadog, Grafana |
| `devex` | Developer experience | Backstage, GitHub Actions |

## Built-in Plugins

```bash
# List installed plugins
pf plugin list

# Output:
# NAME        TYPE            VERSION  STATUS
# aws         infrastructure  1.0.0    enabled (builtin)
# kubernetes  orchestrator    1.0.0    enabled (builtin)
# terraform   infrastructure  1.0.0    enabled (builtin)
# pulumi      infrastructure  1.0.0    enabled
# argocd      orchestrator    1.0.0    enabled
# prometheus  observability   1.0.0    enabled
# backstage   devex           1.0.0    enabled
```

### Builtin Provider Details

Platform Foundry includes three production-ready builtin providers:

#### AWS Provider

Direct AWS infrastructure provisioning using AWS SDK v2.

**Supported Resources:**
- `VPC` - Virtual Private Cloud with CIDR configuration
- `Subnet` - Public/private subnets with availability zone
- `SecurityGroup` - Security groups with ingress/egress rules
- `RDS` - PostgreSQL/MySQL databases
- `ElastiCache` - Redis/Memcached clusters
- `S3` - Object storage buckets

```yaml
spec:
  infrastructure:
    provider: aws
    config:
      region: us-east-1
      resources:
        - type: RDS
          name: my-db
          engine: postgres
          instanceClass: db.t3.micro
```

#### Kubernetes Provider

Native Kubernetes resource management using client-go.

**Supported Resources:**
- `Deployment` - Application deployments with rolling updates
- `Service` - ClusterIP, NodePort, LoadBalancer services
- `ConfigMap` - Configuration data
- `Secret` - Sensitive data (base64 encoded)
- `Ingress` - HTTP routing with TLS support
- `HorizontalPodAutoscaler` - Auto-scaling based on metrics

```yaml
spec:
  orchestrator:
    provider: kubernetes
    config:
      namespace: production
```

#### Terraform Provider

Adapter wrapping the Terraform CLI for any Terraform-managed resource.

**Features:**
- Parses JSON output from `terraform plan` and `terraform apply`
- Supports any Terraform provider ecosystem
- State management via Terraform backends

```yaml
spec:
  infrastructure:
    provider: terraform
    config:
      workingDir: ./terraform
      varFile: prod.tfvars
```

## Installing Plugins

### From Registry

```bash
# Install from official registry
pf plugin install datadog

# Install specific version
pf plugin install datadog@2.1.0
```

### From URL

```bash
pf plugin install https://example.com/plugins/custom-plugin.tar.gz
```

### From Local Path

```bash
pf plugin install ./my-plugin/
```

## Plugin Management

```bash
# Enable plugin
pf plugin enable datadog

# Disable plugin
pf plugin disable datadog

# Update plugin
pf plugin update datadog

# Remove plugin
pf plugin remove datadog

# Show plugin info
pf plugin info terraform
```

## Creating Plugins

### Plugin Structure

```
my-plugin/
├── plugin.yaml          # Plugin manifest
├── main.go              # Plugin entrypoint
├── handlers/
│   ├── apply.go
│   ├── plan.go
│   └── destroy.go
└── templates/
    └── default.yaml
```

### Plugin Manifest

```yaml
# plugin.yaml
apiVersion: platformfoundry.io/v1
kind: Plugin
metadata:
  name: my-provider
  version: 1.0.0
spec:
  type: infrastructure
  description: Custom infrastructure provider
  author: Platform Team
  license: Apache-2.0

  # Plugin capabilities
  capabilities:
    - apply
    - plan
    - destroy
    - import

  # Configuration schema
  config:
    schema:
      type: object
      properties:
        apiUrl:
          type: string
          description: API endpoint URL
        apiKey:
          type: string
          secret: true
          description: API key for authentication
      required:
        - apiUrl
        - apiKey

  # Dependencies
  dependencies:
    - name: terraform
      version: ">=1.5.0"
```

### Plugin Interface

```go
// main.go
package main

import (
    "context"

    "github.com/platformfoundry/pf-ce/pkg/plugin"
)

type MyPlugin struct {
    config Config
}

func (p *MyPlugin) Name() string {
    return "my-provider"
}

func (p *MyPlugin) Version() string {
    return "1.0.0"
}

func (p *MyPlugin) Type() plugin.Type {
    return plugin.TypeInfrastructure
}

func (p *MyPlugin) Init(ctx context.Context, config map[string]interface{}) error {
    // Initialize plugin
    return nil
}

func (p *MyPlugin) Apply(ctx context.Context, resources []plugin.Resource) (*plugin.Result, error) {
    // Apply resources
    return &plugin.Result{Status: "success"}, nil
}

func (p *MyPlugin) Plan(ctx context.Context, resources []plugin.Resource) (*plugin.Plan, error) {
    // Generate plan
    return &plugin.Plan{}, nil
}

func (p *MyPlugin) Destroy(ctx context.Context, resources []plugin.Resource) error {
    // Destroy resources
    return nil
}

func main() {
    plugin.Serve(&MyPlugin{})
}
```

### Build Plugin

```bash
# Build plugin
go build -o my-plugin ./main.go

# Package plugin
pf plugin package ./my-plugin --output=my-plugin.tar.gz
```

## Plugin Configuration

### In Platform Definition

```yaml
apiVersion: platformfoundry.io/v1
kind: Platform
metadata:
  name: my-platform
spec:
  infrastructure:
    provider: my-provider
    config:
      apiUrl: https://api.example.com
      apiKey: ${secrets.my-provider-key}
```

### Global Plugin Config

```yaml
# ~/.platformfoundry/plugins.yaml
plugins:
  my-provider:
    enabled: true
    config:
      apiUrl: https://api.example.com
      timeout: 30s
```

## Plugin Hooks

```go
func (p *MyPlugin) PreApply(ctx context.Context, resources []plugin.Resource) error {
    // Run before apply
    return nil
}

func (p *MyPlugin) PostApply(ctx context.Context, result *plugin.Result) error {
    // Run after apply
    return nil
}

func (p *MyPlugin) OnError(ctx context.Context, err error) error {
    // Handle errors
    return nil
}
```

## Plugin Events

```go
func (p *MyPlugin) Subscribe() []plugin.EventType {
    return []plugin.EventType{
        plugin.EventResourceCreated,
        plugin.EventResourceUpdated,
        plugin.EventResourceDeleted,
    }
}

func (p *MyPlugin) HandleEvent(ctx context.Context, event plugin.Event) error {
    switch event.Type {
    case plugin.EventResourceCreated:
        // Handle creation
    case plugin.EventResourceUpdated:
        // Handle update
    }
    return nil
}
```

## Plugin Testing

```go
// main_test.go
package main

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestApply(t *testing.T) {
    p := &MyPlugin{}
    err := p.Init(context.Background(), map[string]interface{}{
        "apiUrl": "https://test.example.com",
        "apiKey": "test-key",
    })
    assert.NoError(t, err)

    result, err := p.Apply(context.Background(), []plugin.Resource{
        {Name: "test-resource"},
    })
    assert.NoError(t, err)
    assert.Equal(t, "success", result.Status)
}
```

Run tests:

```bash
go test ./...
```

## Publishing Plugins

### To Registry

```bash
# Login to registry
pf registry login

# Publish plugin
pf plugin publish my-plugin.tar.gz
```

### Plugin Registry

```yaml
# Configure custom registry
registry:
  plugins:
    - url: https://plugins.example.com
      auth:
        type: token
        token: ${secrets.registry-token}
```

## Debugging Plugins

```bash
# Enable debug mode
pf plugin debug my-provider

# View plugin logs
pf plugin logs my-provider

# Test plugin locally
pf plugin test my-provider --input=test-resources.yaml
```

## Security

### Plugin Signing

```bash
# Sign plugin
pf plugin sign my-plugin.tar.gz --key=signing-key.pem

# Verify signature
pf plugin verify my-plugin.tar.gz
```

### Plugin Permissions

```yaml
spec:
  permissions:
    - network:outbound
    - filesystem:read
    - secrets:read
```

## Next Steps

- [CLI Reference](../reference/cli.md) - Full CLI documentation
- [API Reference](../reference/api.md) - Plugin API details
