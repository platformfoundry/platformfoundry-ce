# Platform Foundry - Feature Rationalization

## Executive Summary

Platform Foundry has grown to 67 packages and 75+ CLI commands. Analysis reveals significant scope creep, with ~40% of features either duplicating ecosystem tools or serving niche use cases. This document identifies features for removal, consolidation, or deprioritization to sharpen the product focus.

**Core Value Proposition**: Declarative platform provisioning that orchestrates tools, not replaces them.

**Target State**: 45 packages, 40 CLI commands, 40% reduction in maintenance burden.

---

## Feature Categories

### Tier 1: Core (Keep & Invest)

These features define Platform Foundry's value:

| Feature | Package | CLI Commands | Rationale |
|---------|---------|--------------|-----------|
| Platform Definition | `internal/platform/` | `apply`, `plan`, `delete` | Core declarative engine |
| Engine Orchestration | `internal/engine/` | - | Parallel execution with DAG |
| State Management | `internal/state/` | `get`, `describe` | Track platform state |
| GitOps Integration | `internal/gitops/` | `gitops` | ArgoCD/Flux orchestration |
| Policy Engine | `internal/policy/` | `policy` | OPA-based enforcement |
| Secrets Management | `internal/secrets/` | `secrets` | Vault/AWS SM integration |
| Authentication | `internal/auth/` | `auth` | JWT, SAML, API keys |
| RBAC | `internal/rbac/` | `rbac` | Role-based access |
| Audit Logging | `internal/audit/` | `audit` | Compliance trail |
| Environment Management | `internal/environment/` | `environment`, `preview` | Multi-env + ephemeral |
| Golden Paths | `internal/goldenpath/` | `golden-path`, `app` | Developer self-service |
| Plugin System | `internal/plugin/` | - | Extensibility |
| Deployment | `internal/deploy/` | `deploy`, `rollback` | Deployment strategies |
| Workflow/Approvals | `internal/workflow/` | `workflow` | Change management |

### Tier 2: Remove (No Clear Value)

These features should be removed entirely:

#### 1. AI/LLM Integration

**Packages**: `internal/ai/`, `internal/copilot/`

**CLI Commands**: `ask`, `copilot`

**Current State**:
- Two overlapping AI implementations
- Chat interface with token tracking
- LLM-powered recommendations

**Why Remove**:
- Nice-to-have, not core platform functionality
- Doubles maintenance with Claude/OpenAI provider support
- Interactive chat mode is feature bloat for infrastructure tooling
- Users can integrate their own LLM via plugins if needed
- Rapidly evolving space - hard to keep current

**Migration Path**: Remove packages. Document how to integrate external LLM tools via webhooks or plugins.

---

#### 2. Chaos Engineering

**Package**: `internal/chaos/`

**CLI Commands**: `chaos run`, `chaos list`, `chaos stop`, `chaos report`

**Current State**:
- Experiment definition and execution
- Mock executors (not production-ready)
- Steady-state validation

**Why Remove**:
- Specialized tools exist: LitmusChaos, Gremlin, Chaos Monkey
- Very narrow use case within platform engineering
- Mock executors indicate incomplete implementation
- Maintenance burden for chaos experiment compatibility

**Migration Path**: Document integration with LitmusChaos. Provide example ChaosExperiment CRDs that work with external tools.

---

#### 3. Marketplace

**Package**: `internal/marketplace/`

**CLI Commands**: `marketplace search`, `marketplace install`, `marketplace list`

**Current State**:
- Plugin discovery and installation
- Version management
- No actual backend marketplace

**Why Remove**:
- No marketplace backend exists
- Plugin system (`internal/plugin/`) sufficient for loading extensions
- GitHub releases or container registries serve discovery needs
- Creates complexity without infrastructure to support it

**Migration Path**: Use plugin registry pattern with GitHub releases. Document plugin installation from URLs/paths.

---

#### 4. Service Mesh Abstraction

**Package**: `internal/mesh/`

**CLI Commands**: `mesh apply`, `mesh get`, `mesh list`, `mesh status`

**Current State**:
- Abstraction over Istio/Linkerd
- Traffic policy management
- mTLS configuration

