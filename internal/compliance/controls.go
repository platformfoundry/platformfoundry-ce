package compliance

import (
	"fmt"
	"time"
)

// ControlMapping maps compliance controls to platform features
type ControlMapping struct {
	controls  map[string]*Control
	mappings  map[string][]PlatformFeature
	coverage  map[string]*ControlCoverage
}

// Control represents a compliance control
type Control struct {
	ID           string              `json:"id" yaml:"id"`
	Framework    string              `json:"framework" yaml:"framework"`
	Category     string              `json:"category" yaml:"category"`
	Name         string              `json:"name" yaml:"name"`
	Description  string              `json:"description" yaml:"description"`
	Requirement  string              `json:"requirement" yaml:"requirement"`
	Guidance     string              `json:"guidance,omitempty" yaml:"guidance,omitempty"`
	References   []string            `json:"references,omitempty" yaml:"references,omitempty"`
	SubControls  []string            `json:"subControls,omitempty" yaml:"subControls,omitempty"`
}

// PlatformFeature represents a platform capability
type PlatformFeature struct {
	ID          string            `json:"id" yaml:"id"`
	Name        string            `json:"name" yaml:"name"`
	Component   string            `json:"component" yaml:"component"`
	Description string            `json:"description" yaml:"description"`
	ConfigPath  string            `json:"configPath,omitempty" yaml:"configPath,omitempty"`
	Status      FeatureStatus     `json:"status" yaml:"status"`
	Coverage    CoverageLevel     `json:"coverage" yaml:"coverage"`
	Automation  AutomationLevel   `json:"automation" yaml:"automation"`
}

// FeatureStatus indicates feature implementation status
type FeatureStatus string

const (
	FeatureStatusActive      FeatureStatus = "active"
	FeatureStatusPartial     FeatureStatus = "partial"
	FeatureStatusPlanned     FeatureStatus = "planned"
	FeatureStatusNotAvailable FeatureStatus = "not_available"
)

// CoverageLevel indicates how well a control is covered
type CoverageLevel string

const (
	CoverageFull    CoverageLevel = "full"
	CoveragePartial CoverageLevel = "partial"
	CoverageMinimal CoverageLevel = "minimal"
	CoverageNone    CoverageLevel = "none"
)

// AutomationLevel indicates level of automation
type AutomationLevel string

const (
	AutomationFull       AutomationLevel = "full"
	AutomationPartial    AutomationLevel = "partial"
	AutomationManual     AutomationLevel = "manual"
	AutomationAssisted   AutomationLevel = "assisted"
)

// ControlCoverage tracks coverage for a control
type ControlCoverage struct {
	ControlID       string            `json:"controlId"`
	Framework       string            `json:"framework"`
	OverallCoverage CoverageLevel     `json:"overallCoverage"`
	Features        []PlatformFeature `json:"features"`
	Gaps            []string          `json:"gaps,omitempty"`
	LastAssessed    time.Time         `json:"lastAssessed"`
}

// NewControlMapping creates a new control mapping
func NewControlMapping() *ControlMapping {
	cm := &ControlMapping{
		controls: make(map[string]*Control),
		mappings: make(map[string][]PlatformFeature),
		coverage: make(map[string]*ControlCoverage),
	}

	// Load built-in controls and mappings
	cm.loadSOC2Controls()
	cm.loadHIPAAControls()
	cm.loadPCIDSSControls()
	cm.loadGDPRControls()
	cm.loadPlatformMappings()

	return cm
}

