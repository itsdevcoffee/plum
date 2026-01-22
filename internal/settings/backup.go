package settings

import (
	"fmt"
	"os"
)

// ensureBackup creates a one-time backup of settings.json before first modification.
// This is a safety feature - if backup fails, we warn but don't block the write.
// Returns nil if backup already exists, original doesn't exist, or backup succeeds.
func ensureBackup(path string) error {
	backupPath := path + ".backup-plum"

	// Check if backup already exists - nothing to do
	if _, err := os.Stat(backupPath); err == nil {
		return nil
	}

	// Check if original exists
	// #nosec G304 -- path is validated via ScopePath which uses known config dirs
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No original file to backup
		}
		return fmt.Errorf("failed to read original for backup: %w", err)
	}

	// Create backup with secure permissions
	if err := os.WriteFile(backupPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write backup: %w", err)
	}

	return nil
}
