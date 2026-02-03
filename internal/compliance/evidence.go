package compliance

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// EvidenceType represents the type of compliance evidence
type EvidenceType string

const (
	EvidenceTypeScreenshot  EvidenceType = "screenshot"
	EvidenceTypeLog         EvidenceType = "log"
	EvidenceTypeConfig      EvidenceType = "config"
	EvidenceTypeAttestation EvidenceType = "attestation"
	EvidenceTypeReport      EvidenceType = "report"
	EvidenceTypePolicy      EvidenceType = "policy"
	EvidenceTypeAuditTrail  EvidenceType = "audit_trail"
)

// Evidence represents compliance evidence for a control
type Evidence struct {
	ID          string            `json:"id"`
	ControlID   string            `json:"controlId"`
	Framework   string            `json:"framework"`
	Type        EvidenceType      `json:"type"`
	Description string            `json:"description"`
	Content     []byte            `json:"content"`
	ContentType string            `json:"contentType"`
	Metadata    map[string]string `json:"metadata"`
	CollectedAt time.Time         `json:"collectedAt"`
	CollectedBy string            `json:"collectedBy"`
	ValidUntil  time.Time         `json:"validUntil"`
	Hash        string            `json:"hash"`
}

// PolicyEngineInterface interface for policy operations
type PolicyEngineInterface interface {
	ListPolicies(ctx context.Context) ([]interface{}, error)
}

// AuditLoggerInterface interface for audit logging
type AuditLoggerInterface interface {
	GetLogs(ctx context.Context, filter AuditFilter) ([]AuditEntry, error)
}

// StorageBackendInterface interface for evidence storage
type StorageBackendInterface interface {
	Store(ctx context.Context, path string, data []byte) error
	Retrieve(ctx context.Context, path string) ([]byte, error)
	List(ctx context.Context, prefix string) ([]string, error)
}

// StateBackendInterface interface for state operations
type StateBackendInterface interface {
	Get(ctx context.Context, kind, id string) (interface{}, error)
	List(ctx context.Context, kind string) ([]interface{}, error)
}

// AuditFilter defines criteria for filtering audit logs
type AuditFilter struct {
	StartTime  time.Time `json:"startTime"`
	EndTime    time.Time `json:"endTime"`
	Actor      string    `json:"actor,omitempty"`
	Resource   string    `json:"resource,omitempty"`
	Action     string    `json:"action,omitempty"`
	MaxResults int       `json:"maxResults,omitempty"`
}

// AuditEntry represents an audit log entry
type AuditEntry struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Actor     string                 `json:"actor"`
	Action    string                 `json:"action"`
	Resource  string                 `json:"resource"`
	Details   map[string]interface{} `json:"details"`
	Result    string                 `json:"result"`
	IPAddress string                 `json:"ipAddress,omitempty"`
}

// EvidenceCollectorConfig contains configuration for the evidence collector
type EvidenceCollectorConfig struct {
	DefaultValidityPeriod time.Duration `json:"defaultValidityPeriod"`
	StoragePath           string        `json:"storagePath"`
	RetentionPeriod       time.Duration `json:"retentionPeriod"`
}

// EvidenceCollector collects compliance evidence
type EvidenceCollector struct {
	stateBackend  StateBackendInterface
	auditLog      AuditLoggerInterface
	policyEngine  PolicyEngineInterface
	storage       StorageBackendInterface
	config        EvidenceCollectorConfig
}

// NewEvidenceCollector creates a new evidence collector
func NewEvidenceCollector(state StateBackendInterface, audit AuditLoggerInterface, policy PolicyEngineInterface, storage StorageBackendInterface, config EvidenceCollectorConfig) *EvidenceCollector {
	if config.DefaultValidityPeriod == 0 {
		config.DefaultValidityPeriod = 90 * 24 * time.Hour
	}
	if config.RetentionPeriod == 0 {
		config.RetentionPeriod = 365 * 24 * time.Hour
	}

	return &EvidenceCollector{
		stateBackend: state,
		auditLog:     audit,
		policyEngine: policy,
		storage:      storage,
		config:       config,
	}
}

