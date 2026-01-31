# Enterprise Edition Overview

PlatformFoundry Enterprise Edition (EE) extends the Community Edition with advanced features for large organizations.

## Feature Comparison

| Feature | Community | Enterprise |
|---------|-----------|------------|
| Core Platform Management | ✅ | ✅ |
| Multi-Cloud Support | ✅ | ✅ |
| GitOps Integration | ✅ | ✅ |
| Policy Engine (OPA) | ✅ | ✅ |
| Secrets Management | ✅ | ✅ |
| **SSO Integration** | ❌ | ✅ |
| **DORA Metrics** | ❌ | ✅ |
| **Cost Tracking** | ❌ | ✅ |
| **Audit Logging** | Basic | Advanced |
| **RBAC** | Basic | Advanced |
| **Support** | Community | 24/7 |

## Enterprise Features

### Single Sign-On (SSO)

Enterprise SSO support includes:

- SAML 2.0 integration
- OIDC providers (Okta, Azure AD, Google)
- Role mapping from IdP groups
- Just-in-time user provisioning

[Learn more →](sso.md)

### DORA Metrics

Track software delivery performance:

- Deployment Frequency
- Lead Time for Changes
- Mean Time to Recovery (MTTR)
- Change Failure Rate

[Learn more →](dora.md)

### Cost Tracking

Real-time cloud cost management:

- Per-environment cost breakdown
- Cost allocation by team/project
- Budget alerts and forecasting
- Optimization recommendations

[Learn more →](cost-tracking.md)

### Advanced Audit Logging

Comprehensive audit trail:

- All API operations logged
- User activity tracking
- Resource change history
- Compliance reporting

### Advanced RBAC

Fine-grained access control:

- Custom roles and permissions
- Namespace-level access
- Resource-level policies
- Integration with IdP groups

## Installation

### License

Contact sales@platformfoundry.io for licensing.

### Setup

```bash
# Install EE binary
curl -sSL https://download.platformfoundry.io/ee/pf-linux-amd64 -o pf
chmod +x pf
sudo mv pf /usr/local/bin/

# Activate license
pf license activate --key=<license-key>

# Verify
pf version
# PlatformFoundry Enterprise v0.1.0
```

### Configuration

```yaml
# ~/.platformfoundry/config.yaml
enterprise:
  license:
    key: ${PF_LICENSE_KEY}

  # Enable enterprise features
  features:
    sso: true
    dora: true
    costTracking: true
    advancedAudit: true

  # SSO Configuration
  sso:
    provider: okta
    domain: mycompany.okta.com
    clientId: ${PF_SSO_CLIENT_ID}
    clientSecret: ${PF_SSO_CLIENT_SECRET}

  # Cost tracking
  costTracking:
    providers:
      - aws
      - gcp
    refreshInterval: 1h
    budgetAlerts: true
```

## Enterprise Support

### Support Tiers

| Tier | Response Time | Availability |
|------|---------------|--------------|
| Standard | 24 hours | Business hours |
| Premium | 4 hours | 24/7 |
| Critical | 1 hour | 24/7 |

### Contact

- Email: support@platformfoundry.io
- Portal: https://support.platformfoundry.io
- Slack: Enterprise customers channel

## Upgrade from Community

```bash
# Backup current configuration
pf config export > config-backup.yaml

# Install Enterprise Edition
curl -sSL https://download.platformfoundry.io/ee/pf-linux-amd64 -o pf-ee
chmod +x pf-ee
sudo mv pf-ee /usr/local/bin/pf

# Activate license
pf license activate --key=<license-key>

# Restore configuration
pf config import < config-backup.yaml

# Enable enterprise features
pf config set enterprise.features.sso=true
pf config set enterprise.features.dora=true
pf config set enterprise.features.costTracking=true
```

## Next Steps

- [Configure SSO](sso.md)
- [Enable Cost Tracking](cost-tracking.md)
- [Set Up DORA Metrics](dora.md)
