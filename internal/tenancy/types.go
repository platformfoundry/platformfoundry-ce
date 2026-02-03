package tenancy

import (
	"time"
)

// Tenant represents an isolated organizational unit
type Tenant struct {
	APIVersion string         `json:"apiVersion" yaml:"apiVersion"`
	Kind       string         `json:"kind" yaml:"kind"`
	Metadata   TenantMetadata `json:"metadata" yaml:"metadata"`
	Spec       TenantSpec     `json:"spec" yaml:"spec"`
	Status     *TenantStatus  `json:"status,omitempty" yaml:"status,omitempty"`
}

// TenantMetadata contains tenant identification
type TenantMetadata struct {
	ID          string            `json:"id" yaml:"id"`
	Name        string            `json:"name" yaml:"name"`
	DisplayName string            `json:"displayName,omitempty" yaml:"displayName,omitempty"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	CreatedAt   time.Time         `json:"createdAt" yaml:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt" yaml:"updatedAt"`
}

// TenantSpec defines tenant configuration
type TenantSpec struct {
	Isolation    IsolationLevel   `json:"isolation" yaml:"isolation"`
	Quotas       *ResourceQuotas  `json:"quotas,omitempty" yaml:"quotas,omitempty"`
	Networks     *NetworkConfig   `json:"networks,omitempty" yaml:"networks,omitempty"`
	Compliance   *TenantCompliance `json:"compliance,omitempty" yaml:"compliance,omitempty"`
	CostCenter   string           `json:"costCenter,omitempty" yaml:"costCenter,omitempty"`
	BillingEmail string           `json:"billingEmail,omitempty" yaml:"billingEmail,omitempty"`
	Owners       []string         `json:"owners,omitempty" yaml:"owners,omitempty"`
	Clusters     []string         `json:"clusters,omitempty" yaml:"clusters,omitempty"`
}

// IsolationLevel defines tenant isolation
type IsolationLevel string

const (
	IsolationNamespace IsolationLevel = "namespace"
	IsolationCluster   IsolationLevel = "cluster"
	IsolationVPC       IsolationLevel = "vpc"
)

// ResourceQuotas defines resource limits for a tenant
type ResourceQuotas struct {
	CPU          string `json:"cpu,omitempty" yaml:"cpu,omitempty"`               // Total CPU cores
	Memory       string `json:"memory,omitempty" yaml:"memory,omitempty"`         // Total memory
	Storage      string `json:"storage,omitempty" yaml:"storage,omitempty"`       // Total storage
	Pods         int    `json:"pods,omitempty" yaml:"pods,omitempty"`             // Max pods
	Services     int    `json:"services,omitempty" yaml:"services,omitempty"`     // Max services
	Environments int    `json:"environments,omitempty" yaml:"environments,omitempty"` // Max environments
	Secrets      int    `json:"secrets,omitempty" yaml:"secrets,omitempty"`       // Max secrets
	ConfigMaps   int    `json:"configMaps,omitempty" yaml:"configMaps,omitempty"` // Max configmaps
	PVCs         int    `json:"pvcs,omitempty" yaml:"pvcs,omitempty"`             // Max PVCs
}

// NetworkConfig defines tenant network configuration
type NetworkConfig struct {
	CIDRBlocks   []string     `json:"cidrBlocks,omitempty" yaml:"cidrBlocks,omitempty"`
	EgressPolicy EgressPolicy `json:"egressPolicy,omitempty" yaml:"egressPolicy,omitempty"`
	IngressRules []string     `json:"ingressRules,omitempty" yaml:"ingressRules,omitempty"`
	DNSPolicy    string       `json:"dnsPolicy,omitempty" yaml:"dnsPolicy,omitempty"`
}

// EgressPolicy defines outbound traffic policy
type EgressPolicy string

const (
	EgressPolicyAllow      EgressPolicy = "allow"
	EgressPolicyRestricted EgressPolicy = "restricted"
	EgressPolicyDeny       EgressPolicy = "deny"
)

// TenantCompliance defines compliance requirements
type TenantCompliance struct {
	DataResidency []string `json:"dataResidency,omitempty" yaml:"dataResidency,omitempty"` // Allowed regions
	Frameworks    []string `json:"frameworks,omitempty" yaml:"frameworks,omitempty"`       // Required frameworks
	Encryption    bool     `json:"encryption,omitempty" yaml:"encryption,omitempty"`       // Require encryption
	AuditLogging  bool     `json:"auditLogging,omitempty" yaml:"auditLogging,omitempty"`   // Require audit logs
}

