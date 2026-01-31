package state

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Backend implements Backend using S3 for storage and DynamoDB for locking
type S3Backend struct {
	bucket       string
	prefix       string
	region       string
	tableName    string
	s3Client     *s3.Client
	dynamoClient *dynamodb.Client
}

// S3Config represents S3 backend configuration
type S3Config struct {
	Bucket    string `yaml:"bucket" json:"bucket"`
	Prefix    string `yaml:"prefix" json:"prefix"` // Optional prefix for S3 keys
	Region    string `yaml:"region" json:"region"`
	TableName string `yaml:"tableName" json:"tableName"` // DynamoDB table for locking
}

// NewS3Backend creates a new S3 backend
func NewS3Backend(cfg *S3Config) (*S3Backend, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("bucket is required")
	}
	if cfg.Region == "" {
		return nil, fmt.Errorf("region is required")
	}
	if cfg.TableName == "" {
		return nil, fmt.Errorf("table name is required for locking")
	}

	// Load AWS configuration
	awsConfig, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 and DynamoDB clients
	s3Client := s3.NewFromConfig(awsConfig)
	dynamoClient := dynamodb.NewFromConfig(awsConfig)

	prefix := cfg.Prefix
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}

	return &S3Backend{
		bucket:       cfg.Bucket,
		prefix:       prefix,
		region:       cfg.Region,
		tableName:    cfg.TableName,
		s3Client:     s3Client,
		dynamoClient: dynamoClient,
	}, nil
}

// Save stores a resource in S3 (uses background context)
func (sb *S3Backend) Save(resource *Resource) error {
	return sb.SaveWithContext(context.Background(), resource)
}

// SaveWithContext stores a resource in S3 with context support
func (sb *S3Backend) SaveWithContext(ctx context.Context, resource *Resource) error {
	// Update timestamps
	resource.UpdatedAt = time.Now()
	if resource.CreatedAt.IsZero() {
		resource.CreatedAt = resource.UpdatedAt
	}

	// Increment version
	resource.Version++

	// Marshal resource to JSON
	data, err := json.MarshalIndent(resource, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal resource: %w", err)
	}

	// Save current version
	currentKey := sb.createS3Key(resource.Name)
	_, err = sb.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(sb.bucket),
		Key:         aws.String(currentKey),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return fmt.Errorf("failed to save resource to S3: %w", err)
	}

	// Save versioned copy
	versionKey := sb.createVersionKey(resource.Name, resource.Version)
	_, err = sb.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(sb.bucket),
		Key:         aws.String(versionKey),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return fmt.Errorf("failed to save resource version to S3: %w", err)
	}

	return nil
}

// Get retrieves a resource from S3 (uses background context)
func (sb *S3Backend) Get(name string) (*Resource, error) {
	return sb.GetWithContext(context.Background(), name)
}

// GetWithContext retrieves a resource from S3 with context support
func (sb *S3Backend) GetWithContext(ctx context.Context, name string) (*Resource, error) {
	key := sb.createS3Key(name)
	result, err := sb.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(sb.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get resource from S3: %w", err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read resource data: %w", err)
	}

	var resource Resource
	if err := json.Unmarshal(data, &resource); err != nil {
		return nil, fmt.Errorf("failed to unmarshal resource: %w", err)
	}

	return &resource, nil
}

// List returns all resources from S3 (uses background context)
func (sb *S3Backend) List() ([]*Resource, error) {
	return sb.ListWithContext(context.Background())
}

// ListWithContext returns all resources from S3 with context support
func (sb *S3Backend) ListWithContext(ctx context.Context) ([]*Resource, error) {
	prefix := sb.prefix + "resources/"
	result, err := sb.s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(sb.bucket),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list resources from S3: %w", err)
	}

	var resources []*Resource
	for _, obj := range result.Contents {
		// Skip version objects
		if strings.Contains(*obj.Key, "/versions/") {
			continue
		}

		// Extract resource name from key
		name := strings.TrimPrefix(*obj.Key, prefix)
		name = strings.TrimSuffix(name, ".json")

		// Get resource with context
		resource, err := sb.GetWithContext(ctx, name)
		if err != nil {
			continue // Skip resources that fail to load
		}

		resources = append(resources, resource)
	}

	return resources, nil
}

// Delete removes a resource from S3 (uses background context)
func (sb *S3Backend) Delete(name string) error {
	return sb.DeleteWithContext(context.Background(), name)
}

// DeleteWithContext removes a resource from S3 with context support
func (sb *S3Backend) DeleteWithContext(ctx context.Context, name string) error {
	// Delete current resource
	currentKey := sb.createS3Key(name)
	_, err := sb.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(sb.bucket),
		Key:    aws.String(currentKey),
	})
	if err != nil {
		return fmt.Errorf("failed to delete resource from S3: %w", err)
	}

	// Delete all versions
	versionPrefix := sb.prefix + fmt.Sprintf("resources/%s/versions/", name)
	versions, err := sb.s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(sb.bucket),
		Prefix: aws.String(versionPrefix),
	})
	if err != nil {
		return fmt.Errorf("failed to list versions: %w", err)
	}

	for _, obj := range versions.Contents {
		_, err := sb.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(sb.bucket),
			Key:    obj.Key,
		})
		if err != nil {
			// Log error but continue deleting other versions
			continue
		}
	}

	return nil
}

