// Package crd provides Custom Resource Definition support for Platform Foundry.
// This allows the community to define new resource types without modifying the core,
// inspired by Kubernetes CRDs.
package crd

import (
	"time"
)

// CustomResourceDefinition defines a custom resource type
type CustomResourceDefinition struct {
	// APIVersion is the API version (e.g., "platformfoundry.io/v1")
	APIVersion string `yaml:"apiVersion" json:"apiVersion"`

	// Kind is always "CustomResourceDefinition"
	Kind string `yaml:"kind" json:"kind"`

	// Metadata contains CRD metadata
	Metadata Metadata `yaml:"metadata" json:"metadata"`

	// Spec defines the CRD specification
	Spec CRDSpec `yaml:"spec" json:"spec"`

	// Status contains the CRD status (set by the system)
	Status *CRDStatus `yaml:"status,omitempty" json:"status,omitempty"`
}

// Metadata contains resource metadata
type Metadata struct {
	// Name is the CRD name (e.g., "databases.acme.io")
	Name string `yaml:"name" json:"name"`

	// Namespace is the optional namespace
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty"`

	// Labels are key-value labels
	Labels map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`

	// Annotations are key-value annotations
	Annotations map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`

	// CreationTimestamp is when the resource was created
	CreationTimestamp *time.Time `yaml:"creationTimestamp,omitempty" json:"creationTimestamp,omitempty"`

	// UID is the unique identifier
	UID string `yaml:"uid,omitempty" json:"uid,omitempty"`

	// ResourceVersion is the version of this resource
	ResourceVersion string `yaml:"resourceVersion,omitempty" json:"resourceVersion,omitempty"`

	// Generation is incremented on spec changes
	Generation int64 `yaml:"generation,omitempty" json:"generation,omitempty"`

	// Finalizers are cleanup hooks
	Finalizers []string `yaml:"finalizers,omitempty" json:"finalizers,omitempty"`
}

// CRDSpec defines the CRD specification
type CRDSpec struct {
	// Group is the API group (e.g., "acme.io")
	Group string `yaml:"group" json:"group"`

	// Version is the API version (e.g., "v1", "v1beta1")
	Version string `yaml:"version" json:"version"`

	// Kind is the resource kind (e.g., "Database")
	Kind string `yaml:"kind" json:"kind"`

	// Plural is the plural name (e.g., "databases")
	Plural string `yaml:"plural,omitempty" json:"plural,omitempty"`

	// Singular is the singular name (e.g., "database")
	Singular string `yaml:"singular,omitempty" json:"singular,omitempty"`

	// ShortNames are abbreviated names (e.g., ["db"])
	ShortNames []string `yaml:"shortNames,omitempty" json:"shortNames,omitempty"`

	// Scope is "Cluster" or "Namespaced"
	Scope ResourceScope `yaml:"scope" json:"scope"`

	// Schema defines the resource schema
	Schema *JSONSchemaProps `yaml:"schema,omitempty" json:"schema,omitempty"`

	// Handler specifies the plugin that handles this CRD
	Handler *CRDHandler `yaml:"handler,omitempty" json:"handler,omitempty"`

	// Validation specifies additional validation rules
	Validation *CRDValidation `yaml:"validation,omitempty" json:"validation,omitempty"`

	// AdditionalPrinterColumns defines columns for tabular output
	AdditionalPrinterColumns []PrinterColumn `yaml:"additionalPrinterColumns,omitempty" json:"additionalPrinterColumns,omitempty"`
}

// ResourceScope defines the scope of a resource
type ResourceScope string

const (
	// ClusterScope means the resource is cluster-wide
	ClusterScope ResourceScope = "Cluster"

	// NamespacedScope means the resource is namespaced
	NamespacedScope ResourceScope = "Namespaced"
)

// CRDHandler specifies the plugin that handles a CRD
type CRDHandler struct {
	// Plugin is the plugin name
	Plugin string `yaml:"plugin" json:"plugin"`

	// Version is the plugin version constraint
	Version string `yaml:"version,omitempty" json:"version,omitempty"`

	// Config contains handler-specific configuration
	Config map[string]interface{} `yaml:"config,omitempty" json:"config,omitempty"`
}

// CRDValidation specifies additional validation rules
type CRDValidation struct {
	// OpenAPIV3Schema is the validation schema
	OpenAPIV3Schema *JSONSchemaProps `yaml:"openAPIV3Schema,omitempty" json:"openAPIV3Schema,omitempty"`
}

