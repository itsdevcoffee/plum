//go:build unix

package settings

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// FileLock provides advisory file locking for concurrent access protection
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
func (l *FileLock) Lock() error {
	// Ensure directory exists
	dir := filepath.Dir(l.path)
	// #nosec G301 -- Lock directory needs same permissions as settings
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create lock directory: %w", err)
	}

	// Open or create lock file
	// #nosec G304 -- path is derived from settings path, not user input
	// #nosec G302 -- Lock file permissions are intentional (readable for debugging)
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("failed to open lock file: %w", err)
	}
	l.file = f

	// Try to acquire lock with timeout
	deadline := time.Now().Add(lockTimeout)
	for {
		err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return nil // Lock acquired
		}

		if time.Now().After(deadline) {
			_ = f.Close()
			l.file = nil
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

	err := syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	closeErr := l.file.Close()
	l.file = nil

	if err != nil {
		return fmt.Errorf("failed to unlock: %w", err)
	}
	if closeErr != nil {
		return fmt.Errorf("failed to close lock file: %w", closeErr)
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
