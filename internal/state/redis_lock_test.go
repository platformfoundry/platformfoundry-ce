package state

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisLock_AcquireAndRelease(t *testing.T) {
	// Note: This test requires a running Redis instance
	// Skip if Redis is not available
	cfg := &RedisLockConfig{
		Address: "localhost:6379",
		DB:      0,
		TTL:     30 * time.Second,
	}

	lock1, err := NewRedisLock(cfg, "test-lock")
	if err != nil {
		t.Skip("Redis not available, skipping test")
		return
	}
	defer lock1.Close()

	ctx := context.Background()

	// Test acquiring lock
	err = lock1.Acquire(ctx)
	require.NoError(t, err, "should acquire lock successfully")

	// Test that second instance cannot acquire same lock
	lock2, err := NewRedisLock(cfg, "test-lock")
	require.NoError(t, err)
	defer lock2.Close()

	err = lock2.Acquire(ctx)
	assert.Error(t, err, "should not acquire lock when already held")

	// Release first lock
	err = lock1.Release(ctx)
	require.NoError(t, err, "should release lock successfully")

	// Now second instance should be able to acquire
	err = lock2.Acquire(ctx)
	assert.NoError(t, err, "should acquire lock after release")

	// Clean up
	lock2.Release(ctx)
}

func TestRedisLock_Heartbeat(t *testing.T) {
	cfg := &RedisLockConfig{
		Address: "localhost:6379",
		DB:      0,
		TTL:     3 * time.Second, // Short TTL for testing
	}

	lock, err := NewRedisLock(cfg, "test-heartbeat")
	if err != nil {
		t.Skip("Redis not available, skipping test")
		return
	}
	defer lock.Close()

	ctx := context.Background()

	err = lock.Acquire(ctx)
	require.NoError(t, err)

	// Wait longer than TTL to ensure heartbeat keeps it alive
	time.Sleep(5 * time.Second)

	// Lock should still be held
	err = lock.Release(ctx)
	assert.NoError(t, err, "heartbeat should have kept lock alive")
}

func TestRedisLock_ReleaseUnowned(t *testing.T) {
	cfg := &RedisLockConfig{
		Address: "localhost:6379",
		DB:      0,
		TTL:     30 * time.Second,
	}

	lock1, err := NewRedisLock(cfg, "test-unowned")
	if err != nil {
		t.Skip("Redis not available, skipping test")
		return
	}
	defer lock1.Close()

	lock2, err := NewRedisLock(cfg, "test-unowned")
	require.NoError(t, err)
	defer lock2.Close()

	ctx := context.Background()

	// Lock1 acquires
	err = lock1.Acquire(ctx)
	require.NoError(t, err)

	// Lock2 tries to release (should fail)
	err = lock2.Release(ctx)
	assert.Error(t, err, "should not release lock not owned")

	// Cleanup
	lock1.Release(ctx)
}
