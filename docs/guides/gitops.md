# GitOps Guide

PlatformFoundry provides native GitOps support for declarative, Git-based platform management.

## Overview

GitOps principles in PlatformFoundry:

1. **Git as source of truth** - All configuration stored in Git
2. **Declarative** - Desired state defined in YAML
3. **Automated sync** - Changes automatically applied
4. **Audit trail** - Full history via Git commits

## Initialize GitOps

```bash
# Initialize with new repository
pf gitops init --repo=https://github.com/org/platform-config

# Initialize with existing repository
pf gitops init --repo=https://github.com/org/platform-config --existing
```

## Repository Structure

Recommended structure:

```
platform-config/
├── base/
│   ├── platform.yaml
│   ├── infrastructure/
│   │   └── main.tf
│   └── observability/
│       └── prometheus.yaml
├── environments/
│   ├── development/
│   │   ├── kustomization.yaml
│   │   └── patches/
│   ├── staging/
│   │   ├── kustomization.yaml
│   │   └── patches/
│   └── production/
│       ├── kustomization.yaml
│       └── patches/
└── .pf/
    └── config.yaml
```

## GitOps Configuration

### Platform Definition

```yaml
apiVersion: platformfoundry.io/v1
kind: Platform
metadata:
  name: my-platform
spec:
  orchestrator:
    type: argocd
    gitops:
      enabled: true
      repo: https://github.com/org/platform-config
      branch: main
      path: environments/production
      syncPolicy:
        automated:
          prune: true
          selfHeal: true
        syncOptions:
          - CreateNamespace=true
```

### GitOps Resource

```yaml
apiVersion: platformfoundry.io/v1
kind: GitOpsConfig
metadata:
  name: platform-gitops
spec:
  repository:
    url: https://github.com/org/platform-config
    branch: main
    auth:
      type: ssh  # ssh, token, none
      secretRef: git-credentials

  sync:
    interval: 5m
    prune: true
    selfHeal: true

  healthChecks:
    enabled: true
    timeout: 10m

  notifications:
    slack:
      channel: "#platform-updates"
      events:
        - sync-success
        - sync-failed
        - health-degraded
```

## CLI Commands

### Status

```bash
# View GitOps status
pf gitops status

# Output:
# REPO                                    BRANCH  SYNC STATUS  HEALTH
# github.com/org/platform-config          main    Synced       Healthy
```

### Sync

```bash
# Trigger manual sync
pf gitops sync

# Sync specific path
pf gitops sync --path=environments/production

# Force sync (ignore cache)
pf gitops sync --force
```

### Diff

```bash
# Show pending changes
pf gitops diff

# Show diff for specific environment
pf gitops diff --environment=production
```

### History

```bash
# View sync history
pf gitops history

# Output:
# REVISION  COMMIT   MESSAGE                    STATUS   TIME
# 5         abc123   Update replicas to 3       Synced   10m ago
# 4         def456   Add monitoring config      Synced   2h ago
# 3         ghi789   Initial platform setup     Synced   1d ago
```

### Rollback

```bash
# Rollback to previous revision
pf gitops rollback

# Rollback to specific revision
pf gitops rollback --revision=3
```

## Branching Strategies

### Environment Branches

```
main (production)
├── staging
└── development
```

```yaml
spec:
  gitops:
    branches:
      development: development
      staging: staging
      production: main
```

### Directory-based

```yaml
spec:
  gitops:
    branch: main
    paths:
      development: environments/development
      staging: environments/staging
      production: environments/production
```

## Pull Request Workflow

### Enable PR Preview

```yaml
spec:
  gitops:
    pullRequests:
      enabled: true
      preview:
        enabled: true
        ttl: 24h
      checks:
        - validate
        - plan
        - security-scan
```

### PR Commands

```bash
# Create PR for changes
pf gitops pr create --title="Update production replicas"

# Preview PR changes
pf gitops pr preview --pr=123

# Merge PR
pf gitops pr merge --pr=123
```

## Sync Waves

Control deployment order:

```yaml
apiVersion: platformfoundry.io/v1
kind: Platform
metadata:
  name: my-platform
  annotations:
    platformfoundry.io/sync-wave: "0"
spec:
  infrastructure:
    annotations:
      platformfoundry.io/sync-wave: "1"
  orchestrator:
    annotations:
      platformfoundry.io/sync-wave: "2"
  observability:
    annotations:
      platformfoundry.io/sync-wave: "3"
```

## Health Checks

```yaml
spec:
  gitops:
    healthChecks:
      - name: api-ready
        type: http
        endpoint: /health
        successThreshold: 3
      - name: pods-ready
        type: kubernetes
        resource: deployment/api
        condition: Available
```

## Notifications

### Slack

```yaml
spec:
  gitops:
    notifications:
      slack:
        channel: "#platform"
        events: [sync-success, sync-failed, health-degraded]
```

### Webhook

```yaml
spec:
  gitops:
    notifications:
      webhook:
        url: https://api.example.com/webhooks/gitops
        events: [all]
        headers:
          Authorization: Bearer ${secrets.webhook-token}
```

## ArgoCD Integration

```yaml
spec:
  orchestrator:
    type: argocd
    config:
      server: https://argocd.example.com
      project: default
      application:
        name: my-platform
        namespace: argocd
```

## Flux Integration

```yaml
spec:
  orchestrator:
    type: flux
    config:
      namespace: flux-system
      source:
        kind: GitRepository
        name: platform-config
      kustomization:
        name: platform
        path: ./environments/production
```

## Troubleshooting

```bash
# View sync logs
pf gitops logs

# Debug sync issues
pf gitops debug

# Force refresh
pf gitops refresh --hard
```

## Next Steps

- [Policies Guide](policies.md) - GitOps with policy enforcement
- [CI/CD Tutorial](../tutorials/cicd.md) - Integrate with CI/CD
