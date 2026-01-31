// Package pse defines the Platform Secrets Engine (PSE).
// This is inspired by Vault secrets engines and handles secrets management.
package pse

import (
	"context"
	"time"
)

// SecretsEngine is the main interface for secrets management.
// Implementations handle storage, retrieval, and generation of secrets.
type SecretsEngine interface {
	// Type returns the type of this secrets engine
	Type() string

	// Secret Operations
	Get(ctx context.Context, path string) (*Secret, error)
	Put(ctx context.Context, path string, data map[string]interface{}) error
	Delete(ctx context.Context, path string) error
	List(ctx context.Context, prefix string) ([]string, error)

	// Dynamic Secrets (optional, return ErrNotSupported if not implemented)
	GenerateCredentials(ctx context.Context, role string) (*Credentials, error)
	RevokeCredentials(ctx context.Context, leaseID string) error
	RenewCredentials(ctx context.Context, leaseID string, increment time.Duration) (*Credentials, error)

	// Close releases engine resources
	Close() error
}

// Secret represents a retrieved secret
type Secret struct {
	// Data contains the secret data
	Data map[string]interface{}

	// Metadata contains secret metadata
	Metadata SecretMetadata

	// LeaseID is set for dynamic secrets
	LeaseID string

	// LeaseDuration is how long until the secret expires
	LeaseDuration time.Duration

	// Renewable indicates if the lease can be renewed
	Renewable bool
}

// SecretMetadata contains metadata about a secret
type SecretMetadata struct {
	// CreatedAt is when the secret was created
	CreatedAt time.Time

	// UpdatedAt is when the secret was last updated
	UpdatedAt time.Time

	// Version is the secret version
	Version int64

	// CustomMetadata contains user-defined metadata
	CustomMetadata map[string]string

	// DeletedAt is set if the secret is soft-deleted
	DeletedAt *time.Time
}

// Credentials represents dynamically generated credentials
type Credentials struct {
	// Username is the generated username
	Username string

	// Password is the generated password
	Password string

	// LeaseID is the lease identifier
	LeaseID string

	// LeaseDuration is how long until the credentials expire
	LeaseDuration time.Duration

	// Renewable indicates if the lease can be renewed
	Renewable bool

	// Extra contains additional credential data
	Extra map[string]interface{}
}

// EngineConfig holds configuration for a secrets engine
type EngineConfig struct {
	// Type is the engine type (e.g., "vault", "aws", "local")
	Type string

	// MountPath is the path where this engine is mounted
	MountPath string

	// Config contains engine-specific configuration
	Config map[string]interface{}
}

// EngineMetadata contains metadata about a secrets engine
type EngineMetadata struct {
	// Type is the engine type
	Type string

	// Description describes this engine
	Description string

	// Version is the engine version
	Version string

	// SupportsDynamicSecrets indicates if the engine supports dynamic secrets
	SupportsDynamicSecrets bool
}

// Common errors for secrets operations
var (
	// ErrSecretNotFound indicates the secret doesn't exist
	ErrSecretNotFound = secretsError("secret not found")

	// ErrPermissionDenied indicates insufficient permissions
	ErrPermissionDenied = secretsError("permission denied")

	// ErrNotSupported indicates the operation is not supported
	ErrNotSupported = secretsError("operation not supported")

	// ErrLeaseNotFound indicates the lease doesn't exist
	ErrLeaseNotFound = secretsError("lease not found")

	// ErrLeaseExpired indicates the lease has expired
	ErrLeaseExpired = secretsError("lease expired")

	// ErrInvalidPath indicates an invalid secret path
	ErrInvalidPath = secretsError("invalid path")
)

type secretsError string

func (e secretsError) Error() string {
	return string(e)
}

// SecretReference is a reference to a secret that can be resolved later
type SecretReference struct {
	// Engine is the secrets engine to use
	Engine string

	// Path is the path to the secret
	Path string

	// Key is the specific key within the secret (optional)
	Key string

	// Version is a specific version to retrieve (optional)
	Version int64
}

// Resolve resolves the secret reference using the provided engine
func (r *SecretReference) Resolve(ctx context.Context, engine SecretsEngine) (interface{}, error) {
	secret, err := engine.Get(ctx, r.Path)
	if err != nil {
		return nil, err
	}
	if r.Key != "" {
		return secret.Data[r.Key], nil
	}
	return secret.Data, nil
}
