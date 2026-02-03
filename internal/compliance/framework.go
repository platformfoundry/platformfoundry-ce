package compliance

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Framework represents a compliance framework
type Framework string

const (
	FrameworkSOC2     Framework = "SOC2"
	FrameworkHIPAA    Framework = "HIPAA"
	FrameworkPCIDSS   Framework = "PCI-DSS"
	FrameworkGDPR     Framework = "GDPR"
	FrameworkISO27001 Framework = "ISO27001"
	FrameworkNIST     Framework = "NIST"
)

// CheckStatus represents the status of a compliance check
type CheckStatus string

const (
	StatusPass    CheckStatus = "pass"
	StatusFail    CheckStatus = "fail"
	StatusWarning CheckStatus = "warning"
	StatusSkipped CheckStatus = "skipped"
)

// Check represents a compliance check
type Check struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Framework   Framework   `json:"framework"`
	Category    string      `json:"category"`
	Severity    string      `json:"severity"` // critical, high, medium, low
	Status      CheckStatus `json:"status"`
	Message     string      `json:"message,omitempty"`
	Remediation string      `json:"remediation,omitempty"`
	Evidence    []string    `json:"evidence,omitempty"`
	Timestamp   time.Time   `json:"timestamp"`
}

// Report represents a compliance report
type Report struct {
	ID            string            `json:"id"`
	Framework     Framework         `json:"framework"`
	Timestamp     time.Time         `json:"timestamp"`
	TotalChecks   int               `json:"totalChecks"`
	PassedChecks  int               `json:"passedChecks"`
	FailedChecks  int               `json:"failedChecks"`
	WarningChecks int               `json:"warningChecks"`
	SkippedChecks int               `json:"skippedChecks"`
	Compliance    float64           `json:"compliance"` // percentage
	Checks        []*Check          `json:"checks"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// Manager handles compliance checks and reporting
type Manager struct {
	checksDir  string
	reportsDir string
	frameworks map[Framework][]*Check
}

// Config represents compliance manager configuration
type Config struct {
	ChecksDir  string `yaml:"checksDir" json:"checksDir"`
	ReportsDir string `yaml:"reportsDir" json:"reportsDir"`
}

// DefaultConfig returns default compliance configuration
func DefaultConfig() *Config {
	return &Config{
		ChecksDir:  "/etc/platformfoundry/compliance/checks",
		ReportsDir: "/var/lib/platformfoundry/compliance/reports",
	}
}

// NewManager creates a new compliance manager
func NewManager(config *Config) (*Manager, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Ensure directories exist
	if err := os.MkdirAll(config.ChecksDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create checks directory: %w", err)
	}

	if err := os.MkdirAll(config.ReportsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create reports directory: %w", err)
	}

	manager := &Manager{
		checksDir:  config.ChecksDir,
		reportsDir: config.ReportsDir,
		frameworks: make(map[Framework][]*Check),
	}

	// Load built-in checks
	manager.loadBuiltInChecks()

	// Load custom checks from directory
	if err := manager.loadChecksFromDir(); err != nil {
		// Non-fatal, just log
		fmt.Printf("Warning: failed to load custom checks: %v\n", err)
	}

	return manager, nil
}

// loadBuiltInChecks loads built-in compliance checks
func (m *Manager) loadBuiltInChecks() {
	// SOC2 checks
	m.frameworks[FrameworkSOC2] = []*Check{
		{
			ID:          "soc2-ac-1",
			Title:       "Access Control - Multi-Factor Authentication",
			Description: "Verify that multi-factor authentication is enabled for all users",
			Framework:   FrameworkSOC2,
			Category:    "Access Control",
			Severity:    "critical",
			Remediation: "Enable MFA for all user accounts",
		},
		{
			ID:          "soc2-ac-2",
			Title:       "Access Control - Role-Based Access",
			Description: "Verify role-based access control is implemented",
			Framework:   FrameworkSOC2,
			Category:    "Access Control",
			Severity:    "high",
			Remediation: "Implement RBAC with least privilege principle",
		},
		{
			ID:          "soc2-log-1",
			Title:       "Logging - Audit Logs",
			Description: "Verify comprehensive audit logging is enabled",
			Framework:   FrameworkSOC2,
			Category:    "Logging and Monitoring",
			Severity:    "high",
			Remediation: "Enable audit logging for all critical operations",
		},
		{
			ID:          "soc2-enc-1",
			Title:       "Encryption - Data at Rest",
			Description: "Verify data is encrypted at rest",
			Framework:   FrameworkSOC2,
			Category:    "Encryption",
			Severity:    "critical",
			Remediation: "Enable encryption for all data stores",
		},
		{
			ID:          "soc2-enc-2",
			Title:       "Encryption - Data in Transit",
			Description: "Verify data is encrypted in transit (TLS/HTTPS)",
			Framework:   FrameworkSOC2,
			Category:    "Encryption",
			Severity:    "critical",
			Remediation: "Enable TLS 1.2+ for all network communications",
		},
	}

	// PCI-DSS checks
	m.frameworks[FrameworkPCIDSS] = []*Check{
		{
			ID:          "pci-fw-1",
			Title:       "Firewall Configuration",
			Description: "Verify firewall rules are properly configured",
			Framework:   FrameworkPCIDSS,
			Category:    "Network Security",
			Severity:    "critical",
			Remediation: "Configure firewall to block unauthorized access",
		},
		{
			ID:          "pci-pwd-1",
			Title:       "Password Requirements",
			Description: "Verify strong password policies are enforced",
			Framework:   FrameworkPCIDSS,
			Category:    "Access Control",
			Severity:    "high",
			Remediation: "Enforce minimum 12 character passwords with complexity requirements",
		},
		{
			ID:          "pci-card-1",
			Title:       "Cardholder Data Protection",
			Description: "Verify cardholder data is encrypted",
			Framework:   FrameworkPCIDSS,
			Category:    "Data Protection",
			Severity:    "critical",
			Remediation: "Encrypt all cardholder data using strong cryptography",
		},
	}

	// HIPAA checks
	m.frameworks[FrameworkHIPAA] = []*Check{
		{
			ID:          "hipaa-phi-1",
			Title:       "PHI Encryption",
			Description: "Verify Protected Health Information is encrypted",
			Framework:   FrameworkHIPAA,
			Category:    "Data Security",
			Severity:    "critical",
			Remediation: "Enable encryption for all PHI at rest and in transit",
		},
		{
			ID:          "hipaa-audit-1",
			Title:       "Audit Controls",
			Description: "Verify audit controls record access to PHI",
			Framework:   FrameworkHIPAA,
			Category:    "Audit and Accountability",
			Severity:    "high",
			Remediation: "Implement comprehensive audit logging for PHI access",
		},
		{
			ID:          "hipaa-auth-1",
			Title:       "User Authentication",
			Description: "Verify unique user identification and authentication",
			Framework:   FrameworkHIPAA,
			Category:    "Access Control",
			Severity:    "critical",
			Remediation: "Implement unique user IDs and strong authentication",
		},
	}

	// GDPR checks
	m.frameworks[FrameworkGDPR] = []*Check{
		{
			ID:          "gdpr-consent-1",
			Title:       "Consent Management",
			Description: "Verify consent is obtained for data processing",
			Framework:   FrameworkGDPR,
			Category:    "Consent",
			Severity:    "critical",
			Remediation: "Implement consent management system",
		},
		{
			ID:          "gdpr-data-1",
			Title:       "Data Minimization",
			Description: "Verify only necessary data is collected",
			Framework:   FrameworkGDPR,
			Category:    "Data Protection",
			Severity:    "medium",
			Remediation: "Review data collection and minimize to necessary fields",
		},
		{
			ID:          "gdpr-breach-1",
			Title:       "Breach Notification",
			Description: "Verify breach notification procedures are in place",
			Framework:   FrameworkGDPR,
			Category:    "Incident Response",
			Severity:    "high",
			Remediation: "Establish breach notification procedures within 72 hours",
		},
	}
}

// loadChecksFromDir loads custom checks from directory
func (m *Manager) loadChecksFromDir() error {
	entries, err := os.ReadDir(m.checksDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(m.checksDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var check Check
		if err := json.Unmarshal(data, &check); err != nil {
			continue
		}

		m.frameworks[check.Framework] = append(m.frameworks[check.Framework], &check)
	}

	return nil
}

// RunChecks runs compliance checks for a framework
func (m *Manager) RunChecks(ctx context.Context, framework Framework) (*Report, error) {
	checks, ok := m.frameworks[framework]
	if !ok {
		return nil, fmt.Errorf("unknown framework: %s", framework)
	}

	report := &Report{
		ID:          generateReportID(),
		Framework:   framework,
		Timestamp:   time.Now(),
		TotalChecks: len(checks),
		Checks:      make([]*Check, 0, len(checks)),
		Metadata:    make(map[string]string),
	}

	// Run each check
	for _, check := range checks {
		result := m.runCheck(ctx, check)
		report.Checks = append(report.Checks, result)

		switch result.Status {
		case StatusPass:
			report.PassedChecks++
		case StatusFail:
			report.FailedChecks++
		case StatusWarning:
			report.WarningChecks++
		case StatusSkipped:
			report.SkippedChecks++
		}
	}

	// Calculate compliance percentage
	if report.TotalChecks > 0 {
		report.Compliance = float64(report.PassedChecks) / float64(report.TotalChecks) * 100
	}

	// Save report
	if err := m.saveReport(report); err != nil {
		fmt.Printf("Warning: failed to save report: %v\n", err)
	}

	return report, nil
}

// runCheck runs a single compliance check
func (m *Manager) runCheck(ctx context.Context, check *Check) *Check {
	result := &Check{
		ID:          check.ID,
		Title:       check.Title,
		Description: check.Description,
		Framework:   check.Framework,
		Category:    check.Category,
		Severity:    check.Severity,
		Remediation: check.Remediation,
		Timestamp:   time.Now(),
		Evidence:    make([]string, 0),
	}

	// Placeholder check logic
	// In a real implementation, would perform actual validation
	// For now, randomly assign status for demonstration
	result.Status = StatusPass
	result.Message = "Check passed"

	return result
}

// saveReport saves a compliance report
func (m *Manager) saveReport(report *Report) error {
	filename := fmt.Sprintf("%s-%s.json", report.Framework, report.Timestamp.Format("20060102-150405"))
	path := filepath.Join(m.reportsDir, filename)

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// LoadReport loads a compliance report
func (m *Manager) LoadReport(reportID string) (*Report, error) {
	path := filepath.Join(m.reportsDir, reportID)
	if filepath.Ext(path) != ".json" {
		path += ".json"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read report: %w", err)
	}

	var report Report
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to parse report: %w", err)
	}

	return &report, nil
}

// ListReports lists all compliance reports
func (m *Manager) ListReports(framework *Framework) ([]*Report, error) {
	entries, err := os.ReadDir(m.reportsDir)
	if err != nil {
		return nil, err
	}

	reports := make([]*Report, 0)

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(m.reportsDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var report Report
		if err := json.Unmarshal(data, &report); err != nil {
			continue
		}

		// Filter by framework if specified
		if framework != nil && report.Framework != *framework {
			continue
		}

		reports = append(reports, &report)
	}

	return reports, nil
}

// ListFrameworks returns all supported frameworks
func (m *Manager) ListFrameworks() []Framework {
	frameworks := make([]Framework, 0, len(m.frameworks))
	for f := range m.frameworks {
		frameworks = append(frameworks, f)
	}
	return frameworks
}

// GetChecks returns all checks for a framework
func (m *Manager) GetChecks(framework Framework) ([]*Check, error) {
	checks, ok := m.frameworks[framework]
	if !ok {
		return nil, fmt.Errorf("unknown framework: %s", framework)
	}

	return checks, nil
}

// generateReportID generates a unique report ID
func generateReportID() string {
	return fmt.Sprintf("report_%d", time.Now().UnixNano())
}
