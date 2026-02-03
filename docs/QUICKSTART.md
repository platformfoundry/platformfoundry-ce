# Platform Foundry - Quick Start Guide

Get started with Platform Foundry in under 5 minutes! This guide will help you create your first complete Internal Developer Platform locally.

## Table of Contents

- [Try Platform Foundry in 5 Minutes](#try-platform-foundry-in-5-minutes)
- [Understanding Platform Foundry](#understanding-platform-foundry)
- [Demo Modes](#demo-modes)
- [Next Steps](#next-steps)

---

## Try Platform Foundry in 5 Minutes

### Prerequisites

- **Docker Desktop** installed and running
- **8GB RAM** available
- That's it! No cloud account needed for the demo.

### Step 1: Install Platform Foundry

**Build from Source**
```bash
git clone https://github.com/platformfoundry/pf-ce.git
cd PlatformFoundry
go run build.go build

# Binary created at ./bin/pf
# Add to PATH (optional)
export PATH=$PATH:$(pwd)/bin

# Verify installation
./bin/pf --version
```

### Step 2: Run the Demo

```bash
pf demo
```

This single command will:
1. ✅ Check prerequisites (Docker, kubectl, kind)
2. ✅ Create a local Kubernetes cluster using kind (30 seconds)
3. ✅ Install ArgoCD for GitOps (1 minute)
4. ✅ Install Prometheus + Grafana for monitoring (2 minutes)
5. ✅ Install Backstage developer portal (2 minutes)
6. ✅ Configure all integrations automatically (30 seconds)

**Total time: ~6 minutes**

### Step 3: Explore Your Platform

After the demo completes, you'll see:

```
===============================================
✅ Platform ready in 5m 47s!
===============================================

🌐 Access your platform:

  🎯 Backstage:   http://localhost:7007
  📊 Grafana:     http://localhost:3000
       User: admin / Password: admin
  🚀 ArgoCD:      http://localhost:8080
       User: admin / Password: (see below)
  📈 Prometheus:  http://localhost:9090

💡 Note: Services are accessible via kubectl port-forward
   Run in separate terminals:
   kubectl port-forward -n backstage svc/backstage 7007:7007
   kubectl port-forward -n monitoring svc/grafana 3000:3000
   kubectl port-forward -n argocd svc/argocd-server 8080:8080
   kubectl port-forward -n monitoring svc/prometheus 9090:9090
```

### Step 4: Verify Integrations

**Open Grafana** (http://localhost:3000)
- Login: admin / admin
- Navigate to: Data Sources → You'll see Prometheus **automatically configured** ✨
- Navigate to: Dashboards → See Kubernetes dashboard **already loaded** ✨

**Open Backstage** (http://localhost:7007)
- See **all tools integrated** in the developer portal ✨
- ArgoCD, Grafana, and Prometheus links **auto-configured** ✨

**This is the Platform Foundry magic:**
- ✅ No manual configuration needed
- ✅ Components auto-discover each other
- ✅ Platform ready in minutes, not days

### Step 5: Clean Up

When you're done exploring:

```bash
pf demo clean
```

This removes the kind cluster and all resources.

---

## Understanding Platform Foundry

### What is Platform Foundry?

Platform Foundry is a **platform orchestration tool** that helps you create complete Internal Developer Platforms from simple YAML configuration.

**Think of it as Terraform, but for entire platforms** - not just infrastructure.

### Key Concepts

#### 1. **Plugin-Based Architecture**

Like Terraform providers, Platform Foundry uses plugins:

```
Platform Foundry Core (Orchestration)
    ↓
Plugin Interface
    ↓
├── Infrastructure Plugins (AWS, GCP, Azure, Terraform)
├── Orchestrator Plugins (ArgoCD, Flux)
├── Observability Plugins (Prometheus, Datadog)
└── DevEx Plugins (Backstage, Port)
```

**You don't need to know how each tool works** - the plugins handle that!

#### 2. **Automatic Integration**

Platform Foundry's **integration engine** automatically connects components:

```yaml
# You just define what you want:
observability:
  prometheus: enabled
  grafana: enabled

# Platform Foundry automatically:
# ✅ Configures Grafana → Prometheus datasource
# ✅ Loads Kubernetes dashboards
# ✅ Sets up alerting
```

#### 3. **Declarative Configuration**

Define your entire platform in YAML:

```yaml
apiVersion: platformfoundry.io/v1
kind: Platform
metadata:
  name: my-platform
spec:
  infrastructure:
    provider: aws
    region: us-east-1

  components:
    orchestrator: argocd
    observability: prometheus-stack
    devex: backstage
```

**That's it!** Platform Foundry handles:
- Component installation
- Network configuration
- Integration setup
- Access management

---

## Demo Modes

Platform Foundry supports multiple demo/testing modes:

### 1. Local Mode (Full Demo)

Creates real Kubernetes cluster locally using kind:

```bash
pf demo
```

**Use when:**
- You want to see the full platform working
- You have Docker installed
- You have 5-10 minutes

### 2. Quick Mode (Lightweight)

Faster demo with only Prometheus + Grafana:

```bash
pf demo quick
```

**Use when:**
- You want faster setup (2-3 minutes)
- You have limited resources
- You want to test monitoring only

### 3. Mock/Simulation Mode

Simulates everything without deploying resources:

```bash
pf apply -f examples/mock-platform.yaml --mode=mock

# Or use plan to see what would be created
pf plan -f examples/mock-platform.yaml
```

**Use when:**
- You want to test YAML configurations
- You want to see what would be created
- You're learning Platform Foundry
- Running in CI/CD pipelines

**Example output:**
```
📦 Simulating platform creation...

✓ Would create AWS VPC in us-east-1
✓ Would provision EKS cluster (3 nodes, t3.large)
✓ Would install ArgoCD via Helm
✓ Would configure Prometheus monitoring
✓ Would create Backstage portal

Estimated cost: $450/month
Estimated setup time: ~15 minutes
All integrations would be configured automatically
```

---

## Next Steps

### Try with Real Cloud

Once you understand how Platform Foundry works, try it with your cloud:

```yaml
# my-aws-platform.yaml
apiVersion: platformfoundry.io/v1
kind: Platform
metadata:
  name: my-company-platform
spec:
  infrastructure:
    provider: aws
    region: us-west-2
    cloud:
      vpc:
        cidr: "10.0.0.0/16"
    cluster:
      provider: eks
      version: "1.27"
      nodeGroups:
        - name: system
          instanceType: t3.large
          minSize: 3
          maxSize: 10

  components:
    orchestrator:
      provider: argocd
      gitops:
        repo: https://github.com/my-org/platform-config
        branch: main

    observability:
      provider: prometheus-stack
      prometheus:
        retention: 30d
        storage: 100Gi
      grafana:
        dashboards:
          - kubernetes-cluster
          - applications

    devex:
      provider: backstage
      intelligentGeneration: true
```

**Apply it:**
```bash
# First, see what would be created
pf plan -f my-aws-platform.yaml

# Apply (creates real resources)
pf apply -f my-aws-platform.yaml

# Track progress
pf jobs list
pf jobs logs <job-id>

# When done
pf get all
```

**Platform Foundry will:**
1. Provision VPC and EKS cluster (~10 minutes)
2. Install ArgoCD, Prometheus, Grafana, Backstage (~5 minutes)
3. Configure all integrations automatically
4. Provide access URLs and credentials

**Total time: ~15 minutes** vs. 2-3 weeks manually!

### Customize Your Platform

Edit the YAML to match your needs:

```yaml
# Add more components
components:
  orchestrator: argocd
  observability: prometheus-stack
  devex: backstage
  cicd: jenkins          # ← Add CI/CD
  mesh: istio            # ← Add service mesh
  security: vault        # ← Add secrets management
```

### Learn More

- **Architecture**: See [ARCHITECTURE.md](ARCHITECTURE.md) for system design
- **Development**: See [DEVELOPMENT.md](DEVELOPMENT.md) for plugin development and testing
- **Examples**: Browse `examples/` directory for more configurations

### Community

- **GitHub**: https://github.com/platformfoundry/pf-ce
- **Issues**: https://github.com/platformfoundry/pf-ce/issues
- **Discussions**: https://github.com/platformfoundry/pf-ce/discussions

---

## Troubleshooting

### Demo won't start

```bash
# Check prerequisites
pf demo

# Output will show what's missing:
🔍 Checking prerequisites...
  ❌ Docker: docker daemon not running
  ✓ kubectl
  ✓ kind

Missing prerequisites:
  ❌ Docker: docker daemon not running - please start Docker Desktop
```

**Fix**: Start Docker Desktop and try again.

### Port already in use

If you see "port already in use" errors:

```bash
# Check what's using the port
lsof -i :9090  # or 3000, 8080, 7007

# Clean up old demo
pf demo clean

# Try again
pf demo
```

### Cluster creation fails

```bash
# Remove existing cluster
kind delete cluster --name pf-demo

# Try again
pf demo
```

### Still stuck?

Open an issue: https://github.com/platformfoundry/pf-ce/issues

---

## The Platform Foundry Advantage

| Without Platform Foundry | With Platform Foundry |
|-------------------------|----------------------|
| 2-3 weeks to set up platform | ✅ 15 minutes to complete platform |
| Manual integration configuration | ✅ Automatic integrations |
| Complex networking | ✅ Network auto-configured |
| Requires deep expertise in 8+ tools | ✅ Just write YAML |
| Tribal knowledge required | ✅ Declarative, version-controlled |
| Hard to replicate across environments | ✅ One YAML works everywhere |

**Platform Foundry brings Terraform-like simplicity to entire platform management.**

---

## What's Next?

1. ✅ **You've tried the demo** - You understand how it works
2. 📝 **Customize the YAML** - Edit `examples/local-demo.yaml` to your needs
3. ☁️ **Deploy to cloud** - Try with real AWS/GCP/Azure
4. 🔌 **Explore plugins** - See what's available in the ecosystem
5. 🛠️ **Build your own plugin** - Extend Platform Foundry for your tools

Happy platforming! 🚀
