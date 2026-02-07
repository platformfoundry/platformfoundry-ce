# Platform Foundry Innovation Design Document

**Version**: 1.0
**Date**: January 2026
**Status**: Implementation Ready

## Executive Summary

This document outlines the innovation roadmap to position Platform Foundry as the market leader in Internal Developer Platform (IDP) orchestration. The design takes inspiration from market leaders (Humanitec, Backstage, Port, Kratix, Crossplane) while leveraging Platform Foundry's existing codebase.

## Market Analysis Summary

| Competitor | Strength | Gap |
|------------|----------|-----|
| **Humanitec** | Score workload spec, Platform Orchestrator | Closed source, expensive |
| **Backstage** | Software catalog, plugin ecosystem | No orchestration, complex setup |
| **Port** | No-code portal, AI integration | No infrastructure orchestration |
| **Kratix** | Kubernetes-native Promises | Limited to K8s, no multi-IaC |
| **Crossplane** | Self-healing, control plane | Infrastructure only |

**Platform Foundry's Unique Position**: Full-stack orchestration (infra + orchestration + observability + devex) with multi-IaC support and existing enterprise features.

---

## Phase 1: Quick Wins (Wire Existing Components)

### 1.1 Platform Health Score

**Objective**: Provide a single metric representing overall platform health.

**Existing Components Used**:
- `internal/lint/` - Configuration linting
- `internal/drift/` - Drift detection
- `internal/policy/` - Policy compliance
- `internal/cost/` - Cost estimation
- `internal/compliance/` - Compliance checks

**Design**:

```go
// internal/health/health.go
package health

type HealthScore struct {
    Overall          int                    `json:"overall"`           // 0-100
    Categories       map[string]CategoryScore `json:"categories"`
    Issues           []Issue                `json:"issues"`
    Recommendations  []Recommendation       `json:"recommendations"`
    LastChecked      time.Time              `json:"last_checked"`
}

type CategoryScore struct {
    Name        string  `json:"name"`
    Score       int     `json:"score"`       // 0-100
    Weight      float64 `json:"weight"`      // Contribution to overall
    Status      string  `json:"status"`      // healthy, warning, critical
    IssueCount  int     `json:"issue_count"`
}

type HealthChecker struct {
    linter      *lint.Linter
    drift       *drift.Detector
    policy      *policy.Engine
    cost        *cost.Estimator
    compliance  *compliance.Checker
}

func (h *HealthChecker) Check(platform string) (*HealthScore, error) {
    // Aggregate scores from all subsystems
    // Apply weights: lint(20%), drift(25%), policy(25%), cost(15%), compliance(15%)
}
```

**CLI Command**:
```bash
pf health [platform-name]
pf health --watch           # Continuous monitoring
pf health --export json     # Export for dashboards
```

**Output Format**:
```
Platform Health: my-platform
═══════════════════════════════════════════════════════════════

Overall Score: 73/100 ⚠️

┌─────────────────────┬───────┬────────┬────────────────────┐
│ Category            │ Score │ Status │ Issues             │
├─────────────────────┼───────┼────────┼────────────────────┤
│ Configuration       │ 85    │ ✅     │ 2 warnings         │
│ Drift               │ 60    │ ⚠️     │ 3 resources        │
│ Policy Compliance   │ 90    │ ✅     │ 1 violation        │
│ Cost Efficiency     │ 62    │ ⚠️     │ over budget        │
│ Security Compliance │ 70    │ ⚠️     │ 2 findings         │
└─────────────────────┴───────┴────────┴────────────────────┘

Top Issues:
1. [DRIFT] ArgoCD app out of sync for 3 days
2. [COST] Staging nodes oversized (save $340/mo)
3. [SECURITY] 2 container images with HIGH CVEs

Recommendations:
• Run 'pf drift fix' to remediate drift
• Run 'pf cost optimize' for cost savings
```

**Effort**: ~150-200 lines of Go code

---

### 1.2 Auto-Remediation Engine

**Objective**: Automatically fix detected drift based on policies.

**Existing Components Used**:
- `internal/drift/` - Detection
- `internal/policy/` - Allow/deny rules
- `internal/orchestrator/` - Apply changes
- `internal/notifications/` - Alert on actions
- `internal/workflow/` - Approval workflows

**Design**:

```go
// internal/remediation/remediation.go
package remediation

type RemediationRule struct {
    Name        string              `yaml:"name"`
    Trigger     RemediationTrigger  `yaml:"trigger"`
    Conditions  []Condition         `yaml:"conditions"`
    Action      RemediationAction   `yaml:"action"`
    Notify      []string            `yaml:"notify"`
}

type RemediationTrigger struct {
    Type        string  `yaml:"type"`      // drift, policy_violation, health_score
    Severity    string  `yaml:"severity"`  // low, medium, high, critical
    MaxAge      string  `yaml:"max_age"`   // e.g., "1h", "24h"
    Resource    string  `yaml:"resource"`  // optional filter
}

type RemediationAction struct {
    Type            string `yaml:"type"`   // auto_fix, alert_only, create_ticket, require_approval
    ApprovalPolicy  string `yaml:"approval_policy,omitempty"`
}

type Reconciler struct {
    drift       *drift.Detector
    policy      *policy.Engine
    apply       *orchestrator.Engine
    notify      *notifications.Manager
    workflow    *workflow.Engine
    rules       []RemediationRule
}

func (r *Reconciler) Run(ctx context.Context) error {
    // 1. Detect drift
    // 2. Match against rules
    // 3. Check policy allows auto-fix
    // 4. Execute action (fix, alert, or request approval)
    // 5. Notify stakeholders
}
```

**YAML Configuration**:
```yaml
apiVersion: platformfoundry.io/v1
kind: RemediationPolicy
metadata:
  name: auto-remediation
spec:
  rules:
    - name: auto-sync-low-severity
      trigger:
        type: drift
        severity: low
        maxAge: 1h
      conditions:
        - environment: ["dev", "staging"]
      action:
        type: auto_fix
      notify: [slack]

    - name: alert-production-drift
      trigger:
        type: drift
        severity: [medium, high]
      conditions:
        - environment: production
      action:
        type: require_approval
        approvalPolicy: platform-leads
      notify: [pagerduty, slack]

    - name: critical-immediate-alert
      trigger:
        type: drift
        severity: critical
      action:
        type: alert_only
      notify: [pagerduty]
      escalate:
        after: 15m
        to: on-call-lead
```

**CLI Commands**:
```bash
pf remediation status              # Show remediation status
pf remediation run                 # Run remediation check
pf remediation history             # View remediation history
pf remediation enable/disable      # Toggle auto-remediation
```

**Effort**: ~250-300 lines of Go code

---

### 1.3 Platform Diff (Environment Comparison)

**Objective**: Compare platform state across environments.

**Existing Components Used**:
- `internal/state/` - State management
- `internal/parser/` - Configuration parsing

**Design**:

