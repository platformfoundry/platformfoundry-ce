# Platforms Guide

A Platform is the top-level resource in PlatformFoundry that defines your entire infrastructure stack.

## Platform Structure

```yaml
apiVersion: platformfoundry.io/v1
kind: Platform
metadata:
  name: production-platform
  environment: production
  labels:
    team: platform-engineering
    cost-center: engineering
spec:
  infrastructure:
    # Infrastructure provisioning
  orchestrator:
    # Container orchestration
  observability:
    # Monitoring and alerting
  devex:
    # Developer experience tools
```

## Metadata

### Required Fields

| Field | Description |
|-------|-------------|
| `name` | Unique platform identifier |
| `environment` | Target environment (dev/staging/prod) |

### Optional Fields

| Field | Description |
|-------|-------------|
| `labels` | Key-value pairs for organization |
| `annotations` | Additional metadata |
| `owner` | Team or user responsible |

## Infrastructure Section

Define infrastructure provisioning:

```yaml
spec:
  infrastructure:
    provider: terraform  # terraform, pulumi, crossplane, awscdk
    config:
      version: ">=1.5.0"
      backend:
        type: s3
        bucket: terraform-state
        region: us-east-1
        dynamodb_table: terraform-locks
      workdir: ./infrastructure
      variables:
        environment: production
        region: us-east-1
```

### Supported Providers

| Provider | Description |
|----------|-------------|
| `terraform` | HashiCorp Terraform |
| `pulumi` | Pulumi IaC |
| `crossplane` | Kubernetes-native IaC |
| `awscdk` | AWS Cloud Development Kit |

## Orchestrator Section

Configure container orchestration:

```yaml
spec:
  orchestrator:
    type: argocd  # kubernetes, argocd, flux
    config:
      namespace: argocd
      server: https://argocd.example.com
      project: default
    gitops:
      enabled: true
      repo: https://github.com/org/platform-gitops
      branch: main
      path: environments/production
```

### Supported Orchestrators

| Type | Description |
|------|-------------|
| `kubernetes` | Direct Kubernetes deployment |
| `argocd` | Argo CD GitOps |
| `flux` | Flux CD GitOps |

## Observability Section

Set up monitoring and alerting:

```yaml
spec:
  observability:
    provider: prometheus  # prometheus, datadog, opentelemetry
    config:
      retention: 30d
      alertmanager:
        enabled: true
        receivers:
          - name: slack
            slack_configs:
              - channel: "#alerts"
    dashboards:
      provider: grafana
      config:
        url: https://grafana.example.com
```

## DevEx Section

Developer experience tooling:

```yaml
spec:
  devex:
    catalog:
      provider: backstage
      config:
        url: https://backstage.example.com
    ci:
      provider: github-actions
      config:
        org: my-org
```

## Multi-Environment Platforms

Use environment overlays:

```yaml
# base/platform.yaml
apiVersion: platformfoundry.io/v1
kind: Platform
metadata:
  name: my-platform
spec:
  infrastructure:
    provider: terraform
    config:
      backend:
        type: s3
        bucket: terraform-state
---
# overlays/production/patch.yaml
metadata:
  environment: production
spec:
  infrastructure:
    config:
      variables:
        instance_type: m5.xlarge
        replicas: 3
```

Apply with:

```bash
pf apply -f base/platform.yaml -f overlays/production/patch.yaml
```

## Platform Lifecycle

```bash
# Create/Update
pf apply -f platform.yaml

# View status
pf get platforms
pf describe platform my-platform

# View history
pf history platform my-platform

# Rollback
pf rollback platform my-platform --revision=2

# Delete
pf delete platform my-platform
```

## Dependencies

Define dependencies between components:

```yaml
spec:
  infrastructure:
    provider: terraform
  orchestrator:
    type: argocd
    dependsOn:
      - infrastructure  # Wait for infra before deploying
  observability:
    provider: prometheus
    dependsOn:
      - orchestrator  # Deploy after orchestrator
```

## Health Checks

Configure health validation:

```yaml
spec:
  healthChecks:
    enabled: true
    interval: 5m
    checks:
      - name: api-health
        type: http
        endpoint: https://api.example.com/health
        timeout: 30s
      - name: db-connection
        type: tcp
        host: db.example.com
        port: 5432
```

## Next Steps

- [Environments Guide](environments.md)
- [GitOps Guide](gitops.md)
- [Multi-Cloud Tutorial](../tutorials/multi-cloud.md)
