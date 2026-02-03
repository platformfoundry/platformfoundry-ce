# Platform Foundry

**Orchestrate Complete Internal Developer Platforms with Simple YAML**

Platform Foundry is a declarative platform orchestration tool that provisions and manages complete Internal Developer Platforms (IDPs) using a plugin-based architecture.

[![Go Version](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Edition](https://img.shields.io/badge/Edition-Community-green.svg)]()

> **This is the open-source Community Edition.** For Enterprise features (cost tracking, DORA metrics, SSO), see [Platform Foundry Enterprise](https://platformfoundry.io/enterprise).

## Quick Start

```bash
# Build
go run build.go build

# Run local demo (creates Kind cluster + full platform)
./bin/pf demo

# Or apply your own configuration
./bin/pf apply -f platform.yaml
```

See [Quick Start Guide](docs/QUICKSTART.md) for detailed instructions.

## What It Does

Platform Foundry orchestrates **complete platforms** from a single YAML configuration:

- **Infrastructure** - VPCs, Kubernetes clusters, registries (Terraform, Pulumi, CDK)
- **Orchestration** - GitOps and CD pipelines (ArgoCD, Flux)
- **Observability** - Metrics, logs, traces (Prometheus, Grafana, OpenTelemetry)
- **Developer Portal** - Self-service platform (Backstage)
- **Security** - Authentication, TLS, secrets management

```yaml
apiVersion: platformfoundry.io/v1
kind: Platform
metadata:
  name: my-platform
spec:
  components:
    infrastructure: terraform-aws
    orchestrator: argocd-gitops
    observability: prometheus-stack
    devex: backstage-portal
```

## Key Features

- **Declarative YAML** - Kubernetes-style resource definitions
- **Plugin Architecture** - 15+ production plugins for infrastructure, GitOps, observability
- **Builtin Providers** - AWS, Kubernetes, and Terraform plugins out of the box
- **Workload Abstraction** - Developer-friendly workload specs that auto-provision infrastructure
- **Auto-Integration** - Components automatically configure each other
- **Multi-Tenant** - Organizations, environments, RBAC
- **Approval Workflows** - Multi-approver workflows with conditions
- **Ephemeral Environments** - PR-based preview environments
- **Enterprise Security** - JWT/OAuth/SAML auth, TLS, Vault/AWS secrets, audit logs
- **State Management** - S3 backend with DynamoDB locking
- **REST API** - Full API with SSE event streaming

## Edition Comparison

| Feature | Community (This) | Enterprise |
|---------|-----------------|------------|
| CLI + YAML Orchestration | Yes | Yes |
| All Plugins | Yes | Yes |
| Approval Workflows | Yes | Yes |
| Ephemeral Environments | Yes | Yes |
| Basic RBAC | Yes | Yes |
| Basic Cost Estimation | Yes | Yes |
| Real-time Cost Tracking | - | Yes |
| DORA Metrics | - | Yes |
| Visual Resource Graph | - | Yes |
| Enterprise SSO | - | Yes |
| AI Recommendations | - | Yes |
| SLA Support | - | Yes |

## CLI Commands

```bash
pf apply -f platform.yaml    # Apply configuration
pf plan -f platform.yaml     # Preview changes
pf get all                   # List resources
pf delete platform my-plat   # Delete resources
pf jobs list                 # View async jobs
pf auth login                # Authenticate
pf org use acme-corp         # Switch organization
pf env create --pr 123       # Create PR environment
pf workflow approve <id>     # Approve workflow

# Workload commands (developer-friendly abstractions)
pf workload apply -f app.yaml     # Deploy workload with infrastructure
pf workload plan -f app.yaml      # Preview workload resources
pf workload generate -f app.yaml  # Generate K8s manifests
pf workload list                  # List deployed workloads
```

## Documentation

| Document | Description |
|----------|-------------|
| [Quick Start](docs/QUICKSTART.md) | Get running in 5 minutes |
| [Architecture](docs/ARCHITECTURE.md) | System design and components |
| [Development](docs/DEVELOPMENT.md) | Plugin development, engines, testing |

## Project Structure

```
platformfoundry-ce/
├── cmd/pf/              # CLI entry point
├── internal/
│   ├── auth/            # Authentication (JWT, OAuth, SAML)
│   ├── cli/             # 40+ CLI commands
│   ├── cost/            # Cost estimation
│   ├── engine/          # Component engines + coordinator
│   ├── environment/     # Environment resolver + ephemeral
│   ├── mock/            # Mock plugin for testing
│   ├── orchestrator/    # Core orchestration
│   ├── plugins/         # Plugin implementations
│   ├── policy/          # OPA policy engine
│   ├── secrets/         # Secrets management
│   ├── service/         # Service catalog
│   ├── state/           # State backends
│   ├── workflow/        # Approval workflows
│   └── ...              # 37 packages total
├── pkg/
│   ├── plugin/          # Plugin interface
│   └── types/           # Resource types
└── examples/            # YAML configurations
```

## Contributing

We welcome contributions! Platform Foundry is community-driven.

```bash
git clone https://github.com/platformfoundry/pf-ce.git
cd platformfoundry-ce
go mod download
go test ./...
go run build.go build
./bin/pf demo
```

### Ways to Contribute

- Report bugs and request features via [GitHub Issues](https://github.com/platformfoundry/pf-ce/issues)
- Submit pull requests for bug fixes and features
- Write and improve documentation
- Create plugins for new providers
- Share your Platform Foundry configurations

## License

Apache License 2.0 - see [LICENSE](LICENSE)

## Community

- [GitHub Issues](https://github.com/platformfoundry/pf-ce/issues)
- [GitHub Discussions](https://github.com/platformfoundry/pf-ce/discussions)

## Enterprise

Need enterprise features? [Platform Foundry Enterprise](https://platformfoundry.io/enterprise) includes:
- Real-time cost tracking and optimization
- DORA metrics and platform analytics
- Visual resource graph
- Enterprise SSO (Okta, Azure AD, Google)
- SLA-backed support

Contact: sales@platformfoundry.io
