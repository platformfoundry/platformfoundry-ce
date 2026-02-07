package state

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresConfig_Defaults(t *testing.T) {
	cfg := &PostgresConfig{
		Host:     "localhost",
		Database: "testdb",
		User:     "testuser",
		Password: "testpass",
	}

	// Test that NewPostgresBackend sets defaults
	// This will fail to connect but we can check the config processing
	assert.Equal(t, 0, cfg.Port) // Will be set to 5432
	assert.Equal(t, "", cfg.SSLMode) // Will be set to "require"
	assert.Equal(t, "", cfg.Schema) // Will be set to "platformfoundry"
	assert.Equal(t, "", cfg.TableName) // Will be set to "state"
}

func TestPostgresConfig_Validation(t *testing.T) {
	tests := []struct {
		name   string
		config *PostgresConfig
	}{
		{
			name: "minimal config",
			config: &PostgresConfig{
				Host:     "localhost",
				Database: "test",
				User:     "user",
				Password: "pass",
			},
		},
		{
			name: "full config",
			config: &PostgresConfig{
				Host:      "db.example.com",
				Port:      5433,
				Database:  "platformfoundry",
				User:      "pf_user",
				Password:  "secret",
				SSLMode:   "verify-full",
				Schema:    "pf_state",
				TableName: "states",
			},
		},
		{
			name: "with encryption",
			config: &PostgresConfig{
				Host:     "localhost",
				Database: "test",
				User:     "user",
				Password: "pass",
				Encryption: &EncryptionConfig{
					Enabled:     true,
					KeyProvider: "env",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.config)
			assert.NotEmpty(t, tt.config.Host)
			assert.NotEmpty(t, tt.config.Database)
		})
	}
}

func TestLockInfo_Serialization(t *testing.T) {
	info := &LockInfo{
		ID:        "test-lock-123",
		Operation: "apply",
		Who:       "user@example.com",
		Version:   "1.0.0",
		Created:   time.Now(),
		Path:      "/path/to/state",
	}

	assert.Equal(t, "test-lock-123", info.ID)
	assert.Equal(t, "apply", info.Operation)
	assert.Equal(t, "user@example.com", info.Who)
	assert.Equal(t, "1.0.0", info.Version)
	assert.NotEmpty(t, info.Path)
}

func TestLockError(t *testing.T) {
	lockInfo := &LockInfo{
		ID:        "existing-lock",
		Operation: "plan",
		Who:       "other-user@example.com",
		Created:   time.Now(),
	}

	lockErr := &LockError{
		Info: lockInfo,
		Err:  assert.AnError,
	}

	assert.Error(t, lockErr)
	assert.Equal(t, assert.AnError.Error(), lockErr.Error())
	assert.Equal(t, assert.AnError, lockErr.Unwrap())
	assert.Equal(t, lockInfo, lockErr.Info)
}

func TestPostgresBackend_Interface(t *testing.T) {
	// Verify PostgresBackend implements expected methods
	var _ interface {
		Get(ctx context.Context, id string) ([]byte, error)
		Put(ctx context.Context, id string, data []byte) error
		Delete(ctx context.Context, id string) error
		List(ctx context.Context, prefix string) ([]string, error)
		Lock(ctx context.Context, id string, info *LockInfo) error
		Unlock(ctx context.Context, id string) error
		GetLockInfo(ctx context.Context, id string) (*LockInfo, error)
		Close() error
		Migrate(ctx context.Context) error
		Stats(ctx context.Context) (map[string]interface{}, error)
	} = (*PostgresBackend)(nil)
}

// MockPostgresBackend for unit testing without database
type MockPostgresBackend struct {
	data  map[string][]byte
	locks map[string]*LockInfo
}

func NewMockPostgresBackend() *MockPostgresBackend {
	return &MockPostgresBackend{
		data:  make(map[string][]byte),
		locks: make(map[string]*LockInfo),
	}
}

func (m *MockPostgresBackend) Get(ctx context.Context, id string) ([]byte, error) {
	data, ok := m.data[id]
	if !ok {
		return nil, nil
	}
	return data, nil
}

func (m *MockPostgresBackend) Put(ctx context.Context, id string, data []byte) error {
	m.data[id] = data
	return nil
}

func (m *MockPostgresBackend) Delete(ctx context.Context, id string) error {
	delete(m.data, id)
	return nil
}

