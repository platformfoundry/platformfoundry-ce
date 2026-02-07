# Platform Foundry Architecture

This document provides a detailed overview of the Platform Foundry architecture.

## High-Level Overview

Platform Foundry is designed with a modular, plugin-based architecture. The core of the system is the **Orchestration Engine**, which takes declarative YAML files as input and uses plugins to provision and manage complete developer platforms.

```
                    ┌─────────────────────────────┐
                    │     CLI (30+ commands)      │
                    │  + Authentication Layer     │
                    └─────────────┬───────────────┘
                                  ↓
                    ┌─────────────────────────────┐
                    │  Orchestration Engine       │
                    │  - Parser & Validator       │
                    │  - Dependency Resolver      │
                    │  - Environment Resolver     │
                    │  - Execution Planner        │
                    │  - Job Queue                │
                    └─────────────┬───────────────┘
                                  ↓
                    ┌─────────────────────────────┐
                    │     Plugin Manager          │
                    │  + Config Validation        │
                    └─────────────┬───────────────┘
                                  ↓
                    ┌─────────────────────────────┐
                    │  Plugins (15+ production)   │
                    │  Terraform, ArgoCD, etc.    │
                    └─────────────────────────────┘

┌────────────────┐  ┌────────────────┐  ┌────────────────┐
│ Security Layer │  │ State Mgmt     │  │ Web Server     │
│ - Auth         │  │ - S3 Backend   │  │ - REST API     │
│ - TLS          │  │ - DynamoDB     │  │ - Dashboard    │
│ - Secrets      │  │ - Versioning   │  │                │
│ - Audit        │  │                │  │                │
└────────────────┘  └────────────────┘  └────────────────┘
```

## Component Engines

Platform Foundry uses specialized component engines for different platform concerns:

### Engine Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Coordinator                              │
│  - Orchestrates all engines                                  │
│  - Manages execution order via DependencyGraph               │
│  - Provides EventBus for inter-component communication       │
└─────────────────────────────────────────────────────────────┘
                              │
       ┌──────────────────────┼──────────────────────┐
       ↓                      ↓                      ↓
┌─────────────┐      ┌─────────────┐      ┌─────────────┐
│ Infra       │      │ Orchestrator│      │ Observability│
│ Engine      │      │ Engine      │      │ Engine       │
│ (Terraform) │      │ (ArgoCD)    │      │ (Prometheus) │
└─────────────┘      └─────────────┘      └─────────────┘
       ↓                      ↓                      ↓
┌─────────────┐      ┌─────────────┐      ┌─────────────┐
│ DevEx       │      │ Security    │      │ Workflow    │
│ Engine      │      │ Engine      │      │ Engine      │
│ (Backstage) │      │ (Vault)     │      │ (Approvals) │
└─────────────┘      └─────────────┘      └─────────────┘
```

### BaseEngine Interface

All engines implement the BaseEngine interface:

```go
type BaseEngine interface {
    Name() string
    Type() EngineType
    Priority() int

    // Lifecycle
    Initialize(ctx context.Context, config EngineConfig) error
    Validate(spec map[string]interface{}) error
    Plan(ctx context.Context, spec map[string]interface{}) (*Plan, error)
    Apply(ctx context.Context, spec map[string]interface{}) (*Result, error)
    Destroy(ctx context.Context) error

    // Status
    Status() EngineStatus
    Capabilities() []Capability
}
```

### Engine Types

| Engine | Responsibility | Default Provider |
|--------|---------------|------------------|
| Infrastructure | VPCs, clusters, registries | Terraform |
| Orchestrator | GitOps, deployments | ArgoCD |
| Observability | Metrics, logs, traces | Prometheus |
| DevEx | Developer portal | Backstage |
| Security | Auth, secrets, policies | Vault |
| Workflow | Approvals, change windows | Built-in |

## Core Components

### 1. CLI (`internal/cli`)

The Command Line Interface provides 30+ commands for platform management:

- **Platform ops**: `apply`, `plan`, `get`, `delete`, `describe`
- **Authentication**: `auth login`, `auth create-user`, `auth create-api-key`
- **Organizations**: `org create`, `org use`, `org list`
- **Jobs**: `jobs list`, `jobs logs`, `wait`
- **Security**: `tls`, `secrets`, `audit`

### 2. Orchestration Engine (`internal/orchestrator`)

The brain of Platform Foundry:

- **Parser**: Parses YAML into Go structs, validates structure
- **Dependency Resolver**: Builds DAG from `dependsOn` fields
- **Environment Resolver**: Merges base config with environment overrides
- **Execution Planner**: Creates step-by-step plan based on current state
- **Job Queue**: Async job execution with progress tracking
- **Orchestrator Service**: Bridges workloads to coordinator with plugin management

#### Orchestrator Service

The `orchestrator.Service` connects the CLI and API to the engine coordinator:

```go
type Service struct {
    config      Config
    coordinator *engine.Coordinator
    plugins     *plugin.Manager
    state       state.Backend
    eventBus    *engine.EventBus
}