// loadSOC2Controls loads SOC2 Trust Services Criteria
func (cm *ControlMapping) loadSOC2Controls() {
	controls := []*Control{
		{
			ID:          "CC6.1",
			Framework:   "SOC2",
			Category:    "Logical and Physical Access Controls",
			Name:        "Logical Access Security Software",
			Description: "The entity implements logical access security software, infrastructure, and architectures over protected information assets to protect them from security events.",
			Requirement: "Implement software for identification and authentication, role-based access control, and access control policies.",
		},
		{
			ID:          "CC6.2",
			Framework:   "SOC2",
			Category:    "Logical and Physical Access Controls",
			Name:        "User Registration and Authorization",
			Description: "Prior to issuing system credentials and granting system access, the entity registers and authorizes new internal and external users.",
			Requirement: "Formal user registration process with documented authorization.",
		},
		{
			ID:          "CC6.3",
			Framework:   "SOC2",
			Category:    "Logical and Physical Access Controls",
			Name:        "User Access Reviews",
			Description: "The entity performs periodic reviews of user access to protected information and system components.",
			Requirement: "Regular access reviews with documented evidence.",
		},
		{
			ID:          "CC6.6",
			Framework:   "SOC2",
			Category:    "Logical and Physical Access Controls",
			Name:        "System Boundaries",
			Description: "The entity implements logical access security measures to protect against threats from sources outside its system boundaries.",
			Requirement: "Network segmentation, firewalls, and perimeter security.",
		},
		{
			ID:          "CC6.7",
			Framework:   "SOC2",
			Category:    "Logical and Physical Access Controls",
			Name:        "Information Transmission",
			Description: "The entity restricts the transmission, movement, and removal of information to authorized internal and external users and processes.",
			Requirement: "Data loss prevention and encryption in transit.",
		},
		{
			ID:          "CC7.1",
			Framework:   "SOC2",
			Category:    "System Operations",
			Name:        "Security Event Detection",
			Description: "To meet its objectives, the entity uses detection and monitoring procedures to identify changes to configurations that result in the introduction of new vulnerabilities.",
			Requirement: "Continuous monitoring and vulnerability scanning.",
		},
		{
			ID:          "CC7.2",
			Framework:   "SOC2",
			Category:    "System Operations",
			Name:        "Security Event Response",
			Description: "The entity monitors system components and the operation of those components for anomalies that are indicative of malicious acts.",
			Requirement: "Security monitoring, alerting, and incident response.",
		},
		{
			ID:          "CC8.1",
			Framework:   "SOC2",
			Category:    "Change Management",
			Name:        "Change Management Process",
			Description: "The entity authorizes, designs, develops or acquires, configures, documents, tests, approves, and implements changes to infrastructure.",
			Requirement: "Formal change management with testing and approval.",
		},
	}

	for _, c := range controls {
		cm.controls[c.ID] = c
	}
}

// loadHIPAAControls loads HIPAA Security Rule controls
func (cm *ControlMapping) loadHIPAAControls() {
	controls := []*Control{
		{
			ID:          "164.312(a)(1)",
			Framework:   "HIPAA",
			Category:    "Technical Safeguards",
			Name:        "Access Control",
			Description: "Implement technical policies and procedures for electronic information systems that maintain electronic protected health information to allow access only to those persons or software programs that have been granted access rights.",
			Requirement: "Unique user identification, emergency access, automatic logoff, encryption.",
		},
		{
			ID:          "164.312(a)(2)(i)",
			Framework:   "HIPAA",
			Category:    "Technical Safeguards",
			Name:        "Unique User Identification",
			Description: "Assign a unique name and/or number for identifying and tracking user identity.",
			Requirement: "Unique identifiers for all users accessing ePHI.",
		},
		{
			ID:          "164.312(b)",
			Framework:   "HIPAA",
			Category:    "Technical Safeguards",
			Name:        "Audit Controls",
			Description: "Implement hardware, software, and/or procedural mechanisms that record and examine activity in information systems that contain or use electronic protected health information.",
			Requirement: "Comprehensive audit logging of ePHI access.",
		},
		{
			ID:          "164.312(c)(1)",
			Framework:   "HIPAA",
			Category:    "Technical Safeguards",
			Name:        "Integrity",
			Description: "Implement policies and procedures to protect electronic protected health information from improper alteration or destruction.",
			Requirement: "Mechanisms to authenticate ePHI and detect unauthorized changes.",
		},
		{
			ID:          "164.312(e)(1)",
			Framework:   "HIPAA",
			Category:    "Technical Safeguards",
			Name:        "Transmission Security",
			Description: "Implement technical security measures to guard against unauthorized access to electronic protected health information that is being transmitted over an electronic communications network.",
			Requirement: "Encryption and integrity controls for data in transit.",
		},
	}

	for _, c := range controls {
		cm.controls[c.ID] = c
	}
}