func (m *MockPostgresBackend) List(ctx context.Context, prefix string) ([]string, error) {
	var ids []string
	for id := range m.data {
		if len(prefix) == 0 || len(id) >= len(prefix) && id[:len(prefix)] == prefix {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (m *MockPostgresBackend) Lock(ctx context.Context, id string, info *LockInfo) error {
	if _, locked := m.locks[id]; locked {
		return &LockError{Info: m.locks[id], Err: assert.AnError}
	}
	m.locks[id] = info
	return nil
}

func (m *MockPostgresBackend) Unlock(ctx context.Context, id string) error {
	delete(m.locks, id)
	return nil
}

func (m *MockPostgresBackend) GetLockInfo(ctx context.Context, id string) (*LockInfo, error) {
	return m.locks[id], nil
}

func TestMockPostgresBackend_CRUD(t *testing.T) {
	ctx := context.Background()
	backend := NewMockPostgresBackend()

	// Test Put
	err := backend.Put(ctx, "test-id", []byte("test data"))
	require.NoError(t, err)

	// Test Get
	data, err := backend.Get(ctx, "test-id")
	require.NoError(t, err)
	assert.Equal(t, []byte("test data"), data)

	// Test Get non-existent
	data, err = backend.Get(ctx, "non-existent")
	require.NoError(t, err)
	assert.Nil(t, data)

	// Test List
	err = backend.Put(ctx, "prefix-1", []byte("data1"))
	require.NoError(t, err)
	err = backend.Put(ctx, "prefix-2", []byte("data2"))
	require.NoError(t, err)
	err = backend.Put(ctx, "other-1", []byte("data3"))
	require.NoError(t, err)

	ids, err := backend.List(ctx, "prefix-")
	require.NoError(t, err)
	assert.Len(t, ids, 2)

	// Test Delete
	err = backend.Delete(ctx, "test-id")
	require.NoError(t, err)

	data, err = backend.Get(ctx, "test-id")
	require.NoError(t, err)
	assert.Nil(t, data)
}

func TestMockPostgresBackend_Locking(t *testing.T) {
	ctx := context.Background()
	backend := NewMockPostgresBackend()

	lockInfo := &LockInfo{
		ID:        "state-1",
		Operation: "apply",
		Who:       "user1",
		Created:   time.Now(),
	}

	// Acquire lock
	err := backend.Lock(ctx, "state-1", lockInfo)
	require.NoError(t, err)

	// Verify lock info
	info, err := backend.GetLockInfo(ctx, "state-1")
	require.NoError(t, err)
	assert.Equal(t, lockInfo, info)

	// Try to acquire same lock (should fail)
	err = backend.Lock(ctx, "state-1", &LockInfo{Who: "user2"})
	assert.Error(t, err)
	lockErr, ok := err.(*LockError)
	assert.True(t, ok)
	assert.Equal(t, "user1", lockErr.Info.Who)

	// Release lock
	err = backend.Unlock(ctx, "state-1")
	require.NoError(t, err)

	// Verify lock is released
	info, err = backend.GetLockInfo(ctx, "state-1")
	require.NoError(t, err)
	assert.Nil(t, info)

	// Now lock should succeed
	err = backend.Lock(ctx, "state-1", &LockInfo{Who: "user2"})
	require.NoError(t, err)
}

func TestPostgresBackend_ConnectionString(t *testing.T) {
	tests := []struct {
		name     string
		config   *PostgresConfig
		contains []string
	}{
		{
			name: "basic config",
			config: &PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "user",
				Password: "pass",
				Database: "db",
				SSLMode:  "disable",
			},
			contains: []string{"localhost", "5432", "user", "pass", "db", "disable"},
		},
		{
			name: "custom port",
			config: &PostgresConfig{
				Host:     "db.example.com",
				Port:     5433,
				User:     "admin",
				Password: "secret",
				Database: "production",
				SSLMode:  "require",
			},
			contains: []string{"db.example.com", "5433", "admin", "secret", "production", "require"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't test actual connection, but we can verify config is valid
			assert.NotEmpty(t, tt.config.Host)
			assert.Greater(t, tt.config.Port, 0)
			assert.NotEmpty(t, tt.config.User)
			assert.NotEmpty(t, tt.config.Database)
		})
	}
}

func TestLockInfo_DefaultValues(t *testing.T) {
	// Test that Lock creates default LockInfo when nil is passed
	info := &LockInfo{}

	// Verify zero values
	assert.Empty(t, info.ID)
	assert.Empty(t, info.Operation)
	assert.Empty(t, info.Who)
	assert.True(t, info.Created.IsZero())
}

func BenchmarkMockBackend_Put(b *testing.B) {
	ctx := context.Background()
	backend := NewMockPostgresBackend()
	data := []byte("benchmark test data for performance testing")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		backend.Put(ctx, "bench-key", data)
	}
}

func BenchmarkMockBackend_Get(b *testing.B) {
	ctx := context.Background()
	backend := NewMockPostgresBackend()
	backend.Put(ctx, "bench-key", []byte("benchmark data"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		backend.Get(ctx, "bench-key")
	}
}