// TenantStatus tracks tenant state
type TenantStatus struct {
	Phase        TenantPhase       `json:"phase" yaml:"phase"`
	Conditions   []TenantCondition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	ResourceUsage *ResourceUsage   `json:"resourceUsage,omitempty" yaml:"resourceUsage,omitempty"`
	CostSummary   *CostSummary     `json:"costSummary,omitempty" yaml:"costSummary,omitempty"`
}

// TenantPhase indicates tenant lifecycle phase
type TenantPhase string

const (
	TenantPhaseActive      TenantPhase = "Active"
	TenantPhasePending     TenantPhase = "Pending"
	TenantPhaseSuspended   TenantPhase = "Suspended"
	TenantPhaseTerminating TenantPhase = "Terminating"
)

// TenantCondition represents a tenant state condition
type TenantCondition struct {
	Type               string    `json:"type" yaml:"type"`
	Status             string    `json:"status" yaml:"status"`
	LastTransitionTime time.Time `json:"lastTransitionTime" yaml:"lastTransitionTime"`
	Reason             string    `json:"reason,omitempty" yaml:"reason,omitempty"`
	Message            string    `json:"message,omitempty" yaml:"message,omitempty"`
}

// ResourceUsage tracks tenant resource consumption
type ResourceUsage struct {
	CPU            string    `json:"cpu" yaml:"cpu"`
	CPUPercent     float64   `json:"cpuPercent" yaml:"cpuPercent"`
	Memory         string    `json:"memory" yaml:"memory"`
	MemoryPercent  float64   `json:"memoryPercent" yaml:"memoryPercent"`
	Storage        string    `json:"storage" yaml:"storage"`
	StoragePercent float64   `json:"storagePercent" yaml:"storagePercent"`
	Pods           int       `json:"pods" yaml:"pods"`
	PodsPercent    float64   `json:"podsPercent" yaml:"podsPercent"`
	UpdatedAt      time.Time `json:"updatedAt" yaml:"updatedAt"`
}

// CostSummary provides cost information
type CostSummary struct {
	CurrentMonth  float64   `json:"currentMonth" yaml:"currentMonth"`
	PreviousMonth float64   `json:"previousMonth" yaml:"previousMonth"`
	Projected     float64   `json:"projected" yaml:"projected"`
	Currency      string    `json:"currency" yaml:"currency"`
	UpdatedAt     time.Time `json:"updatedAt" yaml:"updatedAt"`
}

// Role defines RBAC role within a tenant
type Role struct {
	APIVersion string       `json:"apiVersion" yaml:"apiVersion"`
	Kind       string       `json:"kind" yaml:"kind"`
	Metadata   RoleMetadata `json:"metadata" yaml:"metadata"`
	Spec       RoleSpec     `json:"spec" yaml:"spec"`
}