// loadPCIDSSControls loads PCI-DSS controls
func (cm *ControlMapping) loadPCIDSSControls() {
	controls := []*Control{
		{
			ID:          "PCI-1.1",
			Framework:   "PCI-DSS",
			Category:    "Build and Maintain a Secure Network",
			Name:        "Firewall Configuration",
			Description: "Install and maintain a firewall configuration to protect cardholder data.",
			Requirement: "Formal process for firewall rule management and review.",
		},
		{
			ID:          "PCI-3.4",
			Framework:   "PCI-DSS",
			Category:    "Protect Stored Cardholder Data",
			Name:        "PAN Protection",
			Description: "Render PAN unreadable anywhere it is stored.",
			Requirement: "Strong cryptography with associated key management.",
		},
		{
			ID:          "PCI-4.1",
			Framework:   "PCI-DSS",
			Category:    "Encrypt Transmission of Cardholder Data",
			Name:        "Secure Transmission",
			Description: "Use strong cryptography and security protocols to safeguard sensitive cardholder data during transmission over open, public networks.",
			Requirement: "TLS 1.2+ for all cardholder data transmission.",
		},
		{
			ID:          "PCI-8.3",
			Framework:   "PCI-DSS",
			Category:    "Identify and Authenticate Access",
			Name:        "Multi-Factor Authentication",
			Description: "Secure all individual non-console administrative access and all remote access to the CDE using multi-factor authentication.",
			Requirement: "MFA for administrative and remote access.",
		},
		{
			ID:          "PCI-10.1",
			Framework:   "PCI-DSS",
			Category:    "Track and Monitor All Access",
			Name:        "Audit Trail",
			Description: "Implement audit trails to link all access to system components to each individual user.",
			Requirement: "User accountability through comprehensive logging.",
		},
	}

	for _, c := range controls {
		cm.controls[c.ID] = c
	}
}

// loadGDPRControls loads GDPR articles
func (cm *ControlMapping) loadGDPRControls() {
	controls := []*Control{
		{
			ID:          "GDPR-32",
			Framework:   "GDPR",
			Category:    "Security of Processing",
			Name:        "Security of Processing",
			Description: "Implement appropriate technical and organizational measures to ensure a level of security appropriate to the risk.",
			Requirement: "Pseudonymization, encryption, confidentiality, integrity, availability, resilience, testing.",
		},
		{
			ID:          "GDPR-33",
			Framework:   "GDPR",
			Category:    "Data Breach",
			Name:        "Breach Notification",
			Description: "Notify supervisory authority within 72 hours of becoming aware of a personal data breach.",
			Requirement: "Breach detection and notification procedures.",
		},
		{
			ID:          "GDPR-25",
			Framework:   "GDPR",
			Category:    "Data Protection by Design",
			Name:        "Data Protection by Design and Default",
			Description: "Implement appropriate technical and organizational measures designed to implement data-protection principles.",
			Requirement: "Privacy by design in all processing activities.",
		},
	}

	for _, c := range controls {
		cm.controls[c.ID] = c
	}
}

