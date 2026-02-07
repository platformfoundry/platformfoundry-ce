# Cost Limits Policy
# Prevents resource configurations that exceed budget limits

package platformfoundry.cost

import future.keywords

# Default deny for production
default allow = false

# Cost estimates per instance type (monthly USD)
instance_costs := {
    "t3.micro": 7,
    "t3.small": 15,
    "t3.medium": 30,
    "t3.large": 60,
    "t3.xlarge": 120,
    "t3.2xlarge": 240,
    "m5.large": 70,
    "m5.xlarge": 140,
    "m5.2xlarge": 280,
    "m5.4xlarge": 560,
}

# Budget limits per environment (monthly USD)
budget_limits := {
    "development": 500,
    "staging": 2000,
    "production": 10000,
}

# Allow if within budget
allow {
    estimated_cost <= environment_budget
}

# Calculate estimated monthly cost
estimated_cost = cost {
    infrastructure := input.spec.infrastructure
    node_count := infrastructure.nodeCount
    instance_type := infrastructure.instanceType
    unit_cost := instance_costs[instance_type]
    cost := node_count * unit_cost
}

# Get environment budget
environment_budget = limit {
    env := input.metadata.labels.environment
    limit := budget_limits[env]
}

# Deny if exceeds budget
deny[msg] {
    estimated_cost > environment_budget
    msg := sprintf(
        "Estimated cost $%d/month exceeds %s budget of $%d/month",
        [estimated_cost, input.metadata.labels.environment, environment_budget]
    )
}

# Deny if instance type not in allowed list
deny[msg] {
    infrastructure := input.spec.infrastructure
    instance_type := infrastructure.instanceType
    not instance_costs[instance_type]
    msg := sprintf("Instance type '%s' is not in the approved list", [instance_type])
}

# Warn if cost is over 80% of budget
warn[msg] {
    threshold := environment_budget * 0.8
    estimated_cost > threshold
    estimated_cost <= environment_budget
    msg := sprintf(
        "Cost $%d/month is over 80%% of %s budget ($%d/month)",
        [estimated_cost, input.metadata.labels.environment, environment_budget]
    )
}

# Production-specific rules
deny[msg] {
    input.metadata.labels.environment == "production"
    infrastructure := input.spec.infrastructure
    node_count := infrastructure.nodeCount
    node_count < 3
    msg := "Production environments must have at least 3 nodes for high availability"
}
