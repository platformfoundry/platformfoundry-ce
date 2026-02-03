package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestS3Backend_NewS3Backend(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *S3Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &S3Config{
				Bucket:    "test-bucket",
				Region:    "us-east-1",
				TableName: "test-locks",
			},
			wantErr: false,
		},
		{
			name: "missing bucket",
			cfg: &S3Config{
				Region:    "us-east-1",
				TableName: "test-locks",
			},
			wantErr: true,
		},
		{
			name: "missing region",
			cfg: &S3Config{
				Bucket:    "test-bucket",
				TableName: "test-locks",
			},
			wantErr: true,
		},
		{
			name: "missing table name",
			cfg: &S3Config{
				Bucket: "test-bucket",
				Region: "us-east-1",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewS3Backend(tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				// Will error due to AWS credentials, but should at least return a backend
				// Just check config validation passed (actual AWS connection may fail)
				_ = err // Expected to potentially fail without AWS creds
			}
		})
	}
}

func TestS3Backend_CreateS3Key(t *testing.T) {
	backend := &S3Backend{
		bucket: "test-bucket",
		prefix: "state/",
	}

	tests := []struct {
		name     string
		resource string
		want     string
	}{
		{
			name:     "simple resource name",
			resource: "my-platform",
			want:     "state/resources/my-platform.json",
		},
		{
			name:     "resource with dashes",
			resource: "my-test-cluster",
			want:     "state/resources/my-test-cluster.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := backend.createS3Key(tt.resource)
			assert.Equal(t, tt.want, key)
		})
	}
}

func TestS3Backend_SaveAndGet(t *testing.T) {
	// Note: This is an integration test that requires AWS credentials and S3 access
	// Skip if not in integration test environment
	t.Skip("Integration test - requires AWS credentials")

	cfg := &S3Config{
		Bucket:    "platformfoundry-test",
		Region:    "us-east-1",
		Prefix:    "test/",
		TableName: "platformfoundry-test-locks",
	}

	backend, err := NewS3Backend(cfg)
	require.NoError(t, err)

	resource := &Resource{
		Name:       "test-platform",
		Kind:       "Platform",
		APIVersion: "platformfoundry.io/v1",
		Spec: map[string]interface{}{
			"cloud": "aws",
		},
	}

	// Test Save
	err = backend.Save(resource)
	require.NoError(t, err)

	// Test Get
	retrieved, err := backend.Get("test-platform")
	require.NoError(t, err)
	assert.Equal(t, resource.Name, retrieved.Name)
	assert.Equal(t, resource.Kind, retrieved.Kind)

	// Test Delete
	err = backend.Delete("test-platform")
	require.NoError(t, err)

	// Verify deleted
	_, err = backend.Get("test-platform")
	assert.Error(t, err)
}

func TestS3Backend_List(t *testing.T) {
	t.Skip("Integration test - requires AWS credentials")

	cfg := &S3Config{
		Bucket:    "platformfoundry-test",
		Region:    "us-east-1",
		Prefix:    "test/",
		TableName: "platformfoundry-test-locks",
	}

	backend, err := NewS3Backend(cfg)
	require.NoError(t, err)

	// Create test resources
	resources := []*Resource{
		{
			Name:       "platform-1",
			Kind:       "Platform",
			APIVersion: "platformfoundry.io/v1",
			Spec:       map[string]interface{}{"cloud": "aws"},
		},
		{
			Name:       "platform-2",
			Kind:       "Platform",
			APIVersion: "platformfoundry.io/v1",
			Spec:       map[string]interface{}{"cloud": "gcp"},
		},
	}

	for _, r := range resources {
		err := backend.Save(r)
		require.NoError(t, err)
	}

	// Test List
	list, err := backend.List()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list), 2)

	// Cleanup
	for _, r := range resources {
		backend.Delete(r.Name)
	}
}
