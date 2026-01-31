package state

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// DynamoLock implements distributed locking using DynamoDB
type DynamoLock struct {
	client     *dynamodb.Client
	tableName  string
	lockKey    string
	sessionID  string
	ttl        time.Duration
	heartbeat  *time.Ticker
	stop       chan struct{}
}

// DynamoLockConfig represents DynamoDB lock configuration
type DynamoLockConfig struct {
	TableName string        `yaml:"tableName" json:"tableName"`
	Region    string        `yaml:"region" json:"region"`
	TTL       time.Duration `yaml:"ttl" json:"ttl"`
}

// lockItem represents a lock entry in DynamoDB
type lockItem struct {
	LockKey   string `dynamodbav:"lockKey"`
	SessionID string `dynamodbav:"sessionID"`
	TTL       int64  `dynamodbav:"ttl"` // Unix timestamp
}

// NewDynamoLock creates a new DynamoDB-based distributed lock
func NewDynamoLock(cfg *DynamoLockConfig, key string) (*DynamoLock, error) {
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := dynamodb.NewFromConfig(awsCfg)

	// Generate unique session ID
	sessionID := fmt.Sprintf("%s-%d", key, time.Now().UnixNano())

	return &DynamoLock{
		client:    client,
		tableName: cfg.TableName,
		lockKey:   key,
		sessionID: sessionID,
		ttl:       cfg.TTL,
		stop:      make(chan struct{}),
	}, nil
}

// Acquire attempts to acquire the distributed lock
func (l *DynamoLock) Acquire(ctx context.Context) error {
	item := lockItem{
		LockKey:   l.lockKey,
		SessionID: l.sessionID,
		TTL:       time.Now().Add(l.ttl).Unix(),
	}

	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("failed to marshal lock item: %w", err)
	}

	// Try to put item with condition: lockKey does not exist or TTL expired
	_, err = l.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(l.tableName),
		Item:      av,
		ConditionExpression: aws.String(
			"attribute_not_exists(lockKey) OR #ttl < :now",
		),
		ExpressionAttributeNames: map[string]string{
			"#ttl": "ttl",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":now": &types.AttributeValueMemberN{
				Value: fmt.Sprintf("%d", time.Now().Unix()),
			},
		},
	})

	if err != nil {
		// Check if it's a conditional check failure (lock already held)
		var condErr *types.ConditionalCheckFailedException
		if ok := err.(*types.ConditionalCheckFailedException); ok == condErr {
			return fmt.Errorf("lock already held by another process")
		}
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	// Start heartbeat to keep lock alive
	l.startHeartbeat(ctx)

	return nil
}

// Release releases the distributed lock
func (l *DynamoLock) Release(ctx context.Context) error {
	// Stop heartbeat
	if l.heartbeat != nil {
		l.heartbeat.Stop()
		close(l.stop)
	}

	// Delete item only if sessionID matches (we own it)
	_, err := l.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(l.tableName),
		Key: map[string]types.AttributeValue{
			"lockKey": &types.AttributeValueMemberS{Value: l.lockKey},
		},
		ConditionExpression: aws.String("sessionID = :sessionID"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":sessionID": &types.AttributeValueMemberS{Value: l.sessionID},
		},
	})

	if err != nil {
		var condErr *types.ConditionalCheckFailedException
		if ok := err.(*types.ConditionalCheckFailedException); ok == condErr {
			return fmt.Errorf("lock not owned by this instance")
		}
		return fmt.Errorf("failed to release lock: %w", err)
	}

	return nil
}

// startHeartbeat starts a background goroutine to refresh the lock TTL
func (l *DynamoLock) startHeartbeat(ctx context.Context) {
	heartbeatInterval := l.ttl / 3 // Refresh at 1/3 of TTL

	l.heartbeat = time.NewTicker(heartbeatInterval)

	go func() {
		for {
			select {
			case <-l.heartbeat.C:
				// Update TTL only if we still own the lock
				_, err := l.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
					TableName: aws.String(l.tableName),
					Key: map[string]types.AttributeValue{
						"lockKey": &types.AttributeValueMemberS{Value: l.lockKey},
					},
					UpdateExpression: aws.String("SET #ttl = :newTTL"),
					ConditionExpression: aws.String("sessionID = :sessionID"),
					ExpressionAttributeNames: map[string]string{
						"#ttl": "ttl",
					},
					ExpressionAttributeValues: map[string]types.AttributeValue{
						":newTTL": &types.AttributeValueMemberN{
							Value: fmt.Sprintf("%d", time.Now().Add(l.ttl).Unix()),
						},
						":sessionID": &types.AttributeValueMemberS{Value: l.sessionID},
					},
				})

				if err != nil {
					// Lock lost, stop heartbeat
					l.heartbeat.Stop()
					return
				}
			case <-l.stop:
				return
			}
		}
	}()
}

// Close stops the heartbeat
func (l *DynamoLock) Close() error {
	if l.heartbeat != nil {
		l.heartbeat.Stop()
		close(l.stop)
	}
	return nil
}
