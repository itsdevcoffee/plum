//go:build windows

package settings

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// FileLock provides advisory file locking for concurrent access protection
// On Windows, this uses exclusive file creation as a simple locking mechanism
type FileLock struct {
	path string
	file *os.File
}

// lockTimeout is how long to wait for a lock before giving up
const lockTimeout = 10 * time.Second

// NewFileLock creates a new file lock for the given path
// The lock file will be created at path + ".lock"
func NewFileLock(path string) *FileLock {
	return &FileLock{
		path: path + ".lock",
	}
}

// Lock acquires an exclusive lock, blocking until available or timeout
// On Windows, this uses exclusive file creation
func (l *FileLock) Lock() error {
	// Ensure directory exists
	dir := filepath.Dir(l.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create lock directory: %w", err)
	}

	// Try to acquire lock with timeout
	deadline := time.Now().Add(lockTimeout)
	for {
		// Try to create lock file exclusively
		// O_CREATE|O_EXCL ensures the file is created only if it doesn't exist
		f, err := os.OpenFile(l.path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
		if err == nil {
			l.file = f
			return nil // Lock acquired
		}

		// If file exists, check if it's stale (older than lockTimeout)
		// This helps recover from crashes that leave lock files behind
		if info, statErr := os.Stat(l.path); statErr == nil {
			if time.Since(info.ModTime()) > lockTimeout {
				// Stale lock, try to remove it
				_ = os.Remove(l.path)
			}
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for lock on %s", l.path)
		}

		// Wait a bit before retrying
		time.Sleep(50 * time.Millisecond)
	}
}

// Unlock releases the lock
func (l *FileLock) Unlock() error {
	if l.file == nil {
		return nil
	}

	closeErr := l.file.Close()
	l.file = nil

	// Remove the lock file
	removeErr := os.Remove(l.path)

	if closeErr != nil {
		return fmt.Errorf("failed to close lock file: %w", closeErr)
	}
	if removeErr != nil && !os.IsNotExist(removeErr) {
		return fmt.Errorf("failed to remove lock file: %w", removeErr)
	}

	return nil
}

// WithLock executes a function while holding the lock
func WithLock(path string, fn func() error) error {
	lock := NewFileLock(path)
	if err := lock.Lock(); err != nil {
		return err
	}
	defer func() { _ = lock.Unlock() }()

	return fn()
}
