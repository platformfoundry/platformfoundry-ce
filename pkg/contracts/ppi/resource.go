// Package ppi defines the Platform Provider Interface (PPI).
package ppi

import (
	"time"
)

// ResourceConfig holds the desired configuration for a resource
type ResourceConfig struct {
	// TypeName is the resource type (e.g., "aws_instance")
	TypeName string

	// Values contains the configuration values
	Values map[string]interface{}

	// Sensitive contains paths to sensitive values
	Sensitive []string
}

// ResourceState represents the current state of a resource
type ResourceState struct {
	// ID is the unique identifier for this resource instance
	ID string

	// TypeName is the resource type
	TypeName string

	// Attributes contains the current attribute values
	Attributes map[string]interface{}

	// Private contains provider-specific data not visible to users
	Private []byte

	// Dependencies lists resources this resource depends on
	Dependencies []string

	// Status indicates the current status of the resource
	Status ResourceStatus

	// Timestamps for tracking
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ResourceStatus represents the current status of a resource
type ResourceStatus string

const (
	ResourceStatusUnknown    ResourceStatus = "unknown"
	ResourceStatusPending    ResourceStatus = "pending"
	ResourceStatusCreating   ResourceStatus = "creating"
	ResourceStatusUpdating   ResourceStatus = "updating"
	ResourceStatusDeleting   ResourceStatus = "deleting"
	ResourceStatusReady      ResourceStatus = "ready"
	ResourceStatusError      ResourceStatus = "error"
	ResourceStatusDegraded   ResourceStatus = "degraded"
	ResourceStatusNotFound   ResourceStatus = "not_found"
	ResourceStatusTainted    ResourceStatus = "tainted"
)

// Plan represents the changes that will be made to a resource
type Plan struct {
	// Action describes what will happen to the resource
	Action PlanAction

	// Prior is the state before the change
	Prior *ResourceState

	// Proposed is the desired state after the change
	Proposed *ResourceState

	// RequiresReplace indicates attributes that force replacement
	RequiresReplace []string

	// PlannedPrivate is provider-specific planning data
	PlannedPrivate []byte
}

// PlanAction describes the type of change being planned
type PlanAction string

const (
	PlanActionNoop    PlanAction = "noop"
	PlanActionCreate  PlanAction = "create"
	PlanActionUpdate  PlanAction = "update"
	PlanActionDelete  PlanAction = "delete"
	PlanActionReplace PlanAction = "replace"
	PlanActionRead    PlanAction = "read"
)

// DataSourceConfig holds the configuration for reading a data source
type DataSourceConfig struct {
	// TypeName is the data source type
	TypeName string

	// Values contains the query parameters
	Values map[string]interface{}
}

// DataSourceState holds the result of reading a data source
type DataSourceState struct {
	// ID is the unique identifier for this data source result
	ID string

	// TypeName is the data source type
	TypeName string

	// Attributes contains the read attribute values
	Attributes map[string]interface{}
}
