package secrets

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Manager defines the interface for secret management
type Manager interface {
	// GetSecret retrieves a secret by path
	GetSecret(ctx context.Context, path string) (*Secret, error)

	// PutSecret stores a secret at the given path
	PutSecret(ctx context.Context, path string, data map[string]string) error

	// DeleteSecret removes a secret
	DeleteSecret(ctx context.Context, path string) error

	// ListSecrets lists all secret paths
	ListSecrets(ctx context.Context, prefix string) ([]string, error)

	// Close closes the manager and any connections
	Close() error
}

// Secret represents a secret value
type Secret struct {
	Path      string            `json:"path"`
	Data      map[string]string `json:"data"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Version   int               `json:"version,omitempty"`
	CreatedAt time.Time         `json:"createdAt,omitempty"`
	UpdatedAt time.Time         `json:"updatedAt,omitempty"`
}

// Config represents secrets manager configuration
type Config struct {
	// Provider specifies the secrets backend (vault, aws, local)
	Provider string `yaml:"provider" json:"provider"`

	// Vault configuration
	Vault *VaultConfig `yaml:"vault,omitempty" json:"vault,omitempty"`

	// AWS Secrets Manager configuration
	AWS *AWSConfig `yaml:"aws,omitempty" json:"aws,omitempty"`

	// Local encrypted storage configuration
	Local *LocalConfig `yaml:"local,omitempty" json:"local,omitempty"`
}

// VaultConfig represents HashiCorp Vault configuration
type VaultConfig struct {
	Address   string `yaml:"address" json:"address"`
	Token     string `yaml:"token,omitempty" json:"token,omitempty"`
	TokenFile string `yaml:"tokenFile,omitempty" json:"tokenFile,omitempty"`
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	Mount     string `yaml:"mount,omitempty" json:"mount,omitempty"` // KV mount path, default: "secret"
}

// AWSConfig represents AWS Secrets Manager configuration
type AWSConfig struct {
	Region    string `yaml:"region" json:"region"`
	AccessKey string `yaml:"accessKey,omitempty" json:"accessKey,omitempty"`
	SecretKey string `yaml:"secretKey,omitempty" json:"secretKey,omitempty"`
	Profile   string `yaml:"profile,omitempty" json:"profile,omitempty"`
}

// LocalConfig represents local encrypted secrets configuration
type LocalConfig struct {
	Path          string `yaml:"path,omitempty" json:"path,omitempty"`
	EncryptionKey string `yaml:"encryptionKey,omitempty" json:"encryptionKey,omitempty"`
}

// NewManager creates a new secrets manager based on configuration
func NewManager(config *Config) (Manager, error) {
	if config == nil {
		return nil, fmt.Errorf("secrets configuration is required")
	}

	switch config.Provider {
	case "vault":
		if config.Vault == nil {
			return nil, fmt.Errorf("vault configuration is required")
		}
		return NewVaultManager(config.Vault)

	case "aws":
		if config.AWS == nil {
			return nil, fmt.Errorf("AWS configuration is required")
		}
		return NewAWSManager(config.AWS)

	case "local":
		localConfig := config.Local
		if localConfig == nil {
			localConfig = &LocalConfig{}
		}
		return NewLocalManager(localConfig)

	default:
		return nil, fmt.Errorf("unsupported secrets provider: %s", config.Provider)
	}
}

// SecretReference represents a reference to a secret in YAML
// Format: ${secret:provider:path:key}
// Examples:
//   ${secret:vault:database/prod:password}
//   ${secret:aws:prod/db:username}
//   ${secret:local:api:token}
type SecretReference struct {
	Provider string
	Path     string
	Key      string
	Raw      string
}

var secretRefRegex = regexp.MustCompile(`\$\{secret:([^:]+):([^:]+)(?::([^}]+))?\}`)

// ParseSecretReference parses a secret reference string
func ParseSecretReference(ref string) (*SecretReference, error) {
	matches := secretRefRegex.FindStringSubmatch(ref)
	if matches == nil {
		return nil, fmt.Errorf("invalid secret reference format: %s", ref)
	}

	provider := matches[1]
	path := matches[2]
	key := "value" // Default key
	if len(matches) > 3 && matches[3] != "" {
		key = matches[3]
	}

	return &SecretReference{
		Provider: provider,
		Path:     path,
		Key:      key,
		Raw:      ref,
	}, nil
}

// FindSecretReferences finds all secret references in a string
func FindSecretReferences(text string) []*SecretReference {
	matches := secretRefRegex.FindAllStringSubmatch(text, -1)
	refs := make([]*SecretReference, 0, len(matches))

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		provider := match[1]
		path := match[2]
		key := "value"
		if len(match) > 3 && match[3] != "" {
			key = match[3]
		}

		refs = append(refs, &SecretReference{
			Provider: provider,
			Path:     path,
			Key:      key,
			Raw:      match[0],
		})
	}

	return refs
}

// ResolveSecretReferences resolves secret references in a string
func ResolveSecretReferences(ctx context.Context, text string, manager Manager) (string, error) {
	refs := FindSecretReferences(text)
	if len(refs) == 0 {
		return text, nil
	}

	result := text
	for _, ref := range refs {
		secret, err := manager.GetSecret(ctx, ref.Path)
		if err != nil {
			return "", fmt.Errorf("failed to resolve secret %s: %w", ref.Raw, err)
		}

		value, ok := secret.Data[ref.Key]
		if !ok {
			return "", fmt.Errorf("secret %s does not contain key %s", ref.Path, ref.Key)
		}

		result = strings.ReplaceAll(result, ref.Raw, value)
	}

	return result, nil
}

// ResolveSecretReferencesInMap resolves secret references in a map recursively
func ResolveSecretReferencesInMap(ctx context.Context, data map[string]interface{}, manager Manager) error {
	for key, value := range data {
		switch v := value.(type) {
		case string:
			resolved, err := ResolveSecretReferences(ctx, v, manager)
			if err != nil {
				return err
			}
			data[key] = resolved

		case map[string]interface{}:
			if err := ResolveSecretReferencesInMap(ctx, v, manager); err != nil {
				return err
			}

		case []interface{}:
			for i, item := range v {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if err := ResolveSecretReferencesInMap(ctx, itemMap, manager); err != nil {
						return err
					}
				} else if itemStr, ok := item.(string); ok {
					resolved, err := ResolveSecretReferences(ctx, itemStr, manager)
					if err != nil {
						return err
					}
					v[i] = resolved
				}
			}
		}
	}

	return nil
}