// loadPlatformMappings loads control-to-feature mappings
func (cm *ControlMapping) loadPlatformMappings() {
	// SOC2 CC6.1 - Logical Access Security
	cm.mappings["CC6.1"] = []PlatformFeature{
		{
			ID:          "auth-jwt",
			Name:        "JWT Authentication",
			Component:   "auth",
			Description: "JSON Web Token based authentication for API access",
			ConfigPath:  "auth.jwt",
			Status:      FeatureStatusActive,
			Coverage:    CoverageFull,
			Automation:  AutomationFull,
		},
		{
			ID:          "auth-saml",
			Name:        "SAML SSO",
			Component:   "auth",
			Description: "SAML 2.0 single sign-on integration",
			ConfigPath:  "auth.saml",
			Status:      FeatureStatusActive,
			Coverage:    CoverageFull,
			Automation:  AutomationFull,
		},
		{
			ID:          "rbac",
			Name:        "Role-Based Access Control",
			Component:   "auth",
			Description: "Fine-grained RBAC with policy enforcement",
			ConfigPath:  "auth.rbac",
			Status:      FeatureStatusActive,
			Coverage:    CoverageFull,
			Automation:  AutomationFull,
		},
		{
			ID:          "policy-opa",
			Name:        "OPA Policy Engine",
			Component:   "policy",
			Description: "Open Policy Agent integration for access decisions",
			ConfigPath:  "policy.opa",
			Status:      FeatureStatusActive,
			Coverage:    CoverageFull,
			Automation:  AutomationFull,
		},
	}

	// SOC2 CC6.6 - System Boundaries
	cm.mappings["CC6.6"] = []PlatformFeature{
		{
			ID:          "network-policy",
			Name:        "Network Policy Management",
			Component:   "security",
			Description: "Kubernetes NetworkPolicy management",
			Status:      FeatureStatusActive,
			Coverage:    CoverageFull,
			Automation:  AutomationFull,
		},
		{
			ID:          "federation-isolation",
			Name:        "Cluster Federation Isolation",
			Component:   "federation",
			Description: "Multi-cluster network isolation",
			Status:      FeatureStatusActive,
			Coverage:    CoveragePartial,
			Automation:  AutomationPartial,
		},
	}

	// SOC2 CC7.1 - Security Event Detection
	cm.mappings["CC7.1"] = []PlatformFeature{
		{
			ID:          "security-scan",
			Name:        "Security Scanning",
			Component:   "security",
			Description: "Container and configuration security scanning",
			Status:      FeatureStatusActive,
			Coverage:    CoverageFull,
			Automation:  AutomationFull,
		},
		{
			ID:          "compliance-monitor",
			Name:        "Continuous Compliance Monitoring",
			Component:   "compliance",
			Description: "Continuous compliance drift detection",
			Status:      FeatureStatusActive,
			Coverage:    CoverageFull,
			Automation:  AutomationFull,
		},
	}

	// SOC2 CC8.1 - Change Management
	cm.mappings["CC8.1"] = []PlatformFeature{
		{
			ID:          "gitops",
			Name:        "GitOps Workflow",
			Component:   "deployment",
			Description: "Git-based change management with ArgoCD/Flux",
			Status:      FeatureStatusActive,
			Coverage:    CoverageFull,
			Automation:  AutomationFull,
		},
		{
			ID:          "approval-workflow",
			Name:        "Approval Workflows",
			Component:   "workflow",
			Description: "Multi-stage approval workflows for changes",
			Status:      FeatureStatusActive,
			Coverage:    CoverageFull,
			Automation:  AutomationPartial,
		},
	}

	// HIPAA audit controls
	cm.mappings["164.312(b)"] = []PlatformFeature{
		{
			ID:          "audit-log",
			Name:        "Audit Logging",
			Component:   "observability",
			Description: "Comprehensive audit trail logging",
			Status:      FeatureStatusActive,
			Coverage:    CoverageFull,
			Automation:  AutomationFull,
		},
	}

	// PCI-DSS encryption
	cm.mappings["PCI-3.4"] = []PlatformFeature{
		{
			ID:          "secrets-vault",
			Name:        "Vault Integration",
			Component:   "secrets",
			Description: "HashiCorp Vault for secrets management",
			Status:      FeatureStatusActive,
			Coverage:    CoverageFull,
			Automation:  AutomationFull,
		},
		{
			ID:          "secrets-aws",
			Name:        "AWS Secrets Manager",
			Component:   "secrets",
			Description: "AWS Secrets Manager integration",
			Status:      FeatureStatusActive,
			Coverage:    CoverageFull,
			Automation:  AutomationFull,
		},
	}
}

// GetControl returns a control by ID
func (cm *ControlMapping) GetControl(controlID string) (*Control, error) {
	control, ok := cm.controls[controlID]
	if !ok {
		return nil, fmt.Errorf("control not found: %s", controlID)
	}
	return control, nil
}

// GetControlsByFramework returns all controls for a framework
func (cm *ControlMapping) GetControlsByFramework(framework string) []*Control {
	result := make([]*Control, 0)
	for _, c := range cm.controls {
		if c.Framework == framework {
			result = append(result, c)
		}
	}
	return result
}

// GetFeatures returns platform features for a control
func (cm *ControlMapping) GetFeatures(controlID string) []PlatformFeature {
	return cm.mappings[controlID]
}

// GetCoverage returns coverage assessment for a control
func (cm *ControlMapping) GetCoverage(controlID string) (*ControlCoverage, error) {
	control, err := cm.GetControl(controlID)
	if err != nil {
		return nil, err
	}

	features := cm.GetFeatures(controlID)

	coverage := &ControlCoverage{
		ControlID:    controlID,
		Framework:    control.Framework,
		Features:     features,
		Gaps:         make([]string, 0),
		LastAssessed: time.Now(),
	}

	// Calculate overall coverage
	if len(features) == 0 {
		coverage.OverallCoverage = CoverageNone
		coverage.Gaps = append(coverage.Gaps, "No platform features mapped to this control")
	} else {
		fullCount := 0
		partialCount := 0
		for _, f := range features {
			switch f.Coverage {
			case CoverageFull:
				fullCount++
			case CoveragePartial:
				partialCount++
			}
			if f.Status != FeatureStatusActive {
				coverage.Gaps = append(coverage.Gaps, fmt.Sprintf("Feature '%s' is not active", f.Name))
			}
		}

		if fullCount == len(features) {
			coverage.OverallCoverage = CoverageFull
		} else if fullCount > 0 || partialCount > 0 {
			coverage.OverallCoverage = CoveragePartial
		} else {
			coverage.OverallCoverage = CoverageMinimal
		}
	}

	return coverage, nil
}