// Key methods
func (s *Service) ApplyWorkload(ctx context.Context, wl *types.Workload, result *workload.TranslationResult) (*ApplyResult, error)
func (s *Service) GetWorkloadStatus(ctx context.Context, name string) (*WorkloadStatus, error)
func (s *Service) Subscribe(listener engine.EventListener)
```

### 3. Plugin Manager (`internal/plugin`)

Manages plugin lifecycle:

- Dynamic plugin loading from `internal/plugins`
- Consistent interface for engine-plugin interaction
- Configuration validation using plugin-defined schemas
- **Builtin Provider Registry**: Pre-registers AWS, Kubernetes, and Terraform plugins

### 4. Plugins (`internal/plugins`)

Plugins implement provider-specific logic:

**Infrastructure**: Terraform, Pulumi, AWS CDK, Crossplane
**Orchestration**: Kubernetes, ArgoCD, Flux
**Observability**: Prometheus, Grafana, OpenTelemetry, Datadog
**DevEx**: Backstage, GitHub Actions, GitLab CI

#### Builtin Providers (`internal/plugin/providers`)

Three production-ready plugins are built into Platform Foundry:

| Provider | Resources | Implementation |
|----------|-----------|----------------|
| **AWS** | VPC, Subnet, SecurityGroup, RDS, ElastiCache, S3 | AWS SDK v2 |
| **Kubernetes** | Deployment, Service, ConfigMap, Secret, Ingress, HPA | client-go |
| **Terraform** | Any Terraform-managed resource | CLI wrapper |

```go
// Register all builtin providers
import "github.com/platformfoundry/pf-ce/internal/plugin/providers"