```go
// internal/diff/diff.go
package diff

type PlatformDiff struct {
    Source      string           `json:"source"`
    Target      string           `json:"target"`
    Differences []ResourceDiff   `json:"differences"`
    OnlyInSource []string        `json:"only_in_source"`
    OnlyInTarget []string        `json:"only_in_target"`
    Summary     DiffSummary      `json:"summary"`
}

type ResourceDiff struct {
    Resource    string            `json:"resource"`
    Type        string            `json:"type"`
    Path        string            `json:"path"`
    SourceValue interface{}       `json:"source_value"`
    TargetValue interface{}       `json:"target_value"`
    Severity    string            `json:"severity"`
}

type Differ struct {
    stateBackend state.Backend
}

func (d *Differ) Compare(source, target string) (*PlatformDiff, error) {
    // Load both states
    // Deep compare resources
    // Identify additions, deletions, modifications
}
```

**CLI Command**:
```bash
pf diff staging production
pf diff staging production --resource argocd
pf diff staging production --output yaml
```

**Output**:
```
Platform Diff: staging → production
═══════════════════════════════════════════════════════════════

Summary: 8 differences, 2 only in staging, 1 only in production

┌────────────────────┬─────────────────┬─────────────────┬──────────┐
│ Resource           │ Staging         │ Production      │ Severity │
├────────────────────┼─────────────────┼─────────────────┼──────────┤
│ argocd.replicas    │ 1               │ 3               │ info     │
│ prometheus.retention│ 7d             │ 30d             │ info     │
│ web-api.replicas   │ 2               │ 5               │ info     │
│ web-api.resources  │ 500m/512Mi      │ 2000m/2Gi       │ warning  │
│ grafana.alerts     │ disabled        │ enabled         │ warning  │
└────────────────────┴─────────────────┴─────────────────┴──────────┘

Only in Staging:
• feature-flag-service (PR environment)
• test-database

Only in Production:
• disaster-recovery-replica
```

**Effort**: ~100-150 lines of Go code

---

### 1.4 Enhanced Service Catalog

**Objective**: Add ownership, dependencies, and API tracking to the existing service catalog.

**Existing Components Used**:
- `internal/service/registry.go`
- `internal/service/manager.go`
- `pkg/types/service.go`

**Design Enhancement**:

```go
// Enhance pkg/types/service.go
type ServiceMetadata struct {
    // Existing fields...

    // NEW: Ownership
    Owner       OwnerInfo       `yaml:"owner" json:"owner"`

    // NEW: Dependencies
    Dependencies []DependencyRef `yaml:"dependencies" json:"dependencies"`
    Dependents   []DependencyRef `yaml:"dependents,omitempty" json:"dependents,omitempty"`

    // NEW: APIs
    APIs        []APISpec       `yaml:"apis,omitempty" json:"apis,omitempty"`

    // NEW: Links
    Links       ServiceLinks    `yaml:"links,omitempty" json:"links,omitempty"`
}

type OwnerInfo struct {
    Team        string   `yaml:"team" json:"team"`
    Slack       string   `yaml:"slack,omitempty" json:"slack,omitempty"`
    Email       string   `yaml:"email,omitempty" json:"email,omitempty"`
    OnCall      string   `yaml:"oncall,omitempty" json:"oncall,omitempty"`
    Escalation  []string `yaml:"escalation,omitempty" json:"escalation,omitempty"`
}

type DependencyRef struct {
    Name        string `yaml:"name" json:"name"`
    Type        string `yaml:"type" json:"type"`         // service, database, api, queue
    Required    bool   `yaml:"required" json:"required"`
    Version     string `yaml:"version,omitempty" json:"version,omitempty"`
}

type APISpec struct {
    Name        string `yaml:"name" json:"name"`
    Type        string `yaml:"type" json:"type"`         // rest, grpc, graphql
    Spec        string `yaml:"spec,omitempty" json:"spec,omitempty"` // URL to OpenAPI/proto
    Version     string `yaml:"version" json:"version"`
}

type ServiceLinks struct {
    Repository  string `yaml:"repository,omitempty" json:"repository,omitempty"`
    Dashboard   string `yaml:"dashboard,omitempty" json:"dashboard,omitempty"`
    Runbook     string `yaml:"runbook,omitempty" json:"runbook,omitempty"`
    Documentation string `yaml:"documentation,omitempty" json:"documentation,omitempty"`
}
```

**YAML Example**:
```yaml
apiVersion: platformfoundry.io/v1
kind: Service
metadata:
  name: payment-service
  owner:
    team: payments
    slack: "#payments-eng"
    email: payments@acme.com
    oncall: "https://pagerduty.com/payments"
spec:
  type: backend
  tier: critical

  dependencies:
    - name: postgres-payments
      type: database
      required: true
    - name: redis-cache
      type: cache
      required: false
    - name: user-service
      type: service
      required: true

  apis:
    - name: Payment API
      type: rest
      spec: "./api/openapi.yaml"
      version: v2

  links:
    repository: https://github.com/acme/payment-service
    dashboard: https://grafana.acme.com/d/payments
    runbook: https://wiki.acme.com/payments/runbook
```

**CLI Enhancements**:
```bash
pf service list --team payments
pf service show payment-service --dependencies
pf service dependencies payment-service      # Show dependency tree
pf service dependents payment-service        # Who depends on this?
pf service owners --oncall                   # Show on-call contacts
```

**Effort**: ~200-250 lines of Go code (enhancements to existing)

---

## Phase 2: Differentiation Features

### 2.1 Workload Specification

**Objective**: Allow developers to define workloads without infrastructure knowledge.

**Design**:

```go
// pkg/types/workload.go
package types

type Workload struct {
    APIVersion string           `yaml:"apiVersion"`
    Kind       string           `yaml:"kind"`       // "Workload"
    Metadata   WorkloadMetadata `yaml:"metadata"`
    Spec       WorkloadSpec     `yaml:"spec"`
}

type WorkloadSpec struct {
    Containers   []Container      `yaml:"containers"`
    Dependencies []WorkloadDep    `yaml:"dependencies"`
    Scaling      *ScalingSpec     `yaml:"scaling,omitempty"`
    Network      *NetworkSpec     `yaml:"network,omitempty"`
}

type Container struct {
    Name      string            `yaml:"name"`
    Image     string            `yaml:"image"`
    Command   []string          `yaml:"command,omitempty"`
    Args      []string          `yaml:"args,omitempty"`
    Env       map[string]string `yaml:"env,omitempty"`
    Resources ResourceSpec      `yaml:"resources,omitempty"`
    Ports     []PortSpec        `yaml:"ports,omitempty"`
}

type WorkloadDep struct {
    Type   string            `yaml:"type"`   // postgres, redis, s3, kafka, etc.
    Name   string            `yaml:"name"`
    Config map[string]interface{} `yaml:"config,omitempty"`
}

// internal/workload/translator.go
type Translator struct {
    pluginRegistry *plugin.Registry
    defaultMappings map[string]string  // e.g., "postgres" -> "terraform-aws-rds"
}

func (t *Translator) Translate(w *Workload) (*Platform, error) {
    // Convert workload spec to platform configuration
    // Map dependencies to appropriate plugins
    // Generate Kubernetes manifests, Terraform configs, etc.
}
```

