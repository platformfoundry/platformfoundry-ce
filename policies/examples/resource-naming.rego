# Resource Naming Policy
# Ensures resources follow naming conventions

package platformfoundry.naming

import future.keywords

# Default deny
default allow = false

# Allow if resource name follows convention
allow {
    valid_name
    valid_prefix
}

# Resource name must be lowercase alphanumeric with hyphens
valid_name {
    regex.match(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`, input.metadata.name)
    count(input.metadata.name) <= 63
}

# Production resources must have "prod-" prefix
valid_prefix {
    input.metadata.labels.environment != "production"
}

valid_prefix {
    input.metadata.labels.environment == "production"
    startswith(input.metadata.name, "prod-")
}

# Development resources must have "dev-" prefix
valid_dev_prefix {
    input.metadata.labels.environment == "development"
    startswith(input.metadata.name, "dev-")
}

# Staging resources must have "staging-" prefix
valid_staging_prefix {
    input.metadata.labels.environment == "staging"
    startswith(input.metadata.name, "staging-")
}

# Reasons for denial
deny[msg] {
    not valid_name
    msg := "Resource name must be lowercase alphanumeric with hyphens and max 63 characters"
}

deny[msg] {
    input.metadata.labels.environment == "production"
    not startswith(input.metadata.name, "prod-")
    msg := "Production resources must have 'prod-' prefix"
}

deny[msg] {
    input.metadata.labels.environment == "development"
    not startswith(input.metadata.name, "dev-")
    msg := "Development resources must have 'dev-' prefix"
}

deny[msg] {
    input.metadata.labels.environment == "staging"
    not startswith(input.metadata.name, "staging-")
    msg := "Staging resources must have 'staging-' prefix"
}
