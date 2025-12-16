package marketplace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateMarketplaceName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid names
		{"valid simple", "my-marketplace", false},
		{"valid with underscore", "test_marketplace", false},
		{"valid with dot", "marketplace.test", false},
		{"valid with numbers", "marketplace123", false},
		{"valid mixed", "My-Market_123.test", false},

		// Invalid names
		{"empty", "", true},
		{"path traversal double-dot", "../etc/passwd", true},
		{"path traversal in middle", "foo/../bar", true},
		{"unix path separator", "foo/bar", true},
		{"windows path separator", "foo\\bar", true},
		{"absolute unix path", "/etc/passwd", true},
		{"absolute windows path", "C:\\Windows", true},
		{"unicode", "marketplace™", true},
		{"space", "my marketplace", true},
		{"special char @", "marketplace@test", true},
		{"special char #", "marketplace#test", true},
		{"too long", strings.Repeat("a", MaxMarketplaceNameLength+1), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMarketplaceName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMarketplaceName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestSaveToCache_AtomicWrite(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := t.TempDir()

	// Override plumCacheDir for testing
	original := plumCacheDir
	plumCacheDir = func() (string, error) {
		return tmpDir, nil
	}
	defer func() { plumCacheDir = original }()

	// Create test manifest
	manifest := &MarketplaceManifest{
		Name: "test-marketplace",
		Owner: MarketplaceOwner{
			Name:  "Test Owner",
			Email: "test@example.com",
		},
		Metadata: MarketplaceMetadata{
			Description: "Test marketplace",
			Version:     "1.0.0",
		},
	}

	marketplaceName := "test-marketplace"

	// Save to cache
	err := SaveToCache(marketplaceName, manifest)
	if err != nil {
		t.Fatalf("SaveToCache failed: %v", err)
	}

	// Verify file exists
	cachePath := filepath.Join(tmpDir, marketplaceName+".json")
	info, err := os.Stat(cachePath)
	if err != nil {
		t.Fatalf("Cache file not created: %v", err)
	}

	// Verify permissions are 0600 (user-only read/write)
	if info.Mode().Perm() != 0600 {
		t.Errorf("Expected 0600 permissions, got %o", info.Mode().Perm())
	}

	// Load from cache and verify
	loaded, err := LoadFromCache(marketplaceName)
	if err != nil {
		t.Fatalf("LoadFromCache failed: %v", err)
	}

	if loaded == nil {
		t.Fatal("LoadFromCache returned nil")
	}

	if loaded.Name != manifest.Name {
		t.Errorf("Expected name %q, got %q", manifest.Name, loaded.Name)
	}
}

func TestSaveToCache_InvalidName(t *testing.T) {
	tmpDir := t.TempDir()

	original := plumCacheDir
	plumCacheDir = func() (string, error) {
		return tmpDir, nil
	}
	defer func() { plumCacheDir = original }()

	manifest := &MarketplaceManifest{Name: "test"}

	// Test path traversal rejection
	err := SaveToCache("../etc/passwd", manifest)
	if err == nil {
		t.Error("SaveToCache should reject path traversal")
	}

	// Test path separator rejection
	err = SaveToCache("foo/bar", manifest)
	if err == nil {
		t.Error("SaveToCache should reject path separator")
	}
}

func TestLoadFromCache_InvalidName(t *testing.T) {
	tmpDir := t.TempDir()

	original := plumCacheDir
	plumCacheDir = func() (string, error) {
		return tmpDir, nil
	}
	defer func() { plumCacheDir = original }()

	// Test path traversal rejection
	_, err := LoadFromCache("../etc/passwd")
	if err == nil {
		t.Error("LoadFromCache should reject path traversal")
	}
}

func TestCacheDirectoryPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")

	original := plumCacheDir
	plumCacheDir = func() (string, error) {
		return cacheDir, nil
	}
	defer func() { plumCacheDir = original }()

	manifest := &MarketplaceManifest{Name: "test"}

	// Save to cache (this should create the directory)
	err := SaveToCache("test", manifest)
	if err != nil {
		t.Fatalf("SaveToCache failed: %v", err)
	}

	// Verify directory permissions are 0700
	info, err := os.Stat(cacheDir)
	if err != nil {
		t.Fatalf("Cache directory not created: %v", err)
	}

	if info.Mode().Perm() != 0700 {
		t.Errorf("Expected cache directory permissions 0700, got %o", info.Mode().Perm())
	}
}
