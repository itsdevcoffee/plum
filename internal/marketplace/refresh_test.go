package marketplace

import (
	"os"
	"path/filepath"
	"testing"
)

// TestClearCache verifies cache clearing functionality
func TestClearCache(t *testing.T) {
	t.Run("clear existing cache", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalPlumCacheDir := plumCacheDir
		plumCacheDir = func() (string, error) {
			return tmpDir, nil
		}
		defer func() { plumCacheDir = originalPlumCacheDir }()

		// Create some cache files
		err := os.MkdirAll(tmpDir, 0700)
		if err != nil {
			t.Fatalf("Failed to create cache dir: %v", err)
		}

		testFile := filepath.Join(tmpDir, "test-cache.json")
		err = os.WriteFile(testFile, []byte("test data"), 0600)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Verify file exists
		if _, err := os.Stat(testFile); os.IsNotExist(err) {
			t.Fatal("Test file should exist before clearing")
		}

		// Clear cache
		err = ClearCache()
		if err != nil {
			t.Fatalf("ClearCache failed: %v", err)
		}

		// Verify cache directory is gone
		if _, err := os.Stat(tmpDir); !os.IsNotExist(err) {
			t.Error("Cache directory should be removed")
		}
	})

	t.Run("clear non-existent cache", func(t *testing.T) {
		tmpDir := filepath.Join(t.TempDir(), "non-existent-cache")
		originalPlumCacheDir := plumCacheDir
		plumCacheDir = func() (string, error) {
			return tmpDir, nil
		}
		defer func() { plumCacheDir = originalPlumCacheDir }()

		// Clear non-existent cache (should not error)
		err := ClearCache()
		if err != nil {
			t.Errorf("ClearCache should not error on non-existent cache: %v", err)
		}
	})

	t.Run("cache dir error handling", func(t *testing.T) {
		originalPlumCacheDir := plumCacheDir
		plumCacheDir = func() (string, error) {
			return "", os.ErrPermission
		}
		defer func() { plumCacheDir = originalPlumCacheDir }()

		err := ClearCache()
		if err == nil {
			t.Error("ClearCache should return error when PlumCacheDir fails")
		}
	})
}
