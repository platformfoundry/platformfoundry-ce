# API Reference

PlatformFoundry provides a REST API for programmatic access.

## Base URL

```
https://api.platformfoundry.example.com/v1
```

## Authentication

### Bearer Token

```bash
curl -H "Authorization: Bearer <token>" \
  https://api.platformfoundry.example.com/v1/platforms
```

### API Key

```bash
curl -H "X-API-Key: <api-key>" \
  https://api.platformfoundry.example.com/v1/platforms
```

## Common Headers

| Header | Description |
|--------|-------------|
| `Authorization` | Bearer token |
| `X-API-Key` | API key |
| `Content-Type` | `application/json` or `application/yaml` |
| `Accept` | Response format |

## Endpoints

### Platforms

#### List Platforms

```http
GET /v1/platforms
```

Query parameters:

| Parameter | Description |
|-----------|-------------|
| `environment` | Filter by environment |
| `labels` | Label selector |
| `limit` | Max results |
| `offset` | Pagination offset |

Response:

```json
{
  "items": [
    {
      "apiVersion": "platformfoundry.io/v1",
      "kind": "Platform",
      "metadata": {
        "name": "production-platform",
        "environment": "production",
        "createdAt": "2024-01-15T10:30:00Z",
        "updatedAt": "2024-01-20T14:45:00Z"
      },
      "status": {
        "phase": "Ready",
        "conditions": []
      }
    }
  ],
  "total": 1
}
```

#### Get Platform

```http
GET /v1/platforms/{name}
```

#### Create Platform

```http
POST /v1/platforms
Content-Type: application/json

{
  "apiVersion": "platformfoundry.io/v1",
  "kind": "Platform",
  "metadata": {
    "name": "my-platform",
    "environment": "production"
  },
  "spec": {
    "infrastructure": {
      "provider": "terraform"
    }
  }
}
```

#### Update Platform

```http
PUT /v1/platforms/{name}
```

#### Delete Platform

```http
DELETE /v1/platforms/{name}
```

Query parameters:

| Parameter | Description |
|-----------|-------------|
| `force` | Force delete |
| `cascade` | Delete dependencies |

### Environments

#### List Environments

```http
GET /v1/environments
```

#### Get Environment

```http
GET /v1/environments/{name}
```

#### Create Environment

```http
POST /v1/environments
```

#### Update Environment

```http
PUT /v1/environments/{name}
```

#### Delete Environment

```http
DELETE /v1/environments/{name}
```

### Secrets

#### List Secrets

```http
GET /v1/secrets
```

Response (metadata only):

```json
{
  "items": [
    {
      "name": "database-password",
      "provider": "vault",
      "createdAt": "2024-01-10T09:00:00Z",
      "updatedAt": "2024-01-15T11:30:00Z"
    }
  ]
}
```

#### Get Secret

```http
GET /v1/secrets/{name}
```

Query parameters:

| Parameter | Description |
|-----------|-------------|
| `reveal` | Include secret value (requires additional auth) |

#### Create Secret

```http
POST /v1/secrets

{
  "name": "api-key",
  "value": "sk-xxx",
  "provider": "vault"
}
```

#### Update Secret

```http
PUT /v1/secrets/{name}

{
  "value": "new-value"
}
```

#### Delete Secret

```http
DELETE /v1/secrets/{name}
```

### Policies

#### List Policies

```http
GET /v1/policies
```

#### Get Policy

```http
GET /v1/policies/{name}
```

#### Create Policy

```http
POST /v1/policies

{
  "name": "require-labels",
  "enforcement": "deny",
  "rego": "package platformfoundry..."
}
```

#### Evaluate Policy

```http
POST /v1/policies/eval

{
  "policy": "require-labels",
  "input": {
    "apiVersion": "platformfoundry.io/v1",
    "kind": "Platform",
    "metadata": {
      "name": "test"
    }
  }
}
```

Response:

```json
{
  "results": [
    {
      "policy": "require-labels",
      "passed": false,
      "messages": ["Missing required labels: [team, cost-center]"]
    }
  ]
}
```

### GitOps

#### Get Status

```http
GET /v1/gitops/status
```

Response:

```json
{
  "repository": "https://github.com/org/config",
  "branch": "main",
  "lastSync": "2024-01-20T15:00:00Z",
  "status": "Synced",
  "health": "Healthy"
}
```

#### Trigger Sync

```http
POST /v1/gitops/sync

{
  "force": false,
  "prune": true
}
```

#### Get History

```http
GET /v1/gitops/history
```

### Operations

#### Apply

```http
POST /v1/apply

{
  "resources": [...],
  "dryRun": false,
  "autoApprove": true
}
```

Response:

```json
{
  "operationId": "op-12345",
  "status": "running"
}
```

#### Get Operation Status

```http
GET /v1/operations/{id}
```

Response:

```json
{
  "id": "op-12345",
  "status": "completed",
  "startedAt": "2024-01-20T15:00:00Z",
  "completedAt": "2024-01-20T15:05:00Z",
  "results": [
    {
      "resource": "platform/my-platform",
      "action": "updated",
      "status": "success"
    }
  ]
}
```

### Workloads

Developer-friendly workload abstraction that auto-provisions infrastructure.

#### List Workloads

```http
GET /v1/workloads
```

