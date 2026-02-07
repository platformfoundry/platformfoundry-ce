# Security and Compliance Policy
# Ensures resources meet security and compliance requirements

package platformfoundry.security

import future.keywords

# Default deny
default allow = false

# Allow if all security requirements are met
allow {
    has_encryption
    has_backup
    has_monitoring
    valid_network_policy
}

# Encryption requirements
has_encryption {
    input.spec.infrastructure.encryption.enabled == true
}

has_encryption {
    input.kind != "Infrastructure"
}

# Backup requirements for production
has_backup {
    input.metadata.labels.environment != "production"
}

has_backup {
    input.metadata.labels.environment == "production"
    input.spec.infrastructure.backup.enabled == true
    input.spec.infrastructure.backup.retention >= 30
}

# Monitoring requirements for production
has_monitoring {
    input.metadata.labels.environment != "production"
}

has_monitoring {
    input.metadata.labels.environment == "production"
    input.spec.observability.monitoring.enabled == true
}

# Network policy requirements
valid_network_policy {
    input.kind != "Infrastructure"
}

valid_network_policy {
    input.kind == "Infrastructure"
    input.spec.infrastructure.networkPolicy.enabled == true
}

# Compliance labels
required_compliance_labels := [
    "data-classification",
    "compliance-scope"
]

has_compliance_labels {
    input.metadata.labels.compliance != "required"
}

has_compliance_labels {
    input.metadata.labels.compliance == "required"
    count([label | label := required_compliance_labels[_]; not input.metadata.labels[label]]) == 0
}

# Collect security violations
deny[msg] {
    input.kind == "Infrastructure"
    not input.spec.infrastructure.encryption.enabled
    msg := "Infrastructure encryption must be enabled"
}

deny[msg] {
    input.metadata.labels.environment == "production"
    input.kind == "Infrastructure"
    not input.spec.infrastructure.backup.enabled
    msg := "Production infrastructure must have backups enabled"
}

deny[msg] {
    input.metadata.labels.environment == "production"
    input.kind == "Infrastructure"
    input.spec.infrastructure.backup.retention < 30
    msg := "Production backups must have minimum 30-day retention"
}

deny[msg] {
    input.metadata.labels.environment == "production"
    not input.spec.observability.monitoring.enabled
    msg := "Production resources must have monitoring enabled"
}

deny[msg] {
    input.kind == "Infrastructure"
    not input.spec.infrastructure.networkPolicy.enabled
    msg := "Network policies must be enabled"
}

deny[msg] {
    input.metadata.labels.compliance == "required"
    label := required_compliance_labels[_]
    not input.metadata.labels[label]
    msg := sprintf("Missing required compliance label: %s", [label])
}

# Data classification validation
valid_data_classification {
    input.metadata.labels["data-classification"]
    input.metadata.labels["data-classification"] in ["public", "internal", "confidential", "restricted"]
}

deny[msg] {
    input.metadata.labels.compliance == "required"
    input.metadata.labels["data-classification"]
    not input.metadata.labels["data-classification"] in ["public", "internal", "confidential", "restricted"]
    msg := "Label 'data-classification' must be one of: public, internal, confidential, restricted"
}
