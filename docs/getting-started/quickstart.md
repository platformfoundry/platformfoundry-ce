# Quickstart

Create and deploy your first platform in minutes.

## Step 1: Initialize Configuration

```bash
pf init
```

This creates a default configuration directory with sample files.

## Step 2: Create a Platform Definition

Create `platform.yaml`:

```yaml
apiVersion: platformfoundry.io/v1
kind: Platform
metadata:
  name: demo-platform
  environment: development
spec:
  infrastructure:
    provider: terraform
    config:
      backend: local
      workdir: ./terraform

  orchestrator:
    type: kubernetes
    config:
      context: kind-demo

  observability:
    provider: prometheus
    config:
      retention: 7d
```

## Step 3: Validate Configuration

```bash
pf validate -f platform.yaml
```

Output:
```
✓ Platform definition is valid
✓ All required fields present
✓ Provider configurations valid
```

## Step 4: Plan Changes

```bash
pf plan -f platform.yaml
```

This shows what changes will be made without applying them.

## Step 5: Apply Configuration

```bash
pf apply -f platform.yaml
```

Output:
```
Applying platform: demo-platform
  ✓ Infrastructure provisioned
  ✓ Orchestrator configured
  ✓ Observability stack deployed

Platform ready!
```

## Step 6: Check Status

```bash
pf get platforms
```

Output:
```
NAME           ENVIRONMENT  STATUS   AGE
demo-platform  development  Ready    2m
```

## Step 7: View Details

```bash
pf describe platform demo-platform
```

## Clean Up

```bash
pf delete -f platform.yaml
```

## Next Steps

- [Configuration Reference](configuration.md) - Deep dive into configuration
- [Platforms Guide](../guides/platforms.md) - Advanced platform features
- [First Platform Tutorial](../tutorials/first-platform.md) - Detailed walkthrough
