# Tutorial: Your First Platform

This tutorial walks you through creating a complete platform from scratch.

## Prerequisites

- PlatformFoundry CLI installed
- Docker Desktop (for local Kubernetes)
- kubectl installed
- ~30 minutes

## What You'll Build

A development platform with:

- Local Kubernetes cluster (Kind)
- Basic monitoring (Prometheus)
- Sample application deployment

## Step 1: Create Project Directory

```bash
mkdir my-first-platform
cd my-first-platform
```

## Step 2: Initialize PlatformFoundry

```bash
pf init
```

This creates the default configuration structure.

## Step 3: Create Local Kubernetes Cluster

We'll use Kind (Kubernetes in Docker) for local development:

```bash
# Create Kind cluster
pf demo create --type=kind --name=dev-cluster

# Verify cluster
kubectl cluster-info --context kind-dev-cluster
```

## Step 4: Define the Environment

Create `environment.yaml`:

```yaml
apiVersion: platformfoundry.io/v1
kind: Environment
metadata:
  name: development
spec:
  type: development

  cluster:
    name: dev-cluster
    context: kind-dev-cluster

  variables:
    LOG_LEVEL: debug
    REPLICAS: "1"

  quotas:
    cpu: "4"
    memory: "8Gi"
    pods: "50"
```

Apply it:

```bash
pf apply -f environment.yaml
```

## Step 5: Define the Platform

Create `platform.yaml`:

```yaml
apiVersion: platformfoundry.io/v1
kind: Platform
metadata:
  name: dev-platform
  environment: development
  labels:
    team: platform-engineering
    purpose: learning
spec:
  # Infrastructure (using local backend for simplicity)
  infrastructure:
    provider: terraform
    config:
      backend:
        type: local
        path: ./terraform-state
      workdir: ./infrastructure

  # Orchestrator (direct Kubernetes)
  orchestrator:
    type: kubernetes
    config:
      context: kind-dev-cluster
      namespace: platform

  # Observability (Prometheus)
  observability:
    provider: prometheus
    config:
      retention: 7d
      scrapeInterval: 30s
```

## Step 6: Create Infrastructure

Create `infrastructure/main.tf`:

```hcl
# infrastructure/main.tf

terraform {
  required_version = ">= 1.0.0"
}

# Create namespace for platform components
resource "kubernetes_namespace" "platform" {
  metadata {
    name = "platform"
    labels = {
      "managed-by" = "platformfoundry"
    }
  }
}

# Create namespace for applications
resource "kubernetes_namespace" "apps" {
  metadata {
    name = "apps"
    labels = {
      "managed-by" = "platformfoundry"
    }
  }
}

output "namespaces" {
  value = {
    platform = kubernetes_namespace.platform.metadata[0].name
    apps     = kubernetes_namespace.apps.metadata[0].name
  }
}
```

Create `infrastructure/providers.tf`:

```hcl
# infrastructure/providers.tf

provider "kubernetes" {
  config_path    = "~/.kube/config"
  config_context = "kind-dev-cluster"
}
```

## Step 7: Validate Configuration

```bash
pf validate -f platform.yaml
```

Expected output:

```
✓ Platform definition is valid
✓ Environment 'development' exists
✓ Infrastructure provider 'terraform' available
✓ Orchestrator 'kubernetes' configured
✓ Observability provider 'prometheus' available
```

## Step 8: Plan Changes

```bash
pf plan -f platform.yaml
```

Review the execution plan:

```
Planning platform: dev-platform

Infrastructure changes:
  + kubernetes_namespace.platform
  + kubernetes_namespace.apps

Orchestrator changes:
  + Create namespace 'platform'

Observability changes:
  + Deploy Prometheus to 'platform' namespace

Plan: 4 to add, 0 to change, 0 to destroy.
```

## Step 9: Apply Platform

```bash
pf apply -f platform.yaml --auto-approve
```

Watch the deployment:

```
Applying platform: dev-platform

[1/3] Infrastructure
  ✓ Terraform initialized
  ✓ kubernetes_namespace.platform created
  ✓ kubernetes_namespace.apps created

[2/3] Orchestrator
  ✓ Kubernetes context verified
  ✓ Namespace 'platform' ready

[3/3] Observability
  ✓ Prometheus deployed
  ✓ ServiceMonitor configured

Platform ready! (2m 15s)
```

## Step 10: Verify Deployment

```bash
# Check platform status
pf get platforms

# Output:
# NAME          ENVIRONMENT  STATUS  AGE
# dev-platform  development  Ready   2m

# Check Kubernetes resources
kubectl get pods -n platform

# Output:
# NAME                          READY   STATUS    RESTARTS   AGE
# prometheus-server-xxx         1/1     Running   0          2m
```

## Step 11: Deploy Sample Application

Create `app.yaml`:

```yaml
apiVersion: platformfoundry.io/v1
kind: Service
metadata:
  name: hello-world
  namespace: apps
spec:
  image: nginx:alpine
  replicas: 2
  ports:
    - name: http
      port: 80
  healthCheck:
    path: /
    port: 80
```

Deploy it:

```bash
pf apply -f app.yaml
```

Verify:

```bash
kubectl get pods -n apps

# Output:
# NAME                           READY   STATUS    RESTARTS   AGE
# hello-world-xxx                1/1     Running   0          30s
# hello-world-yyy                1/1     Running   0          30s
```

## Step 12: Access Prometheus

```bash
# Port-forward Prometheus
kubectl port-forward -n platform svc/prometheus-server 9090:80

# Open http://localhost:9090 in browser
```

## Step 13: Check Platform Health

```bash
pf describe platform dev-platform
```

Output:

```
Name:         dev-platform
Environment:  development
Status:       Ready
Age:          5m

Components:
  Infrastructure:  Ready (Terraform)
  Orchestrator:    Ready (Kubernetes)
  Observability:   Ready (Prometheus)

Resources:
  Namespaces:      2 (platform, apps)
  Deployments:     2
  Services:        2
  Pods:            3/3 Running

Health Checks:
  ✓ Prometheus scraping
  ✓ All pods healthy
```

## Step 14: Make Changes

Update replicas in `app.yaml`:

```yaml
spec:
  replicas: 3  # Changed from 2
```

Apply:

```bash
pf apply -f app.yaml
```

## Step 15: Clean Up

When done, destroy the platform:

```bash
# Delete application
pf delete -f app.yaml

# Delete platform
pf delete -f platform.yaml

# Delete Kind cluster
pf demo delete --name=dev-cluster
```

## What's Next?

- [Multi-Cloud Tutorial](multi-cloud.md) - Deploy across cloud providers
- [GitOps Tutorial](../guides/gitops.md) - Set up Git-based workflows
- [Secrets Guide](../guides/secrets.md) - Manage secrets securely

## Troubleshooting

### Terraform init fails

```bash
# Check Terraform is installed
terraform version

# Manually initialize
cd infrastructure && terraform init
```

### Pods not starting

```bash
# Check pod events
kubectl describe pod -n platform <pod-name>

# Check logs
kubectl logs -n platform <pod-name>
```

### Prometheus not scraping

```bash
# Check ServiceMonitor
kubectl get servicemonitor -n platform

# Check Prometheus targets
# Open http://localhost:9090/targets
```