**Why Remove**:
- Users already know Istio/Linkerd CLIs and APIs
- Adds abstraction layer without simplification
- Version compatibility maintenance burden
- Duplicates service mesh native tooling
- Traffic management overlaps with `federation/traffic.go`

**Migration Path**: Provide Istio/Linkerd as plugin providers. Document native service mesh integration patterns.

---

#### 5. Federation

**Package**: `internal/federation/`

**Files**: `controller.go`, `health.go`, `sync.go`, `traffic.go`

**Current State**:
- Multi-cluster orchestration
- Cross-cluster health monitoring
- Traffic distribution

**Why Remove**:
- Massive scope for MVP
- Kubernetes Federation v2, ArgoCD multi-cluster, Submariner already solve this
- Health monitoring duplicates `internal/health/`
- Traffic management duplicates `internal/mesh/`
- Incomplete implementation

**Migration Path**: Future roadmap item. Document ArgoCD ApplicationSets for multi-cluster GitOps.

---

#### 6. JIT Access

**Package**: `internal/tenancy/jit.go`

**CLI Commands**: `jit request`, `jit approve`, `jit revoke`

**Current State**:
- Just-in-time access requests
- Approval workflows
- Temporary permission grants

**Why Remove**:
- HashiCorp Boundary, Teleport purpose-built for this
- Security-critical feature requires dedicated tooling
- Policy concern, not platform orchestration concern
- Adds approval workflow complexity

**Migration Path**: Document HashiCorp Boundary integration. Provide RBAC patterns for temporary access.

---

#### 7. SLO Management

**Package**: `internal/slo/`

**CLI Commands**: `slo create`, `slo status`, `slo list`

**Current State**:
- SLO definition and tracking
- Error budget calculation
- Burn rate alerting

**Why Remove**:
- Prometheus ecosystem has mature SLO tooling (Sloth, Pyrra)
- Grafana has native SLO dashboards
- Duplicates observability stack capabilities
- Better handled by dedicated SRE tools

**Migration Path**: Document Prometheus SLO recording rules. Provide Grafana dashboard templates.

---

#### 8. Graph Query Engine

**Package**: `internal/graph/`

**CLI Commands**: `graph`

**Current State**:
- Dependency graph visualization
- Query engine for traversal
- Analysis capabilities

**Why Remove**:
- Over-engineered for the use case
- Basic dependency visualization sufficient
- Query engine adds complexity without clear user need
- Overlaps with Kubernetes API introspection

**Migration Path**: Keep simple `pf describe --dependencies` output. Remove query engine.

---

### Tier 3: Consolidate (Duplicated Functionality)

These features overlap and should be merged:

#### 1. Cost Management Duplication

**Current State**:
- `internal/cost/` - Cost estimation
- `internal/finops/` - FinOps analysis, rightsizing, forecasting

**Problem**: Two systems for cost analysis. Unclear source of truth.

**Resolution**:
- Keep `internal/finops/`
- Remove `internal/cost/`
- Rename CLI commands to `cost` (simpler than `finops`)

**Resulting Commands**: `cost report`, `cost forecast`, `cost optimize`

---

#### 2. AI/Intelligence Duplication

**Current State**:
- `internal/ai/` - LLM providers, chat interface
- `internal/intelligence/` - Analysis, recommendations

**Problem**: Overlapping recommendation and analysis logic.

**Resolution**:
- Remove both (per Tier 2 recommendation)
- If keeping any AI: consolidate into single `internal/intelligence/` with no chat interface

---

#### 3. Multi-Tenancy vs RBAC

**Current State**:
- `internal/tenancy/` - Tenant management, quotas, isolation
- `internal/rbac/` - Role-based access control

**Problem**: Multi-tenancy duplicates Kubernetes RBAC + namespace isolation.

**Resolution**:
- Remove `internal/tenancy/`
- Enhance `internal/rbac/` with namespace-scoped roles
- Use Kubernetes ResourceQuotas for tenant quotas

**Resulting Commands**: Remove `tenant` commands. Use `rbac` with namespace scope.

---

#### 4. Compliance vs Policy

**Current State**:
- `internal/compliance/` - Compliance frameworks, scanning
- `internal/policy/` - OPA policy engine

**Problem**: OPA policies already enforce compliance. Separate compliance package adds confusion.