// Lock acquires a distributed lock using DynamoDB (uses background context)
func (sb *S3Backend) Lock(name string) error {
	return sb.LockWithContext(context.Background(), name)
}

// LockWithContext acquires a distributed lock using DynamoDB with context support
func (sb *S3Backend) LockWithContext(ctx context.Context, name string) error {
	now := time.Now()
	expiresAt := now.Add(5 * time.Minute) // Lock expires after 5 minutes
	owner := fmt.Sprintf("pf-%d", now.Unix())

	lock := Lock{
		ResourceName: name,
		Owner:        owner,
		AcquiredAt:   now,
		ExpiresAt:    expiresAt,
	}

	item, err := attributevalue.MarshalMap(lock)
	if err != nil {
		return fmt.Errorf("failed to marshal lock: %w", err)
	}

	// Use conditional write to ensure atomicity (only write if item doesn't exist)
	condition := expression.AttributeNotExists(expression.Name("resourceName"))
	expr, err := expression.NewBuilder().WithCondition(condition).Build()
	if err != nil {
		return fmt.Errorf("failed to build condition: %w", err)
	}

	_, err = sb.dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:                 aws.String(sb.tableName),
		Item:                      item,
		ConditionExpression:       expr.Condition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		// Check if it's a conditional check failure (lock already exists)
		var ccf *types.ConditionalCheckFailedException
		if err.Error() == ccf.Error() {
			return fmt.Errorf("resource %s is already locked", name)
		}
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	return nil
}

// Unlock releases a distributed lock in DynamoDB (uses background context)
func (sb *S3Backend) Unlock(name string) error {
	return sb.UnlockWithContext(context.Background(), name)
}

// UnlockWithContext releases a distributed lock in DynamoDB with context support
func (sb *S3Backend) UnlockWithContext(ctx context.Context, name string) error {
	_, err := sb.dynamoClient.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(sb.tableName),
		Key: map[string]types.AttributeValue{
			"resourceName": &types.AttributeValueMemberS{Value: name},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	return nil
}

// GetVersion retrieves a specific version from S3 (uses background context)
func (sb *S3Backend) GetVersion(name string, version int) (*Resource, error) {
	return sb.GetVersionWithContext(context.Background(), name, version)
}

// GetVersionWithContext retrieves a specific version from S3 with context support
func (sb *S3Backend) GetVersionWithContext(ctx context.Context, name string, version int) (*Resource, error) {
	key := sb.createVersionKey(name, version)
	result, err := sb.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(sb.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get resource version from S3: %w", err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read resource version data: %w", err)
	}

	var resource Resource
	if err := json.Unmarshal(data, &resource); err != nil {
		return nil, fmt.Errorf("failed to unmarshal resource version: %w", err)
	}

	return &resource, nil
}

// ListVersions returns all versions of a resource from S3 (uses background context)
func (sb *S3Backend) ListVersions(name string) ([]*ResourceVersion, error) {
	return sb.ListVersionsWithContext(context.Background(), name)
}

// ListVersionsWithContext returns all versions of a resource from S3 with context support
func (sb *S3Backend) ListVersionsWithContext(ctx context.Context, name string) ([]*ResourceVersion, error) {
	versionPrefix := sb.prefix + fmt.Sprintf("resources/%s/versions/", name)
	result, err := sb.s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(sb.bucket),
		Prefix: aws.String(versionPrefix),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list versions from S3: %w", err)
	}

	var versions []*ResourceVersion
	for _, obj := range result.Contents {
		// Extract version number from key
		key := *obj.Key
		parts := strings.Split(key, "/")
		if len(parts) < 1 {
			continue
		}

		versionStr := strings.TrimSuffix(parts[len(parts)-1], ".json")
		versionNum, err := strconv.Atoi(versionStr)
		if err != nil {
			continue
		}

		// Get version resource with context
		resource, err := sb.GetVersionWithContext(ctx, name, versionNum)
		if err != nil {
			continue
		}

		versions = append(versions, &ResourceVersion{
			Version:   resource.Version,
			Spec:      resource.Spec,
			Status:    resource.Status,
			CreatedAt: resource.CreatedAt,
		})
	}

	return versions, nil
}

// Close closes the S3 backend connection
func (sb *S3Backend) Close() error {
	// No persistent connection to close for S3/DynamoDB
	return nil
}

// createS3Key returns the S3 key for a resource
func (sb *S3Backend) createS3Key(name string) string {
	return fmt.Sprintf("%sresources/%s.json", sb.prefix, name)
}

// createVersionKey returns the S3 key for a resource version
func (sb *S3Backend) createVersionKey(name string, version int) string {
	return fmt.Sprintf("%sresources/%s/versions/%d.json", sb.prefix, name, version)
}
