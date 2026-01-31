# Configuration

PlatformFoundry uses a layered configuration system with multiple sources.

## Configuration Hierarchy

Configuration is loaded in order (later overrides earlier):

1. Built-in defaults
2. System config (`/etc/platformfoundry/config.yaml`)
3. User config (`~/.platformfoundry/config.yaml`)
4. Project config (`./config.yaml`)
5. Environment variables
6. Command-line flags

## Config File Location

```bash
# Show config file locations
pf config paths

# Show current configuration
pf config view
```

## Global Configuration

Create `~/.platformfoundry/config.yaml`:

```yaml
# Default settings
defaults:
  environment: development
  parallelism: 4
  timeout: 30m

# Authentication
auth:
  method: token  # token, saml, api-key
  tokenPath: ~/.platformfoundry/token

# State backend
state:
  backend: local  # local, s3, dynamodb
  path: ~/.platformfoundry/state

# Secrets provider
secrets:
  provider: local  # local, vault, aws
  path: ~/.platformfoundry/secrets

# Logging
logging:
  level: info  # debug, info, warn, error
  format: text  # text, json
  file: ~/.platformfoundry/pf.log

# Telemetry (anonymous usage stats)
telemetry:
  enabled: false
```

## Environment Variables

All config options can be set via environment variables:

| Variable | Description |
|----------|-------------|
| `PF_CONFIG` | Path to config file |
| `PF_LOG_LEVEL` | Log level (debug/info/warn/error) |
| `PF_LOG_FORMAT` | Log format (text/json) |
| `PF_STATE_BACKEND` | State backend type |
| `PF_SECRETS_PROVIDER` | Secrets provider |
| `PF_PARALLELISM` | Parallel operations |
| `PF_TIMEOUT` | Default operation timeout |

Example:

```bash
export PF_LOG_LEVEL=debug
export PF_STATE_BACKEND=s3
pf apply -f platform.yaml
```

## Project Configuration

Create `.pf.yaml` in your project root:

```yaml
# Project-specific settings
project:
  name: my-project
  environment: staging

# Override defaults for this project
defaults:
  timeout: 1h
  parallelism: 8

# Provider configurations
providers:
  terraform:
    version: ">=1.5.0"
    backend:
      type: s3
      bucket: my-terraform-state
      region: us-east-1

  kubernetes:
    context: my-cluster
    namespace: platform
```

## Contexts

Manage multiple environments with contexts:

```bash
# List contexts
pf config contexts

# Switch context
pf config use-context production

# Create new context
pf config set-context staging \
  --environment=staging \
  --state-backend=s3 \
  --secrets-provider=vault
```

## Credentials

Store credentials securely:

```bash
# Set credentials
pf config set-credentials aws \
  --access-key-id=AKIA... \
  --secret-access-key=...

# Use credentials from environment
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
```

## Validation

Validate your configuration:

```bash
pf config validate
```

## Reset Configuration

```bash
# Reset to defaults
pf config reset

# Reset specific section
pf config reset --section=auth
```

## Next Steps

- [Secrets Management](../guides/secrets.md)
- [State Backends](../reference/config.md#state-backends)
