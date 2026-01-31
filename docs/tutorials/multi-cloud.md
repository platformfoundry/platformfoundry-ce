# Tutorial: Multi-Cloud Setup

Deploy a platform across multiple cloud providers for high availability and disaster recovery.

## Prerequisites

- PlatformFoundry CLI installed
- AWS CLI configured
- GCP CLI configured (gcloud)
- Terraform >= 1.5.0
- ~1 hour

## Architecture Overview

```
                    ┌─────────────────────┐
                    │   Load Balancer     │
                    │   (Global/Multi)    │
                    └──────────┬──────────┘
                               │
           ┌───────────────────┼───────────────────┐
           │                   │                   │
    ┌──────▼──────┐    ┌──────▼──────┐    ┌──────▼──────┐
    │    AWS      │    │    GCP      │    │   Azure     │
    │  us-east-1  │    │ us-central1 │    │  eastus     │
    │  (Primary)  │    │ (Secondary) │    │  (DR)       │
    └─────────────┘    └─────────────┘    └─────────────┘
```

## Step 1: Project Setup

```bash
mkdir multi-cloud-platform
cd multi-cloud-platform
pf init
```

## Step 2: Define Environments

Create `environments/aws.yaml`:

```yaml
apiVersion: platformfoundry.io/v1
kind: Environment
metadata:
  name: aws-primary
spec:
  type: production
  cloud:
    provider: aws
    region: us-east-1
    account: "123456789012"
  variables:
    CLOUD_PROVIDER: aws
    REGION: us-east-1
    PRIORITY: primary
```

Create `environments/gcp.yaml`:

```yaml
apiVersion: platformfoundry.io/v1
kind: Environment
metadata:
  name: gcp-secondary
spec:
  type: production
  cloud:
    provider: gcp
    region: us-central1
    project: my-gcp-project
  variables:
    CLOUD_PROVIDER: gcp
    REGION: us-central1
    PRIORITY: secondary
```

Apply environments:

```bash
pf apply -f environments/
```

## Step 3: Create Base Platform

Create `base/platform.yaml`:

```yaml
apiVersion: platformfoundry.io/v1
kind: Platform
metadata:
  name: multi-cloud-platform
spec:
  # Common infrastructure settings
  infrastructure:
    provider: terraform
    config:
      version: ">=1.5.0"
      backend:
        type: s3
        bucket: platform-state-${env.CLOUD_PROVIDER}
        region: ${env.REGION}
        key: platform/terraform.tfstate

  # Kubernetes orchestration
  orchestrator:
    type: argocd
    gitops:
      enabled: true
      repo: https://github.com/org/platform-config
      branch: main

  # Cross-cloud observability
  observability:
    provider: datadog
    config:
      site: datadoghq.com
      clusterName: ${env.CLOUD_PROVIDER}-cluster
      logs:
        enabled: true
      apm:
        enabled: true
```

## Step 4: AWS Infrastructure

Create `infrastructure/aws/main.tf`:

```hcl
# infrastructure/aws/main.tf

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.region
}

variable "region" {
  default = "us-east-1"
}

variable "cluster_name" {
  default = "multi-cloud-aws"
}

# VPC
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "5.0.0"

  name = "${var.cluster_name}-vpc"
  cidr = "10.0.0.0/16"

  azs             = ["${var.region}a", "${var.region}b", "${var.region}c"]
  private_subnets = ["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"]
  public_subnets  = ["10.0.101.0/24", "10.0.102.0/24", "10.0.103.0/24"]

  enable_nat_gateway = true
  single_nat_gateway = false

  tags = {
    Environment = "production"
    ManagedBy   = "platformfoundry"
  }
}

# EKS Cluster
module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "19.0.0"

  cluster_name    = var.cluster_name
  cluster_version = "1.28"

  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnets

  eks_managed_node_groups = {
    default = {
      min_size     = 2
      max_size     = 10
      desired_size = 3

      instance_types = ["m5.large"]
    }
  }
}

output "cluster_endpoint" {
  value = module.eks.cluster_endpoint
}

output "cluster_name" {
  value = module.eks.cluster_name
}
```

## Step 5: GCP Infrastructure

Create `infrastructure/gcp/main.tf`:

```hcl
# infrastructure/gcp/main.tf

terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

variable "project_id" {
  description = "GCP project ID"
}

variable "region" {
  default = "us-central1"
}

variable "cluster_name" {
  default = "multi-cloud-gcp"
}

# VPC
resource "google_compute_network" "vpc" {
  name                    = "${var.cluster_name}-vpc"
  auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "subnet" {
  name          = "${var.cluster_name}-subnet"
  ip_cidr_range = "10.1.0.0/16"
  region        = var.region
  network       = google_compute_network.vpc.id

  secondary_ip_range {
    range_name    = "pods"
    ip_cidr_range = "10.2.0.0/16"
  }

  secondary_ip_range {
    range_name    = "services"
    ip_cidr_range = "10.3.0.0/16"
  }
}

# GKE Cluster
resource "google_container_cluster" "primary" {
  name     = var.cluster_name
  location = var.region

  network    = google_compute_network.vpc.name
  subnetwork = google_compute_subnetwork.subnet.name

  # We manage node pools separately
  remove_default_node_pool = true
  initial_node_count       = 1

  ip_allocation_policy {
    cluster_secondary_range_name  = "pods"
    services_secondary_range_name = "services"
  }
}

resource "google_container_node_pool" "primary_nodes" {
  name       = "${var.cluster_name}-node-pool"
  location   = var.region
  cluster    = google_container_cluster.primary.name
  node_count = 3

  node_config {
    machine_type = "e2-standard-4"

    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform"
    ]

    labels = {
      env       = "production"
      managedby = "platformfoundry"
    }
  }

  autoscaling {
    min_node_count = 2
    max_node_count = 10
  }
}

output "cluster_endpoint" {
  value = google_container_cluster.primary.endpoint
}

output "cluster_name" {
  value = google_container_cluster.primary.name
}
```

