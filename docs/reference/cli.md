# CLI Reference

Complete reference for all PlatformFoundry CLI commands.

## Global Flags

| Flag | Description |
|------|-------------|
| `--config` | Path to config file |
| `--context` | Context to use |
| `-v, --verbose` | Verbose output |
| `--log-level` | Log level (debug/info/warn/error) |
| `--log-format` | Log format (text/json) |
| `-h, --help` | Help for command |

## Core Commands

### pf init

Initialize PlatformFoundry configuration.

```bash
pf init [flags]
```

| Flag | Description |
|------|-------------|
| `--dir` | Directory for config (default: ~/.platformfoundry) |
| `--template` | Template to use |

### pf apply

Apply platform configuration.

```bash
pf apply -f <file> [flags]
```

| Flag | Description |
|------|-------------|
| `-f, --file` | Path to configuration file (required) |
| `--dry-run` | Preview changes without applying |
| `--auto-approve` | Skip confirmation prompt |
| `-p, --parallelism` | Number of parallel operations (default: 4) |
| `--timeout` | Operation timeout (default: 30m) |
| `-e, --environment` | Target environment |

### pf plan

Generate execution plan.

```bash
pf plan -f <file> [flags]
```

| Flag | Description |
|------|-------------|
| `-f, --file` | Path to configuration file |
| `-o, --output` | Output file for plan |
| `--detailed` | Show detailed changes |

### pf destroy

Destroy platform resources.

```bash
pf destroy -f <file> [flags]
```

| Flag | Description |
|------|-------------|
| `-f, --file` | Path to configuration file |
| `--auto-approve` | Skip confirmation |
| `--force` | Force destroy (ignore errors) |

### pf validate

Validate configuration files.

```bash
pf validate -f <file> [flags]
```

| Flag | Description |
|------|-------------|
| `-f, --file` | File to validate |
| `--strict` | Strict validation mode |

### pf diff

Show differences between configurations.

```bash
pf diff <source> <target> [flags]
```

| Flag | Description |
|------|-------------|
| `--format` | Output format (text/json/yaml) |

## Resource Commands

### pf get

List resources.

```bash
pf get <resource-type> [name] [flags]
```

| Resource Types | Description |
|----------------|-------------|
| `platforms` | List platforms |
| `environments` | List environments |
| `secrets` | List secrets |
| `policies` | List policies |
| `plugins` | List plugins |

| Flag | Description |
|------|-------------|
| `-o, --output` | Output format (table/json/yaml) |
| `-l, --selector` | Label selector |
| `-A, --all-namespaces` | All namespaces |

### pf describe

Show resource details.

```bash
pf describe <resource-type> <name> [flags]
```

### pf delete

Delete resources.

```bash
pf delete <resource-type> <name> [flags]
```

| Flag | Description |
|------|-------------|
| `--force` | Force delete |
| `--cascade` | Delete dependent resources |

## Authentication Commands

### pf auth login

Authenticate with PlatformFoundry.

```bash
pf auth login [flags]
```

| Flag | Description |
|------|-------------|
| `--method` | Auth method (token/saml/api-key) |
| `--server` | Server URL |

### pf auth logout

Log out from PlatformFoundry.

```bash
pf auth logout
```

### pf auth status

Show authentication status.

```bash
pf auth status
```

### pf auth token

Manage authentication tokens.

```bash
pf auth token [create|list|revoke] [flags]
```

## Secrets Commands

### pf secrets set

Create or update a secret.

```bash
pf secrets set <name> [flags]
```

| Flag | Description |
|------|-------------|
| `--value` | Secret value |
| `--from-file` | Read from file |
| `--from-env` | Read from environment variable |

### pf secrets get

Get secret metadata.

```bash
pf secrets get <name> [flags]
```

| Flag | Description |
|------|-------------|
| `--show-value` | Display secret value |

### pf secrets list

List all secrets.

```bash
pf secrets list [flags]
```

### pf secrets delete

Delete a secret.

```bash
pf secrets delete <name>
```

## GitOps Commands

### pf gitops init

Initialize GitOps configuration.

```bash
pf gitops init [flags]
```

| Flag | Description |
|------|-------------|
| `--repo` | Git repository URL |
| `--branch` | Branch name |
| `--path` | Path in repository |

### pf gitops sync

Synchronize with Git repository.

```bash
pf gitops sync [flags]
```

| Flag | Description |
|------|-------------|
| `--force` | Force sync |
| `--prune` | Remove deleted resources |