// GenerateCoverageReport generates a coverage report for a framework
func (cm *ControlMapping) GenerateCoverageReport(framework string) *CoverageReport {
	controls := cm.GetControlsByFramework(framework)

	report := &CoverageReport{
		Framework:   framework,
		GeneratedAt: time.Now(),
		Controls:    make([]ControlCoverage, 0),
		Summary: CoverageSummary{
			ByCategory: make(map[string]CategoryCoverage),
		},
	}

	for _, control := range controls {
		coverage, err := cm.GetCoverage(control.ID)
		if err != nil {
			continue
		}

		report.Controls = append(report.Controls, *coverage)
		report.Summary.TotalControls++

		switch coverage.OverallCoverage {
		case CoverageFull:
			report.Summary.FullCoverage++
		case CoveragePartial:
			report.Summary.PartialCoverage++
		case CoverageMinimal:
			report.Summary.MinimalCoverage++
		case CoverageNone:
			report.Summary.NoCoverage++
		}

		// Update category summary
		if _, ok := report.Summary.ByCategory[control.Category]; !ok {
			report.Summary.ByCategory[control.Category] = CategoryCoverage{}
		}
		cat := report.Summary.ByCategory[control.Category]
		cat.Total++
		if coverage.OverallCoverage == CoverageFull || coverage.OverallCoverage == CoveragePartial {
			cat.Covered++
		}
		report.Summary.ByCategory[control.Category] = cat
	}

	// Calculate overall percentage
	if report.Summary.TotalControls > 0 {
		report.Summary.OverallPercentage = float64(report.Summary.FullCoverage+report.Summary.PartialCoverage) / float64(report.Summary.TotalControls) * 100
	}

	return report
}

// CoverageReport represents a framework coverage report
type CoverageReport struct {
	Framework   string            `json:"framework"`
	GeneratedAt time.Time         `json:"generatedAt"`
	Controls    []ControlCoverage `json:"controls"`
	Summary     CoverageSummary   `json:"summary"`
}

// CoverageSummary provides coverage statistics
type CoverageSummary struct {
	TotalControls     int                        `json:"totalControls"`
	FullCoverage      int                        `json:"fullCoverage"`
	PartialCoverage   int                        `json:"partialCoverage"`
	MinimalCoverage   int                        `json:"minimalCoverage"`
	NoCoverage        int                        `json:"noCoverage"`
	OverallPercentage float64                    `json:"overallPercentage"`
	ByCategory        map[string]CategoryCoverage `json:"byCategory"`
}

// CategoryCoverage tracks coverage by category
type CategoryCoverage struct {
	Total   int `json:"total"`
	Covered int `json:"covered"`
}

// ListFrameworks returns all supported frameworks
func (cm *ControlMapping) ListFrameworks() []string {
	frameworks := make(map[string]bool)
	for _, c := range cm.controls {
		frameworks[c.Framework] = true
	}

	result := make([]string, 0, len(frameworks))
	for f := range frameworks {
		result = append(result, f)
	}
	return result
}

// GetGaps returns all coverage gaps across frameworks
func (cm *ControlMapping) GetGaps() []GapAnalysis {
	gaps := make([]GapAnalysis, 0)

	for controlID, control := range cm.controls {
		coverage, _ := cm.GetCoverage(controlID)
		if coverage != nil && len(coverage.Gaps) > 0 {
			gaps = append(gaps, GapAnalysis{
				ControlID:   controlID,
				ControlName: control.Name,
				Framework:   control.Framework,
				Category:    control.Category,
				Coverage:    coverage.OverallCoverage,
				Gaps:        coverage.Gaps,
			})
		}
	}

	return gaps
}

// GapAnalysis represents a gap in compliance coverage
type GapAnalysis struct {
	ControlID   string        `json:"controlId"`
	ControlName string        `json:"controlName"`
	Framework   string        `json:"framework"`
	Category    string        `json:"category"`
	Coverage    CoverageLevel `json:"coverage"`
	Gaps        []string      `json:"gaps"`
}