**YAML Example**:
```yaml
apiVersion: platformfoundry.io/v1
kind: Workload
metadata:
  name: order-service
  team: orders
spec:
  containers:
    - name: api
      image: order-service:latest
      resources:
        cpu: 500m
        memory: 512Mi
      ports:
        - name: http
          port: 8080

  dependencies:
    - type: postgres
      name: orders-db
      config:
        size: medium
        backup: daily

    - type: redis
      name: orders-cache
      config:
        size: small

    - type: kafka
      name: order-events
      config:
        topics: ["orders.created", "orders.updated"]

  scaling:
    min: 2
    max: 10
    targetCPU: 70

  network:
    ingress:
      path: /api/orders
      tls: true
```

**Platform Foundry translates this to**:
- Kubernetes Deployment + Service + HPA
- Terraform RDS PostgreSQL
- Terraform ElastiCache Redis
- MSK Kafka topic configuration
- Ingress configuration

**Effort**: ~400-500 lines of Go code

---

### 2.2 Platform Promises (Self-Service Contracts)

**Objective**: Expose platform capabilities as self-service APIs.

**Design**:

```go
// pkg/types/promise.go
package types

type Promise struct {
    APIVersion string          `yaml:"apiVersion"`
    Kind       string          `yaml:"kind"`       // "Promise"
    Metadata   PromiseMetadata `yaml:"metadata"`
    Spec       PromiseSpec     `yaml:"spec"`
}

type PromiseSpec struct {
    Description string           `yaml:"description"`
    Provider    string           `yaml:"provider"`    // Which plugin fulfills this
    Category    string           `yaml:"category"`    // database, cache, queue, etc.

    // Input schema
    Inputs      []PromiseInput   `yaml:"inputs"`

    // What gets created
    Outputs     []PromiseOutput  `yaml:"outputs"`

    // Constraints
    Policies    []string         `yaml:"policies,omitempty"`
    Approval    *ApprovalSpec    `yaml:"approval,omitempty"`
}

type PromiseInput struct {
    Name        string        `yaml:"name"`
    Type        string        `yaml:"type"`
    Description string        `yaml:"description"`
    Required    bool          `yaml:"required"`
    Default     interface{}   `yaml:"default,omitempty"`
    Enum        []string      `yaml:"enum,omitempty"`
    Validation  string        `yaml:"validation,omitempty"`
}

type PromiseOutput struct {
    Name        string `yaml:"name"`
    Type        string `yaml:"type"`
    Description string `yaml:"description"`
}

// internal/promise/manager.go
type PromiseManager struct {
    promises    map[string]*Promise
    plugins     *plugin.Registry
    policy      *policy.Engine
    workflow    *workflow.Engine
}

func (m *PromiseManager) Request(promiseName string, inputs map[string]interface{}) (*PromiseInstance, error) {
    // Validate inputs against schema
    // Check policies
    // Create workflow if approval required
    // Provision via plugin
    // Return outputs
}
```

**Promise Definition**:
```yaml
apiVersion: platformfoundry.io/v1
kind: Promise
metadata:
  name: postgresql-database
  labels:
    category: database
    tier: production-ready
spec:
  description: "Production-ready PostgreSQL database with backups and monitoring"
  provider: terraform-aws
  category: database

  inputs:
    - name: name
      type: string
      description: "Database name"
      required: true
      validation: "^[a-z][a-z0-9-]{2,20}$"

    - name: size
      type: enum
      description: "Database size"
      required: true
      enum: [small, medium, large, xlarge]
      default: medium

    - name: version
      type: string
      description: "PostgreSQL version"
      default: "15"
      enum: ["13", "14", "15", "16"]

    - name: backup_retention
      type: number
      description: "Backup retention in days"
      default: 7

    - name: multi_az
      type: boolean
      description: "Enable multi-AZ deployment"
      default: false

  outputs:
    - name: connection_string
      type: secret
      description: "Database connection string"
    - name: host
      type: string
      description: "Database hostname"
    - name: port
      type: number
      description: "Database port"
    - name: readonly_endpoint
      type: string
      description: "Read replica endpoint"

  policies:
    - require-team-label
    - cost-limit-database

  approval:
    required: true
    policy: database-provisioning
    environments: [production]
```

**Request a Promise**:
```yaml
apiVersion: platformfoundry.io/v1
kind: PromiseRequest
metadata:
  name: orders-database
  team: orders
spec:
  promise: postgresql-database
  inputs:
    name: orders-db
    size: medium
    version: "15"
    backup_retention: 14
    multi_az: true
```

**CLI Commands**:
```bash
pf promise list                          # List available promises
pf promise show postgresql-database      # Show promise details
pf promise request postgresql-database   # Interactive request
pf promise request -f request.yaml       # Request from file
pf promise instances                     # List my instances
pf promise status orders-database        # Check status
```

**Effort**: ~350-400 lines of Go code

---

## Phase 3: AI Integration

### 3.1 AI Assistant

**Objective**: Natural language interface to the platform.

**Existing Components Used**:
- `internal/intelligence/` - Recommendations engine

**Design**:

```go
// internal/ai/assistant.go
package ai

type Assistant struct {
    provider    LLMProvider
    catalog     *service.Catalog
    state       state.Backend
    health      *health.Checker
    tools       []AssistantTool
}

type LLMProvider interface {
    Complete(ctx context.Context, prompt string) (string, error)
    Stream(ctx context.Context, prompt string) (<-chan string, error)
}

type AssistantTool struct {
    Name        string
    Description string
    Execute     func(args map[string]interface{}) (interface{}, error)
}

func (a *Assistant) Query(ctx context.Context, query string) (*AssistantResponse, error) {
    // 1. Parse natural language query
    // 2. Identify intent (search, status, action, explain)
    // 3. Execute relevant tools
    // 4. Format response
}
```

**Example Queries**:
```bash
pf ask "show me services with high error rates owned by team-payments"
pf ask "what caused the outage yesterday?"
pf ask "recommend cost optimizations for staging"
pf ask "create a new postgres database for the orders team"
pf ask "why is the health score dropping?"
```

**Effort**: ~300-400 lines of Go code (plus LLM API integration)

---

## Implementation Priority

| Feature | Phase | Effort | Impact | Dependencies |
|---------|-------|--------|--------|--------------|
| Health Score | 1 | 3 hrs | High | lint, drift, policy, cost |
| Platform Diff | 1 | 3 hrs | High | state |
| Auto-Remediation | 1 | 1 day | High | drift, policy, notify, workflow |
| Enhanced Catalog | 1 | 1 day | Medium | service registry |
| Workload Spec | 2 | 1 week | High | parser, plugins |
| Promises | 2 | 1 week | High | plugins, workflow, policy |
| AI Assistant | 3 | 1-2 weeks | High | LLM provider, all components |