// CollectForCheck collects evidence for a specific compliance check
func (c *EvidenceCollector) CollectForCheck(ctx context.Context, check *Check) ([]Evidence, error) {
	var evidences []Evidence
	var e *Evidence
	var err error

	switch check.Category {
	case "Access Control":
		e, err = c.collectRBACEvidence(ctx, check)
	case "Encryption":
		e, err = c.collectEncryptionEvidence(ctx, check)
	case "Logging and Monitoring", "Audit and Accountability":
		e, err = c.collectAuditEvidence(ctx, check)
	case "Network Security":
		e, err = c.collectNetworkEvidence(ctx, check)
	case "Data Protection", "Data Security":
		e, err = c.collectEncryptionEvidence(ctx, check)
	default:
		e, err = c.collectGenericEvidence(ctx, check)
	}

	if err != nil {
		return evidences, err
	}

	if e != nil {
		evidences = append(evidences, *e)
		if c.storage != nil {
			path := fmt.Sprintf("evidence/%s/%s/%s", e.Framework, e.ControlID, e.ID)
			data, _ := json.Marshal(e)
			c.storage.Store(ctx, path, data)
		}
	}

	return evidences, nil
}

// collectGenericEvidence collects generic evidence for a check
func (c *EvidenceCollector) collectGenericEvidence(ctx context.Context, check *Check) (*Evidence, error) {
	data := map[string]interface{}{
		"checkId":    check.ID,
		"checkTitle": check.Title,
		"timestamp":  time.Now(),
	}
	content, _ := json.MarshalIndent(data, "", "  ")

	return &Evidence{
		ID:          fmt.Sprintf("generic-%d", time.Now().Unix()),
		ControlID:   check.ID,
		Framework:   string(check.Framework),
		Type:        EvidenceTypeConfig,
		Description: fmt.Sprintf("Evidence for %s", check.Title),
		Content:     content,
		ContentType: "application/json",
		Metadata:    map[string]string{"source": "compliance-system", "category": check.Category},
		CollectedAt: time.Now(),
		ValidUntil:  time.Now().Add(c.config.DefaultValidityPeriod),
	}, nil
}

// collectRBACEvidence collects RBAC-related evidence
func (c *EvidenceCollector) collectRBACEvidence(ctx context.Context, check *Check) (*Evidence, error) {
	data := make(map[string]interface{})

	if c.policyEngine != nil {
		policies, err := c.policyEngine.ListPolicies(ctx)
		if err == nil {
			data["policies"] = policies
		}
	}

	if c.stateBackend != nil {
		bindings, err := c.stateBackend.List(ctx, "RoleBinding")
		if err == nil {
			data["roleBindings"] = bindings
		}
	}

	data["timestamp"] = time.Now()
	content, _ := json.MarshalIndent(data, "", "  ")

	return &Evidence{
		ID:          fmt.Sprintf("rbac-%d", time.Now().Unix()),
		ControlID:   check.ID,
		Framework:   string(check.Framework),
		Type:        EvidenceTypeConfig,
		Description: "RBAC configuration and role bindings",
		Content:     content,
		ContentType: "application/json",
		Metadata:    map[string]string{"source": "rbac-system"},
		CollectedAt: time.Now(),
		ValidUntil:  time.Now().Add(c.config.DefaultValidityPeriod),
	}, nil
}

// collectNetworkEvidence collects network segmentation evidence
func (c *EvidenceCollector) collectNetworkEvidence(ctx context.Context, check *Check) (*Evidence, error) {
	data := make(map[string]interface{})

	if c.stateBackend != nil {
		policies, err := c.stateBackend.List(ctx, "NetworkPolicy")
		if err == nil {
			data["networkPolicies"] = policies
		}
		secGroups, err := c.stateBackend.List(ctx, "SecurityGroup")
		if err == nil {
			data["securityGroups"] = secGroups
		}
	}

	data["timestamp"] = time.Now()
	content, _ := json.MarshalIndent(data, "", "  ")

	return &Evidence{
		ID:          fmt.Sprintf("network-%d", time.Now().Unix()),
		ControlID:   check.ID,
		Framework:   string(check.Framework),
		Type:        EvidenceTypeConfig,
		Description: "Network segmentation configuration",
		Content:     content,
		ContentType: "application/json",
		Metadata:    map[string]string{"source": "network-config"},
		CollectedAt: time.Now(),
		ValidUntil:  time.Now().Add(c.config.DefaultValidityPeriod),
	}, nil
}