## Step 6: Create Platform Overlays

Create `overlays/aws/kustomization.yaml`:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - ../../base/platform.yaml

patches:
  - patch: |
      - op: replace
        path: /metadata/environment
        value: aws-primary
      - op: add
        path: /spec/infrastructure/config/workdir
        value: ./infrastructure/aws
    target:
      kind: Platform
```

Create `overlays/gcp/kustomization.yaml`:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - ../../base/platform.yaml

patches:
  - patch: |
      - op: replace
        path: /metadata/environment
        value: gcp-secondary
      - op: add
        path: /spec/infrastructure/config/workdir
        value: ./infrastructure/gcp
    target:
      kind: Platform
```

## Step 7: Deploy to AWS

```bash
# Deploy AWS infrastructure
pf apply -f overlays/aws/ --environment=aws-primary

# Verify
pf get platforms --environment=aws-primary
```

## Step 8: Deploy to GCP

```bash
# Deploy GCP infrastructure
pf apply -f overlays/gcp/ --environment=gcp-secondary

# Verify
pf get platforms --environment=gcp-secondary
```

## Step 9: Configure Cross-Cloud Networking

Create `networking/cross-cloud.yaml`:

```yaml
apiVersion: platformfoundry.io/v1
kind: NetworkPolicy
metadata:
  name: cross-cloud-mesh
spec:
  type: service-mesh
  provider: istio

  clusters:
    - name: aws-primary
      endpoint: ${platforms.aws.cluster_endpoint}
    - name: gcp-secondary
      endpoint: ${platforms.gcp.cluster_endpoint}

  mesh:
    mtls:
      enabled: true
    trustDomain: multi-cloud.local

  traffic:
    # Primary receives all traffic
    - destination: aws-primary
      weight: 100
    # Failover to secondary
    - destination: gcp-secondary
      weight: 0
      failover: true
```

## Step 10: Set Up Global Load Balancer

Create `loadbalancer/global.yaml`:

```yaml
apiVersion: platformfoundry.io/v1
kind: GlobalLoadBalancer
metadata:
  name: multi-cloud-lb
spec:
  provider: cloudflare  # or aws-global-accelerator

  backends:
    - name: aws-primary
      endpoint: ${platforms.aws.ingress_ip}
      weight: 100
      healthCheck:
        path: /health
        interval: 30s
    - name: gcp-secondary
      endpoint: ${platforms.gcp.ingress_ip}
      weight: 0
      healthCheck:
        path: /health
        interval: 30s

  failover:
    enabled: true
    threshold: 3
    interval: 10s
```

## Step 11: Deploy Application

Create `app/deployment.yaml`:

```yaml
apiVersion: platformfoundry.io/v1
kind: Application
metadata:
  name: multi-cloud-app
spec:
  image: myorg/app:latest
  replicas:
    aws-primary: 3
    gcp-secondary: 2

  resources:
    requests:
      cpu: "500m"
      memory: "512Mi"
    limits:
      cpu: "1"
      memory: "1Gi"

  env:
    - name: CLOUD_PROVIDER
      value: ${env.CLOUD_PROVIDER}
    - name: REGION
      value: ${env.REGION}
```

Deploy:

```bash
pf apply -f app/
```

## Step 12: Verify Multi-Cloud Setup

```bash
# Check all platforms
pf get platforms

# Output:
# NAME                  ENVIRONMENT    STATUS  REGION
# multi-cloud-platform  aws-primary    Ready   us-east-1
# multi-cloud-platform  gcp-secondary  Ready   us-central1

# Check application status
pf get applications

# Test failover
pf failover test --from=aws-primary --to=gcp-secondary --dry-run
```

## Step 13: Monitor Across Clouds

```bash
# View unified metrics
pf metrics --all-environments

# View cross-cloud logs
pf logs --all-environments --follow

# Health dashboard
pf dashboard
```

## Clean Up

```bash
# Delete application
pf delete -f app/

# Delete GCP platform
pf delete -f overlays/gcp/ --environment=gcp-secondary

# Delete AWS platform
pf delete -f overlays/aws/ --environment=aws-primary

# Delete environments
pf delete -f environments/
```

## Best Practices

1. **State Management** - Use separate state backends per cloud
2. **Secrets** - Use cloud-native secrets (AWS Secrets Manager, GCP Secret Manager)
3. **Networking** - Implement proper network segmentation
4. **Monitoring** - Use unified observability (Datadog, Grafana Cloud)
5. **Cost** - Monitor costs across all providers
6. **Compliance** - Ensure data residency requirements

## Next Steps

- [CI/CD Integration](cicd.md)
- [Disaster Recovery](../guides/environments.md#promotion-workflow)
- [Cost Management](../enterprise/cost-tracking.md)
