package types

import "errors"

// Common errors
var (
	ErrMissingAPIVersion = errors.New("missing apiVersion")
	ErrInvalidKind       = errors.New("invalid kind")
	ErrMissingName       = errors.New("missing metadata.name")
	ErrMissingSpec       = errors.New("missing spec")
	ErrNoComponents      = errors.New("no components defined")
	ErrInvalidProvider   = errors.New("invalid provider")
	ErrMissingClusterRef = errors.New("missing clusterRef")

	// Organization/Environment errors
	ErrMissingOrganization    = errors.New("organization is required")
	ErrOrganizationNotFound   = errors.New("organization not found")
	ErrMissingDisplayName     = errors.New("display name is required")
	ErrMissingOwner           = errors.New("owner is required")
	ErrMissingEnvironmentType = errors.New("environment type is required")
	ErrMissingPlatformRef     = errors.New("platform reference is required")
	ErrInvalidEnvironmentType = errors.New("invalid environment type")
	ErrAccessDenied           = errors.New("access denied to organization resource")
	ErrQuotaExceeded          = errors.New("organization quota exceeded")

	// Quota errors
	ErrInvalidScope           = errors.New("invalid quota scope")
	ErrQuotaNotFound          = errors.New("quota not found")
	ErrResourceNotInQuota     = errors.New("resource not defined in quota")
)
