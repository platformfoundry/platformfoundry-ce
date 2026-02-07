// Package psi defines the Platform State Interface (PSI).
package psi

import (
	"errors"
	"time"
)

// LockInfo contains information about a state lock
type LockInfo struct {
	// ID is the unique identifier for this lock
	ID string

	// Operation describes what operation holds the lock
	Operation string

	// Who is the identity that created the lock
	Who string

	// Version is the state version when locked
	Version int64

	// Created is when the lock was created
	Created time.Time

	// Path is the state path being locked
	Path string

	// Info contains additional lock information
	Info string
}

// StateVersion represents a historical state version
type StateVersion struct {
	// Version is the state version number
	Version int64

	// Serial is the state serial number
	Serial int64

	// CreatedAt is when this version was created
	CreatedAt time.Time

	// CreatedBy is who created this version
	CreatedBy string

	// Size is the size of this state in bytes
	Size int64

	// Checksum is a checksum of the state content
	Checksum string

	// Operation describes what operation created this version
	Operation string
}

// Common errors for state operations
var (
	// ErrStateLocked indicates the state is already locked
	ErrStateLocked = errors.New("state is locked")

	// ErrLockNotFound indicates the lock doesn't exist
	ErrLockNotFound = errors.New("lock not found")

	// ErrStateNotFound indicates the state doesn't exist
	ErrStateNotFound = errors.New("state not found")

	// ErrWorkspaceNotFound indicates the workspace doesn't exist
	ErrWorkspaceNotFound = errors.New("workspace not found")

	// ErrWorkspaceExists indicates the workspace already exists
	ErrWorkspaceExists = errors.New("workspace already exists")

	// ErrVersionNotFound indicates the state version doesn't exist
	ErrVersionNotFound = errors.New("state version not found")

	// ErrConcurrentModification indicates a concurrent modification conflict
	ErrConcurrentModification = errors.New("concurrent modification detected")

	// ErrInvalidState indicates the state is invalid
	ErrInvalidState = errors.New("invalid state")
)

// LockError provides details about a lock conflict
type LockError struct {
	Info *LockInfo
	Err  error
}

func (e *LockError) Error() string {
	if e.Info != nil {
		return "state locked by " + e.Info.Who + " for operation " + e.Info.Operation
	}
	return e.Err.Error()
}

func (e *LockError) Unwrap() error {
	return e.Err
}

// NewLockError creates a new lock error
func NewLockError(info *LockInfo, err error) *LockError {
	return &LockError{
		Info: info,
		Err:  err,
	}
}

// Locker provides state locking functionality
type Locker interface {
	// Lock acquires a lock for the given workspace
	Lock(workspace string, info *LockInfo) (string, error)

	// Unlock releases a lock
	Unlock(workspace string, lockID string) error

	// ForceUnlock releases a lock regardless of ownership
	ForceUnlock(workspace string) error

	// GetLockInfo returns information about the current lock
	GetLockInfo(workspace string) (*LockInfo, error)
}
