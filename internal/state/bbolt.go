package state

import (
	"encoding/json"
	"fmt"
	"time"

	"go.etcd.io/bbolt"
)

// BboltBackend implements Backend using bbolt (pure Go, no CGO)
type BboltBackend struct {
	db         *bbolt.DB
	bucketName []byte
}

const (
	defaultBucket = "platform-foundry-state"
)

// NewBboltBackend creates a new bbolt backend
func NewBboltBackend(path string) (*BboltBackend, error) {
	// Open database with a timeout
	db, err := bbolt.Open(path, 0600, &bbolt.Options{
		Timeout: 1 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open bbolt database: %w", err)
	}

	backend := &BboltBackend{
		db:         db,
		bucketName: []byte(defaultBucket),
	}

	// Create bucket if it doesn't exist
	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(backend.bucketName)
		return err
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create bucket: %w", err)
	}

	return backend, nil
}

// Save saves a resource to bbolt
func (b *BboltBackend) Save(resource *Resource) error {
	// Set timestamps
	now := time.Now()
	if resource.CreatedAt.IsZero() {
		resource.CreatedAt = now
	}
	resource.UpdatedAt = now

	// Marshal resource to JSON
	data, err := json.Marshal(resource)
	if err != nil {
		return fmt.Errorf("failed to marshal resource: %w", err)
	}

	// Store in bbolt
	err = b.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(b.bucketName)
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}

		return bucket.Put([]byte(resource.Name), data)
	})

	if err != nil {
		return fmt.Errorf("failed to save resource: %w", err)
	}

	return nil
}

// Get retrieves a resource from bbolt
func (b *BboltBackend) Get(name string) (*Resource, error) {
	var resource Resource

	err := b.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(b.bucketName)
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}

		data := bucket.Get([]byte(name))
		if data == nil {
			return fmt.Errorf("resource not found: %s", name)
		}

		return json.Unmarshal(data, &resource)
	})

	if err != nil {
		return nil, err
	}

	return &resource, nil
}

// List retrieves all resources from bbolt
func (b *BboltBackend) List() ([]*Resource, error) {
	var resources []*Resource

	err := b.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(b.bucketName)
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}

		return bucket.ForEach(func(key, value []byte) error {
			var resource Resource
			if err := json.Unmarshal(value, &resource); err != nil {
				// Skip malformed entries
				return nil
			}

			resources = append(resources, &resource)
			return nil
		})
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	return resources, nil
}

// Delete removes a resource from bbolt
func (b *BboltBackend) Delete(name string) error {
	err := b.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(b.bucketName)
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}

		// Check if resource exists
		if bucket.Get([]byte(name)) == nil {
			return fmt.Errorf("resource not found: %s", name)
		}

		return bucket.Delete([]byte(name))
	})

	if err != nil {
		return fmt.Errorf("failed to delete resource: %w", err)
	}

	return nil
}

// Lock acquires a lock on a resource (bbolt transactions are automatically locked)
func (b *BboltBackend) Lock(name string) error {
	// bbolt uses write locks automatically for Update transactions
	// For explicit locking, we could create a separate locks bucket
	// For now, this is a no-op since transactions provide ACID guarantees
	return nil
}

// Unlock releases a lock on a resource
func (b *BboltBackend) Unlock(name string) error {
	// bbolt transactions are automatically unlocked when committed
	// This is a no-op for the same reason as Lock
	return nil
}

// Close closes the bbolt database
func (b *BboltBackend) Close() error {
	if b.db != nil {
		return b.db.Close()
	}
	return nil
}

// Backup creates a backup of the database
func (b *BboltBackend) Backup(path string) error {
	return b.db.View(func(tx *bbolt.Tx) error {
		return tx.CopyFile(path, 0600)
	})
}

// Stats returns database statistics
func (b *BboltBackend) Stats() (*bbolt.Stats, error) {
	stats := b.db.Stats()
	return &stats, nil
}

// GetVersion retrieves a specific version of a resource
// Note: bbolt doesn't have built-in versioning like SQLite backend
// For now, only return the latest version
func (b *BboltBackend) GetVersion(name string, version int) (*Resource, error) {
	// Get the resource (only latest version is stored)
	resource, err := b.Get(name)
	if err != nil {
		return nil, err
	}

	// If requesting version 1 or current version, return it
	if version == resource.Version || version == 1 {
		return resource, nil
	}

	return nil, fmt.Errorf("version %d not found (only current version %d is available)", version, resource.Version)
}

// ListVersions returns all versions of a resource
// Note: bbolt backend only keeps the latest version
func (b *BboltBackend) ListVersions(name string) ([]*ResourceVersion, error) {
	resource, err := b.Get(name)
	if err != nil {
		return nil, err
	}

	// Return single version
	versions := []*ResourceVersion{
		{
			Version:   resource.Version,
			Spec:      resource.Spec,
			Status:    resource.Status,
			CreatedAt: resource.UpdatedAt,
		},
	}

	return versions, nil
}