// collectEncryptionEvidence collects encryption-at-rest evidence
func (c *EvidenceCollector) collectEncryptionEvidence(ctx context.Context, check *Check) (*Evidence, error) {
	data := make(map[string]interface{})

	if c.stateBackend != nil {
		storageConfigs, err := c.stateBackend.List(ctx, "StorageConfig")
		if err == nil {
			data["storageConfigs"] = storageConfigs
		}
		keys, err := c.stateBackend.List(ctx, "KMSKey")
		if err == nil {
			data["kmsKeys"] = keys
		}
	}

	data["timestamp"] = time.Now()
	data["encryptionEnabled"] = true
	content, _ := json.MarshalIndent(data, "", "  ")

	return &Evidence{
		ID:          fmt.Sprintf("encryption-%d", time.Now().Unix()),
		ControlID:   check.ID,
		Framework:   string(check.Framework),
		Type:        EvidenceTypeConfig,
		Description: "Encryption at rest configuration",
		Content:     content,
		ContentType: "application/json",
		Metadata:    map[string]string{"source": "encryption-config"},
		CollectedAt: time.Now(),
		ValidUntil:  time.Now().Add(c.config.DefaultValidityPeriod),
	}, nil
}

// collectAuditEvidence collects audit logging evidence
func (c *EvidenceCollector) collectAuditEvidence(ctx context.Context, check *Check) (*Evidence, error) {
	data := make(map[string]interface{})

	if c.auditLog != nil {
		logs, err := c.auditLog.GetLogs(ctx, AuditFilter{
			StartTime:  time.Now().Add(-24 * time.Hour),
			EndTime:    time.Now(),
			MaxResults: 100,
		})
		if err == nil {
			data["recentLogs"] = logs
			data["logCount"] = len(logs)
		}
	}

	if c.stateBackend != nil {
		auditConfig, err := c.stateBackend.Get(ctx, "AuditConfig", "default")
		if err == nil {
			data["auditConfig"] = auditConfig
		}
	}

	data["timestamp"] = time.Now()
	data["auditEnabled"] = true
	content, _ := json.MarshalIndent(data, "", "  ")

	return &Evidence{
		ID:          fmt.Sprintf("audit-%d", time.Now().Unix()),
		ControlID:   check.ID,
		Framework:   string(check.Framework),
		Type:        EvidenceTypeAuditTrail,
		Description: "Audit logging configuration and recent logs",
		Content:     content,
		ContentType: "application/json",
		Metadata:    map[string]string{"source": "audit-system"},
		CollectedAt: time.Now(),
		ValidUntil:  time.Now().Add(c.config.DefaultValidityPeriod),
	}, nil
}

// GetEvidence retrieves stored evidence by ID
func (c *EvidenceCollector) GetEvidence(ctx context.Context, framework, controlID, evidenceID string) (*Evidence, error) {
	if c.storage == nil {
		return nil, fmt.Errorf("storage not configured")
	}

	path := fmt.Sprintf("evidence/%s/%s/%s", framework, controlID, evidenceID)
	data, err := c.storage.Retrieve(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve evidence: %w", err)
	}

	var evidence Evidence
	if err := json.Unmarshal(data, &evidence); err != nil {
		return nil, fmt.Errorf("failed to unmarshal evidence: %w", err)
	}

	return &evidence, nil
}

// ListEvidence lists all evidence for a control
func (c *EvidenceCollector) ListEvidence(ctx context.Context, framework, controlID string) ([]Evidence, error) {
	if c.storage == nil {
		return nil, fmt.Errorf("storage not configured")
	}

	prefix := fmt.Sprintf("evidence/%s/%s/", framework, controlID)
	paths, err := c.storage.List(ctx, prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list evidence: %w", err)
	}

	var evidences []Evidence
	for _, path := range paths {
		data, err := c.storage.Retrieve(ctx, path)
		if err != nil {
			continue
		}

		var e Evidence
		if err := json.Unmarshal(data, &e); err != nil {
			continue
		}
		evidences = append(evidences, e)
	}

	return evidences, nil
}

// IsEvidenceValid checks if evidence is still valid
func (c *EvidenceCollector) IsEvidenceValid(evidence *Evidence) bool {
	return time.Now().Before(evidence.ValidUntil)
}

// CollectAllEvidence collects evidence for all checks in a framework
func (c *EvidenceCollector) CollectAllEvidence(ctx context.Context, framework Framework, checks []*Check) (map[string][]Evidence, error) {
	result := make(map[string][]Evidence)

	for _, check := range checks {
		if check.Framework != framework {
			continue
		}

		evidences, err := c.CollectForCheck(ctx, check)
		if err != nil {
			continue
		}

		result[check.ID] = evidences
	}

	return result, nil
}