**Resolution**:
- Merge compliance into `internal/policy/`
- Add compliance framework templates as policy bundles
- Single `policy` CLI with `--framework soc2` flag

**Resulting Commands**: `policy check --framework soc2` instead of `compliance check`

---

### Tier 4: Deprioritize (Keep but Minimize Investment)

These features have value but shouldn't distract from core:

#### 1. Demo System

**Package**: `internal/demo/`

**CLI Commands**: `demo`, `demo quick`, `demo clean`

**Value**: Onboarding, local testing with kind cluster

**Recommendation**: Keep minimal. No new features. Consider moving to separate `pf-demo` repo.

---

#### 2. Scaffolding

**Package**: `internal/scaffold/`, `internal/generator/`

**CLI Commands**: `scaffold platform`, `scaffold component`

**Value**: Accelerates initial setup

**Recommendation**: Simplify to template expansion only. Remove code generation complexity. Consider `pf init` as single entry point.

---

#### 3. Doctor/Troubleshooting

**Package**: `internal/doctor/`

**CLI Commands**: `doctor`, `troubleshoot`

**Value**: Diagnostic convenience

**Recommendation**: Keep lightweight. No expansion. Consider external script instead of package.

---

#### 4. Portal/Web UI

**Package**: `internal/portal/`

**CLI Commands**: `portal`

**Value**: Visual interface

**Recommendation**: Separate product, not CLI concern. Extract to `pf-portal` repo or remove from CLI codebase entirely.

---

## Anti-Pattern: Competing with Ecosystem

Platform Foundry should orchestrate tools, not replace them. These implementations fight the ecosystem:

| PF Implementation | Ecosystem Tool | Problem | Resolution |
|-------------------|----------------|---------|------------|
| `gitops/` manifest sync | ArgoCD | ArgoCD already syncs, detects drift, self-heals | Use ArgoCD as provider, don't reimplement sync |
| `drift/` detection | Terraform | `terraform plan` shows drift | Call Terraform, don't reimplement |
| `mesh/` traffic policies | Istio/Linkerd | Native UIs are mature | Remove abstraction layer |
| `slo/` alerting | Prometheus/Grafana | Mature SLO ecosystem | Use recording rules, not custom engine |
| `chaos/` experiments | LitmusChaos | Purpose-built, Kubernetes-native | Integrate, don't compete |
| `federation/` multi-cluster | ArgoCD ApplicationSets | Already solved | Document integration pattern |

**Principle**: Platform Foundry's value is in orchestrating the provisioning and configuration of these tools, not in reimplementing their runtime functionality.

---

## CLI Command Rationalization

### Current State: 75+ Commands

```
Core (20):
  apply, plan, delete, get, describe, validate, lint
  deploy, rollback, status, logs, wait, jobs
  auth, secrets, rbac, policy, audit
  environment, preview

Platform (10):
  platform, app, golden-path, gitops, infrastructure
  workload, networking, scaling, catalog, service

Observability (8):
  health, monitoring, logging, metrics, alerts, tracing
  dora, slo

Security (6):
  tls, compliance, backup, restore, rotate, scan

Advanced (15):
  chaos, mesh, federation, finops, cost, forecast
  marketplace, portal, copilot, ask
  jit, tenant, promise, graph, scaffold

Utilities (8):
  demo, doctor, troubleshoot, version, init, config
  upgrade, completion
```

### Target State: ~40 Commands

```
Core (18):
  apply, plan, delete, get, describe, validate
  deploy, rollback, status, logs, wait, jobs
  auth, secrets, rbac, policy, audit, gitops

Platform (8):
  platform, app, golden-path, environment, preview
  workload, catalog, infrastructure

Observability (4):
  health, metrics, alerts, dora

Security (4):
  tls, backup, restore, compliance (merged into policy)

Utilities (6):
  init, config, version, upgrade, doctor, completion
```

### Commands to Remove

| Command | Reason |
|---------|--------|
| `ask`, `copilot` | AI chat not core |
| `chaos` | Use LitmusChaos |
| `mesh` | Use Istio/Linkerd directly |
| `federation` | Future scope |
| `jit` | Use Boundary |
| `tenant` | Use RBAC + namespaces |
| `marketplace` | No backend |
| `portal` | Separate product |
| `slo` | Use Prometheus |
| `graph` | Over-engineered |
| `cost` | Merged into finops→cost |
| `finops` | Renamed to cost |
| `promise` | Unclear purpose |
| `scaffold` | Merged into init |
| `demo` | Move to separate tool |

