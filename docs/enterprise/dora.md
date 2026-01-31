# DORA Metrics

Enterprise Edition tracks DORA (DevOps Research and Assessment) metrics to measure software delivery performance.

## Overview

DORA metrics measure four key indicators:

| Metric | Description | Elite | High | Medium | Low |
|--------|-------------|-------|------|--------|-----|
| **Deployment Frequency** | How often code is deployed | On-demand (multiple/day) | Daily to weekly | Weekly to monthly | Monthly+ |
| **Lead Time for Changes** | Time from commit to production | < 1 hour | 1 day - 1 week | 1 week - 1 month | 1 month+ |
| **Mean Time to Recovery** | Time to restore service | < 1 hour | < 1 day | 1 day - 1 week | 1 week+ |
| **Change Failure Rate** | % of deployments causing failure | 0-15% | 16-30% | 31-45% | 46%+ |

## Setup

### Enable DORA Tracking

```bash
# Enable DORA metrics
pf config set enterprise.features.dora=true

# Configure data sources
pf dora configure --git-provider=github --ci-provider=github-actions
```

### Configuration

```yaml
# ~/.platformfoundry/config.yaml
enterprise:
  dora:
    enabled: true

    # Git provider for commit tracking
    git:
      provider: github  # github, gitlab, bitbucket
      org: my-org
      token: ${GITHUB_TOKEN}

    # CI/CD provider for deployment tracking
    ci:
      provider: github-actions  # github-actions, gitlab-ci, jenkins, argocd
      config:
        org: my-org

    # Incident tracking for MTTR
    incidents:
      provider: pagerduty  # pagerduty, opsgenie, custom
      config:
        apiKey: ${PAGERDUTY_API_KEY}
        serviceIds:
          - P123ABC
          - P456DEF

    # Data retention
    retention:
      days: 365

    # Reporting
    reporting:
      schedule: weekly
      recipients:
        - engineering-leads@example.com
```

## CLI Commands

### View Current Metrics

```bash
pf dora show

# Output:
# DORA Metrics Report
# ===================
# Period: Last 30 Days
# Generated: 2024-01-26T23:06:32+05:30
#
# Deployment Frequency:    4.2/day (Elite)
# Lead Time for Changes:   2.1 hours (Elite)
# Mean Time to Recovery:   45 minutes (Elite)
# Change Failure Rate:     3.2% (Elite)
#
# Overall Rating: Elite Performer
```

### Detailed Report

```bash
pf dora report --period=monthly

# Output:
# DORA Metrics - January 2024
# ===========================
#
# DEPLOYMENT FREQUENCY
# ────────────────────
# Total Deployments:     126
# Daily Average:         4.2
# Peak Day:              12 (Jan 15)
# Rating:                Elite
#
# Breakdown by Environment:
#   Production:          84 (67%)
#   Staging:             42 (33%)
#
# Breakdown by Team:
#   Platform:            45 (36%)
#   Backend:             52 (41%)
#   Frontend:            29 (23%)
#
# LEAD TIME FOR CHANGES
# ─────────────────────
# Average:               2.1 hours
# Median:                1.8 hours
# P95:                   6.2 hours
# Rating:                Elite
#
# Breakdown:
#   Code Review:         45 minutes
#   CI/CD:               30 minutes
#   Deployment:          15 minutes
#   Validation:          30 minutes
#
# MEAN TIME TO RECOVERY
# ─────────────────────
# Average:               45 minutes
# Incidents:             8
# Rating:                Elite
#
# Incidents:
#   P1 (Critical):       1 (MTTR: 35 min)
#   P2 (High):           3 (MTTR: 52 min)
#   P3 (Medium):         4 (MTTR: 48 min)
#
# CHANGE FAILURE RATE
# ───────────────────
# Failed Deployments:    4
# Total Deployments:     126
# Failure Rate:          3.2%
# Rating:                Elite
#
# Failure Causes:
#   Configuration:       2 (50%)
#   Infrastructure:      1 (25%)
#   Code Bug:            1 (25%)
```

### Historical Trends

```bash
pf dora trends --period=6months

# Output:
# DORA Trends (6 Months)
# ======================
#
# Month      DF      LT       MTTR     CFR      Rating
# ───────────────────────────────────────────────────
# Aug 2023   2.1/d   4.5h     2.1h     8.2%     High
# Sep 2023   2.8/d   3.2h     1.5h     6.1%     High
# Oct 2023   3.2/d   2.8h     1.2h     5.0%     High
# Nov 2023   3.8/d   2.4h     55m      4.2%     Elite
# Dec 2023   4.0/d   2.2h     48m      3.8%     Elite
# Jan 2024   4.2/d   2.1h     45m      3.2%     Elite
#
# Trend: Improving ↑
```