---

## File Structure

```
internal/
├── health/
│   ├── health.go           # Health score aggregation
│   └── health_test.go
├── remediation/
│   ├── remediation.go      # Auto-remediation engine
│   ├── rules.go            # Rule matching
│   └── remediation_test.go
├── diff/
│   ├── diff.go             # Platform diff
│   └── diff_test.go
├── promise/
│   ├── manager.go          # Promise management
│   ├── validator.go        # Input validation
│   └── promise_test.go
├── workload/
│   ├── translator.go       # Workload to Platform translation
│   ├── mappings.go         # Dependency mappings
│   └── workload_test.go
├── ai/
│   ├── assistant.go        # AI assistant
│   ├── providers/          # LLM providers (OpenAI, Claude, etc.)
│   └── tools.go            # Assistant tools
└── ...

pkg/types/
├── workload.go             # Workload spec types
├── promise.go              # Promise types
└── ...

internal/cli/
├── health.go               # pf health command
├── diff.go                 # pf diff command (enhance existing)
├── remediation.go          # pf remediation commands
├── promise.go              # pf promise commands
├── workload.go             # pf workload commands
└── ask.go                  # pf ask command
```

---

## Success Metrics

| Metric | Target |
|--------|--------|
| Time to first platform | < 30 minutes |
| Developer self-service rate | > 80% |
| Mean time to remediate drift | < 1 hour |
| Platform health score adoption | > 90% |
| Promise fulfillment time | < 5 minutes |

---

## Competitive Positioning

After implementing these features:

```
┌─────────────────────────────────────────────────────────────────┐
│                    PLATFORM FOUNDRY                             │
│                                                                 │
│  "The only platform that combines Humanitec's orchestration,   │
│   Backstage's catalog, Crossplane's self-healing, and          │
│   AI-powered operations - in a single open-source tool."       │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ Workload Spec │ Catalog │ Promises │ Self-Heal │ AI    │   │
│  │   (Score)     │ (BStage)│ (Kratix) │ (Xplane) │ (Port) │   │
│  └─────────────────────────────────────────────────────────┘   │
│                          │                                      │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │         Multi-IaC Orchestration Engine                  │   │
│  │    Terraform │ Pulumi │ CDK │ Crossplane │ K8s          │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

**Key Differentiators**:
1. **Full-stack** - Not just infra, not just portal
2. **Multi-IaC** - Use any tool, not locked to one
3. **Self-healing** - Auto-remediation built-in
4. **Intent-based** - Workload spec, not infrastructure details
5. **AI-native** - Natural language operations

---

## Implementation Status

### Phase 1: Quick Wins - COMPLETED

| Feature | Status | Files | Lines of Code |
|---------|--------|-------|---------------|
| Platform Health Score | Done | `internal/health/health.go`, `internal/cli/health.go` | ~600 |
| Auto-Remediation Engine | Done | `internal/remediation/remediation.go`, `internal/cli/remediation.go` | ~800 |
| Platform Diff | Done | `internal/diff/diff.go`, `internal/cli/diff.go` | ~500 |
| Enhanced Service Catalog | Done | `pkg/types/service.go`, `internal/service/` | ~400 |

**Total Phase 1**: ~2,300 lines of production code

### Phase 2: Differentiation Features - COMPLETED

| Feature | Status | Files | Lines of Code |
|---------|--------|-------|---------------|
| Workload Specification | Done | `pkg/types/workload.go` | ~380 |
| Workload Translator | Done | `internal/workload/translator.go` | ~650 |
| Workload CLI | Done | `internal/cli/workload.go` | ~760 |
| Promise Types | Done | `pkg/types/promise.go` | ~420 |
| Promise Manager | Done | `internal/promise/manager.go` | ~520 |
| Promise CLI | Done | `internal/cli/promise.go` | ~480 |

**Total Phase 2**: ~3,210 lines of production code

### Phase 3: AI Integration - COMPLETED

| Feature | Status | Files | Lines of Code |
|---------|--------|-------|---------------|
| AI Assistant Core | Done | `internal/ai/assistant.go` | ~375 |
| LLM Provider Interface | Done | `internal/ai/provider.go` | ~150 |
| Claude Provider | Done | `internal/ai/providers/claude.go` | ~200 |
| OpenAI Provider | Done | `internal/ai/providers/openai.go` | ~200 |
| Tool Definitions (12 tools) | Done | `internal/ai/tools/tools.go` | ~960 |
| `pf ask` CLI | Done | `internal/cli/ask.go` | ~345 |

**Total Phase 3**: ~2,230 lines of production code

### Phase 4: Advanced Features - COMPLETED

| Feature | Status | Files | Lines of Code |
|---------|--------|-------|---------------|
| GitOps Integration | Done | `internal/gitops/manager.go`, `github.go`, `pkg/types/gitops.go`, `internal/cli/gitops.go` | ~900 |
| Cost Forecasting | Done | `internal/cost/forecast.go`, `internal/cli/forecast.go` | ~550 |
| Chaos Engineering | Done | `internal/chaos/engine.go`, `pkg/types/chaos.go`, `internal/cli/chaos.go` | ~750 |
| Multi-Cloud Abstraction | Done | `internal/multicloud/manager.go`, `pkg/types/multicloud.go` | ~600 |
| Compliance Automation | Done | `internal/compliance/scanner.go`, `pkg/types/compliance.go` | ~650 |

**Total Phase 4**: ~3,450 lines of production code

---

## Phase 3: AI Integration - Detailed Design

### 3.1 Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                      pf ask "query"                              │
└─────────────────────┬───────────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                    AI Assistant                                  │
│  ┌──────────────┬──────────────┬──────────────┬─────────────┐  │
│  │ Query Parser │ Intent       │ Tool         │ Response    │  │
│  │              │ Classifier   │ Executor     │ Formatter   │  │
│  └──────────────┴──────────────┴──────────────┴─────────────┘  │
└─────────────────────┬───────────────────────────────────────────┘
                      │
         ┌────────────┼────────────┐
         ▼            ▼            ▼
┌─────────────┐ ┌──────────┐ ┌─────────────┐
│ LLM Provider│ │ Platform │ │ Knowledge   │
│ (Claude/   │ │ Tools    │ │ Base        │
│  OpenAI)   │ │          │ │             │
└─────────────┘ └──────────┘ └─────────────┘
```

### 3.2 LLM Provider Interface

