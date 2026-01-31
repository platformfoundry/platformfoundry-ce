# Cost Tracking

Enterprise Edition provides real-time cloud cost tracking, allocation, and optimization.

## Overview

Cost tracking features:

- Real-time spend monitoring
- Cost allocation by team/project
- Budget alerts and thresholds
- Cost forecasting
- Optimization recommendations

## Setup

### Enable Cost Tracking

```bash
# Enable cost tracking
pf config set enterprise.features.costTracking=true

# Configure cloud providers
pf cost configure aws --role-arn=arn:aws:iam::123456789012:role/CostExplorer
pf cost configure gcp --project=my-project --service-account=cost-tracker@my-project.iam.gserviceaccount.com
```

### Configuration

```yaml
# ~/.platformfoundry/config.yaml
enterprise:
  costTracking:
    enabled: true
    refreshInterval: 1h

    providers:
      aws:
        enabled: true
        roleArn: arn:aws:iam::123456789012:role/CostExplorer
        regions:
          - us-east-1
          - us-west-2

      gcp:
        enabled: true
        projectId: my-project
        billingAccountId: XXXXX-XXXXX-XXXXX

      azure:
        enabled: true
        subscriptionId: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
        tenantId: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx

    # Cost allocation
    allocation:
      enabled: true
      tagKey: cost-center  # Tag used for allocation
      defaultCostCenter: unallocated

    # Budgets
    budgets:
      enabled: true
      alertThresholds:
        - 50
        - 75
        - 90
        - 100

    # Notifications
    notifications:
      slack:
        channel: "#cost-alerts"
        webhook: ${SLACK_WEBHOOK_URL}
      email:
        recipients:
          - finance@example.com
          - platform-team@example.com
```

## CLI Commands

### View Current Costs

```bash
# Show current month costs
pf cost show

# Output:
# Cost Summary (January 2024)
# ===========================
#
# Provider     This Month    Last Month    Change
# ─────────────────────────────────────────────────
# AWS          $12,450.32    $11,890.45    +4.7%
# GCP          $3,245.67     $3,102.33     +4.6%
# Azure        $1,890.00     $1,750.00     +8.0%
# ─────────────────────────────────────────────────
# Total        $17,585.99    $16,742.78    +5.0%
```

### Cost by Environment

```bash
pf cost show --by=environment

# Output:
# Cost by Environment (January 2024)
# ===================================
#
# Environment    Cost          % of Total
# ────────────────────────────────────────
# production     $12,500.00    71%
# staging        $3,500.00     20%
# development    $1,585.99     9%
```

### Cost by Service

```bash
pf cost show --by=service

# Output:
# Cost by Service (January 2024)
# ==============================
#
# Service            Cost          % of Total
# ───────────────────────────────────────────
# EC2                $6,200.00     35%
# RDS                $4,100.00     23%
# EKS                $3,500.00     20%
# S3                 $1,800.00     10%
# Other              $1,985.99     12%
```

### Cost by Team

```bash
pf cost show --by=team

# Output:
# Cost by Team (January 2024)
# ===========================
#
# Team               Cost          Budget      Status
# ────────────────────────────────────────────────────
# platform           $8,500.00     $10,000     OK
# backend            $5,200.00     $5,000      ⚠️ OVER
# frontend           $2,100.00     $3,000      OK
# data               $1,785.99     $2,000      OK
```

### Forecast

```bash
pf cost forecast

# Output:
# Cost Forecast
# =============
#
# Current Month (January):    $17,585.99
# Forecasted End of Month:    $19,200.00
# Next Month Forecast:        $20,100.00
#
# Trend: +5.2% month-over-month
```

## Budgets

### Create Budget

```bash
# Create monthly budget
pf cost budget create \
  --name=platform-team \
  --amount=10000 \
  --period=monthly \
  --scope=team:platform \
  --alerts=50,75,90,100
```

### Budget Configuration

```yaml
enterprise:
  costTracking:
    budgets:
      - name: platform-team
        amount: 10000
        period: monthly
        scope:
          type: team
          value: platform
        alerts:
          - threshold: 50
            action: notify
          - threshold: 75
            action: notify
          - threshold: 90
            action: warn
          - threshold: 100
            action: alert

      - name: production-environment
        amount: 15000
        period: monthly
        scope:
          type: environment
          value: production
        alerts:
          - threshold: 90
            action: alert
```

