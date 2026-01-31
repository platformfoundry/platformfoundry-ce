package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/platformfoundry/platformfoundry-ce/internal/auth"
	"github.com/platformfoundry/platformfoundry-ce/internal/secrets"
	"github.com/platformfoundry/platformfoundry-ce/internal/state"
	"github.com/platformfoundry/platformfoundry-ce/internal/tls"
	"gopkg.in/yaml.v3"
)

// SecurityConfig represents the complete security configuration
type SecurityConfig struct {
	Auth       AuthConfig       `yaml:"auth"`
	TLS        TLSConfig        `yaml:"tls"`
	Secrets    SecretsConfig    `yaml:"secrets"`
	State      StateConfig      `yaml:"state"`
	Server     ServerConfig     `yaml:"server"`
	Audit      AuditConfig      `yaml:"audit"`
	Security   SecurityHeaders  `yaml:"security"`
	RBAC       RBACConfig       `yaml:"rbac"`
	Compliance ComplianceConfig `yaml:"compliance"`
	Monitoring MonitoringConfig `yaml:"monitoring"`
	Backup     BackupConfig     `yaml:"backup"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	Enabled bool         `yaml:"enabled"`
	JWT     JWTConfig    `yaml:"jwt"`
	APIKeys APIKeyConfig `yaml:"apiKeys"`
	OAuth   OAuthConfig  `yaml:"oauth"`
	SAML    SAMLConfig   `yaml:"saml"`
}

// JWTConfig represents JWT configuration
type JWTConfig struct {
	SecretKey        string `yaml:"secretKey"`
	Issuer           string `yaml:"issuer"`
	Expiration       string `yaml:"expiration"`
	SigningAlgorithm string `yaml:"signingAlgorithm"`
	PrivateKeyPath   string `yaml:"privateKeyPath"`
	PublicKeyPath    string `yaml:"publicKeyPath"`
}

// APIKeyConfig represents API key configuration
type APIKeyConfig struct {
	Enabled           bool   `yaml:"enabled"`
	DefaultExpiration string `yaml:"defaultExpiration"`
}

// OAuthConfig represents OAuth configuration
type OAuthConfig struct {
	Enabled   bool                       `yaml:"enabled"`
	Providers map[string]OAuthProvider   `yaml:"providers"`
}

// OAuthProvider represents an OAuth provider configuration
type OAuthProvider struct {
	Enabled      bool   `yaml:"enabled"`
	ClientID     string `yaml:"clientID"`
	ClientSecret string `yaml:"clientSecret"`
	RedirectURL  string `yaml:"redirectURL"`
	Domain       string `yaml:"domain,omitempty"` // For Okta
}

// SAMLConfig represents SAML configuration
type SAMLConfig struct {
	Enabled                      bool   `yaml:"enabled"`
	EntityID                     string `yaml:"entityID"`
	AssertionConsumerServiceURL  string `yaml:"assertionConsumerServiceURL"`
	SingleLogoutServiceURL       string `yaml:"singleLogoutServiceURL"`
	IDPMetadataURL               string `yaml:"idpMetadataURL"`
}

// TLSConfig represents TLS configuration
type TLSConfig struct {
	Enabled      bool                    `yaml:"enabled"`
	Source       string                  `yaml:"source"`
	Manual       tls.Config              `yaml:"manual"`
	ACME         ACMEConfig              `yaml:"acme"`
	SelfSigned   SelfSignedConfig        `yaml:"selfsigned"`
	MinVersion   string                  `yaml:"minVersion"`
	CipherSuites []string                `yaml:"cipherSuites"`
}

// ACMEConfig represents ACME configuration
type ACMEConfig struct {
	Email         string   `yaml:"email"`
	Domains       []string `yaml:"domains"`
	Staging       bool     `yaml:"staging"`
	RenewBefore   string   `yaml:"renewBefore"`
	ChallengeType string   `yaml:"challengeType"`
	DNSProvider   string   `yaml:"dnsProvider"`
}

// SelfSignedConfig represents self-signed certificate configuration
type SelfSignedConfig struct {
	Organization string   `yaml:"organization"`
	ValidFor     string   `yaml:"validFor"`
	DNSNames     []string `yaml:"dnsNames"`
	IPAddresses  []string `yaml:"ipAddresses"`
}

// SecretsConfig represents secrets configuration
type SecretsConfig struct {
	Provider string              `yaml:"provider"`
	Local    secrets.LocalConfig `yaml:"local"`
	Vault    secrets.VaultConfig `yaml:"vault"`
	AWS      secrets.AWSConfig   `yaml:"aws"`
}

// StateConfig represents state backend configuration
type StateConfig struct {
	Backend string          `yaml:"backend"`
	Local   LocalStateConfig `yaml:"local"`
	S3      S3StateConfig    `yaml:"s3"`
}

// LocalStateConfig represents local state configuration
type LocalStateConfig struct {
	Path string `yaml:"path"`
}

// S3StateConfig represents S3 state configuration
type S3StateConfig struct {
	Bucket          string `yaml:"bucket"`
	Prefix          string `yaml:"prefix"`
	Region          string `yaml:"region"`
	TableName       string `yaml:"tableName"`
	Encryption      bool   `yaml:"encryption"`
	EncryptionKeyID string `yaml:"encryptionKeyID"`
	Versioning      bool   `yaml:"versioning"`
}

// ServerConfig represents web server configuration
type ServerConfig struct {
	Address       string     `yaml:"address"`
	Port          int        `yaml:"port"`
	HTTPSPort     int        `yaml:"httpsPort"`
	RedirectHTTPS bool       `yaml:"redirectHTTPS"`
	RequireAuth   bool       `yaml:"requireAuth"`
	CORS          CORSConfig `yaml:"cors"`
	RateLimit     RateLimitConfig `yaml:"rateLimit"`
	Timeout       string     `yaml:"timeout"`
	MaxRequestSize string    `yaml:"maxRequestSize"`
}

// CORSConfig represents CORS configuration
type CORSConfig struct {
	Enabled          bool     `yaml:"enabled"`
	AllowedOrigins   []string `yaml:"allowedOrigins"`
	AllowedMethods   []string `yaml:"allowedMethods"`
	AllowedHeaders   []string `yaml:"allowedHeaders"`
	AllowCredentials bool     `yaml:"allowCredentials"`
}

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	Enabled            bool `yaml:"enabled"`
	RequestsPerMinute  int  `yaml:"requestsPerMinute"`
	Burst              int  `yaml:"burst"`
}

// AuditConfig represents audit logging configuration
type AuditConfig struct {
	Enabled       bool            `yaml:"enabled"`
	Destination   string          `yaml:"destination"`
	File          AuditFileConfig `yaml:"file"`
	LogAuth       bool            `yaml:"logAuth"`
	LogChanges    bool            `yaml:"logChanges"`
	LogAPIAccess  bool            `yaml:"logAPIAccess"`
	MaskSecrets   bool            `yaml:"maskSecrets"`
}

// AuditFileConfig represents audit file configuration
type AuditFileConfig struct {
	Path       string `yaml:"path"`
	MaxSize    string `yaml:"maxSize"`
	MaxBackups int    `yaml:"maxBackups"`
	MaxAge     string `yaml:"maxAge"`
	Compress   bool   `yaml:"compress"`
}

// SecurityHeaders represents security headers configuration
type SecurityHeaders struct {
	Headers HeadersConfig `yaml:"headers"`
}

// HeadersConfig represents HTTP security headers
type HeadersConfig struct {
	HSTS               HSTSConfig `yaml:"hsts"`
	CSP                CSPConfig  `yaml:"csp"`
	FrameOptions       string     `yaml:"frameOptions"`
	ContentTypeNoSniff bool       `yaml:"contentTypeNoSniff"`
	XSSProtection      bool       `yaml:"xssProtection"`
	ReferrerPolicy     string     `yaml:"referrerPolicy"`
}

// HSTSConfig represents HSTS configuration
type HSTSConfig struct {
	Enabled           bool   `yaml:"enabled"`
	MaxAge            string `yaml:"maxAge"`
	IncludeSubDomains bool   `yaml:"includeSubDomains"`
	Preload           bool   `yaml:"preload"`
}

// CSPConfig represents Content Security Policy configuration
type CSPConfig struct {
	Enabled bool   `yaml:"enabled"`
	Policy  string `yaml:"policy"`
}

// RBACConfig represents RBAC configuration
type RBACConfig struct {
	Enabled     bool              `yaml:"enabled"`
	DefaultRole string            `yaml:"defaultRole"`
	Roles       map[string]Role   `yaml:"roles"`
}

// Role represents a role definition
type Role struct {
	Description string   `yaml:"description"`
	Permissions []string `yaml:"permissions"`
}

// ComplianceConfig represents compliance configuration
type ComplianceConfig struct {
	Enabled             bool              `yaml:"enabled"`
	Frameworks          []string          `yaml:"frameworks"`
	EncryptionAtRest    bool              `yaml:"encryptionAtRest"`
	EncryptionInTransit bool              `yaml:"encryptionInTransit"`
	DataRetention       DataRetention     `yaml:"dataRetention"`
	RequireMFA          bool              `yaml:"requireMFA"`
}

// DataRetention represents data retention policy
type DataRetention struct {
	AuditLogs     string `yaml:"auditLogs"`
	ResourceState string `yaml:"resourceState"`
	UserActivity  string `yaml:"userActivity"`
}

// MonitoringConfig represents monitoring configuration
type MonitoringConfig struct {
	SecurityEvents SecurityEventsConfig `yaml:"securityEvents"`
	Certificates   CertMonitoringConfig `yaml:"certificates"`
	SecretRotation SecretRotationConfig `yaml:"secretRotation"`
}

// SecurityEventsConfig represents security event monitoring
type SecurityEventsConfig struct {
	Enabled                    bool   `yaml:"enabled"`
	FailedLoginThreshold       int    `yaml:"failedLoginThreshold"`
	FailedLoginWindow          string `yaml:"failedLoginWindow"`
	UnauthorizedAccessAlert    bool   `yaml:"unauthorizedAccessAlert"`
	PrivilegeEscalationAlert   bool   `yaml:"privilegeEscalationAlert"`
}

// CertMonitoringConfig represents certificate monitoring
type CertMonitoringConfig struct {
	Enabled           bool   `yaml:"enabled"`
	ExpirationWarning string `yaml:"expirationWarning"`
}

// SecretRotationConfig represents secret rotation monitoring
type SecretRotationConfig struct {
	Enabled          bool   `yaml:"enabled"`
	RotationInterval string `yaml:"rotationInterval"`
}

// BackupConfig represents backup configuration
type BackupConfig struct {
	Enabled     bool              `yaml:"enabled"`
	Schedule    string            `yaml:"schedule"`
	Destination BackupDestination `yaml:"destination"`
	Retention   BackupRetention   `yaml:"retention"`
	Include     []string          `yaml:"include"`
}

// BackupDestination represents backup destination
type BackupDestination struct {
	Type       string `yaml:"type"`
	Bucket     string `yaml:"bucket"`
	Region     string `yaml:"region"`
	Encryption bool   `yaml:"encryption"`
}

// BackupRetention represents backup retention policy
type BackupRetention struct {
	Daily   int `yaml:"daily"`
	Weekly  int `yaml:"weekly"`
	Monthly int `yaml:"monthly"`
}

// LoadSecurityConfig loads security configuration from file
func LoadSecurityConfig(path string) (*SecurityConfig, error) {
	// Expand environment variables in path
	path = os.ExpandEnv(path)

	// Read configuration file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read security config: %w", err)
	}

	// Parse YAML with environment variable expansion
	expanded := os.ExpandEnv(string(data))

	var config SecurityConfig
	if err := yaml.Unmarshal([]byte(expanded), &config); err != nil {
		return nil, fmt.Errorf("failed to parse security config: %w", err)
	}

	// Set defaults
	config.setDefaults()

	return &config, nil
}

// LoadSecurityConfigOrDefault loads security config or returns defaults
func LoadSecurityConfigOrDefault() (*SecurityConfig, error) {
	// Try to load from default locations
	paths := []string{
		"config/security.yaml",
		"/etc/platformfoundry/security.yaml",
		filepath.Join(os.Getenv("HOME"), ".platformfoundry", "security.yaml"),
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return LoadSecurityConfig(path)
		}
	}

	// Return default configuration
	config := &SecurityConfig{}
	config.setDefaults()
	return config, nil
}

// setDefaults sets default values for security configuration
func (c *SecurityConfig) setDefaults() {
	// Auth defaults
	if c.Auth.JWT.Issuer == "" {
		c.Auth.JWT.Issuer = "platformfoundry.io"
	}
	if c.Auth.JWT.Expiration == "" {
		c.Auth.JWT.Expiration = "24h"
	}
	if c.Auth.JWT.SigningAlgorithm == "" {
		c.Auth.JWT.SigningAlgorithm = "HS256"
	}

	// Server defaults
	if c.Server.Address == "" {
		c.Server.Address = "0.0.0.0"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.HTTPSPort == 0 {
		c.Server.HTTPSPort = 8443
	}
	if c.Server.Timeout == "" {
		c.Server.Timeout = "30s"
	}

	// State backend defaults
	if c.State.Backend == "" {
		c.State.Backend = "local"
	}
	if c.State.Local.Path == "" {
		c.State.Local.Path = filepath.Join(os.Getenv("HOME"), ".platformfoundry", "state")
	}

	// Secrets defaults
	if c.Secrets.Provider == "" {
		c.Secrets.Provider = "local"
	}

	// RBAC defaults
	if c.RBAC.DefaultRole == "" {
		c.RBAC.DefaultRole = "developer"
	}
}

// GetJWTExpiration returns JWT expiration duration
func (c *SecurityConfig) GetJWTExpiration() time.Duration {
	duration, err := time.ParseDuration(c.Auth.JWT.Expiration)
	if err != nil {
		return 24 * time.Hour
	}
	return duration
}

// GetServerTimeout returns server timeout duration
func (c *SecurityConfig) GetServerTimeout() time.Duration {
	duration, err := time.ParseDuration(c.Server.Timeout)
	if err != nil {
		return 30 * time.Second
	}
	return duration
}

// IsAuthEnabled returns whether authentication is enabled
func (c *SecurityConfig) IsAuthEnabled() bool {
	return c.Auth.Enabled
}

// IsTLSEnabled returns whether TLS is enabled
func (c *SecurityConfig) IsTLSEnabled() bool {
	return c.TLS.Enabled
}

// GetSecretsConfig returns secrets configuration for manager initialization
func (c *SecurityConfig) GetSecretsConfig() *secrets.Config {
	return &secrets.Config{
		Provider: c.Secrets.Provider,
		Local:    &c.Secrets.Local,
		Vault:    &c.Secrets.Vault,
		AWS:      &c.Secrets.AWS,
	}
}

// GetS3StateConfig returns S3 state backend configuration
func (c *SecurityConfig) GetS3StateConfig() *state.S3Config {
	return &state.S3Config{
		Bucket:    c.State.S3.Bucket,
		Prefix:    c.State.S3.Prefix,
		Region:    c.State.S3.Region,
		TableName: c.State.S3.TableName,
	}
}

// CreateAuthMiddleware creates authentication middleware from configuration
func (c *SecurityConfig) CreateAuthMiddleware() (*auth.AuthMiddleware, error) {
	if !c.Auth.Enabled {
		return nil, nil
	}

	// Create JWT manager
	jwtManager := auth.NewJWTManager(
		c.Auth.JWT.SecretKey,
		c.Auth.JWT.Issuer,
		c.GetJWTExpiration(),
	)

	// Create user store
	userStore, err := auth.NewUserStore("")
	if err != nil {
		return nil, fmt.Errorf("failed to create user store: %w", err)
	}

	// Create API key store
	apiKeyStore, err := auth.NewAPIKeyStore("")
	if err != nil {
		return nil, fmt.Errorf("failed to create API key store: %w", err)
	}

	// Create middleware
	middleware := auth.NewAuthMiddleware(jwtManager, apiKeyStore, userStore, c.Auth.Enabled)

	return middleware, nil
}