```go
// internal/ai/provider.go
package ai

import (
    "context"
)

// LLMProvider defines the interface for language model providers
type LLMProvider interface {
    // Name returns the provider name (e.g., "claude", "openai")
    Name() string

    // Complete sends a prompt and returns a complete response
    Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)

    // Stream sends a prompt and streams the response
    Stream(ctx context.Context, req *CompletionRequest) (<-chan StreamChunk, error)

    // SupportsTools returns whether the provider supports tool/function calling
    SupportsTools() bool
}

// CompletionRequest represents a request to the LLM
type CompletionRequest struct {
    Messages    []Message           `json:"messages"`
    Tools       []ToolDefinition    `json:"tools,omitempty"`
    MaxTokens   int                 `json:"max_tokens,omitempty"`
    Temperature float64             `json:"temperature,omitempty"`
    SystemPrompt string             `json:"system_prompt,omitempty"`
}

// Message represents a conversation message
type Message struct {
    Role    string      `json:"role"`    // system, user, assistant, tool
    Content string      `json:"content"`
    Name    string      `json:"name,omitempty"`
    ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ToolDefinition defines a tool the LLM can use
type ToolDefinition struct {
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Parameters  map[string]interface{} `json:"parameters"`
}

// ToolCall represents a tool invocation by the LLM
type ToolCall struct {
    ID        string                 `json:"id"`
    Name      string                 `json:"name"`
    Arguments map[string]interface{} `json:"arguments"`
}

// CompletionResponse represents the LLM response
type CompletionResponse struct {
    Content   string     `json:"content"`
    ToolCalls []ToolCall `json:"tool_calls,omitempty"`
    Usage     TokenUsage `json:"usage"`
    Model     string     `json:"model"`
}

// TokenUsage tracks token consumption
type TokenUsage struct {
    PromptTokens     int `json:"prompt_tokens"`
    CompletionTokens int `json:"completion_tokens"`
    TotalTokens      int `json:"total_tokens"`
}

// StreamChunk represents a streaming response chunk
type StreamChunk struct {
    Content string `json:"content"`
    Done    bool   `json:"done"`
    Error   error  `json:"error,omitempty"`
}
```

### 3.3 Claude Provider Implementation

```go
// internal/ai/providers/claude.go
package providers

import (
    "context"
    "encoding/json"
    "net/http"

    "github.com/platformfoundry/pf-ce/internal/ai"
)

// ClaudeProvider implements the LLM provider interface for Anthropic Claude
type ClaudeProvider struct {
    apiKey     string
    model      string
    httpClient *http.Client
    baseURL    string
}

// ClaudeConfig configures the Claude provider
type ClaudeConfig struct {
    APIKey  string `yaml:"apiKey" json:"apiKey"`
    Model   string `yaml:"model" json:"model"`     // claude-3-opus, claude-3-sonnet, etc.
    BaseURL string `yaml:"baseURL" json:"baseURL"` // For enterprise deployments
}

// NewClaudeProvider creates a new Claude provider
func NewClaudeProvider(config ClaudeConfig) (*ClaudeProvider, error) {
    if config.APIKey == "" {
        return nil, fmt.Errorf("API key is required")
    }

    model := config.Model
    if model == "" {
        model = "claude-3-sonnet-20240229"
    }

    baseURL := config.BaseURL
    if baseURL == "" {
        baseURL = "https://api.anthropic.com/v1"
    }

    return &ClaudeProvider{
        apiKey:     config.APIKey,
        model:      model,
        baseURL:    baseURL,
        httpClient: &http.Client{Timeout: 120 * time.Second},
    }, nil
}

func (p *ClaudeProvider) Name() string {
    return "claude"
}

func (p *ClaudeProvider) SupportsTools() bool {
    return true
}

func (p *ClaudeProvider) Complete(ctx context.Context, req *ai.CompletionRequest) (*ai.CompletionResponse, error) {
    // Convert to Claude API format
    // Make HTTP request
    // Parse response
    // Return standardized response
}
```

### 3.4 Platform Tools

The AI assistant has access to these platform tools:

```go
// internal/ai/tools/tools.go
package tools

// Tool represents an executable tool for the AI assistant
type Tool struct {
    Name        string
    Description string
    Parameters  ParameterSchema
    Execute     func(ctx context.Context, args map[string]interface{}) (interface{}, error)
}

// GetPlatformTools returns all available platform tools
func GetPlatformTools() []Tool {
    return []Tool{
        // Service Discovery
        {
            Name:        "list_services",
            Description: "List services in the platform, optionally filtered by team, type, or health status",
            Parameters: ParameterSchema{
                Type: "object",
                Properties: map[string]Property{
                    "team":   {Type: "string", Description: "Filter by team name"},
                    "type":   {Type: "string", Description: "Filter by service type (backend, frontend, etc.)"},
                    "health": {Type: "string", Description: "Filter by health status (healthy, degraded, down)"},
                },
            },
            Execute: executeListServices,
        },

        // Health & Status
        {
            Name:        "get_health_score",
            Description: "Get the platform health score and breakdown by category",
            Parameters: ParameterSchema{
                Type: "object",
                Properties: map[string]Property{
                    "platform": {Type: "string", Description: "Platform name (optional)"},
                },
            },
            Execute: executeGetHealthScore,
        },

        // Drift Detection
        {
            Name:        "check_drift",
            Description: "Check for configuration drift across resources",
            Parameters: ParameterSchema{
                Type: "object",
                Properties: map[string]Property{
                    "resource": {Type: "string", Description: "Specific resource to check (optional)"},
                    "severity": {Type: "string", Description: "Minimum severity to report"},
                },
            },
            Execute: executeCheckDrift,
        },

        // Cost Analysis
        {
            Name:        "analyze_costs",
            Description: "Analyze platform costs and identify optimization opportunities",
            Parameters: ParameterSchema{
                Type: "object",
                Properties: map[string]Property{
                    "environment": {Type: "string", Description: "Environment to analyze"},
                    "timeRange":   {Type: "string", Description: "Time range (7d, 30d, 90d)"},
                },
            },
            Execute: executeAnalyzeCosts,
        },

        // Environment Comparison
        {
            Name:        "compare_environments",
            Description: "Compare configuration between two environments",
            Parameters: ParameterSchema{
                Type: "object",
                Properties: map[string]Property{
                    "source": {Type: "string", Description: "Source environment"},
                    "target": {Type: "string", Description: "Target environment"},
                },
                Required: []string{"source", "target"},
            },
            Execute: executeCompareEnvironments,
        },

        // Promise Management
        {
            Name:        "list_promises",
            Description: "List available platform promises (self-service capabilities)",
            Parameters: ParameterSchema{
                Type: "object",
                Properties: map[string]Property{
                    "category": {Type: "string", Description: "Filter by category (database, cache, queue, etc.)"},
                },
            },
            Execute: executeListPromises,
        },

        // Workload Management
        {
            Name:        "list_workloads",
            Description: "List deployed workloads",
            Parameters: ParameterSchema{
                Type: "object",
                Properties: map[string]Property{
                    "team":        {Type: "string", Description: "Filter by team"},
                    "environment": {Type: "string", Description: "Filter by environment"},
                },
            },
            Execute: executeListWorkloads,
        },

        // Troubleshooting
        {
            Name:        "get_recent_events",
            Description: "Get recent platform events and changes",
            Parameters: ParameterSchema{
                Type: "object",
                Properties: map[string]Property{
                    "hours":    {Type: "number", Description: "Look back hours (default: 24)"},
                    "severity": {Type: "string", Description: "Minimum severity"},
                    "resource": {Type: "string", Description: "Filter by resource"},
                },
            },
            Execute: executeGetRecentEvents,
        },

        // Recommendations
        {
            Name:        "get_recommendations",
            Description: "Get actionable recommendations for the platform",
            Parameters: ParameterSchema{
                Type: "object",
                Properties: map[string]Property{
                    "category": {Type: "string", Description: "Category: cost, security, reliability, performance"},
                },
            },
            Execute: executeGetRecommendations,
        },

        // Actions (with confirmation)
        {
            Name:        "request_promise",
            Description: "Request a new infrastructure resource via a promise",
            Parameters: ParameterSchema{
                Type: "object",
                Properties: map[string]Property{
                    "promise": {Type: "string", Description: "Promise name"},
                    "name":    {Type: "string", Description: "Instance name"},
                    "team":    {Type: "string", Description: "Team name"},
                    "inputs":  {Type: "object", Description: "Promise inputs"},
                },
                Required: []string{"promise", "name", "team"},
            },
            Execute: executeRequestPromise,
        },
    }
}
```

