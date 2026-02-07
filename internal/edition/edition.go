// Package edition provides Platform Foundry edition information.
// This is the Community Edition - free and open source.
package edition

// Edition constants
const (
	Edition     = "community"
	EditionName = "Platform Foundry Community"
)

// IsEnterprise returns false - this is the community edition
func IsEnterprise() bool {
	return false
}

// IsCommunity returns true - this is the community edition
func IsCommunity() bool {
	return true
}

// Features lists community edition features
var Features = []string{
	"cli",
	"yaml_orchestration",
	"plugins",
	"approval_workflows",
	"ephemeral_environments",
	"basic_rbac",
	"basic_cost_estimation",
	"multi_tenancy",
	"state_management",
}

// EnterpriseFeatures lists features only available in Enterprise
var EnterpriseFeatures = []string{
	"cost_tracking",
	"analytics",
	"visualization",
	"sso",
	"advanced_rbac",
	"audit_export",
	"compliance_reports",
	"managed_state",
	"ai_recommendations",
	"dora_metrics",
}

// RequireEnterprise returns an error message for enterprise-only features
func RequireEnterprise(feature string) string {
	return "Feature '" + feature + "' requires Platform Foundry Enterprise. " +
		"Visit https://platformfoundry.io/enterprise for more information."
}
