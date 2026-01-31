# Environments Guide

Environments allow you to manage multiple deployment targets with shared configurations.

## Environment Resource

```yaml
apiVersion: platformfoundry.io/v1
kind: Environment
metadata:
  name: production
spec:
  type: production  # development, staging, production

  # Cloud provider configuration
  cloud:
    provider: aws
    region: us-east-1
    account: "123456789012"

  # Kubernetes cluster
  cluster:
    name: prod-cluster
    context: arn:aws:eks:us-east-1:123456789012:cluster/prod

  # Environment-specific variables
  variables:
    LOG_LEVEL: warn
    REPLICAS: "3"
    ENABLE_DEBUG: "false"

  # Resource quotas
  quotas:
    cpu: "100"
    memory: "200Gi"
    pods: "500"

  # Promotion rules
  promotion:
    from: staging
    approval:
      required: true
      approvers:
        - platform-team
        - security-team
```

## Environment Types

### Development

```yaml
apiVersion: platformfoundry.io/v1
kind: Environment
metadata:
  name: development
spec:
  type: development
  cloud:
    provider: aws
    region: us-west-2
  variables:
    LOG_LEVEL: debug
    REPLICAS: "1"
```

### Staging

```yaml
apiVersion: platformfoundry.io/v1
kind: Environment
metadata:
  name: staging
spec:
  type: staging
  cloud:
    provider: aws
    region: us-east-1
  promotion:
    from: development
    auto: true  # Auto-promote from dev
```

### Production

```yaml
apiVersion: platformfoundry.io/v1
kind: Environment
metadata:
  name: production
spec:
  type: production
  cloud:
    provider: aws
    region: us-east-1
  promotion:
    from: staging
    approval:
      required: true
```

## Managing Environments

```bash
# List environments
pf get environments

# Create environment
pf apply -f environment.yaml

# Describe environment
pf describe environment production

# Delete environment
pf delete environment development
```

## Environment Variables

### Setting Variables

```yaml
spec:
  variables:
    DATABASE_URL: postgres://db.example.com:5432/app
    REDIS_URL: redis://cache.example.com:6379
    API_KEY: ${secrets.api-key}  # Reference from secrets
```

### Using Variables in Platform

```yaml
apiVersion: platformfoundry.io/v1
kind: Platform
metadata:
  name: my-platform
  environment: production  # Links to environment
spec:
  infrastructure:
    config:
      variables:
        db_url: ${env.DATABASE_URL}
        replicas: ${env.REPLICAS}
```

## Environment Inheritance

Create base environments and extend them:

```yaml
# base-environment.yaml
apiVersion: platformfoundry.io/v1
kind: Environment
metadata:
  name: base
spec:
  cloud:
    provider: aws
  variables:
    LOG_FORMAT: json
    METRICS_ENABLED: "true"
---
# production-environment.yaml
apiVersion: platformfoundry.io/v1
kind: Environment
metadata:
  name: production
spec:
  inherits: base
  type: production
  cloud:
    region: us-east-1
  variables:
    LOG_LEVEL: warn  # Override base
```

## Promotion Workflow

### Manual Promotion

```bash
# Promote from staging to production
pf promote --from=staging --to=production

# With specific version
pf promote --from=staging --to=production --version=v1.2.3
```

### Automatic Promotion

```yaml
spec:
  promotion:
    from: staging
    auto: true
    conditions:
      - allTestsPassing: true
      - minAge: 24h  # Must be in staging for 24h
```

### Approval Gates

```yaml
spec:
  promotion:
    from: staging
    approval:
      required: true
      approvers:
        - platform-team
      timeout: 48h
      notifications:
        slack: "#platform-approvals"
```

## Environment Drift Detection

```bash
# Check for drift
pf drift detect --environment=production

# Show drift details
pf drift show --environment=production

# Remediate drift
pf drift remediate --environment=production
```

## Environment Comparison

```bash
# Compare environments
pf diff environments staging production

# Compare specific resources
pf diff environments staging production --resource=configmap/app-config
```

## Resource Quotas

Limit resources per environment:

```yaml
spec:
  quotas:
    cpu: "50"           # Total CPU cores
    memory: "100Gi"     # Total memory
    pods: "200"         # Max pods
    services: "50"      # Max services
    secrets: "100"      # Max secrets
    configmaps: "100"   # Max configmaps
```

## Network Policies

Define network isolation:

```yaml
spec:
  network:
    isolation: strict  # none, moderate, strict
    allowedNamespaces:
      - monitoring
      - logging
    egressRules:
      - to:
          - ipBlock:
              cidr: 10.0.0.0/8
```

## Next Steps

- [Secrets Management](secrets.md)
- [GitOps Integration](gitops.md)