### pf gitops status

Show GitOps status.

```bash
pf gitops status
```

### pf gitops diff

Show pending changes.

```bash
pf gitops diff
```

## Policy Commands

### pf policy list

List policies.

```bash
pf policy list [flags]
```

| Flag | Description |
|------|-------------|
| `--builtin` | Show built-in policies |
| `--custom` | Show custom policies only |

### pf policy create

Create a policy.

```bash
pf policy create <name> [flags]
```

| Flag | Description |
|------|-------------|
| `--from-file` | Rego file path |
| `--enforcement` | Enforcement level |

### pf policy test

Test a policy.

```bash
pf policy test <name> [flags]
```

| Flag | Description |
|------|-------------|
| `--input` | Input file to test against |

### pf policy eval

Evaluate policies against input.

```bash
pf policy eval [flags]
```

| Flag | Description |
|------|-------------|
| `--input` | Input file |
| `--policy` | Specific policy to evaluate |

## Plugin Commands

### pf plugin list

List plugins.

```bash
pf plugin list [flags]
```

### pf plugin install

Install a plugin.

```bash
pf plugin install <name|url> [flags]
```

| Flag | Description |
|------|-------------|
| `--version` | Plugin version |

### pf plugin remove

Remove a plugin.

```bash
pf plugin remove <name>
```

### pf plugin enable/disable

Enable or disable a plugin.

```bash
pf plugin enable <name>
pf plugin disable <name>
```

## Configuration Commands

### pf config view

View current configuration.

```bash
pf config view [flags]
```

### pf config set

Set configuration value.

```bash
pf config set <key> <value>
```

### pf config contexts

List available contexts.

```bash
pf config contexts
```

### pf config use-context

Switch context.

```bash
pf config use-context <name>
```

## Workload Commands

Workloads provide a developer-friendly abstraction that automatically provisions both Kubernetes resources and required infrastructure.

### pf workload apply

Apply a workload configuration.

```bash
pf workload apply -f <file> [flags]
```

| Flag | Description |
|------|-------------|
| `-f, --file` | Path to workload file (required) |
| `--dry-run` | Preview generated resources |
| `--cloud` | Target cloud provider (default: aws) |
| `--region` | Cloud region (default: us-east-1) |
| `--env` | Environment name (default: default) |

Example workload file:

```yaml
apiVersion: platformfoundry.io/v1
kind: Workload
metadata:
  name: my-api
  namespace: production
spec:
  runtime: container
  image: myregistry/api:v1.0.0
  replicas: 3
  resources:
    cpu: "500m"
    memory: "512Mi"
  ports:
    - port: 8080
      protocol: HTTP
  scaling:
    minReplicas: 2
    maxReplicas: 10
    targetCPU: 70
  infrastructure:
    database:
      type: postgres
      size: small
    cache:
      type: redis
      size: small
```

### pf workload plan

Preview workload resources without applying.

```bash
pf workload plan -f <file> [flags]
```

### pf workload generate

Generate Kubernetes manifests from a workload.

```bash
pf workload generate -f <file> [flags]
```

| Flag | Description |
|------|-------------|
| `-o, --output` | Output directory for manifests |
| `--format` | Output format (yaml/json) |

### pf workload list

List deployed workloads.

```bash
pf workload list [flags]
```

| Flag | Description |
|------|-------------|
| `-n, --namespace` | Filter by namespace |
| `-o, --output` | Output format (table/json/yaml) |

### pf workload status

Get workload status and health.

```bash
pf workload status <name> [flags]
```

### pf workload delete

Delete a workload and its resources.

```bash
pf workload delete <name> [flags]
```

| Flag | Description |
|------|-------------|
| `--cascade` | Delete infrastructure resources |
| `--force` | Force delete without confirmation |

## Utility Commands

### pf version

Show version information.

```bash
pf version
```

### pf doctor

Check system health.

```bash
pf doctor [flags]
```

### pf completion

Generate shell completion scripts.

```bash
pf completion [bash|zsh|fish|powershell]
```

## Output Formats

Most commands support multiple output formats:

```bash
# Table (default)
pf get platforms

# JSON
pf get platforms -o json

# YAML
pf get platforms -o yaml

# Wide (more columns)
pf get platforms -o wide
```

## Exit Codes

| Code | Description |
|------|-------------|
| 0 | Success |
| 1 | General error |
| 2 | Invalid arguments |
| 3 | Resource not found |
| 4 | Authentication error |
| 5 | Policy violation |
