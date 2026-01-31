# Configuration Reference

Complete reference for all PlatformFoundry configuration options.

## Platform Resource

```yaml
apiVersion: platformfoundry.io/v1
kind: Platform
metadata:
  name: string                    # Required: Unique name
  environment: string             # Required: Target environment
  labels:                         # Optional: Key-value labels
    key: value
  annotations:                    # Optional: Annotations
    key: value
spec:
  infrastructure:                 # Infrastructure configuration
    provider: string              # terraform|pulumi|crossplane|awscdk
    config: object                # Provider-specific config
  orchestrator:                   # Orchestrator configuration
    type: string                  # kubernetes|argocd|flux
    config: object                # Orchestrator-specific config
    gitops:                       # GitOps settings
      enabled: boolean
      repo: string
      branch: string
      path: string
  observability:                  # Observability configuration
    provider: string              # prometheus|datadog|opentelemetry
    config: object                # Provider-specific config
  devex:                          # Developer experience
    catalog:                      # Service catalog
      provider: string            # backstage
      config: object
    ci:                           # CI/CD integration
      provider: string            # github-actions|gitlab-ci
      config: object
```

## Environment Resource

```yaml
apiVersion: platformfoundry.io/v1
kind: Environment
metadata:
  name: string                    # Required: Unique name
spec:
  type: string                    # development|staging|production
  cloud:
    provider: string              # aws|gcp|azure
    region: string
    account: string
  cluster:
    name: string
    context: string
  variables:                      # Environment variables
    KEY: value
  quotas:                         # Resource quotas
    cpu: string
    memory: string
    pods: string
  promotion:
    from: string                  # Source environment
    auto: boolean                 # Auto-promote
    approval:
      required: boolean
      approvers: [string]
```

## Infrastructure Providers

### Terraform

```yaml
infrastructure:
  provider: terraform
  config:
    version: string               # Terraform version constraint
    backend:
      type: string                # local|s3|gcs|azurerm
      bucket: string              # S3/GCS bucket name
      region: string
      key: string                 # State file path
      dynamodb_table: string      # Lock table (S3 backend)
    workdir: string               # Working directory
    varFiles: [string]            # Variable files
    variables:                    # Inline variables
      key: value
    parallelism: integer          # Parallel resource operations
    refresh: boolean              # Refresh state before plan
    targets: [string]             # Target specific resources
```

### Pulumi

```yaml
infrastructure:
  provider: pulumi
  config:
    stack: string                 # Stack name
    backend:
      type: string                # local|s3|pulumi
      url: string
    workdir: string
    config:                       # Pulumi config
      key: value
    secrets:
      provider: string            # passphrase|awskms|gcpkms
```

### Crossplane

```yaml
infrastructure:
  provider: crossplane
  config:
    providerConfigs: [string]     # Provider configurations
    compositionRef:
      name: string
      namespace: string
```

## Orchestrator Configuration

### Kubernetes

```yaml
orchestrator:
  type: kubernetes
  config:
    context: string               # Kubeconfig context
    namespace: string             # Default namespace
    kubeconfig: string            # Path to kubeconfig
```

### ArgoCD

```yaml
orchestrator:
  type: argocd
  config:
    server: string                # ArgoCD server URL
    namespace: string             # ArgoCD namespace
    project: string               # ArgoCD project
    application:
      name: string
      namespace: string
      syncPolicy:
        automated:
          prune: boolean
          selfHeal: boolean
        syncOptions: [string]
```

### Flux

```yaml
orchestrator:
  type: flux
  config:
    namespace: string             # Flux namespace
    source:
      kind: string                # GitRepository|HelmRepository
      name: string
      namespace: string
    kustomization:
      name: string
      path: string
      interval: string
      prune: boolean
```

## Observability Configuration

### Prometheus

```yaml
observability:
  provider: prometheus
  config:
    retention: string             # Data retention period
    scrapeInterval: string        # Scrape interval
    alertmanager:
      enabled: boolean
      config: object              # Alertmanager config
    serviceMonitors: [object]     # ServiceMonitor definitions
```

### Datadog

```yaml
observability:
  provider: datadog
  config:
    apiKey: string                # API key (use secrets ref)
    site: string                  # Datadog site
    clusterName: string
    logs:
      enabled: boolean
    apm:
      enabled: boolean
```

## State Backends

### Local

```yaml
state:
  backend: local
  config:
    path: string                  # State file path
```

### S3

```yaml
state:
  backend: s3
  config:
    bucket: string
    region: string
    key: string
    encrypt: boolean
    dynamodb_table: string        # For locking
```

### DynamoDB

```yaml
state:
  backend: dynamodb
  config:
    table: string
    region: string
```

## Secrets Configuration

### Local

```yaml
secrets:
  provider: local
  config:
    path: string
    encryption: string            # aes-256-gcm
```

### HashiCorp Vault

```yaml
secrets:
  provider: vault
  config:
    address: string
    namespace: string
    auth:
      method: string              # token|kubernetes|aws|approle
      role: string
      mountPath: string
    mount: string
    path: string
```

### AWS Secrets Manager

```yaml
secrets:
  provider: aws
  config:
    region: string
    prefix: string
    kmsKeyId: string
```

## Authentication

### Token

```yaml
auth:
  method: token
  tokenPath: string
```

### SAML

```yaml
auth:
  method: saml
  config:
    idpMetadataURL: string
    spEntityID: string
    acsURL: string
```

### API Key

```yaml
auth:
  method: api-key
  keyPath: string
```

## Logging

```yaml
logging:
  level: string                   # debug|info|warn|error
  format: string                  # text|json
  file: string                    # Log file path
  maxSize: integer                # Max file size (MB)
  maxBackups: integer             # Max backup files
  maxAge: integer                 # Max age (days)
```

## Telemetry

```yaml
telemetry:
  enabled: boolean
  endpoint: string                # Custom endpoint
  sampleRate: float               # 0.0-1.0
```

## Resource Variables

Reference variables in configuration:

```yaml
# Environment variables
${env.VARIABLE_NAME}

# Secrets
${secrets.secret-name}

# Environment-specific
${environment.variables.KEY}

# Platform outputs
${platform.outputs.key}

# File contents
${file:/path/to/file}
```

## Validation Rules

| Field | Validation |
|-------|------------|
| `metadata.name` | `^[a-z][a-z0-9-]*$`, max 63 chars |
| `metadata.environment` | Must match defined environment |
| `spec.infrastructure.provider` | Must be valid provider |
| `spec.orchestrator.type` | Must be valid orchestrator |