// RoleMetadata contains role identification
type RoleMetadata struct {
	Name        string            `json:"name" yaml:"name"`
	Tenant      string            `json:"tenant,omitempty" yaml:"tenant,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
}

// RoleSpec defines role permissions
type RoleSpec struct {
	Permissions  []Permission  `json:"permissions" yaml:"permissions"`
	Constraints  []Constraint  `json:"constraints,omitempty" yaml:"constraints,omitempty"`
	InheritFrom  []string      `json:"inheritFrom,omitempty" yaml:"inheritFrom,omitempty"`
}

// Permission defines what actions can be performed on which resources
type Permission struct {
	Resources    []string `json:"resources" yaml:"resources"`         // deployment, service, secret, etc.
	Verbs        []string `json:"verbs" yaml:"verbs"`                 // get, list, create, update, delete
	Environments []string `json:"environments,omitempty" yaml:"environments,omitempty"` // dev, staging, production
	Namespaces   []string `json:"namespaces,omitempty" yaml:"namespaces,omitempty"`     // specific namespaces
	Condition    string   `json:"condition,omitempty" yaml:"condition,omitempty"`       // OPA/CEL condition
}

// Constraint defines limitations on permission
type Constraint struct {
	Type  string `json:"type" yaml:"type"`   // maxReplicas, deniedImages, etc.
	Value string `json:"value" yaml:"value"` // constraint value
}

// RoleBinding binds roles to users/groups
type RoleBinding struct {
	APIVersion string              `json:"apiVersion" yaml:"apiVersion"`
	Kind       string              `json:"kind" yaml:"kind"`
	Metadata   RoleBindingMetadata `json:"metadata" yaml:"metadata"`
	Spec       RoleBindingSpec     `json:"spec" yaml:"spec"`
}

// RoleBindingMetadata contains binding identification
type RoleBindingMetadata struct {
	Name   string            `json:"name" yaml:"name"`
	Tenant string            `json:"tenant,omitempty" yaml:"tenant,omitempty"`
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

// RoleBindingSpec defines binding configuration
type RoleBindingSpec struct {
	RoleRef  RoleRef   `json:"roleRef" yaml:"roleRef"`
	Subjects []Subject `json:"subjects" yaml:"subjects"`
}

// RoleRef references a role
type RoleRef struct {
	Kind string `json:"kind" yaml:"kind"` // Role or ClusterRole
	Name string `json:"name" yaml:"name"`
}

// Subject is a user, group, or service account
type Subject struct {
	Kind      string `json:"kind" yaml:"kind"` // User, Group, ServiceAccount
	Name      string `json:"name" yaml:"name"`
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
}

// JITAccessRequest represents just-in-time access
type JITAccessRequest struct {
	ID           string        `json:"id" yaml:"id"`
	Requester    string        `json:"requester" yaml:"requester"`
	Tenant       string        `json:"tenant" yaml:"tenant"`
	Role         string        `json:"role" yaml:"role"`
	Reason       string        `json:"reason" yaml:"reason"`
	Duration     time.Duration `json:"duration" yaml:"duration"`
	Status       JITStatus     `json:"status" yaml:"status"`
	RequestedAt  time.Time     `json:"requestedAt" yaml:"requestedAt"`
	ApprovedBy   string        `json:"approvedBy,omitempty" yaml:"approvedBy,omitempty"`
	ApprovedAt   *time.Time    `json:"approvedAt,omitempty" yaml:"approvedAt,omitempty"`
	ExpiresAt    *time.Time    `json:"expiresAt,omitempty" yaml:"expiresAt,omitempty"`
}

// JITStatus represents JIT request status
type JITStatus string

const (
	JITStatusPending  JITStatus = "pending"
	JITStatusApproved JITStatus = "approved"
	JITStatusDenied   JITStatus = "denied"
	JITStatusExpired  JITStatus = "expired"
	JITStatusRevoked  JITStatus = "revoked"
)

// AccessReview represents an access review request
type AccessReview struct {
	ID         string       `json:"id" yaml:"id"`
	Tenant     string       `json:"tenant" yaml:"tenant"`
	Type       ReviewType   `json:"type" yaml:"type"`
	Status     ReviewStatus `json:"status" yaml:"status"`
	StartedAt  time.Time    `json:"startedAt" yaml:"startedAt"`
	DueDate    time.Time    `json:"dueDate" yaml:"dueDate"`
	CompletedAt *time.Time  `json:"completedAt,omitempty" yaml:"completedAt,omitempty"`
	Reviewer   string       `json:"reviewer,omitempty" yaml:"reviewer,omitempty"`
	Entries    []ReviewEntry `json:"entries" yaml:"entries"`
}

// ReviewType defines access review type
type ReviewType string

const (
	ReviewTypePeriodic ReviewType = "periodic"
	ReviewTypeOnDemand ReviewType = "on_demand"
	ReviewTypePromotion ReviewType = "promotion"
)

// ReviewStatus indicates review completion status
type ReviewStatus string

const (
	ReviewStatusPending   ReviewStatus = "pending"
	ReviewStatusInProgress ReviewStatus = "in_progress"
	ReviewStatusCompleted ReviewStatus = "completed"
	ReviewStatusOverdue   ReviewStatus = "overdue"
)

// ReviewEntry is a single item in an access review
type ReviewEntry struct {
	Subject      Subject     `json:"subject" yaml:"subject"`
	Role         string      `json:"role" yaml:"role"`
	LastUsed     *time.Time  `json:"lastUsed,omitempty" yaml:"lastUsed,omitempty"`
	Decision     string      `json:"decision,omitempty" yaml:"decision,omitempty"` // keep, revoke, modify
	DecidedBy    string      `json:"decidedBy,omitempty" yaml:"decidedBy,omitempty"`
	DecidedAt    *time.Time  `json:"decidedAt,omitempty" yaml:"decidedAt,omitempty"`
	Notes        string      `json:"notes,omitempty" yaml:"notes,omitempty"`
}