pluginManager := plugin.NewManager()
providers.RegisterBuiltins(pluginManager)
```

### 5. Security Layer

Enterprise-grade security components:

- **Authentication** (`internal/auth`): JWT, API keys, OAuth 2.0, SAML
- **TLS** (`internal/tls`): Certificate management, Let's Encrypt
- **Secrets** (`internal/secrets`): Vault, AWS Secrets Manager, local encrypted
- **Audit** (`internal/audit`): Event logging for compliance
- **RBAC** (`internal/rbac`): Role-based access control

### 6. State Management (`internal/state`)

Platform state tracking:

- **Backends**: Local (bbolt) for dev, S3 for production
- **Locking**: File-based local, DynamoDB for distributed
- **Versioning**: State history for rollback capability

### 7. Workflow Engine (`internal/workflow`)

Approval workflows and change management:

- **Approvals**: Multi-approver with role requirements
- **Conditions**: Test passing, security scans, coverage thresholds
- **Change Windows**: Time-based deployment restrictions
- **Rollback**: Automatic rollback on failure

### 8. Environment Management (`internal/environment`)

Ephemeral and preview environments:

- **PR Environments**: Auto-create on pull request
- **Branch Environments**: Feature branch previews
- **TTL Management**: Auto-cleanup after expiration
- **PR Events**: Handle merge/close events

## Data Flow

A typical `pf apply` command:

1. CLI authenticates and uploads YAML to Web Server
2. Web Server creates job in Job Queue
3. Worker passes YAML to Orchestration Engine
4. Parser validates YAML structure
5. Environment Resolver applies overrides
6. Dependency Resolver builds resource graph
7. Planner compares desired vs current state
8. Engine iterates plan, calls Plugin Manager for each resource
9. Plugin Manager invokes appropriate Plugin
10. Plugin executes actions (e.g., `terraform apply`)
11. State Backend updated after each step
12. Job completes, results available via `pf jobs logs`

## Project Structure

```
PlatformFoundry/
├── cmd/pf/                  # CLI entry point
├── internal/
│   ├── audit/               # Audit logging
│   ├── auth/                # Authentication (JWT, OAuth, SAML, API keys)
│   ├── backup/              # Backup and restore
│   ├── cli/                 # 40+ CLI commands
│   ├── compliance/          # Compliance frameworks
│   ├── config/              # Security configuration
│   ├── context/             # Context management
│   ├── cost/                # Cost estimation
│   ├── demo/                # Demo mode
│   ├── engine/              # Component engines + coordinator
│   ├── environment/         # Environment resolver + ephemeral
│   ├── errors/              # Error handling
│   ├── generator/           # Code generation
│   ├── integration/         # Integration helpers
│   ├── intelligence/        # Recommendations engine
│   ├── jobs/                # Async job system
│   ├── lifecycle/           # Resource lifecycle
│   ├── mock/                # Mock plugin for testing
│   ├── orchestrator/        # Core orchestration engine
│   ├── parser/              # YAML parsing
│   ├── planner/             # Execution planning
│   ├── plugin/              # Plugin manager
│   ├── plugins/             # Plugin implementations
│   ├── policy/              # OPA policy engine
│   ├── progress/            # Progress tracking
│   ├── rbac/                # Access control
│   ├── registry/            # Plugin registry
│   ├── scaffold/            # Project scaffolding
│   ├── secrets/             # Secrets management
│   ├── service/             # Service catalog
│   ├── state/               # State backends
│   ├── store/               # Data storage
│   ├── telemetry/           # Telemetry collection
│   ├── tls/                 # TLS/certificate management
│   ├── web/                 # REST API + dashboard
│   └── workflow/            # Approval workflows
├── pkg/
│   ├── plugin/              # Plugin interface
│   └── types/               # Resource types
├── examples/                # YAML configurations
└── policies/                # OPA policy examples
```

## Integration Points

### Plugin Integration

Plugins expose capabilities through a standard interface:

```go
type Plugin interface {
    Name() string
    Type() string
    Version() string
    Validate(spec map[string]interface{}) error
    Plan(spec map[string]interface{}) (*Plan, error)
    Apply(spec map[string]interface{}) (*Result, error)
    Delete(name string) error
    Status(name string) (*Status, error)
}
```

### Event Bus

Inter-component communication via EventBus:

```go
type Event struct {
    Type      EventType
    Source    string
    Resource  string
    Status    string
    Message   string
    Timestamp time.Time
    Metadata  map[string]interface{}
}
```

### Secrets Integration

Secret references in YAML configs:

```yaml
spec:
  database:
    password: ${secret:vault:database/prod:password}
    # Resolved at apply time from configured provider
```

## Security Architecture

```
┌──────────────────────────────────────────────────────────┐
│                    Request Flow                          │
├──────────────────────────────────────────────────────────┤
│  Client → TLS → Auth Middleware → RBAC → Handler → Audit │
└──────────────────────────────────────────────────────────┘

Authentication Methods:
  - JWT tokens (HS256, RS256)
  - API keys with expiration
  - OAuth 2.0 (Google, GitHub, Okta)
  - SAML SSO for enterprise

Authorization:
  - Global roles (admin, operator, developer, viewer)
  - Organization-scoped roles
  - Resource-level permissions

Secrets:
  - HashiCorp Vault
  - AWS Secrets Manager
  - Local encrypted (AES-256)
```

## Comparison with Other Tools

| Feature | Platform Foundry | Terraform | Crossplane |
|---------|-----------------|-----------|------------|
| Infrastructure | Via plugins | Native | Native |
| GitOps | ArgoCD/Flux | Manual | Manual |
| Observability | Integrated | Separate | Separate |
| Auto-Integration | Yes | No | No |
| Multi-Tenancy | Native | Workspaces | Namespaces |
| Approval Workflows | Built-in | External | External |

Platform Foundry's advantage: Complete platform orchestration with security built-in.