### View Budgets

```bash
pf cost budget list

# Output:
# NAME                  BUDGET      SPENT       REMAINING   STATUS
# platform-team         $10,000     $8,500      $1,500      85% ⚠️
# production-env        $15,000     $12,500     $2,500      83%
# staging-env           $5,000      $3,500      $1,500      70%
```

## Cost Allocation

### Tag Resources

```yaml
apiVersion: platformfoundry.io/v1
kind: Platform
metadata:
  name: my-platform
  labels:
    cost-center: platform-team
    project: infrastructure
spec:
  infrastructure:
    config:
      tags:
        cost-center: platform-team
        project: infrastructure
        environment: production
```

### Allocation Rules

```yaml
enterprise:
  costTracking:
    allocation:
      rules:
        # Allocate by tag
        - match:
            tag: cost-center
          allocateTo: team

        # Allocate untagged resources
        - match:
            tag: cost-center
            missing: true
          allocateTo: unallocated

        # Split shared resources
        - match:
            service: eks
            tag: shared
            equals: "true"
          split:
            - team: platform
              percentage: 40
            - team: backend
              percentage: 30
            - team: frontend
              percentage: 30
```

## Optimization Recommendations

```bash
pf cost optimize

# Output:
# Optimization Recommendations
# ============================
#
# HIGH IMPACT:
# ───────────
# 1. Right-size EC2 instances
#    Resource: i-abc123 (m5.xlarge → m5.large)
#    Estimated Savings: $150/month
#
# 2. Delete unused EBS volumes
#    Resources: 5 volumes (250 GB total)
#    Estimated Savings: $25/month
#
# MEDIUM IMPACT:
# ─────────────
# 3. Convert to Reserved Instances
#    Resources: 10 EC2 instances
#    Estimated Savings: $2,400/year
#
# 4. Enable S3 Intelligent-Tiering
#    Buckets: logs-bucket, backup-bucket
#    Estimated Savings: $80/month
#
# Total Potential Savings: $495/month ($5,940/year)
```

### Apply Recommendations

```bash
# View recommendation details
pf cost optimize show --id=1

# Apply recommendation
pf cost optimize apply --id=1 --auto-approve

# Apply all safe recommendations
pf cost optimize apply --all --safe-only
```

## Reports

### Generate Report

```bash
# Monthly report
pf cost report --period=monthly --format=pdf --output=cost-report.pdf

# Custom date range
pf cost report --from=2024-01-01 --to=2024-01-31 --format=html
```

### Scheduled Reports

```yaml
enterprise:
  costTracking:
    reports:
      - name: monthly-summary
        schedule: "0 9 1 * *"  # 9 AM on 1st of month
        period: monthly
        format: pdf
        recipients:
          - finance@example.com
          - cto@example.com

      - name: weekly-team-report
        schedule: "0 9 * * 1"  # 9 AM on Mondays
        period: weekly
        groupBy: team
        format: email
        recipients:
          - platform-leads@example.com
```

## Anomaly Detection

```yaml
enterprise:
  costTracking:
    anomalyDetection:
      enabled: true
      sensitivity: medium  # low, medium, high
      notifications:
        slack:
          channel: "#cost-alerts"
        email:
          recipients:
            - platform-team@example.com
```

### View Anomalies

```bash
pf cost anomalies

# Output:
# Cost Anomalies (Last 7 Days)
# ============================
#
# DATE        SERVICE    EXPECTED    ACTUAL      VARIANCE
# 2024-01-18  EC2        $200        $450        +125% ⚠️
# 2024-01-20  RDS        $150        $280        +87% ⚠️
```

## API Integration

```bash
# Export cost data
pf cost export --format=json --from=2024-01-01 --to=2024-01-31 > costs.json

# Push to external system
pf cost export --format=json | curl -X POST -d @- https://api.example.com/costs
```

## Next Steps

- [DORA Metrics](dora.md)
- [Budgets and Alerts](../reference/config.md#cost-tracking)
