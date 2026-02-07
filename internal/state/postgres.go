package state

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

// PostgresBackend implements Backend using PostgreSQL
type PostgresBackend struct {
	db         *sql.DB
	encryptor  *Encryptor
	schema     string
	tableName  string
	lockTable  string
	mu         sync.RWMutex
	config     *PostgresConfig
}

// PostgresConfig configures the PostgreSQL backend
type PostgresConfig struct {
	Host       string            `yaml:"host" json:"host"`
	Port       int               `yaml:"port" json:"port"`
	Database   string            `yaml:"database" json:"database"`
	User       string            `yaml:"user" json:"user"`
	Password   string            `yaml:"password" json:"password"`
	SSLMode    string            `yaml:"sslMode" json:"sslMode"`
	Schema     string            `yaml:"schema" json:"schema"`
	TableName  string            `yaml:"tableName" json:"tableName"`
	Encryption *EncryptionConfig `yaml:"encryption,omitempty" json:"encryption,omitempty"`
}

// NewPostgresBackend creates a new PostgreSQL backend
func NewPostgresBackend(config *PostgresConfig) (*PostgresBackend, error) {
	if config.Port == 0 {
		config.Port = 5432
	}
	if config.SSLMode == "" {
		config.SSLMode = "require"
	}
	if config.Schema == "" {
		config.Schema = "platformfoundry"
	}
	if config.TableName == "" {
		config.TableName = "state"
	}

	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.Database, config.SSLMode,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Initialize encryptor if configured
	var encryptor *Encryptor
	if config.Encryption != nil && config.Encryption.Enabled {
		encryptor, err = NewEncryptor(config.Encryption)
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to initialize encryptor: %w", err)
		}
	}

	backend := &PostgresBackend{
		db:        db,
		encryptor: encryptor,
		schema:    config.Schema,
		tableName: config.TableName,
		lockTable: config.TableName + "_locks",
		config:    config,
	}

	// Initialize schema and tables
	if err := backend.initSchema(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return backend, nil
}