### Compare Teams

```bash
pf dora compare --by=team

# Output:
# Team Comparison (January 2024)
# ==============================
#
# Team        DF       LT      MTTR    CFR     Rating
# ─────────────────────────────────────────────────────
# Platform    4.5/d    1.8h    35m     2.1%    Elite
# Backend     4.2/d    2.5h    55m     3.8%    Elite
# Frontend    3.8/d    2.8h    1.2h    4.5%    High
```

## Data Sources

### Git Integration

Track commits and pull requests:

```yaml
enterprise:
  dora:
    git:
      provider: github
      org: my-org
      token: ${GITHUB_TOKEN}

      # Repositories to track
      repositories:
        - platform-config
        - backend-api
        - frontend-app

      # Branch patterns for production
      productionBranches:
        - main
        - master
        - "release/*"
```

### CI/CD Integration

Track deployments:

```yaml
enterprise:
  dora:
    ci:
      provider: github-actions
      config:
        org: my-org
        workflows:
          - name: deploy-production
            environment: production
          - name: deploy-staging
            environment: staging

    # Or ArgoCD
    argocd:
      server: https://argocd.example.com
      token: ${ARGOCD_TOKEN}
      applications:
        - production-app
        - staging-app
```

### Incident Integration

Track incidents for MTTR:

```yaml
enterprise:
  dora:
    incidents:
      provider: pagerduty
      config:
        apiKey: ${PAGERDUTY_API_KEY}
        serviceIds:
          - P123ABC

      # Map severity to incident type
      severityMapping:
        P1: critical
        P2: high
        P3: medium
        P4: low
```

## Custom Tracking

### Manual Deployment Recording

```bash
# Record deployment
pf dora record deployment \
  --environment=production \
  --commit=abc123 \
  --status=success

# Record failed deployment
pf dora record deployment \
  --environment=production \
  --commit=def456 \
  --status=failed \
  --reason="Configuration error"
```

### Manual Incident Recording

```bash
# Record incident
pf dora record incident \
  --severity=P2 \
  --started="2024-01-20T10:00:00Z" \
  --resolved="2024-01-20T10:45:00Z" \
  --cause="Database connection pool exhausted"
```

## Dashboards

### Built-in Dashboard

```bash
# Open DORA dashboard
pf dora dashboard

# Serve dashboard on specific port
pf dora dashboard --port=8080
```

### Grafana Integration

```yaml
enterprise:
  dora:
    export:
      prometheus:
        enabled: true
        port: 9090
        path: /metrics

      # Metrics exported:
      # pf_dora_deployment_frequency
      # pf_dora_lead_time_seconds
      # pf_dora_mttr_seconds
      # pf_dora_change_failure_rate
```

### Custom Metrics Export

```bash
# Export to JSON
pf dora export --format=json --period=monthly > dora-metrics.json

# Export to CSV
pf dora export --format=csv --period=weekly > dora-metrics.csv
```

## Alerts

### Configure Alerts

```yaml
enterprise:
  dora:
    alerts:
      # Alert when metrics degrade
      - metric: deployment_frequency
        condition: below
        threshold: 1  # per day
        severity: warning

      - metric: lead_time
        condition: above
        threshold: 24h
        severity: warning

      - metric: mttr
        condition: above
        threshold: 4h
        severity: critical

      - metric: change_failure_rate
        condition: above
        threshold: 15%
        severity: warning

    notifications:
      slack:
        channel: "#engineering-metrics"
      email:
        recipients:
          - engineering-leads@example.com
```

## Reports

### Generate PDF Report

```bash
pf dora report --format=pdf --output=dora-report.pdf
```

### Scheduled Reports

```yaml
enterprise:
  dora:
    reports:
      - name: weekly-summary
        schedule: "0 9 * * 1"  # Monday 9 AM
        format: email
        recipients:
          - engineering@example.com

      - name: monthly-executive
        schedule: "0 9 1 * *"  # 1st of month
        format: pdf
        recipients:
          - cto@example.com
          - vp-engineering@example.com
```

## Best Practices

1. **Track all deployments** - Include all environments
2. **Link incidents to deployments** - Understand failure causes
3. **Review weekly** - Monitor trends, not just snapshots
4. **Set realistic goals** - Improve incrementally
5. **Share with teams** - Visibility drives improvement

## Next Steps

- [Cost Tracking](cost-tracking.md)
- [SSO Configuration](sso.md)
