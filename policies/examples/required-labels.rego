# Required Labels Policy
# Ensures all resources have required labels

package platformfoundry.labels

import future.keywords

# Default deny
default allow = false

# Required labels for all resources
required_labels := [
    "team",
    "cost-center",
    "environment"
]

# Optional but recommended labels
recommended_labels := [
    "owner",
    "project",
    "compliance"
]

# Allow if all required labels are present
allow {
    has_all_required_labels
}

# Check if all required labels exist
has_all_required_labels {
    count([label | label := required_labels[_]; not input.metadata.labels[label]]) == 0
}

# Validation for specific label values
valid_environment {
    input.metadata.labels.environment
    input.metadata.labels.environment in ["development", "staging", "production"]
}

valid_cost_center {
    input.metadata.labels["cost-center"]
    regex.match(`^CC-[0-9]{4}$`, input.metadata.labels["cost-center"])
}

valid_team {
    input.metadata.labels.team
    count(input.metadata.labels.team) > 0
}

# Collect all violations
deny[msg] {
    label := required_labels[_]
    not input.metadata.labels[label]
    msg := sprintf("Missing required label: %s", [label])
}

deny[msg] {
    input.metadata.labels.environment
    not input.metadata.labels.environment in ["development", "staging", "production"]
    msg := "Label 'environment' must be one of: development, staging, production"
}

deny[msg] {
    input.metadata.labels["cost-center"]
    not regex.match(`^CC-[0-9]{4}$`, input.metadata.labels["cost-center"])
    msg := "Label 'cost-center' must match format: CC-XXXX (e.g., CC-1234)"
}

# Warnings for missing recommended labels
warn[msg] {
    label := recommended_labels[_]
    not input.metadata.labels[label]
    msg := sprintf("Recommended label missing: %s", [label])
}