---

## Package Rationalization

### Current: 67 Packages

### Target: ~45 Packages

#### Remove (14 packages):
```
internal/ai/
internal/copilot/
internal/chaos/
internal/marketplace/
internal/mesh/
internal/federation/
internal/slo/
internal/graph/
internal/cost/           # Merge into finops
internal/tenancy/        # Merge into rbac
internal/compliance/     # Merge into policy
internal/portal/         # Separate repo
internal/scaffold/       # Simplify into init
internal/generator/      # Remove
```

#### Consolidate (4 merges):
```
internal/cost/ → internal/finops/
internal/tenancy/ → internal/rbac/
internal/compliance/ → internal/policy/
internal/ai/ + internal/intelligence/ → Remove both
```

---

## Migration Plan

### Phase 1: Immediate Removal (Week 1-2)

1. Remove `internal/ai/`, `internal/copilot/`
2. Remove `internal/marketplace/`
3. Remove `internal/chaos/`
4. Remove CLI commands: `ask`, `copilot`, `marketplace`, `chaos`

**Impact**: Minimal user impact. Features incomplete or unused.

### Phase 2: Consolidation (Week 3-4)

1. Merge `internal/cost/` into `internal/finops/`, rename CLI to `cost`
2. Merge `internal/compliance/` into `internal/policy/`
3. Merge `internal/tenancy/` into `internal/rbac/`
4. Update documentation

**Impact**: CLI command changes. Provide aliases for transition.

### Phase 3: Ecosystem Alignment (Week 5-6)

1. Remove `internal/mesh/` - document Istio integration
2. Remove `internal/slo/` - document Prometheus SLO patterns
3. Remove `internal/federation/` - document ArgoCD multi-cluster
4. Remove `internal/graph/` query engine - keep basic visualization

**Impact**: Feature removal. Provide migration guides to ecosystem tools.

### Phase 4: Extraction (Week 7-8)

1. Extract `internal/portal/` to separate repository
2. Extract `internal/demo/` to separate repository
3. Simplify `internal/scaffold/` into `init` command

**Impact**: Organizational. No user-facing changes.

---

## Impact Summary

| Metric | Before | After | Change |
|--------|--------|-------|--------|
| Packages | 67 | 45 | -33% |
| CLI Commands | 75+ | 40 | -47% |
| Lines of Code (est.) | 18,600 | 12,000 | -35% |
| Maintenance Burden | 100% | 60% | -40% |
| Cognitive Load | High | Medium | Significant |

---

## Appendix: Decision Matrix

| Feature | Users | Ecosystem Alternative | Implementation Quality | Verdict |
|---------|-------|----------------------|------------------------|---------|
| AI Chat | Few | External LLMs | Incomplete | Remove |
| Chaos | Few | LitmusChaos | Mock only | Remove |
| Marketplace | None | GitHub/Registry | No backend | Remove |
| Mesh | Some | Istio/Linkerd | Abstraction overhead | Remove |
| Federation | Few | ArgoCD | Incomplete | Remove |
| JIT | Few | Boundary | Security concern | Remove |
| SLO | Some | Prometheus | Duplication | Remove |
| Graph | Few | kubectl | Over-engineered | Remove |
| Cost/FinOps | Some | Cloud tools | Duplicated | Consolidate |
| Tenancy | Some | K8s RBAC | Duplicated | Consolidate |
| Compliance | Some | OPA | Duplicated | Consolidate |
| Portal | Some | N/A | Separate concern | Extract |
| Demo | Some | N/A | Onboarding only | Deprioritize |

---

## Conclusion

Platform Foundry's strength is declarative platform orchestration. The current feature set dilutes this focus by competing with specialized ecosystem tools. By removing 14 packages and consolidating 4 more, the project can:

1. **Sharpen focus** on core value: platform provisioning
2. **Reduce maintenance** by 40%
3. **Simplify UX** with fewer, more purposeful commands
4. **Align with ecosystem** by orchestrating tools, not replacing them

The recommended approach: **orchestrate, don't replicate**.
