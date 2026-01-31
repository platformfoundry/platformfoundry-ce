# Secrets Management

PlatformFoundry provides secure secrets management with multiple backend providers.

## Supported Providers

| Provider | Description | Use Case |
|----------|-------------|----------|
| `local` | Encrypted local storage | Development |
| `vault` | HashiCorp Vault | Production |
| `aws` | AWS Secrets Manager | AWS environments |

## Configuration

### Local Provider

```yaml
# ~/.platformfoundry/config.yaml
secrets:
  provider: local
  config:
    path: ~/.platformfoundry/secrets
    encryption: aes-256-gcm
```

### HashiCorp Vault

```yaml
secrets:
  provider: vault
  config:
    address: https://vault.example.com
    auth:
      method: kubernetes  # token, kubernetes, aws
      role: platform-foundry
    mount: secret
    path: platformfoundry
```

### AWS Secrets Manager

```yaml
secrets:
  provider: aws
  config:
    region: us-east-1
    prefix: /platformfoundry/
    kmsKeyId: alias/platformfoundry
```

## CLI Commands

### Create Secrets

```bash
# Interactive
pf secrets set database-password

# From value
pf secrets set api-key --value="sk-xxx"

# From file
pf secrets set tls-cert --from-file=./cert.pem

# From environment
pf secrets set aws-creds --from-env=AWS_SECRET_ACCESS_KEY
```

### List Secrets

```bash
pf secrets list

# Output:
# NAME              PROVIDER  CREATED      UPDATED
# database-password vault     2024-01-15   2024-01-20
# api-key           vault     2024-01-10   2024-01-10
# tls-cert          vault     2024-01-05   2024-01-05
```

### Get Secrets

```bash
# Show metadata only
pf secrets get database-password

# Show value (requires confirmation)
pf secrets get database-password --show-value
```

### Delete Secrets

```bash
pf secrets delete api-key
```

### Rotate Secrets

```bash
# Manual rotation
pf secrets rotate database-password

# Auto-rotation schedule
pf secrets set database-password --rotate-every=30d
```

## Using Secrets in Platforms

### Direct Reference

```yaml
apiVersion: platformfoundry.io/v1
kind: Platform
metadata:
  name: my-platform
spec:
  infrastructure:
    config:
      variables:
        db_password: ${secrets.database-password}
        api_key: ${secrets.api-key}
```

### Secret Resource

```yaml
apiVersion: platformfoundry.io/v1
kind: Secret
metadata:
  name: app-secrets
  namespace: default
spec:
  provider: vault
  data:
    DATABASE_URL: ${secrets.database-url}
    REDIS_PASSWORD: ${secrets.redis-password}
  sync:
    enabled: true
    interval: 5m
    target:
      type: kubernetes
      namespace: app
      name: app-secrets
```

## Secret Sync

Sync secrets to Kubernetes:

```bash
# One-time sync
pf secrets sync database-password --to-namespace=app

# Continuous sync
pf secrets sync database-password --to-namespace=app --watch
```

### Sync Configuration

```yaml
apiVersion: platformfoundry.io/v1
kind: SecretSync
metadata:
  name: app-secrets-sync
spec:
  source:
    provider: vault
    path: platformfoundry/app
  target:
    type: kubernetes
    namespace: app
    secretName: app-secrets
  schedule: "*/5 * * * *"  # Every 5 minutes
  transform:
    - key: db_password
      targetKey: DATABASE_PASSWORD
```

## Encryption

### Local Encryption

```bash
# Initialize encryption key
pf secrets init

# Backup encryption key
pf secrets backup-key > encryption-key.backup
```

### Key Rotation

```bash
# Rotate encryption key
pf secrets rotate-key

# Re-encrypt all secrets with new key
pf secrets reencrypt
```

## Access Control

### RBAC for Secrets

```yaml
apiVersion: platformfoundry.io/v1
kind: Role
metadata:
  name: secrets-reader
spec:
  rules:
    - resources: ["secrets"]
      verbs: ["get", "list"]
      paths: ["production/*"]
---
apiVersion: platformfoundry.io/v1
kind: RoleBinding
metadata:
  name: dev-team-secrets
spec:
  role: secrets-reader
  subjects:
    - kind: Group
      name: developers
```

## Audit Logging

All secret access is logged:

```bash
# View secret access logs
pf audit logs --resource=secrets

# Output:
# TIME                 USER      ACTION  SECRET
# 2024-01-20 10:30:00  alice     get     database-password
# 2024-01-20 10:25:00  bob       set     api-key
```

## Best Practices

1. **Never commit secrets** - Use `.gitignore` for local secret files
2. **Use least privilege** - Grant minimal access to secrets
3. **Enable rotation** - Rotate secrets regularly
4. **Audit access** - Review secret access logs
5. **Use namespaces** - Organize secrets by environment/team

## Migration

### From Environment Variables

```bash
pf secrets import --from-env --prefix=APP_
```

### From Files

```bash
pf secrets import --from-file=secrets.env
```

### To Another Provider

```bash
pf secrets migrate --from=local --to=vault
```

## Next Steps

- [Policies Guide](policies.md) - Enforce secret policies
- [GitOps Guide](gitops.md) - Secrets in GitOps workflows
