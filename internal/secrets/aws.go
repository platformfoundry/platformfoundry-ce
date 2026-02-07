package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// AWSManager implements Manager using AWS Secrets Manager
type AWSManager struct {
	client *secretsmanager.Client
	region string
}

// NewAWSManager creates a new AWS Secrets Manager
func NewAWSManager(cfg *AWSConfig) (*AWSManager, error) {
	// Build AWS config options
	var configOptions []func(*config.LoadOptions) error

	// Set region
	if cfg.Region != "" {
		configOptions = append(configOptions, config.WithRegion(cfg.Region))
	}

	// Set credentials if provided
	if cfg.AccessKey != "" && cfg.SecretKey != "" {
		configOptions = append(configOptions, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		))
	}

	// Set profile if provided
	if cfg.Profile != "" {
		configOptions = append(configOptions, config.WithSharedConfigProfile(cfg.Profile))
	}

	// Load AWS config
	awsConfig, err := config.LoadDefaultConfig(context.Background(), configOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create Secrets Manager client
	client := secretsmanager.NewFromConfig(awsConfig)

	return &AWSManager{
		client: client,
		region: cfg.Region,
	}, nil
}

// GetSecret retrieves a secret by path
func (m *AWSManager) GetSecret(ctx context.Context, path string) (*Secret, error) {
	// Get secret value
	output, err := m.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(path),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get secret from AWS: %w", err)
	}

	// Parse secret string as JSON
	var data map[string]string
	if output.SecretString != nil {
		if err := json.Unmarshal([]byte(*output.SecretString), &data); err != nil {
			// If not JSON, treat as single value
			data = map[string]string{
				"value": *output.SecretString,
			}
		}
	} else if output.SecretBinary != nil {
		// Binary secret
		data = map[string]string{
			"value": string(output.SecretBinary),
		}
	} else {
		return nil, fmt.Errorf("secret has no value")
	}

	// Extract metadata
	metadata := make(map[string]string)
	if output.Name != nil {
		metadata["name"] = *output.Name
	}
	if output.ARN != nil {
		metadata["arn"] = *output.ARN
	}

	// Parse version
	version := 0
	if output.VersionId != nil {
		metadata["version_id"] = *output.VersionId
	}

	secret := &Secret{
		Path:     path,
		Data:     data,
		Metadata: metadata,
		Version:  version,
	}

	// Set timestamps
	if output.CreatedDate != nil {
		secret.CreatedAt = *output.CreatedDate
	}

	return secret, nil
}

// PutSecret stores a secret at the given path
func (m *AWSManager) PutSecret(ctx context.Context, path string, data map[string]string) error {
	// Serialize data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal secret data: %w", err)
	}

	secretString := string(jsonData)

	// Check if secret exists
	_, err = m.client.DescribeSecret(ctx, &secretsmanager.DescribeSecretInput{
		SecretId: aws.String(path),
	})

	if err != nil {
		// Secret doesn't exist, create it
		_, err = m.client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
			Name:         aws.String(path),
			SecretString: aws.String(secretString),
		})
		if err != nil {
			return fmt.Errorf("failed to create secret in AWS: %w", err)
		}
	} else {
		// Secret exists, update it
		_, err = m.client.PutSecretValue(ctx, &secretsmanager.PutSecretValueInput{
			SecretId:     aws.String(path),
			SecretString: aws.String(secretString),
		})
		if err != nil {
			return fmt.Errorf("failed to update secret in AWS: %w", err)
		}
	}

	return nil
}

// DeleteSecret removes a secret
func (m *AWSManager) DeleteSecret(ctx context.Context, path string) error {
	_, err := m.client.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{
		SecretId:                   aws.String(path),
		ForceDeleteWithoutRecovery: aws.Bool(false), // Allow 30-day recovery
	})
	if err != nil {
		return fmt.Errorf("failed to delete secret from AWS: %w", err)
	}

	return nil
}

// ListSecrets lists all secret paths
func (m *AWSManager) ListSecrets(ctx context.Context, prefix string) ([]string, error) {
	var paths []string

	// List secrets with pagination
	paginator := secretsmanager.NewListSecretsPaginator(m.client, &secretsmanager.ListSecretsInput{})

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list secrets from AWS: %w", err)
		}

		for _, secret := range output.SecretList {
			if secret.Name != nil {
				name := *secret.Name
				if prefix == "" || strings.HasPrefix(name, prefix) {
					paths = append(paths, name)
				}
			}
		}
	}

	return paths, nil
}

// Close closes the AWS client
func (m *AWSManager) Close() error {
	// AWS SDK v2 doesn't require explicit closing
	return nil
}
