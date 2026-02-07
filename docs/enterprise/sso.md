# Single Sign-On (SSO)

Enterprise Edition supports SAML 2.0 and OIDC authentication with major identity providers.

## Supported Providers

| Provider | Protocol | Status |
|----------|----------|--------|
| Okta | SAML/OIDC | ✅ Supported |
| Azure AD | SAML/OIDC | ✅ Supported |
| Google Workspace | OIDC | ✅ Supported |
| OneLogin | SAML | ✅ Supported |
| PingFederate | SAML | ✅ Supported |
| Custom SAML | SAML 2.0 | ✅ Supported |
| Custom OIDC | OIDC | ✅ Supported |

## Quick Setup

### Okta

```bash
# Configure Okta SSO
pf sso configure okta \
  --domain=mycompany.okta.com \
  --client-id=<client-id> \
  --client-secret=<client-secret>

# Test connection
pf sso test

# Enable SSO
pf sso enable
```

### Azure AD

```bash
# Configure Azure AD
pf sso configure azure-ad \
  --tenant-id=<tenant-id> \
  --client-id=<client-id> \
  --client-secret=<client-secret>

# Enable SSO
pf sso enable
```

### Google Workspace

```bash
# Configure Google
pf sso configure google \
  --domain=mycompany.com \
  --client-id=<client-id> \
  --client-secret=<client-secret>

# Enable SSO
pf sso enable
```

## Configuration

### YAML Configuration

```yaml
# ~/.platformfoundry/config.yaml
enterprise:
  sso:
    enabled: true
    provider: okta  # okta, azure-ad, google, custom-saml, custom-oidc

    # Okta configuration
    okta:
      domain: mycompany.okta.com
      clientId: ${PF_OKTA_CLIENT_ID}
      clientSecret: ${PF_OKTA_CLIENT_SECRET}
      scopes:
        - openid
        - profile
        - email
        - groups

    # Role mapping
    roleMapping:
      enabled: true
      defaultRole: viewer
      mappings:
        - group: "Platform Admins"
          role: admin
        - group: "Platform Engineers"
          role: editor
        - group: "Developers"
          role: viewer

    # Session settings
    session:
      timeout: 8h
      refreshEnabled: true
      maxAge: 24h
```

### SAML Configuration

```yaml
enterprise:
  sso:
    enabled: true
    provider: custom-saml

    saml:
      idpMetadataURL: https://idp.example.com/metadata
      # Or provide metadata directly
      # idpMetadata: |
      #   <EntityDescriptor>...</EntityDescriptor>

      spEntityID: https://platformfoundry.example.com
      acsURL: https://platformfoundry.example.com/auth/saml/callback
      sloURL: https://platformfoundry.example.com/auth/saml/logout

      attributeMapping:
        email: http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress
        firstName: http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname
        lastName: http://schemas.xmlsoap.org/ws/2005/05/identity/claims/surname
        groups: http://schemas.xmlsoap.org/claims/Group
```

### OIDC Configuration

```yaml
enterprise:
  sso:
    enabled: true
    provider: custom-oidc

    oidc:
      issuer: https://idp.example.com
      clientId: ${PF_OIDC_CLIENT_ID}
      clientSecret: ${PF_OIDC_CLIENT_SECRET}
      redirectURL: https://platformfoundry.example.com/auth/oidc/callback

      scopes:
        - openid
        - profile
        - email
        - groups

      claims:
        email: email
        name: name
        groups: groups
```

## CLI Commands

```bash
# List configured providers
pf sso list

# Output:
# NAME          TYPE      STATUS    USERS
# okta-prod     SAML      Active    142
# azure-ad      OIDC      Active    89

# Configure new provider
pf sso configure <provider> [flags]

# Test SSO connection
pf sso test

# Login via SSO
pf auth login --sso

# View SSO status
pf sso status

# Disable SSO
pf sso disable

# Remove provider
pf sso remove okta-prod
```

## Role Mapping

### From IdP Groups

```yaml
roleMapping:
  enabled: true
  groupAttribute: groups  # Attribute containing group memberships

  mappings:
    # Admin access
    - group: "cn=platform-admins,ou=groups,dc=example,dc=com"
      role: admin
      namespaces: ["*"]

    # Editor access to specific namespaces
    - group: "Platform Engineers"
      role: editor
      namespaces: ["production", "staging"]

    # Viewer access
    - group: "Developers"
      role: viewer
      namespaces: ["development"]

  # Default role for unmapped users
  defaultRole: viewer
  defaultNamespaces: ["development"]
```

### Custom Role Mapping

```yaml
roleMapping:
  enabled: true
  custom:
    # Map by email domain
    - condition:
        attribute: email
        matches: ".*@engineering.example.com"
      role: editor

    # Map by specific attribute
    - condition:
        attribute: department
        equals: "Platform"
      role: admin
```

## Just-in-Time Provisioning

Automatically create users on first login:

```yaml
enterprise:
  sso:
    jit:
      enabled: true
      createUser: true
      updateOnLogin: true  # Update user attributes on each login

      # Default settings for new users
      defaults:
        role: viewer
        namespaces: ["development"]
        notifications: true
```

## Multi-Provider Setup

Configure multiple SSO providers:

```yaml
enterprise:
  sso:
    enabled: true
    providers:
      - name: okta-internal
        type: oidc
        config:
          issuer: https://internal.okta.com
          clientId: ${OKTA_INTERNAL_CLIENT_ID}
        domains:
          - "@internal.example.com"

      - name: azure-external
        type: oidc
        config:
          issuer: https://login.microsoftonline.com/tenant
          clientId: ${AZURE_CLIENT_ID}
        domains:
          - "@external.example.com"
          - "@partner.example.com"
```

## Security Settings

```yaml
enterprise:
  sso:
    security:
      # Require SSO for all users
      enforced: true

      # Allow local accounts for break-glass
      allowLocalAccounts: true
      localAccountDomains:
        - "@admin.example.com"

      # MFA settings
      mfa:
        required: true
        methods:
          - totp
          - webauthn

      # Session security
      session:
        secure: true
        httpOnly: true
        sameSite: strict
```

## Troubleshooting

### Debug Mode

```bash
# Enable SSO debug logging
pf config set logging.level=debug

# Test SSO with verbose output
pf sso test --verbose

# View SSO logs
pf logs --component=sso
```

### Common Issues

**SAML signature validation failed:**
```bash
# Update IdP certificate
pf sso update-cert --provider=okta --cert-file=idp-cert.pem
```

**User not found in role mapping:**
```bash
# Check user's groups
pf sso debug-user --email=user@example.com

# Output shows:
# Email: user@example.com
# Groups: ["Developers", "Team-A"]
# Mapped Role: viewer
# Mapped Namespaces: [development]
```

**Session timeout issues:**
```yaml
# Increase session timeout
enterprise:
  sso:
    session:
      timeout: 12h
      refreshEnabled: true
```

## Next Steps

- [RBAC Configuration](../reference/config.md#rbac)
- [Audit Logging](overview.md#advanced-audit-logging)
