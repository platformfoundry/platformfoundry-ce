package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/platformfoundry/platformfoundry-ce/internal/workflow"
	"go.etcd.io/bbolt"
)

const (
	workflowBucket   = "dag-workflows"
	executionBucket  = "dag-executions"
	stepExecBucket   = "dag-step-executions"
)

// BoltStore implements Store using BoltDB
type BoltStore struct {
	db *bbolt.DB
}

// NewBoltStore creates a new BoltDB-based store
func NewBoltStore(path string) (*BoltStore, error) {
	db, err := bbolt.Open(path, 0600, &bbolt.Options{
		Timeout: 1 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create buckets
	err = db.Update(func(tx *bbolt.Tx) error {
		for _, bucket := range []string{workflowBucket, executionBucket, stepExecBucket} {
			if _, err := tx.CreateBucketIfNotExists([]byte(bucket)); err != nil {
				return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
			}
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, err
	}

	return &BoltStore{db: db}, nil
}

// SaveWorkflow saves a workflow definition
func (s *BoltStore) SaveWorkflow(ctx context.Context, wf *workflow.DAGWorkflow) error {
	data, err := json.Marshal(wf)
	if err != nil {
		return fmt.Errorf("failed to marshal workflow: %w", err)
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(workflowBucket))
		return bucket.Put([]byte(wf.Metadata.Name), data)
	})
}

// GetWorkflow retrieves a workflow by name
func (s *BoltStore) GetWorkflow(ctx context.Context, name string) (*workflow.DAGWorkflow, error) {
	var wf workflow.DAGWorkflow

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(workflowBucket))
		data := bucket.Get([]byte(name))
		if data == nil {
			return fmt.Errorf("workflow not found: %s", name)
		}
		return json.Unmarshal(data, &wf)
	})

	if err != nil {
		return nil, err
	}

	return &wf, nil
}

// ListWorkflows returns all workflows
func (s *BoltStore) ListWorkflows(ctx context.Context) ([]*workflow.DAGWorkflow, error) {
	var workflows []*workflow.DAGWorkflow

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(workflowBucket))
		return bucket.ForEach(func(key, value []byte) error {
			var wf workflow.DAGWorkflow
			if err := json.Unmarshal(value, &wf); err != nil {
				return nil // Skip malformed entries
			}
			workflows = append(workflows, &wf)
			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	return workflows, nil
}

// DeleteWorkflow removes a workflow
func (s *BoltStore) DeleteWorkflow(ctx context.Context, name string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(workflowBucket))
		if bucket.Get([]byte(name)) == nil {
			return fmt.Errorf("workflow not found: %s", name)
		}
		return bucket.Delete([]byte(name))
	})
}

// SaveExecution saves an execution
func (s *BoltStore) SaveExecution(ctx context.Context, exec *workflow.DAGExecution) error {
	data, err := json.Marshal(exec)
	if err != nil {
		return fmt.Errorf("failed to marshal execution: %w", err)
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(executionBucket))
		return bucket.Put([]byte(exec.ID), data)
	})
}

// GetExecution retrieves an execution by ID
func (s *BoltStore) GetExecution(ctx context.Context, id string) (*workflow.DAGExecution, error) {
	var exec workflow.DAGExecution

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(executionBucket))
		data := bucket.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("execution not found: %s", id)
		}
		return json.Unmarshal(data, &exec)
	})

	if err != nil {
		return nil, err
	}

	return &exec, nil
}

// ListExecutions returns executions for a workflow
func (s *BoltStore) ListExecutions(ctx context.Context, workflowName string, limit int) ([]*workflow.DAGExecution, error) {
	var executions []*workflow.DAGExecution

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(executionBucket))
		return bucket.ForEach(func(key, value []byte) error {
			var exec workflow.DAGExecution
			if err := json.Unmarshal(value, &exec); err != nil {
				return nil // Skip malformed entries
			}

			// Filter by workflow name if specified
			if workflowName != "" && exec.WorkflowName != workflowName {
				return nil
			}

			executions = append(executions, &exec)

			// Apply limit
			if limit > 0 && len(executions) >= limit {
				return nil
			}

			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	return executions, nil
}

// UpdateExecutionStatus updates just the status of an execution
func (s *BoltStore) UpdateExecutionStatus(ctx context.Context, id string, status workflow.WorkflowStatus) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(executionBucket))
		data := bucket.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("execution not found: %s", id)
		}

		var exec workflow.DAGExecution
		if err := json.Unmarshal(data, &exec); err != nil {
			return err
		}

		exec.Status = status
		if status == workflow.WorkflowStatusCompleted || status == workflow.WorkflowStatusFailed {
			now := time.Now()
			exec.CompletedAt = &now
		}

		updatedData, err := json.Marshal(exec)
		if err != nil {
			return err
		}

		return bucket.Put([]byte(id), updatedData)
	})
}

// SaveStepExecution saves a step execution
func (s *BoltStore) SaveStepExecution(ctx context.Context, execID string, step *workflow.StepExecution) error {
	data, err := json.Marshal(step)
	if err != nil {
		return fmt.Errorf("failed to marshal step execution: %w", err)
	}

	key := fmt.Sprintf("%s:%s", execID, step.StepID)

	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(stepExecBucket))
		return bucket.Put([]byte(key), data)
	})
}

// GetStepExecution retrieves a step execution
func (s *BoltStore) GetStepExecution(ctx context.Context, execID, stepID string) (*workflow.StepExecution, error) {
	var step workflow.StepExecution
	key := fmt.Sprintf("%s:%s", execID, stepID)

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(stepExecBucket))
		data := bucket.Get([]byte(key))
		if data == nil {
			return fmt.Errorf("step execution not found: %s", key)
		}
		return json.Unmarshal(data, &step)
	})

	if err != nil {
		return nil, err
	}

	return &step, nil
}

// Close closes the database
func (s *BoltStore) Close() error {
	return s.db.Close()
}

// Backup creates a backup of the database
func (s *BoltStore) Backup(path string) error {
	return s.db.View(func(tx *bbolt.Tx) error {
		return tx.CopyFile(path, 0600)
	})
}