func (b *PostgresBackend) initSchema(ctx context.Context) error {
	// Create schema if not exists
	_, err := b.db.ExecContext(ctx, fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, b.schema))
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Create state table
	createStateTable := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.%s (
			id VARCHAR(255) PRIMARY KEY,
			data BYTEA NOT NULL,
			version INTEGER NOT NULL DEFAULT 1,
			lock_id VARCHAR(255),
			locked_at TIMESTAMP WITH TIME ZONE,
			locked_by VARCHAR(255),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
	`, b.schema, b.tableName)

	if _, err := b.db.ExecContext(ctx, createStateTable); err != nil {
		return fmt.Errorf("failed to create state table: %w", err)
	}

	// Create index on updated_at for efficient queries
	createIndex := fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS idx_%s_updated_at ON %s.%s(updated_at)
	`, b.tableName, b.schema, b.tableName)

	if _, err := b.db.ExecContext(ctx, createIndex); err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	// Create locks table
	createLocksTable := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.%s (
			id VARCHAR(255) PRIMARY KEY,
			info JSONB NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			expires_at TIMESTAMP WITH TIME ZONE
		)
	`, b.schema, b.lockTable)

	if _, err := b.db.ExecContext(ctx, createLocksTable); err != nil {
		return fmt.Errorf("failed to create locks table: %w", err)
	}

	return nil
}

// Get retrieves state data by ID
func (b *PostgresBackend) Get(ctx context.Context, id string) ([]byte, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	query := fmt.Sprintf(`SELECT data FROM %s.%s WHERE id = $1`, b.schema, b.tableName)

	var data []byte
	err := b.db.QueryRowContext(ctx, query, id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %w", err)
	}

	// Decrypt if encryption is enabled
	if b.encryptor != nil && b.encryptor.IsEnabled() {
		decrypted, err := b.encryptor.Decrypt(data)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt state: %w", err)
		}
		return decrypted, nil
	}

	return data, nil
}

// Put stores state data
func (b *PostgresBackend) Put(ctx context.Context, id string, data []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Encrypt if encryption is enabled
	if b.encryptor != nil && b.encryptor.IsEnabled() {
		encrypted, err := b.encryptor.Encrypt(data)
		if err != nil {
			return fmt.Errorf("failed to encrypt state: %w", err)
		}
		data = encrypted
	}

	query := fmt.Sprintf(`
		INSERT INTO %s.%s (id, data, version, updated_at)
		VALUES ($1, $2, 1, NOW())
		ON CONFLICT (id) DO UPDATE SET
			data = EXCLUDED.data,
			version = %s.%s.version + 1,
			updated_at = NOW()
	`, b.schema, b.tableName, b.schema, b.tableName)

	_, err := b.db.ExecContext(ctx, query, id, data)
	if err != nil {
		return fmt.Errorf("failed to put state: %w", err)
	}

	return nil
}

// Delete removes state data
func (b *PostgresBackend) Delete(ctx context.Context, id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	query := fmt.Sprintf(`DELETE FROM %s.%s WHERE id = $1`, b.schema, b.tableName)

	_, err := b.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete state: %w", err)
	}

	return nil
}

// List returns all state IDs matching a prefix
func (b *PostgresBackend) List(ctx context.Context, prefix string) ([]string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	query := fmt.Sprintf(`SELECT id FROM %s.%s WHERE id LIKE $1 ORDER BY updated_at DESC`, b.schema, b.tableName)

	rows, err := b.db.QueryContext(ctx, query, prefix+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to list states: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		ids = append(ids, id)
	}

	return ids, nil
}

// Lock acquires a lock on the state
func (b *PostgresBackend) Lock(ctx context.Context, id string, info *LockInfo) error {
	if info == nil {
		info = &LockInfo{
			ID:        id,
			Operation: "unknown",
			Who:       "unknown",
			Version:   "1",
			Created:   time.Now(),
		}
	}

	infoJSON, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal lock info: %w", err)
	}

	// Try to acquire lock with advisory lock
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Try to get advisory lock
	var acquired bool
	lockQuery := `SELECT pg_try_advisory_xact_lock(hashtext($1))`
	if err := tx.QueryRowContext(ctx, lockQuery, id).Scan(&acquired); err != nil {
		return fmt.Errorf("failed to acquire advisory lock: %w", err)
	}

	if !acquired {
		// Check existing lock info
		existingQuery := fmt.Sprintf(`SELECT info FROM %s.%s WHERE id = $1`, b.schema, b.lockTable)
		var existingInfo []byte
		err := tx.QueryRowContext(ctx, existingQuery, id).Scan(&existingInfo)
		if err == nil {
			var existing LockInfo
			json.Unmarshal(existingInfo, &existing)
			return &LockError{
				Info: &existing,
				Err:  fmt.Errorf("state is locked by %s", existing.Who),
			}
		}
		return fmt.Errorf("failed to acquire lock")
	}

	// Insert or update lock record
	upsertQuery := fmt.Sprintf(`
		INSERT INTO %s.%s (id, info, created_at, expires_at)
		VALUES ($1, $2, NOW(), NOW() + INTERVAL '1 hour')
		ON CONFLICT (id) DO UPDATE SET
			info = EXCLUDED.info,
			created_at = NOW(),
			expires_at = NOW() + INTERVAL '1 hour'
	`, b.schema, b.lockTable)

	if _, err := tx.ExecContext(ctx, upsertQuery, id, infoJSON); err != nil {
		return fmt.Errorf("failed to insert lock: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit lock: %w", err)
	}

	return nil
}

// Unlock releases a lock on the state
func (b *PostgresBackend) Unlock(ctx context.Context, id string) error {
	query := fmt.Sprintf(`DELETE FROM %s.%s WHERE id = $1`, b.schema, b.lockTable)

	_, err := b.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	return nil
}

// GetLockInfo retrieves lock information
func (b *PostgresBackend) GetLockInfo(ctx context.Context, id string) (*LockInfo, error) {
	query := fmt.Sprintf(`SELECT info FROM %s.%s WHERE id = $1 AND (expires_at IS NULL OR expires_at > NOW())`, b.schema, b.lockTable)

	var infoJSON []byte
	err := b.db.QueryRowContext(ctx, query, id).Scan(&infoJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get lock info: %w", err)
	}

	var info LockInfo
	if err := json.Unmarshal(infoJSON, &info); err != nil {
		return nil, fmt.Errorf("failed to unmarshal lock info: %w", err)
	}

	return &info, nil
}

// Close closes the database connection
func (b *PostgresBackend) Close() error {
	return b.db.Close()
}

// Migrate performs any necessary database migrations
func (b *PostgresBackend) Migrate(ctx context.Context) error {
	// Add any migration logic here
	// This is a placeholder for future schema migrations

	// Example: Add a new column if it doesn't exist
	alterQuery := fmt.Sprintf(`
		ALTER TABLE %s.%s
		ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}'
	`, b.schema, b.tableName)

	_, err := b.db.ExecContext(ctx, alterQuery)
	if err != nil {
		// Ignore error if column already exists (for older PostgreSQL versions)
		return nil
	}

	return nil
}

// Stats returns statistics about the backend
func (b *PostgresBackend) Stats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get total count
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM %s.%s`, b.schema, b.tableName)
	var count int64
	if err := b.db.QueryRowContext(ctx, countQuery).Scan(&count); err != nil {
		return nil, err
	}
	stats["total_states"] = count

	// Get total size
	sizeQuery := fmt.Sprintf(`SELECT COALESCE(SUM(LENGTH(data)), 0) FROM %s.%s`, b.schema, b.tableName)
	var size int64
	if err := b.db.QueryRowContext(ctx, sizeQuery).Scan(&size); err != nil {
		return nil, err
	}
	stats["total_size_bytes"] = size

	// Get active locks
	lockQuery := fmt.Sprintf(`SELECT COUNT(*) FROM %s.%s WHERE expires_at IS NULL OR expires_at > NOW()`, b.schema, b.lockTable)
	var locks int64
	if err := b.db.QueryRowContext(ctx, lockQuery).Scan(&locks); err != nil {
		return nil, err
	}
	stats["active_locks"] = locks

	// Get database stats
	dbStats := b.db.Stats()
	stats["open_connections"] = dbStats.OpenConnections
	stats["in_use_connections"] = dbStats.InUse
	stats["idle_connections"] = dbStats.Idle

	return stats, nil
}

// LockInfo contains information about a state lock
type LockInfo struct {
	ID        string    `json:"id"`
	Operation string    `json:"operation"`
	Who       string    `json:"who"`
	Version   string    `json:"version"`
	Created   time.Time `json:"created"`
	Path      string    `json:"path,omitempty"`
}

// LockError is returned when a lock operation fails
type LockError struct {
	Info *LockInfo
	Err  error
}

func (e *LockError) Error() string {
	return e.Err.Error()
}

func (e *LockError) Unwrap() error {
	return e.Err
}