// PrinterColumn defines a column for tabular output
type PrinterColumn struct {
	// Name is the column header
	Name string `yaml:"name" json:"name"`

	// Type is the column type (string, integer, date, etc.)
	Type string `yaml:"type" json:"type"`

	// JSONPath is the path to the value
	JSONPath string `yaml:"jsonPath" json:"jsonPath"`

	// Description describes the column
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Priority is the column priority (0 = show always)
	Priority int32 `yaml:"priority,omitempty" json:"priority,omitempty"`
}

// CRDStatus contains the status of a CRD
type CRDStatus struct {
	// Conditions are the CRD conditions
	Conditions []CRDCondition `yaml:"conditions,omitempty" json:"conditions,omitempty"`

	// AcceptedNames are the accepted resource names
	AcceptedNames AcceptedNames `yaml:"acceptedNames,omitempty" json:"acceptedNames,omitempty"`

	// StoredVersions are the versions stored in the backend
	StoredVersions []string `yaml:"storedVersions,omitempty" json:"storedVersions,omitempty"`
}

// CRDCondition represents a condition of a CRD
type CRDCondition struct {
	// Type is the condition type
	Type CRDConditionType `yaml:"type" json:"type"`

	// Status is the condition status
	Status ConditionStatus `yaml:"status" json:"status"`

	// Reason is a brief reason for the condition
	Reason string `yaml:"reason,omitempty" json:"reason,omitempty"`

	// Message is a human-readable message
	Message string `yaml:"message,omitempty" json:"message,omitempty"`

	// LastTransitionTime is when the condition last changed
	LastTransitionTime *time.Time `yaml:"lastTransitionTime,omitempty" json:"lastTransitionTime,omitempty"`
}

// CRDConditionType represents a CRD condition type
type CRDConditionType string

const (
	// Established means the CRD is established
	CRDConditionEstablished CRDConditionType = "Established"

	// NamesAccepted means the names are accepted
	CRDConditionNamesAccepted CRDConditionType = "NamesAccepted"

	// Terminating means the CRD is being deleted
	CRDConditionTerminating CRDConditionType = "Terminating"
)

// ConditionStatus represents a condition status
type ConditionStatus string

const (
	ConditionTrue    ConditionStatus = "True"
	ConditionFalse   ConditionStatus = "False"
	ConditionUnknown ConditionStatus = "Unknown"
)

// AcceptedNames contains the accepted names for a CRD
type AcceptedNames struct {
	// Plural is the accepted plural name
	Plural string `yaml:"plural" json:"plural"`

	// Singular is the accepted singular name
	Singular string `yaml:"singular" json:"singular"`

	// Kind is the accepted kind
	Kind string `yaml:"kind" json:"kind"`

	// ListKind is the accepted list kind
	ListKind string `yaml:"listKind,omitempty" json:"listKind,omitempty"`

	// ShortNames are the accepted short names
	ShortNames []string `yaml:"shortNames,omitempty" json:"shortNames,omitempty"`
}

// CustomResource represents an instance of a custom resource
type CustomResource struct {
	// APIVersion is the API version
	APIVersion string `yaml:"apiVersion" json:"apiVersion"`

	// Kind is the resource kind
	Kind string `yaml:"kind" json:"kind"`

	// Metadata contains resource metadata
	Metadata Metadata `yaml:"metadata" json:"metadata"`

	// Spec contains the resource specification
	Spec map[string]interface{} `yaml:"spec,omitempty" json:"spec,omitempty"`

	// Status contains the resource status
	Status map[string]interface{} `yaml:"status,omitempty" json:"status,omitempty"`
}

// GroupVersionKind identifies a resource type
type GroupVersionKind struct {
	Group   string
	Version string
	Kind    string
}

// String returns a string representation
func (gvk GroupVersionKind) String() string {
	if gvk.Group == "" {
		return gvk.Version + "/" + gvk.Kind
	}
	return gvk.Group + "/" + gvk.Version + "/" + gvk.Kind
}

// GroupVersionResource identifies a resource type by its plural name
type GroupVersionResource struct {
	Group    string
	Version  string
	Resource string
}

// String returns a string representation
func (gvr GroupVersionResource) String() string {
	if gvr.Group == "" {
		return gvr.Version + "/" + gvr.Resource
	}
	return gvr.Group + "/" + gvr.Version + "/" + gvr.Resource
}