### 3.5 AI Assistant Implementation

```go
// internal/ai/assistant.go
package ai

import (
    "context"
    "fmt"
    "strings"

    "github.com/platformfoundry/pf-ce/internal/ai/tools"
    "github.com/platformfoundry/pf-ce/internal/health"
    "github.com/platformfoundry/pf-ce/internal/promise"
    "github.com/platformfoundry/pf-ce/internal/service"
)

// Assistant provides natural language interface to the platform
type Assistant struct {
    provider       LLMProvider
    tools          []tools.Tool
    healthChecker  *health.Checker
    promiseManager *promise.Manager
    serviceRegistry *service.Registry

    // Conversation history for context
    history        []Message
    maxHistory     int
}

// AssistantConfig configures the AI assistant
type AssistantConfig struct {
    Provider      LLMProvider
    MaxHistory    int  // Maximum conversation turns to remember
    Verbose       bool // Show tool calls in output
}

// NewAssistant creates a new AI assistant
func NewAssistant(config AssistantConfig) *Assistant {
    maxHistory := config.MaxHistory
    if maxHistory == 0 {
        maxHistory = 10
    }

    return &Assistant{
        provider:   config.Provider,
        tools:      tools.GetPlatformTools(),
        maxHistory: maxHistory,
        history:    make([]Message, 0),
    }
}

// Query processes a natural language query
func (a *Assistant) Query(ctx context.Context, query string) (*AssistantResponse, error) {
    // Add user message to history
    a.history = append(a.history, Message{
        Role:    "user",
        Content: query,
    })

    // Trim history if too long
    if len(a.history) > a.maxHistory*2 {
        a.history = a.history[len(a.history)-a.maxHistory*2:]
    }

    // Build request with system prompt
    req := &CompletionRequest{
        SystemPrompt: a.buildSystemPrompt(),
        Messages:     a.history,
        Tools:        a.buildToolDefinitions(),
        MaxTokens:    4096,
        Temperature:  0.7,
    }

    // Get completion from LLM
    resp, err := a.provider.Complete(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("LLM completion failed: %w", err)
    }

    // Process tool calls if any
    if len(resp.ToolCalls) > 0 {
        return a.processToolCalls(ctx, resp)
    }

    // Add assistant response to history
    a.history = append(a.history, Message{
        Role:    "assistant",
        Content: resp.Content,
    })

    return &AssistantResponse{
        Content:    resp.Content,
        ToolsUsed:  nil,
        TokenUsage: resp.Usage,
    }, nil
}

// buildSystemPrompt creates the system prompt for the assistant
func (a *Assistant) buildSystemPrompt() string {
    return `You are an AI assistant for Platform Foundry, an Internal Developer Platform (IDP) orchestration tool.

Your capabilities include:
- Querying platform health, services, and workloads
- Analyzing costs and recommending optimizations
- Detecting configuration drift and compliance issues
- Comparing environments
- Managing platform promises (self-service infrastructure)
- Providing troubleshooting guidance

Guidelines:
1. Be concise and actionable in your responses
2. When showing data, use tables or structured formats
3. Always explain what you're doing before executing tools
4. For destructive actions, ask for confirmation
5. Provide next steps and recommendations when appropriate
6. If you don't have enough information, ask clarifying questions

Available environments: development, staging, production
Available promise categories: database, cache, queue, storage, compute`
}

// processToolCalls executes tool calls and returns results
func (a *Assistant) processToolCalls(ctx context.Context, resp *CompletionResponse) (*AssistantResponse, error) {
    var toolResults []ToolResult

    for _, call := range resp.ToolCalls {
        // Find the tool
        var tool *tools.Tool
        for _, t := range a.tools {
            if t.Name == call.Name {
                tool = &t
                break
            }
        }

        if tool == nil {
            toolResults = append(toolResults, ToolResult{
                ToolCall: call,
                Error:    fmt.Sprintf("unknown tool: %s", call.Name),
            })
            continue
        }

        // Execute the tool
        result, err := tool.Execute(ctx, call.Arguments)
        if err != nil {
            toolResults = append(toolResults, ToolResult{
                ToolCall: call,
                Error:    err.Error(),
            })
        } else {
            toolResults = append(toolResults, ToolResult{
                ToolCall: call,
                Result:   result,
            })
        }
    }

    // Send tool results back to LLM for final response
    return a.generateFinalResponse(ctx, toolResults)
}

// AssistantResponse represents the assistant's response
type AssistantResponse struct {
    Content    string       `json:"content"`
    ToolsUsed  []ToolResult `json:"tools_used,omitempty"`
    TokenUsage TokenUsage   `json:"token_usage"`
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
    ToolCall ToolCall    `json:"tool_call"`
    Result   interface{} `json:"result,omitempty"`
    Error    string      `json:"error,omitempty"`
}
```

### 3.6 CLI Command

```go
// internal/cli/ask.go
package cli

import (
    "bufio"
    "context"
    "fmt"
    "os"
    "strings"

    "github.com/platformfoundry/pf-ce/internal/ai"
    "github.com/platformfoundry/pf-ce/internal/ai/providers"
    "github.com/spf13/cobra"
)

func init() {
    rootCmd.AddCommand(askCmd)
    askCmd.Flags().Bool("interactive", false, "Start interactive chat session")
    askCmd.Flags().Bool("verbose", false, "Show tool calls and reasoning")
    askCmd.Flags().String("provider", "claude", "LLM provider (claude, openai)")
}

var askCmd = &cobra.Command{
    Use:   "ask [query]",
    Short: "Ask the AI assistant about your platform",
    Long: `Use natural language to query your platform, get recommendations,
