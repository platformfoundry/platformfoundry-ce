# PlatformFoundry

**Cloud-native platform orchestration framework**

PlatformFoundry is a unified platform engineering tool that orchestrates infrastructure, observability, developer experience, and GitOps workflows through a declarative YAML-based configuration.

## Features

- **Declarative Configuration** - Define your entire platform in YAML
- **Multi-Cloud Support** - AWS, GCP, Azure, and on-premises
- **Plugin Architecture** - Extensible with Terraform, Pulumi, ArgoCD, and more
- **GitOps Native** - Built-in Git-based workflow management
- **Policy Engine** - OPA-based policy enforcement
- **Secrets Management** - Vault, AWS Secrets Manager, local providers
- **Enterprise Ready** - SSO, RBAC, audit logging, cost tracking

## Quick Example

```yaml
apiVersion: platformfoundry.io/v1
kind: Platform
metadata:
  name: my-platform
  environment: production
spec:
  infrastructure:
    provider: terraform
    backend: s3
  orchestrator:
    type: argocd
    gitops:
      repo: https://github.com/org/platform-config
  observability:
    provider: prometheus
    dashboards: grafana
```

```bash
pf apply -f platform.yaml
```

## Getting Started

<div class="grid cards" markdown>

-   :material-download:{ .lg .middle } **Installation**

    ---

    Install PlatformFoundry CLI on your system

    [:octicons-arrow-right-24: Install](getting-started/installation.md)

-   :material-rocket-launch:{ .lg .middle } **Quickstart**

    ---

    Create your first platform in 5 minutes

    [:octicons-arrow-right-24: Quickstart](getting-started/quickstart.md)

-   :material-cog:{ .lg .middle } **Configuration**

    ---

    Learn about configuration options

    [:octicons-arrow-right-24: Configure](getting-started/configuration.md)

</div>

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    PlatformFoundry CLI                   │
├─────────────────────────────────────────────────────────┤
│  Parser  │  Policy Engine  │  State Manager  │  Secrets │
├─────────────────────────────────────────────────────────┤
│                     Plugin System                        │
├──────────┬──────────┬─────────────┬────────────────────┤
│  Infra   │  Orch    │   DevEx     │    Observability   │
│ Terraform│  ArgoCD  │  Backstage  │    Prometheus      │
│  Pulumi  │   Flux   │  GitHub CI  │    Grafana         │
│  CDK     │   K8s    │  GitLab CI  │    Datadog         │
└──────────┴──────────┴─────────────┴────────────────────┘
```

## Community

- [GitHub Repository](https://github.com/platformfoundry/pf-ce)
- [Issue Tracker](https://github.com/platformfoundry/pf-ce/issues)
- [Discussions](https://github.com/platformfoundry/pf-ce/discussions)

## License

PlatformFoundry Community Edition is released under the [Apache 2.0 License](https://github.com/platformfoundry/pf-ce/blob/main/LICENSE).
