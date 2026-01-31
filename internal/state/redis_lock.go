package state

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisLock implements distributed locking using Redis
type RedisLock struct {
	client    *redis.Client
	key       string
	value     string
	ttl       time.Duration
	heartbeat *time.Ticker
	stop      chan struct{}
}

// RedisLockConfig represents Redis lock configuration
type RedisLockConfig struct {
	Address  string        `yaml:"address" json:"address"`
	Password string        `yaml:"password" json:"password"`
	DB       int           `yaml:"db" json:"db"`
	TTL      time.Duration `yaml:"ttl" json:"ttl"`
}

// NewRedisLock creates a new Redis-based distributed lock
func NewRedisLock(cfg *RedisLockConfig, key string) (*RedisLock, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// Generate unique value for this lock instance
	value := fmt.Sprintf("%s-%d", key, time.Now().UnixNano())

	return &RedisLock{
		client: client,
		key:    fmt.Sprintf("platformfoundry:lock:%s", key),
		value:  value,
		ttl:    cfg.TTL,
		stop:   make(chan struct{}),
	}, nil
}

// Acquire attempts to acquire the distributed lock
func (l *RedisLock) Acquire(ctx context.Context) error {
	// Try to set the key with NX (only if not exists) and EX (expiration)
	success, err := l.client.SetNX(ctx, l.key, l.value, l.ttl).Result()
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	if !success {
		return fmt.Errorf("lock already held by another process")
	}

	// Start heartbeat to keep lock alive
	l.startHeartbeat()

	return nil
}

// Release releases the distributed lock
func (l *RedisLock) Release(ctx context.Context) error {
	// Stop heartbeat
	if l.heartbeat != nil {
		l.heartbeat.Stop()
		close(l.stop)
	}

	// Use Lua script to atomically check and delete only if we own the lock
	luaScript := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`

	result, err := l.client.Eval(ctx, luaScript, []string{l.key}, l.value).Result()
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	if result == int64(0) {
		return fmt.Errorf("lock not owned by this instance")
	}

	return nil
}

// startHeartbeat starts a background goroutine to refresh the lock TTL
func (l *RedisLock) startHeartbeat() {
	heartbeatInterval := l.ttl / 3 // Refresh at 1/3 of TTL

	l.heartbeat = time.NewTicker(heartbeatInterval)

	go func() {
		for {
			select {
			case <-l.heartbeat.C:
				ctx := context.Background()
				// Extend TTL only if we still own the lock
				luaScript := `
					if redis.call("get", KEYS[1]) == ARGV[1] then
						return redis.call("expire", KEYS[1], ARGV[2])
					else
						return 0
					end
				`
				_, err := l.client.Eval(ctx, luaScript, []string{l.key}, l.value, int(l.ttl.Seconds())).Result()
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

// Close closes the Redis connection
func (l *RedisLock) Close() error {
	if l.heartbeat != nil {
		l.heartbeat.Stop()
		close(l.stop)
	}
	return l.client.Close()
}