and perform actions. The AI assistant has access to all platform data
and can help with troubleshooting, cost optimization, and more.`,
    Example: `  # Single query
  pf ask "show me services with high error rates"

  # Cost analysis
  pf ask "what are the biggest cost optimization opportunities?"

  # Troubleshooting
  pf ask "why did the deployment fail yesterday?"

  # Interactive mode
  pf ask --interactive

  # Request infrastructure
  pf ask "create a medium postgres database for the orders team"`,
    RunE: runAsk,
}

func runAsk(cmd *cobra.Command, args []string) error {
    interactive, _ := cmd.Flags().GetBool("interactive")
    verbose, _ := cmd.Flags().GetBool("verbose")
    providerName, _ := cmd.Flags().GetString("provider")

    // Initialize provider
    provider, err := initProvider(providerName)
    if err != nil {
        return fmt.Errorf("failed to initialize AI provider: %w", err)
    }

    // Create assistant
    assistant := ai.NewAssistant(ai.AssistantConfig{
        Provider:   provider,
        MaxHistory: 20,
        Verbose:    verbose,
    })

    ctx := context.Background()

    if interactive {
        return runInteractiveMode(ctx, assistant)
    }

    if len(args) == 0 {
        return fmt.Errorf("query is required (or use --interactive)")
    }

    query := strings.Join(args, " ")
    return executeQuery(ctx, assistant, query, verbose)
}

func runInteractiveMode(ctx context.Context, assistant *ai.Assistant) error {
    fmt.Println("Platform Foundry AI Assistant")
    fmt.Println("Type 'exit' or 'quit' to end the session")
    fmt.Println(strings.Repeat("-", 50))
    fmt.Println()

    scanner := bufio.NewScanner(os.Stdin)

    for {
        fmt.Print("You: ")
        if !scanner.Scan() {
            break
        }

        query := strings.TrimSpace(scanner.Text())
        if query == "" {
            continue
        }

        if query == "exit" || query == "quit" {
            fmt.Println("Goodbye!")
            break
        }

        resp, err := assistant.Query(ctx, query)
        if err != nil {
            fmt.Printf("Error: %v\n\n", err)
            continue
        }

        fmt.Printf("\nAssistant: %s\n\n", resp.Content)
    }

    return nil
}

func executeQuery(ctx context.Context, assistant *ai.Assistant, query string, verbose bool) error {
    resp, err := assistant.Query(ctx, query)
    if err != nil {
        return err
    }

    if verbose && len(resp.ToolsUsed) > 0 {
        fmt.Println("Tools used:")
        for _, tool := range resp.ToolsUsed {
            fmt.Printf("  - %s\n", tool.ToolCall.Name)
        }
        fmt.Println()
    }

    fmt.Println(resp.Content)
    return nil
}
```

### 3.7 Example Interactions

```
$ pf ask "what's the health of the platform?"

I'll check the platform health score for you.

Platform Health: production
═══════════════════════════════════════════════════════
Overall Score: 78/100 ⚠️

Category Breakdown:
┌─────────────────────┬───────┬────────┐
│ Category            │ Score │ Status │
├─────────────────────┼───────┼────────┤
│ Configuration       │ 85    │ ✅     │
│ Drift               │ 65    │ ⚠️     │
│ Policy Compliance   │ 92    │ ✅     │
│ Cost Efficiency     │ 68    │ ⚠️     │
│ Security            │ 80    │ ✅     │
└─────────────────────┴───────┴────────┘

Top Issues:
1. 3 resources with configuration drift (ArgoCD, Prometheus)
2. Staging environment 23% over budget
3. 2 container images need security updates

Recommendations:
• Run `pf drift fix` to remediate drift
• Review staging resource allocation
• Update container base images

$ pf ask "compare staging to production"

Comparing environments: staging → production

┌────────────────────────┬─────────────┬─────────────┐
│ Resource               │ Staging     │ Production  │
├────────────────────────┼─────────────┼─────────────┤
│ argocd.replicas        │ 1           │ 3           │
│ order-service.replicas │ 2           │ 5           │
│ prometheus.retention   │ 7d          │ 30d         │
│ web-api.resources      │ 500m/512Mi  │ 2000m/2Gi   │
└────────────────────────┴─────────────┴─────────────┘

Only in Staging: feature-flag-service, test-database
Only in Production: disaster-recovery-replica

The main differences are scaling and retention settings,
which is expected between environments.

$ pf ask "create a postgres database for the orders team"

I can help you create a PostgreSQL database. Let me get the details:

The 'postgresql-database' promise is available with these options:
- Size: small, medium, large, xlarge
- Version: 13, 14, 15, 16
- Backup retention: 1-35 days
- Multi-AZ: true/false

Please provide:
1. Database name (e.g., orders-db)
2. Size tier
3. Environment (staging/production)

Or I can use defaults: medium size, PostgreSQL 15, 7-day backup.

Would you like to proceed with defaults or customize?
```

---

## Phase 4: Advanced Features (Future)

### 4.1 GitOps Integration

**Objective**: Full GitOps workflow with PR-based changes.

```yaml
apiVersion: platformfoundry.io/v1
kind: GitOpsConfig
metadata:
  name: platform-gitops
spec:
  repository:
    url: https://github.com/acme/platform-config
    branch: main
    path: environments/

  pullRequest:
    enabled: true
    autoMerge: false
    requiredApprovals: 2

  sync:
    interval: 5m
    prune: true

  notifications:
    slack: "#platform-changes"
```

### 4.2 Cost Forecasting

**Objective**: Predict future costs based on trends and planned changes.

```go
// internal/cost/forecast.go
type CostForecast struct {
    Current      float64           `json:"current"`
    Predicted    float64           `json:"predicted"`
    Trend        string            `json:"trend"`        // increasing, stable, decreasing
    Confidence   float64           `json:"confidence"`   // 0-1
    BreakdownBy  map[string]float64 `json:"breakdown"`
    Recommendations []CostRecommendation `json:"recommendations"`
}

func (f *Forecaster) Predict(timeframe string) (*CostForecast, error) {
    // Analyze historical data
    // Apply ML model for prediction
    // Consider planned changes (new workloads, scaling)
    // Generate recommendations
}
```

### 4.3 Chaos Engineering Integration

**Objective**: Built-in chaos experiments for reliability testing.

```yaml
apiVersion: platformfoundry.io/v1
kind: ChaosExperiment
metadata:
  name: payment-service-resilience
spec:
  target:
    service: payment-service
    environment: staging

  experiments:
    - name: pod-failure
      type: pod-kill
      probability: 0.3
      duration: 5m

    - name: network-latency
      type: network-delay
      latency: 500ms
      jitter: 100ms
      duration: 10m

    - name: dependency-failure
      type: service-unavailable
      target: postgres-payments
      duration: 2m

  schedule:
    cron: "0 2 * * 1-5"  # Weekdays at 2 AM

  safety:
    maxImpact: 10%
    rollbackOnError: true
    healthCheckInterval: 30s
```

### 4.4 Multi-Cloud Abstraction

**Objective**: Unified interface across cloud providers.

