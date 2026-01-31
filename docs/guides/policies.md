# Policies Guide

PlatformFoundry uses Open Policy Agent (OPA) for policy enforcement across all platform operations.

## Overview

Policies enable you to:

- Enforce security standards
- Validate configurations before apply
- Control resource quotas
- Implement compliance requirements
- Gate deployments

## Policy Structure

```rego
# policies/security/no-privileged.rego
package platformfoundry.security

deny[msg] {
    input.spec.containers[_].securityContext.privileged == true
    msg := "Privileged containers are not allowed"
}
```

## Built-in Policies

PlatformFoundry includes default policies:

| Policy | Description |
|--------|-------------|
| `security/no-privileged` | Deny privileged containers |
| `security/no-root` | Deny running as root |
| `security/require-limits` | Require resource limits |
| `naming/conventions` | Enforce naming standards |
| `cost/quotas` | Enforce cost quotas |

List built-in policies:

```bash
pf policy list --builtin
```

## Creating Policies

### Policy Resource

```yaml
apiVersion: platformfoundry.io/v1
kind: Policy
metadata:
  name: require-labels
  labels:
    category: compliance
spec:
  enforcement: deny  # deny, warn, audit
  match:
    kinds:
      - Platform
      - Environment
  rego: |
    package platformfoundry.compliance

    required_labels := ["team", "cost-center", "environment"]

    deny[msg] {
      provided := {label | input.metadata.labels[label]}
      missing := required_labels - provided
      count(missing) > 0
      msg := sprintf("Missing required labels: %v", [missing])
    }
```

### From File

```bash
# Create policy from Rego file
pf policy create require-labels --from-file=./policies/require-labels.rego
```

## Enforcement Levels

| Level | Behavior |
|-------|----------|
| `deny` | Block operation if policy fails |
| `warn` | Allow but show warning |
| `audit` | Log only, no action |

```yaml
spec:
  enforcement: deny
```

## Policy Commands

```bash
# List policies
pf policy list

# Get policy details
pf policy get require-labels

# Test policy
pf policy test require-labels --input=platform.yaml

# Enable/disable policy
pf policy enable require-labels
pf policy disable require-labels

# Delete policy
pf policy delete require-labels
```

## Policy Testing

### Unit Tests

```rego
# policies/security/no-privileged_test.rego
package platformfoundry.security

test_deny_privileged {
    deny["Privileged containers are not allowed"] with input as {
        "spec": {
            "containers": [{"securityContext": {"privileged": true}}]
        }
    }
}

test_allow_non_privileged {
    count(deny) == 0 with input as {
        "spec": {
            "containers": [{"securityContext": {"privileged": false}}]
        }
    }
}
```

Run tests:

```bash
pf policy test --all
```

### Integration Testing

```bash
# Test policy against real config
pf policy eval require-labels --input=platform.yaml

# Output:
# POLICY          RESULT  MESSAGE
# require-labels  PASS    All required labels present
```

## Policy Bundles

Group related policies:

```yaml
apiVersion: platformfoundry.io/v1
kind: PolicyBundle
metadata:
  name: security-baseline
spec:
  policies:
    - no-privileged
    - no-root
    - require-limits
    - no-host-network
  enforcement: deny
```

### Install Bundle

```bash
# Install from registry
pf policy bundle install security-baseline

# Install from URL
pf policy bundle install https://example.com/policies/security.tar.gz
```

## Exemptions

Create policy exemptions:

```yaml
apiVersion: platformfoundry.io/v1
kind: PolicyExemption
metadata:
  name: allow-privileged-monitoring
spec:
  policy: no-privileged
  match:
    namespaces:
      - monitoring
    labels:
      app: node-exporter
  reason: "Node exporter requires privileged access for host metrics"
  expiration: 2024-12-31
  approvedBy: security-team
```

## Validation Hooks

### Pre-Apply

```yaml
spec:
  hooks:
    preApply:
      - policy: security-baseline
        enforcement: deny
      - policy: cost-limits
        enforcement: warn
```

### Pre-Commit

```bash
# Install pre-commit hook
pf policy hook install --type=pre-commit

# .git/hooks/pre-commit runs:
# pf policy eval --input=. --enforcement=deny
```

## Policy Reports

```bash
# Generate compliance report
pf policy report --format=html --output=compliance-report.html

# Continuous compliance monitoring
pf policy watch --interval=1h --report-to=slack
```

### Report Output

```
Policy Compliance Report
========================
Generated: 2024-01-20 10:30:00

Summary:
  Total Policies: 15
  Passing: 13
  Failing: 1
  Warnings: 1

Failed Policies:
  - require-labels (Platform: staging-platform)
    Missing labels: [cost-center]

Warnings:
  - cost-limits (Platform: dev-platform)
    Approaching quota limit (85%)
```

## OPA Integration

### Remote OPA Server

```yaml
policy:
  provider: opa
  config:
    url: https://opa.example.com
    auth:
      type: bearer
      token: ${secrets.opa-token}
```

### OPA Bundles

```yaml
policy:
  provider: opa
  config:
    bundles:
      - url: https://bundle-server/policies/v1
        polling:
          interval: 5m
```

## Common Policies

### Require Resource Limits

```rego
package platformfoundry.resources

deny[msg] {
    container := input.spec.containers[_]
    not container.resources.limits.cpu
    msg := sprintf("Container %v missing CPU limit", [container.name])
}

deny[msg] {
    container := input.spec.containers[_]
    not container.resources.limits.memory
    msg := sprintf("Container %v missing memory limit", [container.name])
}
```

### Enforce Naming Convention

```rego
package platformfoundry.naming

deny[msg] {
    not regex.match("^[a-z][a-z0-9-]*$", input.metadata.name)
    msg := "Name must start with lowercase letter and contain only lowercase letters, numbers, and hyphens"
}
```

### Cost Limits

```rego
package platformfoundry.cost

deny[msg] {
    input.spec.quotas.monthlyCost > 10000
    msg := sprintf("Monthly cost limit exceeded: $%v > $10000", [input.spec.quotas.monthlyCost])
}
```

## Next Steps

- [Plugins Guide](plugins.md) - Custom policy plugins
- [Enterprise Compliance](../enterprise/overview.md) - Advanced compliance features