Response:

```json
{
  "success": true,
  "data": {
    "workloads": [
      {
        "name": "my-api",
        "namespace": "production",
        "status": "running",
        "replicas": 3,
        "createdAt": "2024-01-15T10:30:00Z"
      }
    ],
    "count": 1
  },
  "timestamp": "2024-01-20T15:00:00Z"
}
```

#### Get Workload

```http
GET /v1/workloads/{name}
```

#### Create Workload

```http
POST /v1/workloads
Content-Type: application/json

{
  "config": "apiVersion: platformfoundry.io/v1\nkind: Workload\nmetadata:\n  name: my-api\nspec:\n  runtime: container\n  image: myregistry/api:v1.0.0",
  "dryRun": false
}
```

Response:

```json
{
  "success": true,
  "data": {
    "workload": "my-api",
    "status": "applied",
    "resources": {
      "kubernetes": ["deployment/my-api", "service/my-api", "hpa/my-api"],
      "infrastructure": ["rds/my-api-db", "elasticache/my-api-cache"]
    }
  },
  "timestamp": "2024-01-20T15:00:00Z"
}
```

#### Get Workload Status

```http
GET /v1/workloads/{name}/status
```

Response:

```json
{
  "success": true,
  "data": {
    "name": "my-api",
    "status": "running",
    "replicas": {
      "desired": 3,
      "ready": 3,
      "available": 3
    },
    "conditions": [
      {
        "type": "Available",
        "status": "True",
        "lastTransitionTime": "2024-01-20T14:00:00Z"
      }
    ]
  },
  "timestamp": "2024-01-20T15:00:00Z"
}
```

#### Delete Workload

```http
DELETE /v1/workloads/{name}
```

### Events

Real-time and historical event access.

#### List Events

```http
GET /v1/events
```

Query parameters:

| Parameter | Description |
|-----------|-------------|
| `since` | RFC3339 timestamp to filter events after |
| `engineId` | Filter by engine ID |
| `type` | Filter by event type |

Response:

```json
{
  "success": true,
  "data": {
    "events": [
      {
        "engineId": "infra-001",
        "type": "resource_created",
        "component": "aws",
        "progress": 100,
        "message": "Created RDS instance my-api-db",
        "timestamp": "2024-01-20T14:55:00Z"
      }
    ],
    "count": 1
  },
  "timestamp": "2024-01-20T15:00:00Z"
}
```

#### Stream Events (SSE)

```http
GET /v1/events/stream
```

Establishes a Server-Sent Events connection for real-time updates.

Response (SSE format):

```
event: connected
data: {"message": "SSE connection established"}

event: resource_created
data: {"engineId":"infra-001","type":"resource_created","component":"aws","progress":50,"message":"Creating RDS instance","timestamp":"2024-01-20T14:55:00Z"}

: keepalive
```

### Health

#### Health Check

```http
GET /v1/health
```

Response:

```json
{
  "status": "healthy",
  "version": "v0.1.0",
  "components": {
    "database": "healthy",
    "secrets": "healthy",
    "gitops": "healthy"
  }
}
```

## Error Responses

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid platform configuration",
    "details": [
      {
        "field": "spec.infrastructure.provider",
        "message": "Unknown provider: invalid"
      }
    ]
  }
}
```

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `NOT_FOUND` | 404 | Resource not found |
| `VALIDATION_ERROR` | 400 | Invalid input |
| `UNAUTHORIZED` | 401 | Authentication required |
| `FORBIDDEN` | 403 | Permission denied |
| `CONFLICT` | 409 | Resource conflict |
| `INTERNAL_ERROR` | 500 | Server error |

## Pagination

```http
GET /v1/platforms?limit=10&offset=20
```

Response includes:

```json
{
  "items": [...],
  "total": 50,
  "limit": 10,
  "offset": 20
}
```

## Filtering

### Label Selector

```http
GET /v1/platforms?labels=team=platform,env=prod
```

### Field Selector

```http
GET /v1/platforms?fields=metadata.environment=production
```

## Webhooks

### Register Webhook

```http
POST /v1/webhooks

{
  "url": "https://api.example.com/webhook",
  "events": ["platform.created", "platform.updated", "platform.deleted"],
  "secret": "webhook-secret"
}
```

### Webhook Payload

```json
{
  "event": "platform.updated",
  "timestamp": "2024-01-20T15:00:00Z",
  "resource": {
    "kind": "Platform",
    "name": "my-platform"
  },
  "changes": {
    "spec.infrastructure.config.replicas": {
      "old": 2,
      "new": 3
    }
  }
}
```

## Rate Limiting

| Endpoint | Limit |
|----------|-------|
| Read operations | 1000/min |
| Write operations | 100/min |
| Bulk operations | 10/min |

Headers:

```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 950
X-RateLimit-Reset: 1705762800
```

## SDK

### Go

```go
import "github.com/platformfoundry/platformfoundry-ce/pkg/sdk"

client := sdk.NewClient(
    sdk.WithBaseURL("https://api.example.com"),
    sdk.WithToken("token"),
)

platforms, err := client.Platforms().List(ctx)
```

### Python

```python
from platformfoundry import Client

client = Client(
    base_url="https://api.example.com",
    token="token"
)

platforms = client.platforms.list()
```