```yaml
apiVersion: platformfoundry.io/v1
kind: MultiCloudPlatform
metadata:
  name: global-platform
spec:
  primary:
    provider: aws
    region: us-east-1

  secondary:
    - provider: gcp
      region: us-central1
      role: disaster-recovery

    - provider: azure
      region: eastus
      role: edge

  services:
    database:
      type: postgres
      provider: primary
      replication:
        - target: secondary[0]
          mode: async

    cache:
      type: redis
      provider: all
      consistency: eventual

  traffic:
    strategy: latency-based
    healthCheck:
      interval: 30s
      threshold: 3
```

### 4.5 Compliance Automation

**Objective**: Automated compliance checking and reporting.

```yaml
apiVersion: platformfoundry.io/v1
kind: ComplianceFramework
metadata:
  name: soc2-compliance
spec:
  framework: SOC2
  scope:
    - type: Trust Services Criteria
      categories: [security, availability, confidentiality]

  controls:
    - id: CC6.1
      name: "Logical Access Controls"
      automated: true
      checks:
        - rbac-enabled
        - mfa-required
        - audit-logging

    - id: CC7.2
      name: "System Monitoring"
      automated: true
      checks:
        - monitoring-enabled
        - alerting-configured
        - log-retention-90d

  reporting:
    schedule: monthly
    recipients:
      - compliance@acme.com
    format: [pdf, json]

  evidence:
    collection: automatic
    storage: s3://acme-compliance/evidence/
```

---

## Security Considerations

### Authentication & Authorization

```go
// All AI operations require authentication
type AISecurityConfig struct {
    // Require authentication for AI queries
    RequireAuth    bool     `yaml:"requireAuth"`

    // Roles that can use AI assistant
    AllowedRoles   []string `yaml:"allowedRoles"`

    // Actions that require additional approval
    ProtectedActions []string `yaml:"protectedActions"`

    // Audit all AI interactions
    AuditEnabled   bool     `yaml:"auditEnabled"`

    // Rate limiting
    RateLimit      int      `yaml:"rateLimit"` // queries per minute
}
```

### Data Privacy

- **No PII in prompts**: Scrub sensitive data before sending to LLM
- **Local processing option**: Support for local LLM models
- **Audit logging**: All AI interactions are logged
- **Data retention**: Configurable conversation history retention

### Tool Execution Safety

```go
// Tools with side effects require confirmation
type ToolSafety struct {
    // Read-only tools execute without confirmation
    ReadOnly       bool

    // Tools that modify state
    RequireConfirm bool

    // Tools restricted to certain environments
    AllowedEnvs    []string

    // Maximum blast radius
    MaxAffected    int
}
```

---

## Performance Considerations

### Caching Strategy

```go
// Cache AI responses for common queries
type AICache struct {
    // Cache identical queries
    QueryCache     *lru.Cache
    QueryTTL       time.Duration

    // Cache tool results
    ToolResultCache *lru.Cache
    ToolResultTTL   time.Duration

    // Invalidation triggers
    InvalidateOn   []string // events that invalidate cache
}
```

### Optimization Targets

| Operation | Target Latency | Strategy |
|-----------|---------------|----------|
| Simple query | < 2s | Direct LLM call |
| Tool-based query | < 5s | Parallel tool execution |
| Complex analysis | < 15s | Streaming response |
| Interactive mode | < 3s per turn | Context caching |

---

## Testing Strategy

### Unit Tests

```go
// Test tool execution
func TestListServiceseTool(t *testing.T) {
    tool := tools.GetPlatformTools()[0]
    assert.Equal(t, "list_services", tool.Name)

    result, err := tool.Execute(context.Background(), map[string]interface{}{
        "team": "orders",
    })

    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```

### Integration Tests

```go
// Test full AI query flow with mock provider
func TestAIQueryIntegration(t *testing.T) {
    mockProvider := NewMockLLMProvider()
    assistant := ai.NewAssistant(ai.AssistantConfig{
        Provider: mockProvider,
    })

    resp, err := assistant.Query(context.Background(), "list services")
    assert.NoError(t, err)
    assert.Contains(t, resp.Content, "service")
}
```

### E2E Tests

```bash
# Test interactive mode
echo "list services" | pf ask --interactive | grep -q "service"

# Test single query
pf ask "what is the health score" | grep -q "Score"
```

---

## Migration Guide

### From Phase 2 to Phase 3

1. **Install AI provider credentials**:
   ```bash
   pf config set ai.provider claude
   pf config set ai.apiKey $ANTHROPIC_API_KEY
   ```

2. **Enable AI features**:
   ```bash
   pf config set ai.enabled true
   ```

3. **Verify installation**:
   ```bash
   pf ask "hello"
   ```

### Configuration

```yaml
# ~/.pf/config.yaml
ai:
  enabled: true
  provider: claude  # or openai
  apiKey: ${ANTHROPIC_API_KEY}
  model: claude-3-sonnet-20240229
  maxTokens: 4096
  temperature: 0.7

  # Privacy settings
  auditEnabled: true
  dataRetention: 30d

  # Safety settings
  confirmDestructive: true
  allowedEnvironments:
    - development
    - staging
```

---

## Appendix: API Reference

### Workload API

```
POST /api/v1/workloads
GET  /api/v1/workloads
GET  /api/v1/workloads/{name}
PUT  /api/v1/workloads/{name}
DELETE /api/v1/workloads/{name}
POST /api/v1/workloads/{name}/translate
```

### Promise API

```
GET  /api/v1/promises
GET  /api/v1/promises/{name}
POST /api/v1/promises/{name}/request
GET  /api/v1/promise-instances
GET  /api/v1/promise-instances/{name}
DELETE /api/v1/promise-instances/{name}
POST /api/v1/promise-instances/{name}/approve
POST /api/v1/promise-instances/{name}/reject
```

### AI API

```
POST /api/v1/ai/query
POST /api/v1/ai/stream
GET  /api/v1/ai/tools
GET  /api/v1/ai/history
DELETE /api/v1/ai/history
```

---

## Changelog

### v2.0.0 (January 2026)

**Phase 1 - Completed**
- Platform Health Score with weighted category scoring
- Auto-Remediation Engine with rule-based policies
- Platform Diff for environment comparison
- Enhanced Service Catalog with ownership tracking

**Phase 2 - Completed**
- Workload Specification for developer-friendly deployments
- Platform Promises for self-service infrastructure
- CLI commands: `pf workload`, `pf promise`
- Example configurations and documentation

**Phase 3 - Completed**
- AI Assistant with conversation management and tool execution
- LLM provider interfaces (Claude, OpenAI)
- 12 platform tools for AI integration
- Interactive CLI mode with `pf ask`

**Phase 4 - Completed**
- GitOps Integration with PR workflows and environment promotions
- Cost Forecasting with prediction models and recommendations
- Chaos Engineering with experiment management and safety controls
- Multi-Cloud Abstraction for unified resource management
- Compliance Automation with policy scanning and violation tracking

### Roadmap

- **Q1 2026**: Phase 3 AI Integration ✅
- **Q1-Q2 2026**: Phase 4 Advanced Features ✅
- **Q3 2026**: Enterprise Features
- **Q4 2026**: GA Release
